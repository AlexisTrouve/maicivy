package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

// ============================================================================
// Tests de la montagne : table des strates, déduction des liens, déterminisme des positions,
// monogramme, assemblage. Aucun appel réseau : la source maiProFiles est simulée en mémoire.
// ============================================================================

// --- Strates -----------------------------------------------------------------

// Catégories RÉELLES (maiProFiles, 2026-10-30) → strate attendue. Chaque cas verrouille une
// décision de la table ; un changement de table qui en déplace un doit être assumé ici.
func TestStrateDuProjet_CategoriesReelles(t *testing.T) {
	cas := []struct {
		id, category string
		attendu      Strate
	}{
		{"grove_engine", "C++ Systems Programming / Modular Runtime / Game Engine", StrateMoteurs},
		{"daedalium", "C++ engine / Navigation & Pathfinding / Crowd simulation", StrateMoteurs},
		{"theomen", "Game / C++ engine / Procedural generation", StrateOeuvres}, // un JEU qui a un moteur
		{"lecodeurfou", "Infrastructure / Internal platform / Multi-tenant billing", StrateFondations},
		{"anthropic_vendor_proxy", "Backend Development / API Infrastructure / DevOps", StrateFondations},
		{"ladderkeg", "Networking / Anti-censorship / Developer tooling", StrateFondations},
		{"etheryale_duo", "Client library / Dart / Multi-agent infrastructure", StrateFondations},
		{"claude_orchestrator", "AI Development / DevOps Automation / Python Backend", StrateOutillage},
		{"projectmind", "AI / LLM Tooling", StrateOutillage},
		{"codeur_notify", "Python / Automation / AI Integration", StrateOutillage}, // « Python » ne parle pas
		{"gitea_commit_brief_mcp", "Developer Tools / MCP Server", StrateOutillage},
		{"hanasuba_nutrition", "REST API / Rust backend / Web application", StrateOeuvres}, // « Rust backend » ≠ fondation
		{"aissia_celuna", "Desktop application / AI agent / Systems programming in C++", StrateOeuvres},
		{"demiurgos_game_master", "AI Application / Game Engine / Full-Stack Development", StrateOeuvres},
		{"openclaw_platform", "Full-Stack Web Application / SaaS / AI Infrastructure", StrateOeuvres},
		{"civjdr", "Worldbuilding / Editorial tooling / AI integration", StrateOeuvres},
		{"goldentoken", "Finance / Algorithmic Trading", StrateOeuvres},
	}
	for _, c := range cas {
		if got := strateDuProjet(c.category, nil); got != c.attendu {
			t.Errorf("%s (%q) : strate %q, attendu %q", c.id, c.category, got, c.attendu)
		}
	}
}

// Dans un même segment, l'ordre de la table tranche : moteur avant jeu, fondation avant outil.
func TestStrateDuProjet_OrdreDansLeSegment(t *testing.T) {
	if got := strateDuProjet("Game Engine", nil); got != StrateMoteurs {
		t.Errorf("« Game Engine » : %q, attendu moteurs", got)
	}
	if got := strateDuProjet("DevOps Automation", nil); got != StrateFondations {
		t.Errorf("« DevOps Automation » : %q, attendu fondations", got)
	}
}

// Les tags ne servent qu'en repli, quand aucun segment de la catégorie ne parle.
func TestStrateDuProjet_TagsEnRepliPuisDefaut(t *testing.T) {
	if got := strateDuProjet("Python", []string{"python", "game development"}); got != StrateOeuvres {
		t.Errorf("repli tags : %q, attendu œuvres", got)
	}
	if got := strateDuProjet("Web application", []string{"devops"}); got != StrateOeuvres {
		t.Errorf("la catégorie doit primer sur les tags : %q", got)
	}
	if got := strateDuProjet("Rust", []string{"rust", "tokio"}); got != strateParDefaut {
		t.Errorf("rien ne parle : %q, attendu %q", got, strateParDefaut)
	}
}

// « Début de mot » : un mot-clé court ne matche pas au milieu d'un mot.
func TestStrateDuProjet_DebutDeMotSeulement(t *testing.T) {
	if got := strateDuTexte("Storage layer"); got != "" { // « rag » dans « storage »
		t.Errorf("« Storage » ne doit rien matcher, obtenu %q", got)
	}
	if got := strateDuTexte("Games"); got != StrateOeuvres {
		t.Errorf("« Games » : %q, attendu œuvres", got)
	}
}

// --- Liens -------------------------------------------------------------------

func lienExiste(liens []MontagneLien, from, to, kind string) bool {
	for _, l := range liens {
		if l.From == from && l.To == to && l.Kind == kind {
			return true
		}
	}
	return false
}

// Règle 1 : l'id (ou le nom) de B dans la stack de A ⇒ B sert à construire A.
func TestDeduireLiens_Stack(t *testing.T) {
	liens := deduireLiens([]projetStructure{
		{ID: "grove_engine", Name: "GroveEngine — Modular C++ runtime"},
		{ID: "aissia_celuna", Name: "AISSIA (CELUNA) — AI assistant", Stack: []string{"c++17", "groveengine"}},
		{ID: "projectmind", Name: "ProjectMind — memory server"},
		{ID: "melodicode", Name: "MelodiCode — agents", Stack: []string{"flutter", "ProjectMind"}},
	})
	if !lienExiste(liens, "grove_engine", "aissia_celuna", lienStack) {
		t.Errorf("groveengine dans la stack d'aissia : lien attendu, obtenu %+v", liens)
	}
	if !lienExiste(liens, "projectmind", "melodicode", lienStack) {
		t.Errorf("nom « ProjectMind » dans la stack : lien attendu, obtenu %+v", liens)
	}
	if len(liens) != 2 {
		t.Errorf("2 liens attendus, obtenu %d : %+v", len(liens), liens)
	}
}

// Égalité d'élément, pas sous-chaîne ; et une clé trop courte ne désigne pas un projet.
func TestDeduireLiens_StackEgaliteStricte(t *testing.T) {
	liens := deduireLiens([]projetStructure{
		{ID: "grove_engine", Name: "GroveEngine"},
		{ID: "go", Name: "Go"}, // clé < 4 caractères : ignorée
		{ID: "app", Name: "App", Stack: []string{"groveengine-tools", "go"}},
	})
	if len(liens) != 0 {
		t.Errorf("aucun lien attendu, obtenu %+v", liens)
	}
}

// Règle 2 : les tournures donnent le sens, dans les deux directions.
func TestDeduireLiens_DescriptionTournures(t *testing.T) {
	liens := deduireLiens([]projetStructure{
		{ID: "drifterra", Name: "Drifterra — colony sim"},
		{ID: "demiurgos_game_master", Name: "Demiurgos — game master"},
		{ID: "theomen", Name: "Theomen — worlds"},
		{ID: "aveg", Name: "AVEG — voices", Description: "Pipeline generating emotional AI voices for the game Drifterra."},
		{ID: "soliano", Name: "Soliano — music", Description: "Adaptive game music. First consumer: Drifterra."},
		{ID: "iliodys", Name: "Iliodys — site", Description: "Public site for a civilization simulation (Demiurgos engine)."},
		{ID: "emergika", Name: "Emergika — terrain", Description: "Unfolds a 20 km cell of a Theomen world into terrain."},
	})
	attendus := [][2]string{
		{"aveg", "drifterra"},                // A sert B : « for the game »
		{"soliano", "drifterra"},             // A sert B : « first consumer: »
		{"demiurgos_game_master", "iliodys"}, // B sert A : « B engine »
		{"theomen", "emergika"},              // B sert A : « of a B world »
	}
	for _, a := range attendus {
		if !lienExiste(liens, a[0], a[1], lienDescription) {
			t.Errorf("lien %s → %s attendu, obtenu %+v", a[0], a[1], liens)
		}
	}
	if len(liens) != len(attendus) {
		t.Errorf("%d liens attendus, obtenu %d : %+v", len(attendus), len(liens), liens)
	}
}

// Une mention NUE (sans tournure) ne produit rien : son sens serait deviné.
func TestDeduireLiens_MentionNueSansLien(t *testing.T) {
	liens := deduireLiens([]projetStructure{
		{ID: "drifterra", Name: "Drifterra"},
		{ID: "blog", Name: "Blog", Description: "Articles about Drifterra and other things. Avalon Drifterra."},
	})
	if len(liens) != 0 {
		t.Errorf("mention nue : aucun lien attendu, obtenu %+v", liens)
	}
}

// Un lien vu par les deux règles garde le kind "stack".
func TestDeduireLiens_StackPrimeSurDescription(t *testing.T) {
	liens := deduireLiens([]projetStructure{
		{ID: "grove_engine", Name: "GroveEngine"},
		{ID: "drifterra", Name: "Drifterra", Stack: []string{"groveengine"}, Description: "Colony sim in C++17 on GroveEngine."},
	})
	if len(liens) != 1 || !lienExiste(liens, "grove_engine", "drifterra", lienStack) {
		t.Errorf("un seul lien stack attendu, obtenu %+v", liens)
	}
}

// La sortie ne dépend pas de l'ordre des fiches.
func TestDeduireLiens_IndependantDeLOrdre(t *testing.T) {
	a := []projetStructure{
		{ID: "grove_engine", Name: "GroveEngine"},
		{ID: "drifterra", Name: "Drifterra", Stack: []string{"groveengine"}},
		{ID: "aveg", Name: "AVEG", Description: "Voices for the game Drifterra."},
	}
	b := []projetStructure{a[2], a[0], a[1]}
	if !reflect.DeepEqual(deduireLiens(a), deduireLiens(b)) {
		t.Errorf("ordre d'entrée différent ⇒ liens différents")
	}
}

// --- Positions ---------------------------------------------------------------

func strateFixture() (map[string]Strate, []MontagneLien) {
	strates := map[string]Strate{
		"infra_a": StrateFondations, "infra_b": StrateFondations,
		"tool_a": StrateOutillage, "tool_b": StrateOutillage, "tool_c": StrateOutillage,
		"engine": StrateMoteurs,
		"game_a": StrateOeuvres, "game_b": StrateOeuvres, "game_c": StrateOeuvres, "app_solo": StrateOeuvres,
	}
	liens := []MontagneLien{
		{From: "engine", To: "game_a", Kind: lienStack},
		{From: "engine", To: "game_b", Kind: lienStack},
		{From: "tool_a", To: "game_c", Kind: lienDescription},
		{From: "infra_a", To: "tool_b", Kind: lienStack},
		{From: "inconnu", To: "game_a", Kind: lienStack}, // extrémité inconnue : ignorée
	}
	return strates, liens
}

// Même entrée ⇒ positions identiques au bit près, y compris liens fournis dans un autre ordre.
func TestCalculerPositions_Deterministe(t *testing.T) {
	strates, liens := strateFixture()
	p1, b1, f1 := calculerPositions(strates, liens)
	inverses := make([]MontagneLien, len(liens))
	for i, l := range liens {
		inverses[len(liens)-1-i] = l
	}
	p2, b2, f2 := calculerPositions(strates, inverses)
	if !reflect.DeepEqual(p1, p2) || !reflect.DeepEqual(b1, b2) || f1 != f2 {
		t.Fatalf("positions différentes pour la même entrée")
	}
	j1, _ := json.Marshal(p1)
	j2, _ := json.Marshal(p2)
	if string(j1) != string(j2) {
		t.Fatalf("JSON différent pour la même entrée")
	}
}

// grandeFixture — la forme RÉELLE de la donnée (2026-10-30) : 9 fondations, 25 outils, 2 moteurs,
// 39 œuvres, un moteur relié à 12 œuvres. C'est sur cette répartition (sommet le plus peuplé)
// que la première disposition s'effondrait ; les invariants doivent y tenir.
func grandeFixture() (map[string]Strate, []MontagneLien) {
	strates := map[string]Strate{}
	ajoute := func(prefixe string, n int, s Strate) {
		for i := 0; i < n; i++ {
			strates[fmt.Sprintf("%s_%02d", prefixe, i)] = s
		}
	}
	ajoute("infra", 9, StrateFondations)
	ajoute("tool", 25, StrateOutillage)
	ajoute("engine", 2, StrateMoteurs)
	ajoute("work", 39, StrateOeuvres)
	var liens []MontagneLien
	for i := 0; i < 12; i++ {
		liens = append(liens, MontagneLien{From: "engine_00", To: fmt.Sprintf("work_%02d", i*3), Kind: lienStack})
	}
	liens = append(liens,
		MontagneLien{From: "tool_04", To: "work_07", Kind: lienStack},
		MontagneLien{From: "tool_04", To: "work_08", Kind: lienStack},
		MontagneLien{From: "infra_02", To: "tool_10", Kind: lienStack},
		MontagneLien{From: "engine_00", To: "engine_01", Kind: lienStack},
	)
	return strates, liens
}

// Les projets sont SUR le cône (rayon = pente × profondeur sous le sommet), les strates sont
// empilées dans l'ordre et séparées par un vide plus grand que l'écart entre deux anneaux.
func TestCalculerPositions_ConeEtStratesSeparees(t *testing.T) {
	strates, liens := grandeFixture()
	pos, bandes, forme := calculerPositions(strates, liens)
	if len(bandes) != len(ordreStrates) {
		t.Fatalf("%d bandes, attendu %d", len(bandes), len(ordreStrates))
	}
	for id, p := range pos {
		attendu := forme.Slope * (forme.ApexY - p.Y)
		if r := math.Hypot(p.X, p.Z); math.Abs(r-attendu) > 2e-3 {
			t.Errorf("%s : rayon %.3f hors du cône (%.3f attendu à y=%.3f)", id, r, attendu, p.Y)
		}
	}
	bandeDe := map[Strate]MontagneStrate{}
	for i, b := range bandes {
		bandeDe[b.ID] = b
		if i > 0 {
			ecart := b.YMin - bandes[i-1].YMax
			if ecart < ecartStrates-1e-3 {
				t.Errorf("%s commence %.3f au-dessus de %s : vide < ecartStrates", b.ID, ecart, bandes[i-1].ID)
			}
			if b.Radius > bandes[i-1].Radius {
				t.Errorf("bande %s plus large que celle du dessous", b.ID)
			}
		}
	}
	for id, p := range pos {
		b := bandeDe[strates[id]]
		if p.Y < b.YMin-1e-9 || p.Y > b.YMax+1e-9 {
			t.Errorf("%s : y=%.3f hors de sa bande [%.3f, %.3f]", id, p.Y, b.YMin, b.YMax)
		}
	}
	if bandes[0].YMin != 0 || forme.ApexY != bandes[len(bandes)-1].YMax {
		t.Errorf("pied à y=%.3f (attendu 0), sommet %.3f ≠ haut des œuvres %.3f", bandes[0].YMin, forme.ApexY, bandes[len(bandes)-1].YMax)
	}
	if bandeDe[StrateOeuvres].Count != 39 {
		t.Errorf("œuvres : %d projets comptés, attendu 39", bandeDe[StrateOeuvres].Count)
	}
}

// Le sommet porte UN projet, l'œuvre la plus reliée ; et deux projets d'un même anneau ne sont
// jamais plus près que espacementArc (la place des étiquettes est garantie par construction).
func TestCalculerPositions_SommetEtEspacement(t *testing.T) {
	strates, liens := grandeFixture()
	pos, _, forme := calculerPositions(strates, liens)
	var auSommet []string
	parNiveau := map[float64][]string{}
	for id, p := range pos {
		if p.Y == forme.ApexY {
			auSommet = append(auSommet, id)
		}
		parNiveau[p.Y] = append(parNiveau[p.Y], id)
	}
	// work_00 est relié au moteur, comme 11 autres : à degré égal, l'id tranche.
	if len(auSommet) != 1 || auSommet[0] != "work_00" {
		t.Errorf("sommet = %v, attendu [work_00]", auSommet)
	}
	for y, ids := range parNiveau {
		for i := 0; i < len(ids); i++ {
			for j := i + 1; j < len(ids); j++ {
				a, b := pos[ids[i]], pos[ids[j]]
				if d := math.Hypot(a.X-b.X, a.Z-b.Z); d < espacementArc*0.95 {
					t.Errorf("y=%.3f : %s et %s à %.3f (< %.2f)", y, ids[i], ids[j], d, espacementArc)
				}
			}
		}
	}
}

// Les liens montent la face : une œuvre reliée au moteur est, en angle, plus près de lui que la
// moyenne des œuvres qui ne lui sont pas reliées.
func TestCalculerPositions_LiensMontentLaFace(t *testing.T) {
	strates, liens := grandeFixture()
	pos, _, _ := calculerPositions(strates, liens)
	ang := func(id string) float64 { return math.Atan2(pos[id].Z, pos[id].X) }
	ecart := func(a, b string) float64 {
		d := math.Abs(ang(a) - ang(b))
		return math.Min(d, 2*math.Pi-d)
	}
	relies := map[string]bool{}
	for _, l := range liens {
		if l.From == "engine_00" && strates[l.To] == StrateOeuvres {
			relies[l.To] = true
		}
	}
	var sr, sl float64
	var nr, nl int
	for id, s := range strates {
		if s != StrateOeuvres || id == "work_00" { // le sommet (rayon nul) n'a pas d'angle
			continue
		}
		if relies[id] {
			sr += ecart(id, "engine_00")
			nr++
		} else {
			sl += ecart(id, "engine_00")
			nl++
		}
	}
	if sr/float64(nr) >= sl/float64(nl) {
		t.Errorf("écart angulaire moyen : reliés %.2f rad, non reliés %.2f rad — les liens devraient rapprocher", sr/float64(nr), sl/float64(nl))
	}
}

// repartir : au moins un projet par anneau, total exact, aucune capacité dépassée.
func TestRepartir(t *testing.T) {
	cas := []struct {
		n    int
		caps []int
	}{{39, []int{1, 1, 3, 5, 6, 8, 10, 11, 13}}, {9, []int{14}}, {2, []int{19}}, {25, []int{17, 18}}}
	for _, c := range cas {
		parts := repartir(c.n, c.caps)
		total := 0
		for i, p := range parts {
			if p < 1 || p > c.caps[i] {
				t.Errorf("repartir(%d, %v) = %v : anneau %d hors [1, %d]", c.n, c.caps, parts, i, c.caps[i])
			}
			total += p
		}
		if total != c.n {
			t.Errorf("repartir(%d, %v) = %v : total %d", c.n, c.caps, parts, total)
		}
	}
}

// placerAnneau : chaque projet sur sa visée quand les visées sont déjà équidistantes, quel que
// soit le tour (angles décalés de 2π).
func TestPlacerAnneau_RespecteLesVisees(t *testing.T) {
	vises := []float64{0.3, 0.3 + 2*math.Pi/3, 0.3 + 4*math.Pi/3}
	for i, a := range placerAnneau(vises) {
		if d := math.Mod(math.Abs(a-vises[i])+1e-9, 2*math.Pi); d > 1e-6 {
			t.Errorf("projet %d : %.4f, visé %.4f", i, a, vises[i])
		}
	}
}

// --- Monogramme --------------------------------------------------------------

func TestMonogramme(t *testing.T) {
	cas := map[string]string{
		"GroveEngine — Modular C++ runtime": "GE",
		"Anthropic Vendor Proxy — gateway":  "AV",
		"Video to MP3 Transcriptor":         "VM",
		"Drifterra — colony sim":            "Dr",
		"ladderKeg":                         "LK",
		"maicivy":                           "Ma",
		"AISSIA (CELUNA) — AI assistant":    "AI",
		"CoconSEO":                          "CS",
		"X":                                 "X",
	}
	for nom, attendu := range cas {
		if got := monogramme(nom); got != attendu {
			t.Errorf("monogramme(%q) = %q, attendu %q", nom, got, attendu)
		}
	}
}

// Noms réels à sous-titre après deux-points : seule l'identité reste ; un « : » entre
// parenthèses ne coupe pas.
func TestNomIdentite(t *testing.T) {
	cas := map[string]string{
		"Theomen: Procedural geological world generator":             "Theomen",
		"DAOS (Dawi: Age of Stone): Physics-driven side-view colony": "DAOS (Dawi: Age of Stone)",
		"GroveEngine — Modular C++ runtime":                          "GroveEngine",
		"Video to MP3 Transcriptor":                                  "Video to MP3 Transcriptor",
		"http://x":                                                   "http://x", // pas de « : » suivi d'une espace
	}
	for nom, attendu := range cas {
		if got := nomIdentite(nom); got != attendu {
			t.Errorf("nomIdentite(%q) = %q, attendu %q", nom, got, attendu)
		}
	}
	if got := monogramme("Theomen: Procedural geological world generator"); got != "Th" {
		t.Errorf("monogramme de Theomen = %q, attendu Th", got)
	}
	if got := monogramme("DAOS (Dawi: Age of Stone): Physics-driven"); got != "DA" {
		t.Errorf("monogramme de DAOS = %q, attendu DA", got)
	}
}

// --- Assemblage --------------------------------------------------------------

// fausseSourceMontagne — maiProFiles en mémoire. Les descriptions FR diffèrent des EN : la
// structure doit pourtant être la même dans les deux langues.
type fausseSourceMontagne struct{ livreDemande string }

func (f *fausseSourceMontagne) ListProjects(_ context.Context, lang string) ([]MPFProject, error) {
	desc := "Pipeline of voices for the game Drifterra."
	if lang == "fr" {
		desc = "Pipeline de voix pour le jeu Drifterra." // ne doit JAMAIS servir
	}
	return []MPFProject{
		{ID: "grove_engine", Name: "GroveEngine — runtime", Category: "C++ / Modular Runtime", Status: "active"},
		{ID: "drifterra", Name: "Drifterra — colony sim", Category: "Game / C++", Stack: []string{"groveengine"}, Status: "wip"},
		{ID: "aveg", Name: "AVEG — voices", Category: "Developer Tools", Description: MPFDescription{Short: desc}, Status: "R&D"},
		{ID: "lecodeurfou", Name: "LeCodeurFou — gateway", Category: "Infrastructure", Status: "production"},
	}, nil
}

func (f *fausseSourceMontagne) GetProjectsIndex(_ context.Context, lang string) ([]MPFProjectIndexEntry, error) {
	if lang == "fr" {
		return []MPFProjectIndexEntry{
			{ID: "grove_engine", Name: "GroveEngine — moteur", Mini: "Moteur C++ modulaire"},
			{ID: "drifterra", Name: "Drifterra — sim", Mini: "Colonie spatiale"},
			{ID: "aveg", Name: "AVEG — voix", Mini: "Voix IA"},
			{ID: "lecodeurfou", Name: "LeCodeurFou", Mini: "Passerelle"},
		}, nil
	}
	return []MPFProjectIndexEntry{
		{ID: "grove_engine", Name: "GroveEngine — engine", Mini: "Modular C++ engine"},
		{ID: "drifterra", Name: "Drifterra — sim", Mini: "Space colony"},
		{ID: "aveg", Name: "AVEG — voices", Mini: "Voix IA"}, // NON traduit : identique au FR
		// lecodeurfou absent de l'index EN : nom anglais de la fiche, sans mini
	}, nil
}

func (f *fausseSourceMontagne) GetPortfolioBook(_ context.Context, lang string) (*MPFBook, error) {
	f.livreDemande = lang
	return &MPFBook{Lang: lang, Projects: []MPFBookProject{{
		ID:     "drifterra",
		Image:  MPFImage{URL: "/portfolio-images/drifterra.png", Alt: "hero"},
		Images: []MPFGalleryImage{{URL: "/portfolio-images/drifterra.png"}, {URL: "/portfolio-images/d2.png", Caption: "fleet"}},
		Video:  &MPFVideo{URL: "/images/vid1", PosterURL: "/images/poster1", Caption: "demo"},
	}}}, nil
}

func (f *fausseSourceMontagne) ListImages(_ context.Context) ([]MPFStoreImage, error) {
	im := func(id, ct, typ, projet string) MPFStoreImage {
		x := MPFStoreImage{ID: id, ContentType: ct, Alt: id}
		x.LinkedTo.Type, x.LinkedTo.ID = typ, projet
		return x
	}
	return []MPFStoreImage{
		im("img_ge", "image/png", "project", "grove_engine"),
		im("img_ge_svg", "image/svg+xml", "project", "grove_engine"),
		// Même fichier que la galerie du livre de drifterra (« d2.png »), téléversé une 2e fois :
		// ne doit pas faire un pétale de plus.
		func() MPFStoreImage {
			x := im("img_d2_copy", "image/png", "project", "drifterra")
			x.OriginalName = "D2.PNG"
			return x
		}(),
		im("img_ge_vid", "video/mp4", "project", "grove_engine"),
		im("img_orphan", "image/png", "", ""), // non rattachée : ignorée
		im("img_blog", "image/png", "blog", "grove_engine"),
	}, nil
}

func TestConstruireMontagne_Assemblage(t *testing.T) {
	src := &fausseSourceMontagne{}
	m, err := construireMontagne(context.Background(), src, "en")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Nodes) != 4 || m.Nodes[0].ID != "aveg" || m.Nodes[3].ID != "lecodeurfou" {
		t.Fatalf("nœuds triés par id attendus, obtenu %+v", m.Nodes)
	}
	par := map[string]MontagneNoeud{}
	for _, n := range m.Nodes {
		par[n.ID] = n
	}

	// Curation : seul le projet du livre a une page.
	if !par["drifterra"].Curated || par["grove_engine"].Curated {
		t.Errorf("curation : drifterra seul attendu")
	}
	// Pétales du livre : hero, galerie dédupliquée, puis la vidéo en dernier.
	dv := par["drifterra"].Visuals
	// (la copie « D2.PNG » du store est reconnue par son nom de fichier et écartée)
	if len(dv) != 3 || dv[0].URL != "/portfolio-images/drifterra.png" || dv[1].URL != "/portfolio-images/d2.png" ||
		dv[2].Kind != "video" || dv[2].PosterURL != "/images/poster1" {
		t.Errorf("pétales de drifterra inattendus : %+v", dv)
	}
	// Store : image et vidéo rattachées au projet ; blog et orphelines ignorées.
	gv := par["grove_engine"].Visuals
	if len(gv) != 3 || gv[0].URL != "/images/img_ge" || gv[0].Vector || !gv[1].Vector || gv[2].Kind != "video" {
		t.Errorf("pétales de grove_engine inattendus (SVG marqué vector attendu) : %+v", gv)
	}
	if cleVisuel("/portfolio-images/Grove.jpg?v=a9") != "grove" || cleVisuel("Grove.png") != "grove" {
		t.Errorf("cleVisuel : « Grove.jpg » et « Grove.png » doivent partager la clé grove")
	}
	if !cheminSVG("/portfolio-images/a.SVG?v=12") || cheminSVG("/portfolio-images/a.png?v=x.svg") {
		t.Errorf("cheminSVG : extension mal lue")
	}
	// Projet sans visuel : tableau vide, jamais null.
	if par["lecodeurfou"].Visuals == nil || len(par["lecodeurfou"].Visuals) != 0 {
		t.Errorf("visuels de lecodeurfou : tableau vide attendu")
	}
	// Textes : nom court localisé ; mini non traduit vidé ; absent de l'index → nom de la fiche.
	if par["grove_engine"].Name != "GroveEngine" || par["grove_engine"].Mini != "Modular C++ engine" {
		t.Errorf("texte de grove_engine : %+v", par["grove_engine"])
	}
	if par["aveg"].Mini != "" {
		t.Errorf("mini non traduit d'aveg affiché : %q", par["aveg"].Mini)
	}
	if par["lecodeurfou"].Name != "LeCodeurFou" || par["lecodeurfou"].Mini != "" {
		t.Errorf("lecodeurfou absent de l'index : %+v", par["lecodeurfou"])
	}
	// Strates et monogramme.
	if par["grove_engine"].Stratum != StrateMoteurs || par["lecodeurfou"].Stratum != StrateFondations {
		t.Errorf("strates inattendues : %+v", par)
	}
	if par["grove_engine"].Monogram != "GE" || par["grove_engine"].IconURL != "" {
		t.Errorf("monogramme / icon_url : %+v", par["grove_engine"])
	}
	// Liens : stack + description anglaise.
	if !lienExiste(m.Links, "grove_engine", "drifterra", lienStack) || !lienExiste(m.Links, "aveg", "drifterra", lienDescription) {
		t.Errorf("liens inattendus : %+v", m.Links)
	}
	// Sommet : seul grove_engine, des quatre cartes, existe dans cette source ; sa preuve est le
	// jeu qu'il sert à construire (drifterra), par son image principale.
	if len(m.Summit) != 1 || m.Summit[0].ID != "grove_engine" || m.Summit[0].Rule != RegleJeuxConstruits {
		t.Fatalf("sommet : [grove_engine] attendu, obtenu %+v", m.Summit)
	}
	if p := m.Summit[0].Proofs; len(p) != 1 || p[0].ProjectID != "drifterra" || p[0].URL != "/portfolio-images/drifterra.png" || !p[0].Curated {
		t.Errorf("preuves de grove_engine inattendues : %+v", p)
	}
}

// Le sommet est une décision éditoriale (montagne_sommet.go) : ce test verrouille les cartes, leur
// ordre et la règle de preuves de chacune, pour qu'aucun changement n'y entre sans qu'on le voie.
func TestSommetMontagne_ListeExplicite(t *testing.T) {
	type carte struct {
		id    string
		regle RegleSommet
	}
	attendu := []carte{
		{"grove_engine", RegleJeuxConstruits},
		{"melodicode", RegleVideo},
		{"theomen", RegleEtages},
		{"maicivy", RegleSite},
	}
	if len(sommetMontagne) != len(attendu) {
		t.Fatalf("sommet : %d cartes, attendu %d", len(sommetMontagne), len(attendu))
	}
	for i, c := range sommetMontagne {
		if c.ID != attendu[i].id || c.Regle != attendu[i].regle {
			t.Errorf("carte %d = %s/%s, attendu %s/%s", i, c.ID, c.Regle, attendu[i].id, attendu[i].regle)
		}
	}
	// La chaîne de Theomen : le monde global, puis le terrain local qu'Emergika en dérive.
	if !reflect.DeepEqual(sommetMontagne[2].Etages, []etageSommet{{"theomen", EtageGlobal}, {"emergika", EtageLocal}}) {
		t.Errorf("étages de theomen : %+v", sommetMontagne[2].Etages)
	}
	// Le site : ses quatre pages, dans l'ordre demandé.
	var pages []string
	for _, p := range sommetMontagne[3].Pages {
		pages = append(pages, p.Page+"="+p.Chemin)
	}
	if !reflect.DeepEqual(pages, []string{"blog=/blog", "gitstats=/gitstats", "chat=/chat", "cv=/cv"}) {
		t.Errorf("pages de maicivy : %v", pages)
	}
}

// noeudTest — un nœud minimal pour les règles du sommet.
func noeudTest(id, statut string, curated bool, visuels ...MontagneVisuel) MontagneNoeud {
	if visuels == nil {
		visuels = []MontagneVisuel{}
	}
	return MontagneNoeud{ID: id, Name: strings.ToUpper(id), Status: statut, Curated: curated, Visuals: visuels}
}

func imageTest(url, alt string) MontagneVisuel {
	return MontagneVisuel{Kind: "image", URL: url, Alt: alt}
}

// RegleJeuxConstruits : seuls les JEUX que le moteur sert à construire, avec un VRAI visuel ;
// une preuve par jeu, la plus aboutie d'abord ; jamais un schéma, un SVG ou un non-jeu.
func TestPreuvesJeuxConstruits_FiltreEtOrdre(t *testing.T) {
	noeuds := []MontagneNoeud{
		noeudTest("eng", "active", true, imageTest("/p/eng.jpg", "engine")),
		// Le plus abouti, mais seulement un schéma d'architecture : écarté.
		noeudTest("g_prod_schema", "production", true, imageTest("/p/g_prod_architecture.jpg", "Game")),
		// Schéma d'abord puis vraie capture : la capture est la preuve.
		noeudTest("g_active", "active", true, imageTest("/p/g_active_architecture.jpg", "arch"), imageTest("/p/g_active.jpg", "world")),
		noeudTest("g_wip", "wip", false, imageTest("/p/g_wip.jpg", "shot")),
		// Même maturité que g_wip mais relié par la description : après lui.
		noeudTest("a_wip_desc", "wip", false, imageTest("/p/a_wip.jpg", "shot")),
		noeudTest("g_svg", "active", true, MontagneVisuel{Kind: "image", URL: "/images/x", Vector: true}),
		noeudTest("g_video", "active", true, MontagneVisuel{Kind: "video", URL: "/images/v"}),
		noeudTest("g_rien", "active", true),
		noeudTest("assistant", "production", true, imageTest("/p/assistant.jpg", "app")),
		noeudTest("g_ailleurs", "production", true, imageTest("/p/ailleurs.jpg", "shot")),
	}
	parID := map[string]MontagneNoeud{}
	for _, n := range noeuds {
		parID[n.ID] = n
	}
	categories := map[string]string{
		"g_prod_schema": "Game / C++", "g_active": "Game / C++ engine / Procedural generation",
		"g_wip": "Game development / C++ / Prototype", "a_wip_desc": "Game", "g_svg": "Game", "g_video": "Game",
		"g_rien": "Game", "assistant": "Desktop application / AI agent", "g_ailleurs": "Game",
	}
	liens := []MontagneLien{
		{From: "eng", To: "a_wip_desc", Kind: lienDescription},
		{From: "eng", To: "assistant", Kind: lienStack},
		{From: "eng", To: "g_active", Kind: lienStack},
		{From: "eng", To: "g_prod_schema", Kind: lienStack},
		{From: "eng", To: "g_rien", Kind: lienStack},
		{From: "eng", To: "g_svg", Kind: lienStack},
		{From: "eng", To: "g_video", Kind: lienStack},
		{From: "eng", To: "g_wip", Kind: lienStack},
		{From: "g_wip", To: "g_ailleurs", Kind: lienStack}, // ne part pas du moteur : ignoré
	}
	got := preuvesJeuxConstruits("eng", parID, liens, categories)
	var ids, urls []string
	for _, p := range got {
		ids = append(ids, p.ProjectID)
		urls = append(urls, p.URL)
	}
	if !reflect.DeepEqual(ids, []string{"g_active", "g_wip", "a_wip_desc"}) {
		t.Fatalf("jeux retenus = %v", ids)
	}
	if urls[0] != "/p/g_active.jpg" || got[0].ProjectName != "G_ACTIVE" || !got[0].Curated || got[1].Curated {
		t.Errorf("preuve du meilleur jeu inattendue : %+v", got[0])
	}
	if !estJeu("Game / C++") || estJeu("C++ engine / Game tooling") || estJeu("") {
		t.Errorf("estJeu : seul le PREMIER segment de la catégorie dit la nature du projet")
	}
}

// Les quatre règles appliquées aux vrais ids : ordre des cartes, vidéo, chaîne d'étages (avec et
// sans visuel pour Emergika), pages du site ; une carte absente de maiProFiles disparaît.
func TestConstruireSommet_Regles(t *testing.T) {
	video := MontagneVisuel{Kind: "video", URL: "/images/demo", PosterURL: "/images/poster", Alt: "Four chats"}
	base := func(emergika MontagneNoeud) []MontagneNoeud {
		return []MontagneNoeud{
			noeudTest("melodicode", "production", true, imageTest("/p/melodicode_architecture.jpg", "arch"), video),
			noeudTest("theomen", "active", true, imageTest("/p/theomen_layers.jpg", "Theomen world layers")),
			emergika,
			noeudTest("maicivy", "production", true, imageTest("/p/maicivy_live.jpg", "live")),
		}
	}

	// 1. Emergika sans visuel : la carte Theomen ne montre que l'étage global, rien n'est absorbé
	//    (Emergika reste une tuile) ; grove_engine, absent, n'a pas de carte.
	cartes := construireSommet(base(noeudTest("emergika", "wip", false)), nil, nil)
	var ids []string
	for _, c := range cartes {
		ids = append(ids, c.ID)
		if c.Proofs == nil || c.Members == nil {
			t.Errorf("%s : proofs/members jamais nil", c.ID)
		}
	}
	if !reflect.DeepEqual(ids, []string{"melodicode", "theomen", "maicivy"}) {
		t.Fatalf("cartes = %v", ids)
	}
	if p := cartes[0].Proofs; len(p) != 1 || p[0].Kind != "video" || p[0].URL != "/images/demo" || p[0].PosterURL != "/images/poster" {
		t.Errorf("melodicode : la vidéo seule attendue, obtenu %+v", p)
	}
	theo := cartes[1]
	if len(theo.Proofs) != 1 || theo.Proofs[0].Stage != EtageGlobal || theo.Proofs[0].ProjectID != "theomen" || len(theo.Members) != 0 {
		t.Errorf("theomen sans emergika illustrée : %+v", theo)
	}
	var pages []string
	for _, p := range cartes[2].Proofs {
		if p.Kind != "page" || p.URL != "" {
			t.Errorf("maicivy : preuve de page attendue, obtenu %+v", p)
		}
		pages = append(pages, p.Path)
	}
	if !reflect.DeepEqual(pages, []string{"/blog", "/gitstats", "/chat", "/cv"}) {
		t.Errorf("maicivy : pages %v", pages)
	}

	// 2. Emergika avec un vrai visuel : l'étage local s'ajoute APRÈS le global, et Emergika est
	//    absorbée par la carte (elle cesse d'être une tuile).
	cartes = construireSommet(base(noeudTest("emergika", "wip", false, imageTest("/p/emergika_terrain.jpg", "terrain"))), nil, nil)
	theo = cartes[1]
	if len(theo.Proofs) != 2 || theo.Proofs[1].Stage != EtageLocal || theo.Proofs[1].ProjectID != "emergika" ||
		!reflect.DeepEqual(theo.Members, []string{"emergika"}) {
		t.Errorf("theomen avec emergika illustrée : %+v", theo)
	}
}

// La structure (liens, strates, positions) est la même dans toutes les langues ; seuls les
// textes changent. Le livre est lu dans la langue demandée.
func TestConstruireMontagne_StructureIndependanteDeLaLangue(t *testing.T) {
	src := &fausseSourceMontagne{}
	en, err := construireMontagne(context.Background(), src, "en")
	if err != nil {
		t.Fatal(err)
	}
	fr, err := construireMontagne(context.Background(), src, "fr")
	if err != nil {
		t.Fatal(err)
	}
	if src.livreDemande != "fr" {
		t.Errorf("livre lu en %q, attendu fr", src.livreDemande)
	}
	if !reflect.DeepEqual(en.Links, fr.Links) || !reflect.DeepEqual(en.Strata, fr.Strata) {
		t.Errorf("structure différente entre en et fr")
	}
	for i := range en.Nodes {
		if en.Nodes[i].PositionNoeud != fr.Nodes[i].PositionNoeud {
			t.Errorf("%s : position différente entre en et fr", en.Nodes[i].ID)
		}
	}
	if fr.Nodes[1].Mini != "Moteur C++ modulaire" && fr.Nodes[2].Mini != "Moteur C++ modulaire" {
		t.Errorf("mini français attendu sur la montagne fr")
	}
}

// de/it/ka → anglais (jamais le français) ; fr/en/zh natifs.
func TestLangueMontagne(t *testing.T) {
	cas := map[string]string{"fr": "fr", "en": "en", "zh": "zh", "de": "en", "it": "en", "ka": "en", "": "en"}
	for in, attendu := range cas {
		if got := langueMontagne(in); got != attendu {
			t.Errorf("langueMontagne(%q) = %q, attendu %q", in, got, attendu)
		}
	}
}
