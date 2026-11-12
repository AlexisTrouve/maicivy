package services

// Régressions de la panne gitstats du 2026-10-11 (page « claquée au sol ») :
//
//  1. COMPTER dépendait de MESURER. Chaque commit était lu avec ses stats (stat=true, ~1,5 s la
//     page contre ~0,08 s sans) sous un délai global partagé par 145 repos. Melodicode (~70 pages)
//     n'aboutissait jamais → 0 commit en permanence ; sous charge (14:15 UTC, 1 690 délais dépassés)
//     la moitié des repos tombait à 0, et un repo en échec était enregistré VIDE.
//  2. Une LISTE en échec perdait sa fenêtre pour de bon : le passage suivant ne relit que depuis son
//     propre fetchedAt, qui avait avancé quand même.
//
// Ces tests passent par GetStats (API publique) et un faux Gitea qui pagine, filtre par `since` et
// sait faire échouer séparément la liste et la mesure.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// commitFaux : un commit du faux Gitea (date auteur = date committer).
type commitFaux struct {
	sha  string
	date time.Time
}

// giteaPaginant : un repo `melo`, des commits du plus récent au plus ancien, 50 par page.
type giteaPaginant struct {
	mu            sync.Mutex
	commits       []commitFaux
	echecListe    bool // toute requête /commits → 500
	echecMesure   bool // requêtes stat=true → 500 (la liste sans stats répond)
	requetesStats int  // nombre de pages lues AVEC stats
}

func (g *giteaPaginant) ajouter(n int, age time.Duration, prefixe string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	base := time.Now().Add(-age)
	nouveaux := make([]commitFaux, 0, n)
	for i := 0; i < n; i++ {
		nouveaux = append(nouveaux, commitFaux{sha: fmt.Sprintf("%s%03d", prefixe, i), date: base.Add(-time.Duration(i) * time.Minute)})
	}
	// Ordre Gitea : du plus récent au plus ancien. Les commits ajoutés sont insérés à leur place.
	tous := append(g.commits, nouveaux...)
	for i := 1; i < len(tous); i++ {
		for j := i; j > 0 && tous[j].date.After(tous[j-1].date); j-- {
			tous[j], tous[j-1] = tous[j-1], tous[j]
		}
	}
	g.commits = tous
}

func (g *giteaPaginant) regler(f func(*giteaPaginant)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	f(g)
}

func (g *giteaPaginant) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/repos/search") {
			_, _ = fmt.Fprint(w, `{"ok":true,"data":[{"name":"melo","description":"d","language":"Dart",
				"stars_count":0,"updated_at":"2026-10-01T00:00:00Z","empty":false,"fork":false,"default_branch":"main"}]}`)
			return
		}
		if !strings.Contains(r.URL.Path, "/commits") {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		g.mu.Lock()
		defer g.mu.Unlock()
		q := r.URL.Query()
		avecStats := q.Get("stat") != "false" // défaut Gitea : stats incluses
		if g.echecListe || (avecStats && g.echecMesure) {
			http.Error(w, "surcharge", http.StatusInternalServerError)
			return
		}
		if avecStats {
			g.requetesStats++
		}
		since, _ := time.Parse(time.RFC3339, q.Get("since"))
		page, _ := strconv.Atoi(q.Get("page"))
		var filtres []commitFaux
		for _, c := range g.commits {
			if !c.date.Before(since) {
				filtres = append(filtres, c)
			}
		}
		debut, fin := (page-1)*50, page*50
		if debut > len(filtres) {
			debut = len(filtres)
		}
		if fin > len(filtres) {
			fin = len(filtres)
		}
		morceaux := make([]string, 0, fin-debut)
		for _, c := range filtres[debut:fin] {
			stats := ""
			if avecStats {
				stats = `,"stats":{"additions":10,"deletions":1}`
			}
			morceaux = append(morceaux, fmt.Sprintf(`{"sha":%q,"commit":{"author":{"date":%q}}%s}`,
				c.sha, c.date.UTC().Format(time.RFC3339), stats))
		}
		_, _ = w.Write([]byte("[" + strings.Join(morceaux, ",") + "]"))
	}
}

func serviceSurGiteaPaginant(t *testing.T) (*GiteaStatsService, *giteaPaginant) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	g := &giteaPaginant{}
	srv := httptest.NewServer(g.handler())
	t.Cleanup(srv.Close)
	svc := NewGiteaStatsService(redis.NewClient(&redis.Options{Addr: mr.Addr()}), srv.URL, "tok", "StillHammer")
	require.NotNil(t, svc)
	return svc, g
}

func commitsDuRepo(resp *GitStatsResponse, nom string) int {
	for _, r := range resp.Repos {
		if r.Name == nom {
			return r.Commits
		}
	}
	return -1
}

// Le compte est complet même quand la mesure des lignes échoue ; les lignes se complètent au
// passage suivant, sans recompter ni relire ce qui est déjà mesuré.
func TestGitstats_LeComptageNeDependPasDeLaMesure(t *testing.T) {
	svc, g := serviceSurGiteaPaginant(t)
	ctx := context.Background()
	g.ajouter(120, 2*time.Hour, "a") // 3 pages
	g.regler(func(g *giteaPaginant) { g.echecMesure = true })

	resp, err := svc.GetStats(ctx, false)
	require.NoError(t, err)
	require.Equal(t, 120, commitsDuRepo(resp, "melo"), "les 120 commits comptent même si leurs lignes sont illisibles")
	require.Equal(t, 120, resp.TotalCommits)
	require.Zero(t, resp.TotalAdded, "lignes inconnues = non comptées, jamais inventées")

	// La mesure revient : le passage suivant complète les lignes.
	g.regler(func(g *giteaPaginant) { g.echecMesure = false; g.requetesStats = 0 })
	resp, err = svc.GetStats(ctx, true)
	require.NoError(t, err)
	require.Equal(t, 120, resp.TotalCommits, "compléter les lignes ne recompte rien")
	require.Equal(t, 1200, resp.TotalAdded)
	require.Equal(t, 3, g.requetesStats, "une lecture avec stats par page à mesurer, pas plus")

	// Tout est mesuré : plus aucune lecture avec stats.
	g.regler(func(g *giteaPaginant) { g.requetesStats = 0 })
	resp, err = svc.GetStats(ctx, true)
	require.NoError(t, err)
	require.Equal(t, 1200, resp.TotalAdded)
	require.Zero(t, g.requetesStats, "rien à mesurer → aucune page relue avec stats")
}

// Une liste en échec ne perd pas sa fenêtre : le passage suivant remonte jusqu'à elle, même si sa
// propre fenêtre est plus courte.
func TestGitstats_UneListeEnEchecEstRelueAuPassageSuivant(t *testing.T) {
	svc, g := serviceSurGiteaPaginant(t)
	ctx := context.Background()
	g.ajouter(10, 30*24*time.Hour, "vieux")
	_, err := svc.GetStats(ctx, false)
	require.NoError(t, err)

	// Commits d'il y a 12 jours, inconnus du cache (le cache date d'il y a 20 jours).
	g.ajouter(5, 12*24*time.Hour, "rate")
	backdateCacheVers(t, svc, 20*24*time.Hour)

	// Passage 1 : la liste échoue (Gitea saturé). Rien n'est perdu de ce qui était connu.
	g.regler(func(g *giteaPaginant) { g.echecListe = true })
	resp, err := svc.GetStats(ctx, true)
	require.NoError(t, err)
	require.Equal(t, 10, commitsDuRepo(resp, "melo"), "un échec ne vide jamais un repo")

	// Passage 2 : Gitea répond. Sa fenêtre propre (7 jours) ne couvre pas les commits d'il y a
	// 12 jours ; ils doivent quand même arriver.
	g.regler(func(g *giteaPaginant) { g.echecListe = false })
	resp, err = svc.GetStats(ctx, true)
	require.NoError(t, err)
	require.Equal(t, 15, commitsDuRepo(resp, "melo"), "la fenêtre du passage en échec est relue")
}

// Un appelant qui a constaté « pas de cache » JUSTE avant qu'un fetch complet ne le pose ne doit pas
// relancer 6 mois de fetch. Mesuré en prod au déploiement de la v9 (2026-10-11, 14:53-14:59 UTC) :
// trois fetches complets de 2 min enchaînés, chacun relancé par un appel arrivé pendant la
// sauvegarde du précédent.
func TestGitstats_PasDeSecondFetchCompletQuandLeCacheVientDEtrePose(t *testing.T) {
	svc, g := serviceSurGiteaPaginant(t)
	ctx := context.Background()
	g.ajouter(3, time.Hour, "c")

	_, err := svc.fullFetchAndCache(ctx)
	require.NoError(t, err)
	g.regler(func(g *giteaPaginant) { g.echecListe = true }) // tout nouvel accès à Gitea échouerait

	// Le retardataire : il a vu « pas de cache » avant la pose, il entre maintenant.
	resp, err := svc.fullFetchAndCache(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, commitsDuRepo(resp, "melo"), "le cache posé entre-temps est servi, Gitea n'est pas relu")
}

// Un passage dont la mesure a échoué ne retire pas des lignes déjà connues.
func TestMergeRepoCommits_UneMesureConnueNeCedePasAUneVersionSansLignes(t *testing.T) {
	cache := map[string]commitStat{"x": {Day: "2026-10-01", Additions: 40, Deletions: 2}}
	frais := map[string]commitStat{"x": {Day: "2026-10-01", SansLignes: true}, "y": {Day: "2026-10-02", SansLignes: true}}
	out := mergeRepoCommits(cache, frais, "2026-04-01")
	require.Equal(t, commitStat{Day: "2026-10-01", Additions: 40, Deletions: 2}, out["x"])
	require.True(t, out["y"].SansLignes, "un commit nouveau compte, lignes à venir")

	// Et l'inverse : une mesure fraîche complète une entrée sans lignes.
	out = mergeRepoCommits(frais, map[string]commitStat{"y": {Day: "2026-10-02", Additions: 7}}, "2026-04-01")
	require.Equal(t, 7, out["y"].Additions)
	require.False(t, out["y"].SansLignes)
}

// backdateCacheVers vieillit le cache réel du service (fetchedAt) sans dépendre de son format interne.
func backdateCacheVers(t *testing.T, svc *GiteaStatsService, age time.Duration) {
	t.Helper()
	ctx := context.Background()
	raw, err := svc.redis.Get(ctx, cacheKey).Result()
	require.NoError(t, err)
	var blob map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &blob))
	blob["fetchedAt"] = time.Now().Add(-age).Format(time.RFC3339Nano)
	out, err := json.Marshal(blob)
	require.NoError(t, err)
	require.NoError(t, svc.redis.Set(ctx, cacheKey, string(out), 0).Err())
}
