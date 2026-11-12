package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// QUOI : un dépôt listé dans l'exclusion (GITEA_STATS_EXCLUDE) n'existe pas pour les stats.
// POURQUOI : certains dépôts privés ne doivent apparaître NULLE PART sur le site public (stats,
// langages). Leur nom ne peut pas vivre dans le code — maicivy est miroité en public — d'où l'env.
// COMMENT : listRepos est la seule entrée des deux caches (gitstats et gitlang) ; on vérifie qu'il
// écarte le dépôt, sans casse, et garde les autres.
func TestListRepos_ExclutLesDepotsConfigures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/repos/search") {
			_, _ = w.Write([]byte(`{"ok":true,"data":[{"name":"Public"},{"name":"Secret"},{"name":"Autre"}]}`))
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	svc := NewGiteaStatsService(nil, srv.URL, "tok", "u")
	svc.SetExcludedRepos(" secret , ,inexistant")

	repos, err := svc.listRepos(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var noms []string
	for _, r := range repos {
		noms = append(noms, r.Name)
	}
	if strings.Join(noms, ",") != "Public,Autre" {
		t.Fatalf("dépôts listés = %v, attendu [Public Autre]", noms)
	}
}
