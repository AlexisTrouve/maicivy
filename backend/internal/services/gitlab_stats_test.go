package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Verrouille le filtre auteur (un repo GitLab partagé : ne compter QUE les commits d'Alexis) + le merge.

// Régression : fetchDaily doit (1) compter le travail des branches de feature — passe all=true — sinon
// « 0 commit » pour du non-mergé ; (2) garder les lignes add/del de la branche par défaut — passe
// with_stats ; (3) dédupliquer par SHA entre les deux passes ; (4) filtrer teammates + commits massifs.
// Le mock distingue les passes par le paramètre `all`.
func TestGitLabFetchDaily_TwoPasses(t *testing.T) {
	var sawAll, sawStats bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("with_stats") == "true" {
			sawStats = true
		}
		if q.Get("page") != "1" { // pas de page 2 → fin de pagination
			_, _ = w.Write([]byte(`[]`))
			return
		}
		if q.Get("all") == "true" {
			// Passe 2 (toutes branches, SANS stats) : "aaa" (déjà vu en passe 1 → dédup) + "ddd"
			// (commit de feature d'Alexis, non mergé, sans lignes).
			sawAll = true
			_, _ = w.Write([]byte(`[
			  {"id":"aaa","author_name":"Alexis Trouvé","committed_date":"2026-07-20T11:00:00Z","stats":{"additions":0,"deletions":0}},
			  {"id":"ddd","author_name":"Alexis Trouvé","committed_date":"2026-07-20T14:00:00Z","stats":{"additions":0,"deletions":0}}
			]`))
			return
		}
		// Passe 1 (branche par défaut, AVEC stats) : "aaa" (Alexis, lignes), "bbb" (teammate), "ccc" (massif).
		_, _ = w.Write([]byte(`[
		  {"id":"aaa","author_name":"Alexis Trouvé","committed_date":"2026-07-20T10:00:00Z","stats":{"additions":10,"deletions":2}},
		  {"id":"bbb","author_name":"Yulin27","committed_date":"2026-07-20T12:00:00Z","stats":{"additions":5,"deletions":0}},
		  {"id":"ccc","author_name":"Alexis Trouvé","committed_date":"2026-07-20T13:00:00Z","stats":{"additions":60000,"deletions":0}}
		]`))
	}))
	defer srv.Close()

	s := NewGitLabStatsService(nil, srv.URL, "tok", "123", "alexis")
	daily, err := s.fetchDaily(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !sawStats {
		t.Error("passe 1 doit demander with_stats=true (lignes de la branche par défaut)")
	}
	if !sawAll {
		t.Error("passe 2 doit demander all=true (commits des branches de feature)")
	}
	// aaa (passe 1, compté 1× avec lignes) + ddd (passe 2, feature, sans lignes) = 2 commits ;
	// bbb (teammate) et ccc (massif) exclus ; aaa NON recompté en passe 2 (dédup SHA).
	if len(daily) != 1 || daily[0].Date != "2026-07-20" || daily[0].Commits != 2 {
		t.Fatalf("daily = %+v, want 2 commits le 2026-07-20 (aaa+ddd, dedup+filtres)", daily)
	}
	// Lignes = celles d'aaa uniquement (ddd n'a pas de stats) → +10/-2.
	if daily[0].Additions != 10 || daily[0].Deletions != 2 {
		t.Errorf("stats = +%d/-%d, want +10/-2 (lignes de la branche défaut seule)", daily[0].Additions, daily[0].Deletions)
	}
}

func TestGitLabMatchesAuthor(t *testing.T) {
	s := NewGitLabStatsService(nil, "", "tok", "76946006", "alexis, stillhammer")
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
	if NewGitLabStatsService(nil, "", "", "proj", "alexis") != nil {
		t.Error("sans token → doit être nil")
	}
	if NewGitLabStatsService(nil, "", "tok", "", "alexis") != nil {
		t.Error("sans projet → doit être nil")
	}
	if NewGitLabStatsService(nil, "", "tok", "proj", "  ") != nil {
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
