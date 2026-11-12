package config

import (
	"strings"
	"testing"
)

// TestThemes_TitreDuBandeau verrouille le titre du bandeau du CV choisi par Alexis le 09/11/2026
// (dossier pf_71da886381ac) : « AI Orchestration Engineer — Technomancer » pour les thèmes ai et
// fullstack, dans chaque langue où le titre a une version propre (NameFor part dans le PDF). Les
// autres thèmes gardent leur intitulé métier, sans la marque : c'était le second choix du dossier.
func TestThemes_TitreDuBandeau(t *testing.T) {
	attendu := map[string]map[string]string{
		"ai":        {"fr": "AI Orchestration Engineer — Technomancer", "en": "AI Orchestration Engineer — Technomancer", "zh": "AI 编排工程师 — Technomancer"},
		"fullstack": {"fr": "AI Orchestration Engineer — Technomancer", "en": "AI Orchestration Engineer — Technomancer", "zh": "AI 编排工程师 — Technomancer"},
	}
	themes := GetAvailableThemes()
	for id, langues := range attendu {
		for lang, titre := range langues {
			if got := themes[id].NameFor(lang); got != titre {
				t.Errorf("thème %q, langue %q : %q, attendu %q", id, lang, got, titre)
			}
		}
	}
	for id, th := range themes {
		if _, marque := attendu[id]; marque {
			continue
		}
		for _, lang := range []string{"fr", "en", "zh"} {
			if strings.Contains(th.NameFor(lang), "Technomancer") {
				t.Errorf("thème %q, langue %q : %q porte la marque, réservée à ai et fullstack", id, lang, th.NameFor(lang))
			}
		}
	}
}
