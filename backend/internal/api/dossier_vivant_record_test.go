package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/config"
	"maicivy/internal/dossiervivant"
	"maicivy/internal/dossiervivant/fakeagents"
	"maicivy/internal/services"
)

// QUOI : le PDF d'un run RÉEL, de bout en bout, sur les routes publiques.
//  1. un visiteur sans cookie lance le run (route publique) ; le serveur l'exécute contre une
//     passerelle simulée (fakeagents, celle du banc) et garde le dossier remis ;
//  2. le flux finit par run.completed portant l'id de cette copie ;
//  3. POST /dossier-vivant/run/pdf {dossierId} imprime CE dossier : visiteur, sections, schéma,
//     preuves — et rien d'un corps forgé, même accompagné d'un id valide ;
//  4. après expiration (24 h), le même id rend l'erreur nommée DOSSIER_NOT_FOUND ;
//  5. un run qui échoue ne laisse aucune copie.

// runSources — la montagne et les fiches que les agents simulés citent (fakeagents.Kept/Dropped).
type runSources struct{}

func (runSources) Montagne(context.Context, string) (*services.Montagne, error) {
	m := &services.Montagne{}
	for _, id := range append(append([]string{}, fakeagents.Kept...), fakeagents.Dropped) {
		m.Nodes = append(m.Nodes, services.MontagneNoeud{ID: id, Name: "Projet " + id, Mini: "mini " + id, Status: "production"})
	}
	return m, nil
}

func (runSources) Project(_ context.Context, id, _ string) (*services.MPFProject, error) {
	p := &services.MPFProject{ID: id, Status: "production", Stats: services.MPFStats{LOC: 1000, Tests: 100}}
	p.Description.Short = "court " + id
	p.Description.Portfolio = "récit " + id
	return p, nil
}

func (runSources) Book(context.Context, string) (*services.MPFBook, error) {
	return &services.MPFBook{Lang: "fr", Projects: []services.MPFBookProject{{ID: "melodicode", Name: "MelodiCode", Paragraphs: []string{"p"}}}}, nil
}

// runAndPDFApp — les routes publiques du run et du PDF, sur UN Redis en mémoire partagé (le run y
// garde, le PDF y lit) ; l'impression capture le HTML.
func runAndPDFApp(t *testing.T, gatewayURL string) (*fiber.App, *capture, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	store := services.NewDossierStore(redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	ai, err := services.NewAIService(&config.AIConfig{AnthropicAPIKey: "eai_test", AnthropicBaseURL: gatewayURL,
		MaxRequestsPerMinute: 600, RequestTimeout: 5 * time.Second}, nil)
	require.NoError(t, err)
	svc := dossiervivant.NewService(ai, runSources{}, dossiervivant.Options{Model: "claude-haiku-5-5", Beat: -1})
	c := &capture{}
	app := fiber.New()
	NewDossierVivantHandler(svc, store, dvSecret, dvKey).RegisterPublicRoutes(app.Group("/api/v1"))
	NewDossierPDFHandler(services.NewDossierPDFService(runFacts{}).WithRenderer(c.render), store, dvSecret, dvKey).RegisterPublicRoutes(app.Group("/api/v1"))
	return app, c, mr
}

// runFacts — les faits relus par le PDF (noms, chiffres) sur la même montagne.
type runFacts struct{}

func (runFacts) Montagne(ctx context.Context, lang string) (*services.Montagne, error) {
	return runSources{}.Montagne(ctx, lang)
}
func (runFacts) ListProjects(context.Context, string) ([]services.MPFProject, error) {
	return []services.MPFProject{{ID: "envoia", Status: "production", Stats: services.MPFStats{LOC: 8971, Tests: 246}}}, nil
}
func (runFacts) Skills(context.Context, string) ([]services.MPFSkill, error) { return nil, nil }

var dossierIDInStream = regexp.MustCompile(`"type":"run.completed","dossierId":"([0-9a-f]{32})"`)

// runToCompletion — lance un run public et rend son flux complet.
func runToCompletion(t *testing.T, app *fiber.App) string {
	t.Helper()
	body := `{"need":"Notre bureau d'études reçoit 300 demandes de devis par semaine ; nous voulons des agents.","who":"Jeanne Martin, Directrice des opérations chez Atelier Nord","lang":"fr"}`
	resp, err := app.Test(dvRequest(dvPublicPath, body, nil), -1)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	stream, _ := io.ReadAll(resp.Body)
	return string(stream)
}

func TestDossierPDF_RunReelExecuteParLeServeur(t *testing.T) {
	gw := fakeagents.NewGateway(0)
	t.Cleanup(gw.Close)
	app, printed, mr := runAndPDFApp(t, gw.URL)

	stream := runToCompletion(t, app)
	m := dossierIDInStream.FindStringSubmatch(stream)
	require.NotNil(t, m, "run.completed porte l'id de la copie serveur :\n%s", stream[max(0, len(stream)-400):])
	id := m[1]

	// 3. Le PDF du run : le contenu que le SERVEUR a produit et gardé.
	resp, err := app.Test(pdfRequest("/api/v1/dossier-vivant/run/pdf", `{"dossierId":"`+id+`","lang":"fr"}`, nil), -1)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	require.Len(t, printed.pages, 1)
	html := printed.pages[0]
	for _, want := range []string{
		"Pour Jeanne Martin, Directrice des opérations chez Atelier Nord.", // le visiteur, tel que validé par le run
		"par 6 agents", // l'équipe du run
		"Traiter chaque demande en minutes, sans perdre la trace.", // section « besoin » (Rédacteur)
		"File humaine",                       // le schéma du Dessinateur
		"Des agents en production, mesurés.", // une preuve (Chercheur)
		"Un appel de 30 minutes, puis un prototype.",
	} {
		assert.Contains(t, html, want)
	}
	assert.NotContains(t, html, "Exemple rejoué", "un run réel ne porte pas la mention du rejeu")

	// Un corps forgé accompagné d'un id VALIDE : seul l'id est lu.
	forged := strings.Replace(forgedBody, `"dossierId": "`+strings.Repeat("ab", 16)+`"`, `"dossierId": "`+id+`"`, 1)
	require.NotEqual(t, forgedBody, forged)
	resp, err = app.Test(pdfRequest("/api/v1/dossier-vivant/run/pdf", forged, nil), -1)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.False(t, strings.Contains(printed.pages[1], forge), "rien du corps forgé n'est imprimé")
	assert.Contains(t, printed.pages[1], "Traiter chaque demande en minutes")

	// 4. Expiré : Redis l'a oublié → erreur nommée.
	mr.FastForward(25 * time.Hour)
	resp, err = app.Test(pdfRequest("/api/v1/dossier-vivant/run/pdf", `{"dossierId":"`+id+`","lang":"fr"}`, nil), -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	var named struct{ Code string }
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&named))
	assert.Equal(t, "DOSSIER_NOT_FOUND", named.Code)
	assert.Len(t, printed.pages, 2, "rien n'est imprimé après expiration")
}

func TestDossierPDF_RunEnEchecNeLaissePasDeDossier(t *testing.T) {
	app, _, mr := runAndPDFApp(t, mustRefusingGateway(t))
	stream := runToCompletion(t, app)
	assert.Contains(t, stream, `"type":"run.failed"`)
	assert.NotContains(t, stream, "dossierId")
	assert.Empty(t, mr.Keys(), "un run qui échoue ne laisse aucun dossier imprimable")
}

// mustRefusingGateway — une passerelle qui refuse tout (refuseAll) : le run s'arrête sur run.failed.
func mustRefusingGateway(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(refuseAll))
	t.Cleanup(srv.Close)
	return srv.URL
}
