package services

import "github.com/anthropics/anthropic-sdk-go"

// QUOI : le prix d'UN appel à la passerelle, poste par poste — entrée hors cache, écriture de cache
// (TTL 5 min ou 1 h), lecture de cache, sortie — chacun à son tarif.
//
// POURQUOI quatre postes et pas « entrée + sortie » : la passerelle pose un cache de prompt d'une
// heure, et son `usage.input_tokens` EXCLUT alors les tokens écrits ou lus en cache (mesuré : 1 454
// d'écriture sur 1 876 tokens d'entrée). Un coût calculé sur `input_tokens` seul en oubliait les trois
// quarts sur un premier appel ; et une écriture 1 h coûte DEUX fois l'entrée, une lecture un dixième :
// un tarif unique « entrée » serait faux dans les deux sens.
//
// SOURCE des tarifs : passerelle LCF, `docs/model-pricing.json` (dépôt LeCodeurFou, origin/main,
// commit a4d56f3 du 2026-10-11) — table en $ par MTok, champs input / output / cacheRead / cacheWrite.
// Sa règle `_regles.cacheWrite` : `cacheWrite` est le tarif d'écriture TTL 5 min, « le TTL 1 h vaut
// 2x » l'entrée. Les gpt-5.6-* y ont un prix catalogue, mais leur accès passe par un forfait (coût
// marginal nul) : ils restent à 0 ici, cf. leur entrée.

// TokenUsage — l'usage facturable d'un appel, tel que la passerelle le ventile.
type TokenUsage struct {
	Input        int // `input_tokens` : entrée HORS cache
	CacheWrite5m int // écriture de cache, TTL 5 min
	CacheWrite1h int // écriture de cache, TTL 1 h (le TTL que pose la passerelle)
	CacheRead    int // `cache_read_input_tokens`
	Output       int
}

// usageOf — lit les quatre postes d'une réponse de la passerelle.
//
// COMMENT l'écriture est ventilée : `cache_creation` porte la part 5 min et la part 1 h. Si la
// passerelle rend un total d'écriture que sa ventilation n'explique pas (ventilation absente ou
// partielle), le reste est imputé au TTL 1 h — c'est celui qu'elle applique, et c'est le plus cher :
// un doute ne doit jamais faire SOUS-estimer un coût ni un budget.
func usageOf(u anthropic.Usage) TokenUsage {
	w5m, w1h := int(u.CacheCreation.Ephemeral5mInputTokens), int(u.CacheCreation.Ephemeral1hInputTokens)
	if rest := int(u.CacheCreationInputTokens) - w5m - w1h; rest > 0 {
		w1h += rest
	}
	return TokenUsage{
		Input:        int(u.InputTokens),
		CacheWrite5m: w5m,
		CacheWrite1h: w1h,
		CacheRead:    int(u.CacheReadInputTokens),
		Output:       int(u.OutputTokens),
	}
}

// CacheWrite — tokens écrits en cache, tous TTL confondus.
func (u TokenUsage) CacheWrite() int { return u.CacheWrite5m + u.CacheWrite1h }

// Prompt — TOUTE l'entrée facturée : hors cache + écriture + lecture. C'est ce qu'un budget compte,
// et ce sur quoi porte un seuil de palier (Haiku 5.5).
func (u TokenUsage) Prompt() int { return u.Input + u.CacheWrite() + u.CacheRead }

// modelRates — $ par MTok. `cacheWrite` = écriture TTL 5 min ; le TTL 1 h vaut 2 × `input` (règle de
// la source, cf. en-tête), il n'a donc pas de colonne propre.
type modelRates struct{ input, output, cacheRead, cacheWrite float64 }

// cost — prix d'un usage à ces tarifs.
func (r modelRates) cost(u TokenUsage) float64 {
	return (float64(u.Input)*r.input +
		float64(u.CacheWrite5m)*r.cacheWrite +
		float64(u.CacheWrite1h)*2*r.input +
		float64(u.CacheRead)*r.cacheRead +
		float64(u.Output)*r.output) / 1_000_000
}

var (
	// Haiku 5.5 — palier « prompts jusqu'à 100 000 tokens » de la source. AU-DELÀ, TOUTE la requête
	// passe au palier ×5 (haiku55Long). Alias `claude-haiku-latest` / `haiku` = la 5.5 (mesuré le
	// 2026-10-16).
	haiku55     = modelRates{input: 0.10, output: 0.50, cacheRead: 0.01, cacheWrite: 0.125}
	haiku55Long = modelRates{input: 0.50, output: 2.50, cacheRead: 0.05, cacheWrite: 0.625}
	// Haiku 4.5 — historique des métriques. `cacheWrite` 1,00 tel que la source le cote (et non
	// 1,25 × l'entrée) : on recopie la table, on ne la « corrige » pas.
	haiku45 = modelRates{input: 1.00, output: 5.00, cacheRead: 0.10, cacheWrite: 1.00}
	// Opus 5 et 4.x — identiques dans la source. `claude-opus-4-5` n'y figure pas : même famille, même
	// prix d'entrée/sortie que ses voisins, rangé avec eux plutôt que laissé au `default`.
	opus = modelRates{input: 5.00, output: 25.00, cacheRead: 0.50, cacheWrite: 6.25}
	// Sonnet 5.5 — CORRIGÉ dans la source le 2026-10-09 : palier tier_2_10 du bundle Claude Code
	// (2/10, écriture 2,50) ; lecture baissée à 0,10 par Anthropic le 2026-10-07. L'ancienne entrée
	// 3/15 recopiait de mauvais chiffres.
	sonnet55 = modelRates{input: 2.00, output: 10.00, cacheRead: 0.10, cacheWrite: 2.50}
	// Sonnet 5 — plein tarif catalogue (la source écarte le tarif d'introduction échu). Sert aussi de
	// `default` : même prix que les Sonnet 4.x de la source.
	sonnet5 = modelRates{input: 3.00, output: 15.00, cacheRead: 0.30, cacheWrite: 3.75}
	fable5  = modelRates{input: 10.00, output: 50.00, cacheRead: 1.00, cacheWrite: 12.50}
	// forfait — modèles de repli atteints via le compte d'abonnement branché sur la passerelle : coût
	// marginal au token NUL. ⚠️ Affirmation sur le MODE D'ACCÈS, pas sur le modèle : si la passerelle
	// bascule sur une clé API facturée, ce 0 devient faux (prix catalogue dans la source).
	forfait = modelRates{}
)

// haiku55Threshold — au-delà de ce prompt TOTAL (cache compris), Haiku 5.5 facture toute la requête
// au palier ×5. Seuil strict : 100 000 pile reste au palier bas.
const haiku55Threshold = 100_000

// ratesFor — les tarifs d'un identifiant de modèle pour CET usage (le palier Haiku 5.5 dépend de la
// taille du prompt).
//
// Chaque alias réellement envoyé à la passerelle est listé : un identifiant absent retombe en silence
// sur le `default` (tarif Sonnet) et les stats mentent sans erreur (cf. ai_pricing_test.go).
func ratesFor(model string, u TokenUsage) modelRates {
	switch model {
	case "claude-haiku-5-5", "claude-haiku-latest", "haiku":
		if u.Prompt() > haiku55Threshold {
			return haiku55Long
		}
		return haiku55
	case "claude-haiku-4-5-20251001", "claude-haiku-4-5":
		return haiku45
	case "claude-opus-5", "claude-opus-latest", "opus", "claude-opus-4-8", "claude-opus-4-7", "claude-opus-4-6", "claude-opus-4-5":
		return opus
	case "claude-sonnet-5-5":
		return sonnet55
	case "claude-sonnet-5":
		return sonnet5
	case "claude-fable-5", "claude-fable-latest", "fable":
		return fable5
	case "gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna":
		return forfait
	default:
		return sonnet5
	}
}

// calculateModelCost — coût en $ d'un appel, selon le modèle effectif (tous fournisseurs : la
// passerelle les sert sous une seule API, donc une seule table).
func calculateModelCost(u TokenUsage, model string) float64 {
	return ratesFor(model, u).cost(u)
}
