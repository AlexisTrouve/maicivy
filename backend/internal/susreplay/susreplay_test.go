package susreplay

import (
	"strings"
	"testing"

	"maicivy/internal/middleware"
)

const trace185 = "../../testdata/attacks/185.177.72.52.jsonl"

// Rejoue le scan réel 185.177 dans les conditions PROD (scope = /api non bloqué par nginx) et
// vérifie que la défense ACTUELLE reproduit le réel : alerte firée, score > seuil, throttle dans
// le ballpark des 82 throttlés observés en prod. C'est le test de fidélité du harness.
func TestReplay_185_ReproduitProd(t *testing.T) {
	tr, err := LoadTrace(trace185)
	if err != nil {
		t.Fatal(err)
	}
	if tr.Meta.Total != 5420 {
		t.Fatalf("trace total inattendu: %d", tr.Meta.Total)
	}

	cfg := middleware.SusConfig{AlertScore: 20} // défauts prod (halflife 48h)
	// PROD : seules les requêtes /api non-444 (non bloquées par nginx badbots) atteignaient le
	// backend. Tous ces paths /api sont inexistants → le backend renvoie 404.
	apiScope := func(r Req) bool { return strings.HasPrefix(r.P, "/api/") && r.St != 444 }
	as404 := func(r Req) int { return 404 }

	m, err := Replay(tr, cfg, apiScope, as404)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("scope PROD /api : fed=%d throttled=%d reached=%d score=%.1f alerted=%v (prod réel: 82 throttlés)",
		m.Fed, m.Throttled, m.Reached, m.FinalScore, m.Alerted)

	if !m.Alerted {
		t.Error("l'alerte filou aurait dû se déclencher")
	}
	if m.FinalScore < 20 {
		t.Errorf("score final %.1f < seuil 20", m.FinalScore)
	}
	// Fourchette autour des 82 throttlés réels (variance probabiliste + déclin négligeable sur 17 min).
	if m.Throttled < 40 || m.Throttled > 120 {
		t.Errorf("throttled=%d hors ballpark prod (~82)", m.Throttled)
	}
}

// FAUX POSITIFS : le corpus LÉGITIME ne doit JAMAIS être throttlé. Un humain avec quelques 404
// incidents (sous le free-score) passe sans throttle, même dans le pire cas où le checkpoint voit
// TOUT son trafic. C'est la garde anti-faux-positif du tuning ③.
func TestReplay_LegitSynthetic_NoThrottle(t *testing.T) {
	tr, err := LoadTrace("../../testdata/legit/legit-synthetic.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	cfg := middleware.SusConfig{AlertScore: 20}
	all := func(r Req) bool { return true } // pire cas : le checkpoint voit TOUT le trafic légitime
	recorded := func(r Req) int { return r.St }

	m, err := Replay(tr, cfg, all, recorded)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("légitime: fed=%d throttled=%d score=%.1f alerted=%v", m.Fed, m.Throttled, m.FinalScore, m.Alerted)
	if m.Throttled != 0 {
		t.Errorf("FAUX POSITIF : %d requêtes légitimes throttlées (attendu 0)", m.Throttled)
	}
	if m.Alerted {
		t.Error("FAUX POSITIF : alerte filou déclenchée sur du trafic légitime")
	}
}

// TUNING ③ : évalue le signal "signature de path" sur attaque + corpus légitimes. La donnée a
// révélé deux choses : (A) avec les statuts réels, le scanner floode des 404 → le signal 4xx
// attrape déjà ~89% une fois le checkpoint sur tout le trafic → le CHOKE-POINT est le levier #1 ;
// (B) la valeur UNIQUE de la signature est l'angle mort des 200 (scanner furtif qui ne génère pas
// d'erreur) où le 4xx est aveugle. Et dans tous les cas : 0 faux positif.
func TestTune_ScannerSignatures(t *testing.T) {
	if testing.Short() {
		t.Skip("tuning lourd — hors mode court")
	}
	attack, err := LoadTrace(trace185)
	if err != nil {
		t.Fatal(err)
	}
	synth, err := LoadTrace("../../testdata/legit/legit-synthetic.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	hist, err := LoadTrace("../../testdata/legit/legit-historical.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	all := func(r Req) bool { return true } // scope TOUT (le checkpoint étendu voit tout)
	recorded := func(r Req) int { return r.St }
	as200 := func(r Req) int { return 200 } // pire cas : un site qui 200 tout → le 4xx est aveugle
	full := middleware.ScannerPathMatcher(middleware.AllScannerPatterns()...)

	// A) Statuts réels : le flood de 404 fait que le 4xx sature déjà → le choke-point est le levier.
	baseA, _ := Replay(attack, middleware.SusConfig{AlertScore: 20}, all, recorded)
	t.Logf("A) statuts réels : baseline 4xx throttled=%d/%d (le flood de 404 sature déjà)", baseA.Throttled, baseA.Fed)

	// B) 200 partout (4xx AVEUGLE) : seule la signature peut attraper le scanner furtif.
	baseB, _ := Replay(attack, middleware.SusConfig{AlertScore: 20}, all, as200)
	sigB, _ := Replay(attack, middleware.SusConfig{AlertScore: 20, ScannerPath: full}, all, as200)
	t.Logf("B) 200 partout   : baseline 4xx=%d  →  +signature=%d (la signature sauve l'angle mort)", baseB.Throttled, sigB.Throttled)

	// Per-signature : faux-positifs sur chaque corpus légitime (statuts réels). Doit être 0 partout.
	t.Logf("%-12s %9s %9s", "SIGNATURE", "FP-synth", "FP-hist")
	for _, sig := range middleware.ScannerSignatureDefs {
		cfg := middleware.SusConfig{AlertScore: 20, ScannerPath: middleware.ScannerPathMatcher(sig.Pattern)}
		fs, _ := Replay(synth, cfg, all, recorded)
		fh, _ := Replay(hist, cfg, all, recorded)
		t.Logf("%-12s %9d %9d", sig.Name, fs.Throttled, fh.Throttled)
	}

	// FP du set combiné — la garantie clé.
	cfgFull := middleware.SusConfig{AlertScore: 20, ScannerPath: full}
	fs, _ := Replay(synth, cfgFull, all, recorded)
	fh, _ := Replay(hist, cfgFull, all, recorded)
	t.Logf("COMBINÉ FP : synth=%d hist=%d", fs.Throttled, fh.Throttled)

	// Assertions : la signature sauve l'angle mort 200, et 0 faux positif (la garantie clé).
	if sigB.Throttled <= baseB.Throttled+100 {
		t.Errorf("la signature n'attrape pas l'angle mort 200 (sig=%d vs base=%d)", sigB.Throttled, baseB.Throttled)
	}
	if fs.Throttled != 0 || fh.Throttled != 0 {
		t.Errorf("FAUX POSITIF : synth=%d hist=%d (attendu 0/0)", fs.Throttled, fh.Throttled)
	}
}

// Illustre la leçon du choke-point : si le checkpoint voyait TOUT le trafic (pas juste /api),
// le throttle couvrirait bien plus. Non-assertif — mesure pédagogique.
func TestReplay_185_ChokePointGap(t *testing.T) {
	tr, err := LoadTrace(trace185)
	if err != nil {
		t.Fatal(err)
	}
	cfg := middleware.SusConfig{AlertScore: 20}
	allScope := func(r Req) bool { return r.St != 444 } // tout sauf ce que nginx bloque déjà
	recorded := func(r Req) int { return r.St }         // status réel (frontend 307/200, backend 404)

	m, err := Replay(tr, cfg, allScope, recorded)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("scope TOUT : fed=%d throttled=%d (vs 82 en /api seul) — gain de couverture du choke-point", m.Fed, m.Throttled)
}
