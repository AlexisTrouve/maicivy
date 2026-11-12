package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// AIBudget : les plafonds IA VISITEURS (ceux d'AIRateLimit) lus et consommés HORS d'une requête.
//
// POURQUOI : AIRateLimit est un middleware Fiber — il garde une route. Une génération lancée EN
// FOND (le CV adapté préparé après un chat ou un dossier vivant, cf. services/cv_context.go) n'a pas
// de route à garder, mais elle coûte la même chose qu'un POST /cv/generate. Elle doit donc tirer sur
// le MÊME budget : mêmes clés Redis, mêmes plafonds. Sans ça, un visiteur obtiendrait des
// générations hors quota en passant par le chat.
//
// COMMENT : les clés sont produites par les mêmes fonctions que celles du middleware
// (aiSessionDailyKey, aiIPDailyKey, globalDailyKey) — une divergence de format ferait compter deux
// budgets distincts sans la moindre erreur. Le cooldown et le verrou in-flight ne sont PAS repris :
// ils protègent une route contre le double-clic et la concurrence, ce que le single-flight du
// service fait déjà ; poser le cooldown ici bloquerait, deux minutes durant, la lettre que le
// visiteur demanderait juste après avoir parlé au chat — un effet de bord invisible pour lui.
type AIBudget struct {
	Redis          *redis.Client
	KeyPrefix      string // vide → "ai", le budget partagé CV/lettres/messages
	MaxPerDay      int    // plafond journalier par session
	MaxPerDayPerIP int    // plafond journalier par IP
	GlobalDailyMax int    // circuit-breaker global (0 = désactivé)
}

func (b AIBudget) prefix() string {
	if b.KeyPrefix == "" {
		return "ai"
	}
	return b.KeyPrefix
}

// Allow : true si la session, l'IP et le compteur global ont encore de la marge.
// Erreur Redis autre que « clé absente » → refus : une génération en fond n'est jamais urgente, et
// la lancer sans pouvoir compter reviendrait à ouvrir le robinet à l'aveugle.
func (b AIBudget) Allow(ctx context.Context, sessionID, ip string) bool {
	checks := []struct {
		key string
		max int
	}{
		{aiSessionDailyKey(b.prefix(), sessionID), b.MaxPerDay},
		{aiIPDailyKey(b.prefix(), ip), b.MaxPerDayPerIP},
	}
	if b.GlobalDailyMax > 0 {
		checks = append(checks, struct {
			key string
			max int
		}{globalDailyKey, b.GlobalDailyMax})
	}
	for _, c := range checks {
		n, err := b.Redis.Get(ctx, c.key).Int()
		if err != nil && err != redis.Nil {
			return false
		}
		if n >= c.max {
			return false
		}
	}
	return true
}

// Consume : compte UNE génération réussie (session, IP, global), TTL à minuit comme
// IncrementAIRateLimit — les deux chemins remettent les compteurs à zéro au même instant.
func (b AIBudget) Consume(ctx context.Context, sessionID, ip string) error {
	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	ttl := midnight.Sub(now)
	for _, key := range []string{aiSessionDailyKey(b.prefix(), sessionID), aiIPDailyKey(b.prefix(), ip), globalDailyKey} {
		n, err := b.Redis.Incr(ctx, key).Result()
		if err != nil {
			return fmt.Errorf("ai budget: incr %s: %w", key, err)
		}
		if n == 1 {
			b.Redis.Expire(ctx, key, ttl)
		}
	}
	return nil
}
