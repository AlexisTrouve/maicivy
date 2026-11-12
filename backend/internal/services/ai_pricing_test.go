package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// QUOI : verrouille la tarification de CHAQUE identifiant de modèle réellement envoyé au proxy,
// alias compris, sur les QUATRE postes facturés (entrée, écriture de cache, lecture, sortie).
//
// POURQUOI : ratesFor tombe sur un `default` (tarif Sonnet) pour tout modèle inconnu. Un alias non
// listé ne provoque donc AUCUNE erreur — il produit juste un coût faux, silencieusement, et les stats
// admin racontent n'importe quoi pendant des mois. C'est exactement ce qui est arrivé avec les anciens
// tarifs Opus ($15/$75 alors que le vrai est $5/$25).
//
// Toute nouvelle valeur possible d'OWNER_MODEL, de CLAUDE_MODEL, de CV_MODEL, du modèle du dossier
// vivant ou d'AI_FALLBACK_MODEL doit être ajoutée ici EN MÊME TEMPS que dans la table, sinon le test
// échoue — c'est le but. Valeurs attendues : LeCodeurFou docs/model-pricing.json (cf. ai_pricing.go).

const unMTok = 1_000_000

func TestCalculateModelCost_TousLesIdentifiantsUtilises(t *testing.T) {
	for _, tc := range []struct {
		model                          string
		in, out, cacheRead, cacheWrite float64 // $ par MTok attendus (cacheWrite = TTL 5 min)
	}{
		// Haiku 5.5 au palier bas (≤ 100k de prompt) ; le palier haut est verrouillé par
		// TestCalculateModelCost_Haiku55Paliers.
		{"claude-haiku-5-5", 0.10, 0.50, 0.01, 0.125},
		{"claude-haiku-latest", 0.10, 0.50, 0.01, 0.125}, // alias proxy — défaut visiteurs
		{"haiku", 0.10, 0.50, 0.01, 0.125},               // alias court du proxy
		// Haiku 4.5 (historique) — écriture cotée 1,00 par la source.
		{"claude-haiku-4-5-20251001", 1, 5, 0.10, 1.00},
		{"claude-haiku-4-5", 1, 5, 0.10, 1.00},
		// Opus
		{"claude-opus-5", 5, 25, 0.50, 6.25},
		{"claude-opus-latest", 5, 25, 0.50, 6.25}, // alias proxy
		{"opus", 5, 25, 0.50, 6.25},
		{"claude-opus-4-8", 5, 25, 0.50, 6.25},
		{"claude-opus-4-6", 5, 25, 0.50, 6.25},
		// Sonnet 5.5 (CV_MODEL, Dessinateur du dossier vivant) : 2/10, corrigé dans la source le 2026-10-09.
		{"claude-sonnet-5-5", 2, 10, 0.10, 2.50},
		{"claude-sonnet-5", 3, 15, 0.30, 3.75},
		{"claude-fable-5", 10, 50, 1.00, 12.50},
		// Modèles de repli (autre fournisseur, via la passerelle) : accès au forfait → coût marginal
		// nul sur TOUS les postes. L'entrée DOIT exister : sans elle, un repli se facturerait au tarif
		// Sonnet du défaut, entièrement inventé.
		{"gpt-5.6-terra", 0, 0, 0, 0},
		{"gpt-5.6-sol", 0, 0, 0, 0},
		{"gpt-5.6-luna", 0, 0, 0, 0},
	} {
		t.Run(tc.model, func(t *testing.T) {
			// Un poste à la fois, à 10 000 tokens (sous le palier Haiku) : chaque tarif se lit isolément.
			const n = 10_000
			poste := func(u TokenUsage) float64 { return calculateModelCost(u, tc.model) * unMTok / n }
			msg := "tarif faux pour %s — un alias absent de la table retombe en silence sur le tarif par défaut"
			assert.InDelta(t, tc.in, poste(TokenUsage{Input: n}), 1e-9, msg, tc.model)
			assert.InDelta(t, tc.out, poste(TokenUsage{Output: n}), 1e-9, msg, tc.model)
			assert.InDelta(t, tc.cacheRead, poste(TokenUsage{CacheRead: n}), 1e-9, msg, tc.model)
			assert.InDelta(t, tc.cacheWrite, poste(TokenUsage{CacheWrite5m: n}), 1e-9, msg, tc.model)
			// Écriture TTL 1 h = 2 × l'entrée (règle de la source) — le TTL que pose la passerelle.
			assert.InDelta(t, 2*tc.in, poste(TokenUsage{CacheWrite1h: n}), 1e-9, msg, tc.model)
		})
	}
}

// TestCalculateModelCost_Haiku55Paliers : sous 100 000 tokens de prompt, Haiku 5.5 coûte $0,10/$0,50 ;
// au-delà, TOUTE la requête passe au palier ×5 (pas seulement l'excédent). Le seuil est strict et porte
// sur le prompt TOTAL, cache compris : 100 000 pile reste au palier bas.
func TestCalculateModelCost_Haiku55Paliers(t *testing.T) {
	// 100k en entrée + 10k en sortie, palier bas : 0,1×0,10 + 0,01×0,50 = 0,015 $
	assert.InDelta(t, 0.015, calculateModelCost(TokenUsage{Input: 100_000, Output: 10_000}, "claude-haiku-5-5"), 1e-9)
	// 100 001 en entrée : tout bascule ×5 → 0,100001×0,50 + 0,01×2,50 ≈ 0,075 $
	assert.InDelta(t, 0.0750005, calculateModelCost(TokenUsage{Input: 100_001, Output: 10_000}, "claude-haiku-5-5"), 1e-9)
	// Même prompt de 100 001, dont 99 001 LUS en cache : le palier haut s'applique aussi (0,50 / 0,05 / 2,50).
	assert.InDelta(t, (1000*0.50+99_001*0.05+10_000*2.50)/1e6,
		calculateModelCost(TokenUsage{Input: 1000, CacheRead: 99_001, Output: 10_000}, "claude-haiku-5-5"), 1e-12)
}

// Le défaut existe pour ne pas planter sur un modèle inattendu, mais il ne doit JAMAIS s'appliquer à
// un identifiant qu'on utilise vraiment. Ce test documente sa valeur pour que la différence saute aux
// yeux si un alias y retombe par accident.
func TestCalculateModelCost_DefautSonnet(t *testing.T) {
	got := calculateModelCost(TokenUsage{Input: unMTok, Output: unMTok}, "modele-inconnu-xyz")

	assert.InDelta(t, 18.0, got, 0.001, "défaut = tarif Sonnet ($3/$15)")
}
