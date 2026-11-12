package services

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// ============================================================================
// swrStore — cache Redis stale-while-revalidate, commun au portfolio et au blog.
// ----------------------------------------------------------------------------
// QUOI : sert une valeur depuis Redis ; si elle est périmée, la sert QUAND MÊME et la rafraîchit
// en fond ; ne fait attendre l'appelant que si rien n'est en cache.
//
// POURQUOI : panne du 2026-09-27 — maiProFiles saturé, et les pages qui le lisaient en direct
// tombaient en erreur. La donnée qu'il sert (curation, articles) change quelques fois par jour au
// plus : une version d'il y a une heure vaut toujours mieux qu'une page d'erreur. Le mécanisme
// vivait d'abord dans PortfolioCache ; il est extrait ici pour que le blog n'en soit pas une copie.
//
// COMMENT :
//   - blob JSON {fetchedAt, data} par clé ;
//   - frais (< fraicheur) → servi tel quel ; périmé → servi + rafraîchi en fond, un seul
//     rafraîchissement par clé à la fois (bgRefresher) ; absent → fetch synchrone ;
//   - seules les réponses RÉUSSIES sont stockées : une erreur n'est jamais mise en cache ;
//   - `estAbsent(err)` vrai au rafraîchissement → la ressource a disparu en amont, la clé est
//     supprimée (sinon elle resterait servie pour toujours) ;
//   - `ttl` : 0 = jamais d'expiration (clés en nombre fini) ; > 0 quand les clés dérivent d'une
//     entrée libre (numéro de page) — une clé non visitée pendant `ttl` disparaît, une clé vivante
//     est réécrite à chaque rafraîchissement et ne meurt donc pas.
//   - redis nil → passe-plat pur (tests, Redis absent).
//
// ============================================================================
// limiteFondMaiProFiles — plafond GLOBAL des lectures en fond vers maiProFiles, partagé par tous les
// caches qui le lisent (portfolio, blog).
//
// POURQUOI : mesuré le 2026-09-27 — au démarrage, le préchauffage du portfolio a déclenché 76
// rafraîchissements SIMULTANÉS. maiProFiles tourne sur UN worker uvicorn, et une requête abandonnée
// par notre client (timeout 8 s) continue d'être traitée de son côté : la file a saturé maiProFiles
// pendant plusieurs minutes (/book 17 s, /blog/posts > 60 s), puis tout est revenu à 0,5 s une fois
// la file écoulée, sans redémarrage. Le cache censé protéger le site d'un amont lent le rendait lent.
// COMMENT : un sémaphore (canal bufferisé). Au-delà de 2, les rafraîchissements attendent CHEZ NOUS,
// dans leur goroutine, au lieu de s'empiler sur maiProFiles. Personne n'attend une lecture en fond,
// donc la file peut être longue sans coût visible.
var limiteFondMaiProFiles = make(chan struct{}, 2)

type swrStore struct {
	redis     *redis.Client
	nom       string // préfixe des logs : "portfolio", "blog"
	fraicheur time.Duration
	ttl       time.Duration
	estAbsent func(error) bool
	// limite — sémaphore des lectures en fond (nil = illimité). Partagé entre les caches d'un même
	// amont : c'est l'amont qu'on protège, pas chaque cache.
	limite chan struct{}
	// refreshers — un bgRefresher PAR CLÉ : le rafraîchissement d'une page ne doit pas bloquer
	// celui d'une autre.
	refreshers sync.Map // clé → *bgRefresher
}

// swrBlob — forme stockée dans Redis. `Data` reste brut ; le décodage typé se fait à la lecture.
type swrBlob struct {
	FetchedAt time.Time       `json:"fetchedAt"`
	Data      json.RawMessage `json:"data"`
}

// lire — voir l'en-tête. `fetch` rend la valeur typée depuis l'amont.
func (s *swrStore) lire(ctx context.Context, cle string, out interface{}, fetch func(context.Context) (interface{}, error)) error {
	if s.redis == nil {
		v, err := fetch(ctx)
		if err != nil {
			return err
		}
		return recopier(v, out)
	}

	if blob, ok := s.charger(ctx, cle); ok {
		if err := json.Unmarshal(blob.Data, out); err == nil {
			if time.Since(blob.FetchedAt) >= s.fraicheur {
				s.rafraichirEnFond(cle, fetch)
			}
			return nil
		}
		// Blob illisible (forme changée sans bump de clé) : traité comme absent.
		log.Warn().Str("cle", cle).Msgf("%s: blob de cache illisible, refetch", s.nom)
	}

	// Rien en cache : le seul cas où l'appelant attend l'amont.
	v, err := fetch(ctx)
	if err != nil {
		return err
	}
	s.stocker(ctx, cle, v)
	return recopier(v, out)
}

// lireSansAttendre — comme `lire`, sauf quand rien n'est en cache : au lieu de faire attendre
// l'appelant, la construction part EN FOND et l'appelant reçoit `false` (rien à servir encore).
//
// POURQUOI : pour une donnée facultative dont la construction est longue (les icônes des projets :
// une lecture Gitea par dépôt, puis le téléchargement de chaque icône), le premier visiteur ne doit
// pas payer la construction — il voit le repli (monogrammes), les suivants la donnée.
// COMMENT : redis nil → passe-plat synchrone, comme `lire` (tests, Redis absent).
func (s *swrStore) lireSansAttendre(ctx context.Context, cle string, out interface{}, fetch func(context.Context) (interface{}, error)) (bool, error) {
	if s.redis == nil {
		v, err := fetch(ctx)
		if err != nil {
			return false, err
		}
		return true, recopier(v, out)
	}
	if blob, ok := s.charger(ctx, cle); ok {
		if err := json.Unmarshal(blob.Data, out); err == nil {
			if time.Since(blob.FetchedAt) >= s.fraicheur {
				s.rafraichirEnFond(cle, fetch)
			}
			return true, nil
		}
	}
	s.rafraichirEnFond(cle, fetch)
	return false, nil
}

// rafraichirEnFond — contexte détaché : celui de la requête est mort une fois le périmé rendu.
func (s *swrStore) rafraichirEnFond(cle string, fetch func(context.Context) (interface{}, error)) {
	r, _ := s.refreshers.LoadOrStore(cle, &bgRefresher{})
	r.(*bgRefresher).trigger(func(bg context.Context) {
		// Attente d'une place AVANT d'appeler l'amont (cf. limiteFondMaiProFiles). Le flag
		// single-flight de la clé reste levé pendant l'attente : pas de doublon en file.
		if s.limite != nil {
			s.limite <- struct{}{}
			defer func() { <-s.limite }()
		}
		v, err := fetch(bg)
		switch {
		case err != nil && s.estAbsent != nil && s.estAbsent(err):
			s.redis.Del(bg, cle)
			log.Info().Str("cle", cle).Msgf("%s: ressource disparue en amont, sortie du cache", s.nom)
		case err != nil:
			// Amont lent ou tombé : on garde le périmé, c'est tout l'objet du cache.
			log.Warn().Err(err).Str("cle", cle).Msgf("%s: rafraîchissement en fond échoué, version en cache conservée", s.nom)
		default:
			s.stocker(bg, cle, v)
		}
	})
}

func (s *swrStore) charger(ctx context.Context, cle string) (*swrBlob, bool) {
	brut, err := s.redis.Get(ctx, cle).Bytes()
	if err != nil {
		return nil, false
	}
	var blob swrBlob
	if err := json.Unmarshal(brut, &blob); err != nil {
		return nil, false
	}
	return &blob, true
}

func (s *swrStore) stocker(ctx context.Context, cle string, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	brut, err := json.Marshal(swrBlob{FetchedAt: time.Now(), Data: data})
	if err != nil {
		return
	}
	if err := s.redis.Set(ctx, cle, brut, s.ttl).Err(); err != nil {
		log.Warn().Err(err).Str("cle", cle).Msgf("%s: écriture du cache échouée", s.nom)
	}
}

// supprimerPrefixe — efface toutes les clés commençant par `prefixe` (SCAN, jamais KEYS : KEYS
// bloque Redis sur toute la base). Sert à l'invalidation après une écriture.
func (s *swrStore) supprimerPrefixe(ctx context.Context, prefixe string) {
	if s.redis == nil {
		return
	}
	iter := s.redis.Scan(ctx, 0, prefixe+"*", 200).Iterator()
	var cles []string
	for iter.Next(ctx) {
		cles = append(cles, iter.Val())
	}
	if err := iter.Err(); err != nil {
		log.Warn().Err(err).Msgf("%s: parcours des clés à invalider échoué", s.nom)
		return
	}
	if len(cles) > 0 {
		s.redis.Del(ctx, cles...)
	}
}

// recopier — passe de la valeur typée rendue par l'amont à la cible de l'appelant.
func recopier(v, out interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}
