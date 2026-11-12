package services

import "testing"

// TestNormalizeLangKey verrouille la canonicalisation des noms de langage Gitea.
func TestNormalizeLangKey(t *testing.T) {
	cases := map[string]string{
		"Go":         "go",
		"  Go  ":     "go",
		"TypeScript": "typescript",
		"C++":        "c++",
		"C#":         "c#",
	}
	for in, want := range cases {
		if got := normalizeLangKey(in); got != want {
			t.Errorf("normalizeLangKey(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestBuildLangStats vérifie l'agrégation octets→LOC + la fusion des noms qui collapse.
func TestBuildLangStats(t *testing.T) {
	// "Go" et "go " doivent fusionner sur la clé "go" (380 + 380 = 760 octets).
	in := map[string]int{
		"Go":         380,
		"go ":        380,
		"TypeScript": 76,
	}

	resp := buildLangStats(in, nil)

	// Fusion : une seule entrée "go", octets sommés.
	goStat, ok := resp.Languages["go"]
	if !ok {
		t.Fatalf("clé 'go' absente du résultat: %+v", resp.Languages)
	}
	if goStat.Bytes != 760 {
		t.Errorf("go bytes = %d, want 760", goStat.Bytes)
	}
	// LOC dérivée des octets fusionnés : 760 / 38 = 20.
	if goStat.LOC != 20 {
		t.Errorf("go loc = %d, want 20 (760/38)", goStat.LOC)
	}

	tsStat := resp.Languages["typescript"]
	if tsStat.Bytes != 76 || tsStat.LOC != 2 {
		t.Errorf("typescript = %+v, want {bytes:76 loc:2}", tsStat)
	}

	// Totaux
	if resp.TotalBytes != 836 {
		t.Errorf("totalBytes = %d, want 836", resp.TotalBytes)
	}
	if resp.TotalLOC != 22 { // 20 + 2
		t.Errorf("totalLoc = %d, want 22", resp.TotalLOC)
	}
	if resp.Period != "all-time" {
		t.Errorf("period = %q, want all-time", resp.Period)
	}
}

// TestBuildLangStatsEmpty : map vide → réponse vide cohérente (pas de nil panic).
func TestBuildLangStatsEmpty(t *testing.T) {
	resp := buildLangStats(map[string]int{}, nil)
	if resp == nil {
		t.Fatal("réponse nil")
	}
	if len(resp.Languages) != 0 || resp.TotalLOC != 0 || resp.TotalBytes != 0 {
		t.Errorf("attendu vide, got %+v", resp)
	}
	if len(resp.Repos) != 0 {
		t.Errorf("attendu Repos vide, got %+v", resp.Repos)
	}
}

// TestBuildLangStatsPerRepo vérifie la ventilation LOC par repo (octets/38, même diviseur que le
// global) — SOURCE du "LOC par projet" affiché sur /gitstats. Indépendante de la fusion par langage :
// un repo garde sa propre conversion, pas d'agrégation croisée entre repos.
func TestBuildLangStatsPerRepo(t *testing.T) {
	byLang := map[string]int{"Go": 380, "TypeScript": 76}
	repoBytes := map[string]int{
		"drifterra":  380, // 380/38 = 10
		"Melodicode": 76,  // 76/38 = 2
		"empty-repo": 0,   // 0 octet → absent du résultat (pas de LOC = pas d'entrée, pas un zéro bruyant)
	}

	resp := buildLangStats(byLang, repoBytes)

	if got := resp.Repos["drifterra"]; got != 10 {
		t.Errorf("Repos[drifterra] = %d, want 10 (380/38)", got)
	}
	if got := resp.Repos["Melodicode"]; got != 2 {
		t.Errorf("Repos[Melodicode] = %d, want 2 (76/38)", got)
	}
	if _, ok := resp.Repos["empty-repo"]; ok {
		t.Errorf("Repos[empty-repo] ne devrait pas exister (0 octet), got %+v", resp.Repos)
	}
}
