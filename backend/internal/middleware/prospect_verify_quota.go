package middleware

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// ProspectVerifyQuota — plafond quotidien des vérifications Composer (/admin/compose/verify) d'un
// prospecteur.
//
// POURQUOI un compteur DISTINCT du budget IA principal (prospectDailyKey) : la vérification
// (traduction de contrôle + checklist, modèle auxiliaire bon marché) part après CHAQUE tour de
// l'agent. La décompter du même budget que les générations diviserait par deux, en pratique, ce
// qu'un prospecteur peut produire. Mais elle appelle le modèle aux frais du propriétaire : elle ne
// peut pas rester illimitée. D'où un plafond propre, fixé par celui qui lance le service (main.go).
//
// COMMENT : INCR atomique sur une clé par prospecteur, TTL posé jusqu'à minuit au premier appel du
// jour. Le compteur est consommé AVANT l'appel modèle : un appel qui échoue compte quand même —
// c'est voulu, une boucle d'erreurs coûte aussi.
//
// Échec FERMÉ partout : plafond ≤ 0, Redis absent ou en erreur ⇒ refus. Un quota qui laisse passer
// quand il ne sait pas compter n'est pas un quota.
type ProspectVerifyQuota struct {
	rdb *redis.Client
	max int
}

// NewProspectVerifyQuota construit le quota. max < 0 = plafond levé ; max = 0 = aucune vérification.
func NewProspectVerifyQuota(rdb *redis.Client, max int) *ProspectVerifyQuota {
	return &ProspectVerifyQuota{rdb: rdb, max: max}
}

// prospectVerifyKey : clé Redis du compteur journalier de vérifications d'un prospecteur.
func prospectVerifyKey(id string) string { return "ratelimit:prospect:" + id + ":verify:daily" }

// errVerifyQuotaIndisponible : le quota ne peut pas compter (non câblé ou sans Redis).
var errVerifyQuotaIndisponible = errors.New("quota de vérification prospecteur indisponible")

// Consume compte une vérification et dit si elle est autorisée.
// Renvoie (false, nil) quand le plafond est atteint, (false, err) quand on ne peut pas compter.
func (q *ProspectVerifyQuota) Consume(ctx context.Context, prospectID string) (bool, error) {
	if q == nil {
		return false, errVerifyQuotaIndisponible
	}
	// Plafond négatif = plafond LEVÉ, choix explicite de celui qui lance le service
	// (PROSPECT_VERIFY_DAILY_MAX, -1 par défaut depuis le 2026-10-15). Rien n'est compté, Redis
	// n'est pas requis. 0 garde son sens : aucune vérification autorisée.
	if q.max < 0 {
		return true, nil
	}
	if q.rdb == nil {
		return false, errVerifyQuotaIndisponible
	}
	if q.max <= 0 {
		return false, nil
	}
	key := prospectVerifyKey(prospectID)
	n, err := q.rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if n == 1 {
		// Premier appel du jour : la clé expire à minuit (même horloge que les autres compteurs
		// journaliers, cf. IncrementAIRateLimit). Sans TTL elle serait immortelle.
		now := time.Now()
		midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		if err := q.rdb.Expire(ctx, key, midnight.Sub(now)).Err(); err != nil {
			return false, err
		}
	}
	return n <= int64(q.max), nil
}
