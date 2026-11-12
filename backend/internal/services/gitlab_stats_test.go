package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Verrouille le filtre auteur (un repo GitLab partagé : ne compter QUE les commits d'Alexis) + le merge.

// fetchDaily lit UNE seule référence — celle configurée (`GITLAB_STATS_REF`, ex. `dev`) — et rien d'autre.
//
// POURQUOI plus de passe « toutes branches » : elle demandait `all=true`, or GitLab IGNORE alors le
// paramètre `page` — mesuré le 2026-08-11 sur le projet réel, les pages 1, 2 et 3 renvoient les MÊMES
// 100 commits. Le service émettait donc 60 requêtes identiques par refresh et ne voyait que les 100
// commits les plus récents TOUS AUTEURS CONFONDUS : le chiffre affiché dépendait de l'activité des
// coéquipiers (plus ils poussaient, plus la part d'Alexis dans cette fenêtre rétrécissait). Un compteur
// dont la valeur bouge à cause du travail de quelqu'un d'autre ne mesure rien.
// Décision : on ne compte que le travail intégré sur la branche de référence.
func TestGitLabFetchDaily_SinglePassOnConfiguredRef(t *testing.T) {
	var calls int
	var sawStats, sawAll bool
	var sawRefs []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		calls++
		if q.Get("with_stats") == "true" {
			sawStats = true
		}
		if q.Get("all") == "true" {
			sawAll = true
		}
		sawRefs = append(sawRefs, q.Get("ref_name"))
		if q.Get("page") != "1" { // pas de page 2 → fin de pagination
			_, _ = w.Write([]byte(`[]`))
			return
		}
		// "aaa" (Alexis, avec lignes), "bbb" (teammate → exclu), "ccc" (massif → compté, lignes ignorées).
		_, _ = w.Write([]byte(`[
		  {"id":"aaa","author_name":"Alexis Trouvé","committed_date":"2026-07-20T10:00:00Z","stats":{"additions":10,"deletions":2}},
		  {"id":"bbb","author_name":"Yulin27","committed_date":"2026-07-20T12:00:00Z","stats":{"additions":5,"deletions":0}},
		  {"id":"ccc","author_name":"Alexis Trouvé","committed_date":"2026-07-20T13:00:00Z","stats":{"additions":60000,"deletions":0}}
		]`))
	}))
	defer srv.Close()

	s := NewGitLabStatsService(nil, srv.URL, "tok", "123", "alexis", "dev")
	daily, err := s.fetchDaily(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !sawStats {
		t.Error("les lignes add/del sont nécessaires → with_stats=true attendu")
	}
	if sawAll {
		t.Error("all=true ne doit PLUS être demandé : GitLab y ignore la pagination")
	}
	if calls != 1 {
		t.Errorf("%d requêtes, want 1 : une page de moins de 100 termine la pagination", calls)
	}
	for _, ref := range sawRefs {
		if ref != "dev" {
			t.Errorf("ref_name=%q, want \"dev\" (la référence configurée, pas la branche par défaut)", ref)
		}
	}
	// aaa + ccc = 2 commits ; bbb (teammate) exclu. Le commit massif COMPTE — même règle que côté
	// Gitea (cf. isMassiveLOC) : seules ses LIGNES sont écartées, sinon les deux séries mergées dans
	// la même courbe ne compteraient pas la même chose.
	if len(daily) != 1 || daily[0].Date != "2026-07-20" || daily[0].Commits != 2 {
		t.Fatalf("daily = %+v, want 2 commits le 2026-07-20 (aaa+ccc, teammate exclu)", daily)
	}
	if daily[0].Additions != 10 || daily[0].Deletions != 2 {
		t.Errorf("stats = +%d/-%d, want +10/-2 (lignes du commit massif exclues)", daily[0].Additions, daily[0].Deletions)
	}
}

// Sans référence configurée, on n'envoie pas `ref_name` : GitLab retombe sur la branche par défaut du
// projet — comportement d'avant, préservé pour tout déploiement qui ne définit pas GITLAB_STATS_REF.
func TestGitLabFetchDaily_NoRefConfigured(t *testing.T) {
	var sawRef string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawRef = r.URL.Query().Get("ref_name")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	s := NewGitLabStatsService(nil, srv.URL, "tok", "123", "alexis", "")
	if _, err := s.fetchDaily(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sawRef != "" {
		t.Errorf("ref_name=%q, want vide (branche par défaut du projet)", sawRef)
	}
}

func TestGitLabMatchesAuthor(t *testing.T) {
	s := NewGitLabStatsService(nil, "", "tok", "76946006", "alexis, stillhammer", "")
	if s == nil {
		t.Fatal("service nil malgré une config valide")
	}
	cases := map[string]bool{
		"Alexis Trouvé": true, // contient "alexis"
		"alexis":        true,
		"StillHammer":   true,  // contient "stillhammer"
		"Yulin27":       false, // teammate → exclu
		"":              false,
	}
	for name, want := range cases {
		if got := s.matchesAuthor(name); got != want {
			t.Errorf("matchesAuthor(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestNewGitLabStatsServiceDisabled(t *testing.T) {
	if NewGitLabStatsService(nil, "", "", "proj", "alexis", "") != nil {
		t.Error("sans token → doit être nil")
	}
	if NewGitLabStatsService(nil, "", "tok", "", "alexis", "") != nil {
		t.Error("sans projet → doit être nil")
	}
	if NewGitLabStatsService(nil, "", "tok", "proj", "  ", "") != nil {
		t.Error("sans auteur → doit être nil (sécurité : pas de filtre = ne pas compter les teammates)")
	}
}

func TestMergeGitLabDaily(t *testing.T) {
	resp := &GitStatsResponse{
		Daily: []DayStat{
			{Date: "2026-06-01", Commits: 2, Additions: 100, Deletions: 10},
			{Date: "2026-06-03", Commits: 1, Additions: 50, Deletions: 5},
		},
		Repos:       []RepoStat{{Name: "gitea-repo"}},
		ActiveRepos: 1,
	}
	gitlab := []DayStat{
		{Date: "2026-06-01", Commits: 3, Additions: 200, Deletions: 20}, // même jour → somme
		{Date: "2026-06-02", Commits: 1, Additions: 30, Deletions: 0},   // nouveau jour
	}
	MergeGitLabDaily(resp, gitlab)

	if resp.Daily[0].Date != "2026-06-01" || resp.Daily[0].Commits != 5 || resp.Daily[0].Additions != 300 || resp.Daily[0].Deletions != 30 {
		t.Errorf("06-01 = %+v, want {5,300,30}", resp.Daily[0])
	}
	if len(resp.Daily) != 3 || resp.Daily[1].Date != "2026-06-02" || resp.Daily[2].Date != "2026-06-03" {
		t.Errorf("daily mal trié/inséré: %+v", resp.Daily)
	}
	if resp.TotalCommits != 7 || resp.TotalAdded != 380 || resp.TotalDeleted != 35 {
		t.Errorf("totaux = %d/%d/%d, want 7/380/35", resp.TotalCommits, resp.TotalAdded, resp.TotalDeleted)
	}
	// Repo GitLab masqué → repos list + ActiveRepos inchangés.
	if len(resp.Repos) != 1 || resp.ActiveRepos != 1 {
		t.Errorf("repos/activeRepos modifiés (%d/%d), doivent rester 1/1", len(resp.Repos), resp.ActiveRepos)
	}
}

// Régression : le bug de réallocation. gitea = 1 jour (cap 1) ; gitlab = un jour NEUF (plus ancien,
// provoque un append→réalloc) PUIS un jour PARTAGÉ. Avec l'ancienne implémentation à pointeurs,
// l'incrément du jour partagé partait dans le tableau orphelin → perdu (06-10 restait à 1).
func TestMergeGitLabDaily_ReallocOverlapLost(t *testing.T) {
	resp := &GitStatsResponse{Daily: []DayStat{{Date: "2026-06-10", Commits: 1, Additions: 10, Deletions: 1}}}
	gitlab := []DayStat{
		{Date: "2026-06-01", Commits: 5, Additions: 50, Deletions: 5}, // NEUF (avant) → append/réalloc
		{Date: "2026-06-10", Commits: 3, Additions: 30, Deletions: 3}, // PARTAGÉ → doit s'additionner
	}
	MergeGitLabDaily(resp, gitlab)

	byDate := map[string]DayStat{}
	for _, d := range resp.Daily {
		byDate[d.Date] = d
	}
	if d := byDate["2026-06-10"]; d.Commits != 4 || d.Additions != 40 {
		t.Errorf("jour partagé perdu (bug réalloc): 06-10 = %+v, want commits=4 add=40", d)
	}
	if d := byDate["2026-06-01"]; d.Commits != 5 {
		t.Errorf("jour neuf: 06-01 = %+v, want commits=5", d)
	}
	if resp.TotalCommits != 9 || resp.TotalAdded != 90 || resp.TotalDeleted != 9 {
		t.Errorf("totaux = %d/%d/%d, want 9/90/9", resp.TotalCommits, resp.TotalAdded, resp.TotalDeleted)
	}
}
