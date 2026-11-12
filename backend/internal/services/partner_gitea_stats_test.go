package services

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Verrouille la source « Gitea tiers » (dépôt d'un jeu partenaire, partagé avec deux autres développeurs) :
// filtre par EMAIL insensible à la casse, lignes des merges écartées, pagination, branche demandée.
// Faux serveur httptest : ces tests n'interrogent jamais le vrai Gitea.

func partnerCommitJSON(sha, email, date string, parents, add, del int) string {
	ps := make([]string, parents)
	for i := range ps {
		ps[i] = fmt.Sprintf(`{"sha":"p%d"}`, i)
	}
	return fmt.Sprintf(`{"sha":%q,"commit":{"author":{"email":%q,"date":%q}},"parents":[%s],"stats":{"additions":%d,"deletions":%d}}`,
		sha, email, date, strings.Join(ps, ","), add, del)
}

func TestPartnerGitea_FiltreEmailEtMerges(t *testing.T) {
	var refs, stats []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/api/v1/repos/studio/jeu/commits" {
			t.Errorf("chemin inattendu %s", r.URL.Path)
		}
		q := r.URL.Query()
		refs = append(refs, q.Get("sha"))
		stats = append(stats, q.Get("stat"))
		if q.Get("page") != "1" {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		_, _ = w.Write([]byte("[" + strings.Join([]string{
			// Alexis, email AVEC majuscule (forme réelle) → compté, lignes comptées.
			partnerCommitJSON("a1", "Alexistrouve.pro@gmail.com", "2026-10-20T10:00:00+08:00", 1, 10, 2),
			// Alexis sous un autre nom, même email minuscule → compté.
			partnerCommitJSON("a2", "alexistrouve.pro@gmail.com", "2026-10-20T11:00:00+08:00", 1, 5, 0),
			// Merge d'Alexis → compté comme commit, ses 844 lignes NE comptent PAS.
			partnerCommitJSON("a3", "Alexistrouve.pro@gmail.com", "2026-10-21T09:00:00+08:00", 2, 150, 694),
			// Coéquipier → exclu.
			partnerCommitJSON("b1", "anthony@example.com", "2026-10-20T12:00:00+08:00", 1, 99, 99),
			// Email qui CONTIENT celui d'Alexis sans l'être → exclu (égalité, pas sous-chaîne).
			partnerCommitJSON("b2", "x.alexistrouve.pro@gmail.com", "2026-10-20T12:00:00+08:00", 1, 7, 7),
			// Doublon de a1 (pages qui se recouvrent) → ne recompte pas.
			partnerCommitJSON("a1", "Alexistrouve.pro@gmail.com", "2026-10-20T10:00:00+08:00", 1, 10, 2),
		}, ",") + "]"))
	}))
	defer srv.Close()

	s := NewPartnerGiteaStatsService(nil, srv.URL, "tok", "studio/jeu", "main", " AlexisTrouve.pro@gmail.com ")
	if s == nil {
		t.Fatal("service configuré attendu")
	}
	daily, err := s.fetchDaily(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || refs[0] != "main" {
		t.Errorf("requêtes sur %v, want une seule sur main (page incomplète = fin)", refs)
	}
	if stats[0] != "true" {
		t.Error("stat=true attendu : les lignes alimentent les totaux")
	}
	want := map[string]DayStat{
		"2026-10-20": {Date: "2026-10-20", Commits: 2, Additions: 15, Deletions: 2},
		"2026-10-21": {Date: "2026-10-21", Commits: 1, Additions: 0, Deletions: 0},
	}
	if len(daily) != len(want) {
		t.Fatalf("daily = %+v, want %d jours", daily, len(want))
	}
	for _, d := range daily {
		if d != want[d.Date] {
			t.Errorf("%s = %+v, want %+v", d.Date, d, want[d.Date])
		}
	}
}

// Pagination : une page pleine (50) appelle la suivante ; elle s'arrête sur la première incomplète.
func TestPartnerGitea_Pagination(t *testing.T) {
	var pages []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Query().Get("page")
		pages = append(pages, p)
		n := 0
		switch p {
		case "1":
			n = partnerGiteaPageSize
		case "2":
			n = 3
		}
		items := make([]string, n)
		for i := range items {
			items[i] = partnerCommitJSON(fmt.Sprintf("s%s-%d", p, i), "alexistrouve.pro@gmail.com", "2026-10-22T10:00:00Z", 1, 1, 0)
		}
		_, _ = w.Write([]byte("[" + strings.Join(items, ",") + "]"))
	}))
	defer srv.Close()

	s := NewPartnerGiteaStatsService(nil, srv.URL, "tok", "studio/jeu", "", "alexistrouve.pro@gmail.com")
	daily, err := s.fetchDaily(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(pages, ",") != "1,2" {
		t.Errorf("pages lues %v, want 1,2", pages)
	}
	if len(daily) != 1 || daily[0].Commits != partnerGiteaPageSize+3 {
		t.Errorf("daily = %+v, want %d commits", daily, partnerGiteaPageSize+3)
	}
}

// Une erreur HTTP remonte (le handler retombe alors sur les stats sans cette source) au lieu de rendre
// une série vide qui ferait chuter les compteurs en silence.
func TestPartnerGitea_ErreurRemonte(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	s := NewPartnerGiteaStatsService(nil, srv.URL, "tok", "studio/jeu", "main", "a@b.c")
	if _, err := s.fetchDaily(context.Background()); err == nil {
		t.Fatal("erreur attendue sur 403")
	}
}

// Configuration incomplète → service désactivé (nil), jamais un service qui compterait tout le monde.
func TestPartnerGitea_ConfigIncompleteDesactive(t *testing.T) {
	cases := []struct{ url, tok, repo, emails string }{
		{"https://g", "", "o/r", "a@b.c"},
		{"https://g", "t", "", "a@b.c"},
		{"https://g", "t", "sans-slash", "a@b.c"},
		{"https://g", "t", "o/r", " , "},
		{"", "t", "o/r", "a@b.c"},
	}
	for _, c := range cases {
		if NewPartnerGiteaStatsService(nil, c.url, c.tok, c.repo, "main", c.emails) != nil {
			t.Errorf("%+v : nil attendu", c)
		}
	}
}
