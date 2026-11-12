package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// CV adapté au besoin d'un visiteur — préparé EN FOND, lu par la page /cv.
//
// QUOI : quand un visiteur a dit ce qu'il cherche (chat public via l'outil note_visitor_need, ou
// besoin saisi pour un dossier vivant), on garde un RÉSUMÉ de ce besoin pour sa session (jamais la
// conversation), et on lance une génération de CV adapté (même moteur que POST /cv/generate). La page
// /cv l'affiche quand il est prêt ; le PDF se rend à la demande depuis le JSON stocké.
//
// POURQUOI en fond : la génération coûte ~14 s d'IA (mesuré le 09/11). Lancée au moment où le besoin
// est exprimé, elle est souvent finie quand le visiteur ouvre /cv — et /cv n'attend jamais : il montre
// toujours d'abord le CV standard.
//
// COMMENT (Redis, tout expire seul) :
//   - cvctx:v1:<session>:need                    → le besoin (résumé, source, langue, empreinte), 24 h.
//   - cvctx:v1:<session>:cv:<empreinte>:<langue> → l'état de la génération ET son résultat.
//     Posé par SETNX à l'état « pending » : c'est le verrou single-flight — une seule génération par
//     session, par besoin et par langue. Le TTL de « pending » (cvContextPendingTTL) dépasse le délai
//     max d'une génération : si le process meurt en vol, l'état s'efface et une lecture suivante la
//     relance. « ready » vit 24 h ; « failed » 15 min (une panne passagère de l'IA ne condamne pas le
//     besoin pour la journée, mais ne la relance pas en boucle non plus).
//   - un budget refusé n'écrit RIEN : la lecture suivante re-vérifie (le quota peut s'être libéré).

const (
	cvContextKeyPrefix   = "cvctx:v1:"
	CVContextTTL         = 24 * time.Hour  // durée de vie du besoin et du CV prêt
	cvContextPendingTTL  = 4 * time.Minute // > cvContextGenTimeout : couvre une génération complète
	cvContextGenTimeout  = 3 * time.Minute
	cvContextFailedTTL   = 15 * time.Minute
	cvContextMaxNeedRune = 1200 // un résumé, pas un document : borne dure côté serveur
	cvContextMinNeedRune = 10
)

// Sources d'un besoin — stables, le front les lit pour son bandeau.
const (
	NeedSourceChat    = "chat"
	NeedSourceDossier = "dossier"
)

// États exposés à la page /cv.
const (
	CVContextNone    = "none"    // aucun besoin connu pour cette session
	CVContextPending = "pending" // génération en cours
	CVContextReady   = "ready"   // CV adapté prêt
	CVContextFailed  = "failed"  // génération échouée (retentée après cvContextFailedTTL)
	CVContextLimited = "limited" // plafonds IA visiteurs atteints : pas de génération
)

// ErrNeedLength : résumé vide ou trop court pour adapter quoi que ce soit.
var ErrNeedLength = errors.New("cv context: need too short")

// CVContextGenerator : le moteur de génération (CVGenerationService en prod, un faux en test).
type CVContextGenerator interface {
	GenerateDynamicCV(ctx context.Context, offer, lang string) (*AdaptiveCVResponse, error)
}

// CVContextBudget : les plafonds IA visiteurs (middleware.AIBudget en prod). Interface ici parce que
// le paquet middleware importe déjà services — l'inverse ferait un cycle.
type CVContextBudget interface {
	Allow(ctx context.Context, sessionID, ip string) bool
	Consume(ctx context.Context, sessionID, ip string) error
}

// CVVisitor : l'identité d'un visiteur côté budget — sa session SIGNÉE et son IP.
type CVVisitor struct {
	SessionID string
	IP        string
}

// visitorNeed : ce qui est gardé du besoin. Le texte est un RÉSUMÉ (outil du chat) ou le besoin tel
// que le visiteur l'a saisi pour son dossier (déjà borné à 2 000 caractères), recoupé ici.
type visitorNeed struct {
	Summary string    `json:"summary"`
	Source  string    `json:"source"`
	Lang    string    `json:"lang"`
	Hash    string    `json:"hash"`
	At      time.Time `json:"at"`
}

type cvContextState struct {
	Status string              `json:"status"`
	CV     *AdaptiveCVResponse `json:"cv,omitempty"`
}

// CVContextView : la réponse de GET /cv/context.
type CVContextView struct {
	Status string              `json:"status"`
	Need   string              `json:"need,omitempty"`
	Source string              `json:"source,omitempty"`
	Lang   string              `json:"lang,omitempty"`
	CV     *AdaptiveCVResponse `json:"cv,omitempty"`
}

// CVContextService : voir l'en-tête du fichier.
type CVContextService struct {
	redis     *redis.Client
	generator CVContextGenerator
	budget    CVContextBudget
	l10n      *LocalizationHelper
	// done : appelé à la fin de chaque génération en fond (tests uniquement — synchronisation sans
	// sleep). nil en prod.
	done func()
}

// NewCVContextService : generator nil (IA non configurée) → le service reste lisible mais ne lance rien.
func NewCVContextService(r *redis.Client, generator CVContextGenerator, budget CVContextBudget) *CVContextService {
	return &CVContextService{redis: r, generator: generator, budget: budget, l10n: NewLocalizationHelper()}
}

func needKey(sessionID string) string { return cvContextKeyPrefix + sessionID + ":need" }
func stateKey(sessionID, hash, lang string) string {
	return cvContextKeyPrefix + sessionID + ":cv:" + hash + ":" + lang
}

// normalizeNeed : espaces repliés, borne de longueur en runes (jamais au milieu d'un caractère).
func normalizeNeed(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) > cvContextMaxNeedRune {
		s = string([]rune(s)[:cvContextMaxNeedRune])
	}
	return s
}

// RecordNeed garde le besoin de la session (remplace le précédent) puis lance la génération en fond
// dans sa langue. Un besoin identique à celui déjà gardé ne relance rien (même empreinte → même
// clé d'état, déjà posée).
func (s *CVContextService) RecordNeed(ctx context.Context, v CVVisitor, source, summary, lang string) error {
	summary = normalizeNeed(summary)
	if utf8.RuneCountInString(summary) < cvContextMinNeedRune {
		return ErrNeedLength
	}
	sum := sha256.Sum256([]byte(summary))
	need := visitorNeed{
		Summary: summary,
		Source:  source,
		Lang:    s.l10n.TailoredLanguage(lang),
		Hash:    hex.EncodeToString(sum[:8]),
		At:      time.Now(),
	}
	raw, err := json.Marshal(need)
	if err != nil {
		return err
	}
	if err := s.redis.Set(ctx, needKey(v.SessionID), raw, CVContextTTL).Err(); err != nil {
		return err
	}
	_, err = s.Ensure(ctx, v, need.Lang)
	return err
}

// Ensure : l'état du CV adapté pour la session dans `lang`, en lançant la génération si aucune n'existe
// encore pour ce besoin et cette langue (et si le budget le permet). Appelé par RecordNeed ET par la
// lecture de /cv : un visiteur qui a parlé en français puis ouvre /en/cv obtient sa version anglaise.
func (s *CVContextService) Ensure(ctx context.Context, v CVVisitor, lang string) (CVContextView, error) {
	need, err := s.loadNeed(ctx, v.SessionID)
	if err != nil || need == nil {
		return CVContextView{Status: CVContextNone}, err
	}
	lang = s.l10n.TailoredLanguage(lang)
	view := CVContextView{Need: need.Summary, Source: need.Source, Lang: lang}

	key := stateKey(v.SessionID, need.Hash, lang)
	if st, err := s.loadState(ctx, key); err != nil || st != nil {
		return withState(view, st), err
	}
	if s.generator == nil || !s.budget.Allow(ctx, v.SessionID, v.IP) {
		view.Status = CVContextLimited
		return view, nil
	}

	// Verrou single-flight : seul le gagnant du SETNX lance la génération.
	pending, _ := json.Marshal(cvContextState{Status: CVContextPending})
	won, err := s.redis.SetNX(ctx, key, pending, cvContextPendingTTL).Result()
	if err != nil {
		return view, err
	}
	if !won {
		st, err := s.loadState(ctx, key)
		return withState(view, st), err
	}
	go s.generate(v, key, need.Summary, lang)
	view.Status = CVContextPending
	return view, nil
}

// AdaptedCV : le CV adapté PRÊT pour la session dans `lang` (nil s'il n'existe pas ou pas encore).
func (s *CVContextService) AdaptedCV(ctx context.Context, sessionID, lang string) (*AdaptiveCVResponse, string, error) {
	need, err := s.loadNeed(ctx, sessionID)
	if err != nil || need == nil {
		return nil, "", err
	}
	lang = s.l10n.TailoredLanguage(lang)
	st, err := s.loadState(ctx, stateKey(sessionID, need.Hash, lang))
	if err != nil || st == nil || st.Status != CVContextReady {
		return nil, lang, err
	}
	return st.CV, lang, nil
}

// generate : la génération elle-même, détachée de la requête qui l'a déclenchée (contexte propre,
// borné). Le besoin est présenté au modèle comme tel — et préfixé, ce qui garantit aussi qu'il ne
// commence jamais par « http » (GenerateDynamicCV irait sinon chercher une URL).
func (s *CVContextService) generate(v CVVisitor, key, summary, lang string) {
	if s.done != nil {
		defer s.done()
	}
	ctx, cancel := context.WithTimeout(context.Background(), cvContextGenTimeout)
	defer cancel()

	offer := "Besoin exprimé par un visiteur du site (résumé, pas une offre d'emploi rédigée) :\n" + summary
	cv, err := s.generator.GenerateDynamicCV(ctx, offer, lang)
	st, ttl := cvContextState{Status: CVContextReady, CV: cv}, CVContextTTL
	if err != nil {
		log.Warn().Err(err).Str("lang", lang).Msg("cv context: génération en fond échouée")
		st, ttl = cvContextState{Status: CVContextFailed}, cvContextFailedTTL
	} else if err := s.budget.Consume(ctx, v.SessionID, v.IP); err != nil {
		// Le CV est payé et prêt : on le garde, mais l'écart de comptage doit se voir.
		log.Error().Err(err).Msg("cv context: génération réussie NON comptée dans le budget IA")
	}
	raw, _ := json.Marshal(st)
	if err := s.redis.Set(context.Background(), key, raw, ttl).Err(); err != nil {
		log.Error().Err(err).Msg("cv context: état non enregistré")
	}
}

func (s *CVContextService) loadNeed(ctx context.Context, sessionID string) (*visitorNeed, error) {
	raw, err := s.redis.Get(ctx, needKey(sessionID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var n visitorNeed
	if err := json.Unmarshal(raw, &n); err != nil {
		return nil, err
	}
	return &n, nil
}

func (s *CVContextService) loadState(ctx context.Context, key string) (*cvContextState, error) {
	raw, err := s.redis.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var st cvContextState
	if err := json.Unmarshal(raw, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

// withState complète la vue avec l'état lu (nil → none, cas d'une erreur de lecture).
func withState(view CVContextView, st *cvContextState) CVContextView {
	if st == nil {
		view.Status = CVContextNone
		return view
	}
	view.Status = st.Status
	view.CV = st.CV
	return view
}
