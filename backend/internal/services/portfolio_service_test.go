package services

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeMaiProFiles monte un serveur maiProFiles minimal pour tester le wiring du PortfolioService
// SANS réseau. On route sur le path (le client ajoute ?lang= en query, ignoré ici). Le PortfolioService
// est construit en pointant son client interne vers ce serveur (champs non exportés, même package).
func fakeMaiProFiles(t *testing.T) *PortfolioService {
	t.Helper()
	mux := http.NewServeMux()

	// /profile — bio/domaines (PAS d'expériences ni de skills riches : c'est tout l'intérêt du fix).
	mux.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"name":"Alexi","headline":"Dev","experience_years":9,` +
			`"bio":{"short":"bio courte","full":"bio longue"},` +
			`"skills":{"strong":["Go"],"familiar":["Rust"]},"domains":["Backend"]}`))
	})

	// /experiences — le parcours pro réel (poste, boîte, dates, technos, accroche, résumé). Un poste
	// clos (avec technos + catchphrase) + un en cours.
	mux.HandleFunc("/experiences", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[` +
			`{"id":"2015-cogesco-stage","title":"Stagiaire Développeur VBA","company":"Cogesco",` +
			`"category":"fullstack","technologies":["VBA","Access","SQL"],` +
			`"catchphrase":"Outils RH et automatisation en VBA",` +
			`"start_date":"2015-06","end_date":"2015-10","description":{"short":"Outils RH en VBA"}},` +
			`{"id":"current","title":"Freelance","company":"Indépendant",` +
			`"start_date":"2024-01","end_date":null,"description":{"short":"Missions dev"}}` +
			`]`))
	})

	// /skills — les skills curés (8 dans le fixture, 27 en prod) avec catégorie/niveau/années.
	mux.HandleFunc("/skills", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[` +
			`{"name":"AI / LLM Integration","level":"expert","category":"AI","years":4},` +
			`{"name":"VBA","level":"intermediate","category":"Languages","years":4},` +
			`{"name":"API Development","level":"expert","category":"Backend","years":6},` +
			`{"name":"Go","level":"advanced","category":"Languages","years":5},` +
			`{"name":"Rust","level":"intermediate","category":"Languages","years":2},` +
			`{"name":"React","level":"advanced","category":"Frontend","years":4},` +
			`{"name":"PostgreSQL","level":"advanced","category":"Backend","years":6},` +
			`{"name":"Docker","level":"advanced","category":"DevOps","years":5}` +
			`]`))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &PortfolioService{client: &MaiProFilesClient{http: srv.Client(), baseURL: srv.URL}}
}

// mapProject DOIT exposer GithubURL/DemoURL à la fiche chat — avant ce fix, ces champs n'existaient
// pas sur PortfolioEntry, donc ProjectFiche.tsx n'avait aucun lien cliquable vers le code/la démo
// (BlogFiche, lui, en a un depuis le début — incohérence). Même logique de filtrage que le CV
// (content_provider.go, mapProjectLinks/isGithubURL) : réutilisée ici, PAS réimplémentée.
func TestMapProject_ExposesGithubAndDemoURL(t *testing.T) {
	p := &MPFProject{
		ID:   "maicivy",
		Name: "maicivy",
		Links: map[string]string{
			"repo":    "github.com/AlexisTrouve/maicivy",
			"demo":    "maicivy.etheryale.com",
			"website": "alexistrouve.pro", // ni repo ni demo → doit rester ignoré ici
		},
	}
	entry := mapProject(p)

	if entry.GithubURL != "https://github.com/AlexisTrouve/maicivy" {
		t.Errorf("GithubURL = %q, attendu https://github.com/AlexisTrouve/maicivy (scheme ajouté)", entry.GithubURL)
	}
	if entry.DemoURL != "https://maicivy.etheryale.com" {
		t.Errorf("DemoURL = %q, attendu https://maicivy.etheryale.com (scheme ajouté)", entry.DemoURL)
	}
}

// TestMapProject_HidesPrivateGiteaRepo verrouille la même règle que le CV : un lien "repo" qui ne
// pointe PAS vers github.com (ex: Gitea privé StillHammer) reste MASQUÉ — pas de mur de login pour
// un visiteur qui clique depuis le chat.
func TestMapProject_HidesPrivateGiteaRepo(t *testing.T) {
	p := &MPFProject{
		ID:   "drifterra",
		Name: "drifterra",
		Links: map[string]string{
			"repo": "git.etheryale.com/StillHammer/drifterra", // Gitea privé, pas github.com
		},
	}
	entry := mapProject(p)

	if entry.GithubURL != "" {
		t.Errorf("GithubURL = %q, attendu vide (repo Gitea privé, pas github.com)", entry.GithubURL)
	}
}

// TestMapProject_NoLinks : projet sans aucun lien → champs vides, pas de panic.
func TestMapProject_NoLinks(t *testing.T) {
	entry := mapProject(&MPFProject{ID: "x", Name: "x"})
	if entry.GithubURL != "" || entry.DemoURL != "" {
		t.Errorf("attendu GithubURL/DemoURL vides sans Links, got %+v", entry)
	}
}

// L'agent DOIT pouvoir parler du parcours pro : GetExperience expose les expériences de /experiences.
// AVANT ce fix, le champ Experience était une slice vide EN DUR → l'agent ne savait rien du parcours.
func TestGetExperience_PopulatesFromExperiencesEndpoint(t *testing.T) {
	ps := fakeMaiProFiles(t)
	data := ps.GetExperience("fr")

	if len(data.Experience) < 2 {
		t.Fatalf("GetExperience: attendu >=2 expériences depuis /experiences, eu %d", len(data.Experience))
	}
	// La 1re expérience (Cogesco) doit être présente et correctement mappée.
	first := data.Experience[0]
	if first.Company != "Cogesco" {
		t.Errorf("Company: attendu Cogesco, eu %q", first.Company)
	}
	if first.Role != "Stagiaire Développeur VBA" {
		t.Errorf("Role: attendu le titre du poste, eu %q", first.Role)
	}
	if !strings.Contains(first.Period, "2015-06") || !strings.Contains(first.Period, "2015-10") {
		t.Errorf("Period: attendu la plage de dates, eu %q", first.Period)
	}
	if first.Summary == "" {
		t.Error("Summary: attendu le résumé du poste, eu vide")
	}
	// Infos enrichies : l'agent doit pouvoir citer la stack + l'accroche du poste.
	if !containsStr(first.Technologies, "VBA") {
		t.Errorf("Technologies: attendu VBA dans la stack, eu %v", first.Technologies)
	}
	if first.Catchphrase == "" {
		t.Error("Catchphrase: attendu l'accroche du poste, eu vide")
	}
	if first.Category != "fullstack" {
		t.Errorf("Category: attendu fullstack, eu %q", first.Category)
	}
}

// containsStr — petit helper de test : la slice contient-elle la valeur ?
func containsStr(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// Un poste en cours (end_date null) doit afficher une période ouverte, pas une fin vide/erronée.
func TestGetExperience_OngoingRoleHasOpenPeriod(t *testing.T) {
	ps := fakeMaiProFiles(t)
	data := ps.GetExperience("fr")

	var ongoing *ExperienceItem
	for i := range data.Experience {
		if data.Experience[i].Company == "Indépendant" {
			ongoing = &data.Experience[i]
		}
	}
	if ongoing == nil {
		t.Fatal("expérience en cours (Indépendant) introuvable")
	}
	if !strings.Contains(ongoing.Period, "2024-01") {
		t.Errorf("Period en cours: attendu la date de début, eu %q", ongoing.Period)
	}
}

// L'agent DOIT voir les skills curés de /skills (27 en prod), PAS les ~7 noms de /profile.
func TestListSkills_SourcedFromRichSkillsEndpoint(t *testing.T) {
	ps := fakeMaiProFiles(t)
	cats := ps.ListSkills("fr")

	total := 0
	for _, c := range cats {
		total += len(c.Skills)
	}
	if total < 8 {
		t.Fatalf("ListSkills: attendu tous les skills de /skills (>=8 dans le fixture), eu %d", total)
	}
	// Groupé par les vraies catégories maiProFiles (AI, Languages, Backend…), pas les libellés /profile.
	names := map[string]bool{}
	for _, c := range cats {
		names[c.Name] = true
	}
	if !names["Languages"] || !names["AI"] {
		t.Errorf("catégories attendues (Languages, AI) absentes: %v", names)
	}

	// Chaque skill porte son niveau + ses années (l'agent doit pouvoir parler de la maîtrise).
	var goSkill *SkillDetail
	for ci := range cats {
		for si := range cats[ci].Skills {
			if cats[ci].Skills[si].Name == "Go" {
				goSkill = &cats[ci].Skills[si]
			}
		}
	}
	if goSkill == nil {
		t.Fatal("skill 'Go' introuvable dans le résultat")
	}
	if goSkill.Level != "advanced" {
		t.Errorf("Go.Level: attendu advanced, eu %q", goSkill.Level)
	}
	if goSkill.Years != 5 {
		t.Errorf("Go.Years: attendu 5, eu %d", goSkill.Years)
	}
}
