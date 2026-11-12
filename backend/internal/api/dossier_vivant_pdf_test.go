package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/services"
)

// QUOI : les portes du PDF du dossier vivant.
//   - la route owner refuse tout non-owner AVANT le rendu : ni Chromium, ni lecture des fiches ;
//   - les routes publiques — rejeu, et run réel d'un visiteur sans cookie — rendent un vrai PDF
//     (`%PDF` en tête) par le Chromium local, à partir de la copie SERVEUR du dossier ; seul le rejeu
//     porte sa mention. (La faille du corps forgé : dossier_vivant_pdf_forge_test.go ; le run réel de
//     bout en bout : dossier_vivant_record_test.go.)
// Le rendu exige un navigateur : sans lui (machine de CI sans Chrome), le test du PDF est sauté,
// sauf si DOSSIER_PDF_REQUIRE_CHROME est posé (le banc local, où il doit passer).

type countingFacts struct{ calls int }

func (f *countingFacts) Montagne(context.Context, string) (*services.Montagne, error) {
	f.calls++
	return &services.Montagne{Nodes: []services.MontagneNoeud{{ID: "envoia", Name: "EnvoIA", Monogram: "EI"}}}, nil
}

func (f *countingFacts) ListProjects(context.Context, string) ([]services.MPFProject, error) {
	f.calls++
	return []services.MPFProject{{ID: "envoia", Status: "production", Stats: services.MPFStats{LOC: 8971, Tests: 246},
		Stack: []string{"python", "sqlite"}, Tags: []string{"Automatisation"}}}, nil
}

func (f *countingFacts) Skills(_ context.Context, lang string) ([]services.MPFSkill, error) {
	f.calls++
	name := "Automatisation"
	if lang == "en" {
		name = "Automation"
	}
	return []services.MPFSkill{{Name: name, Level: "advanced", Years: 5}, {Name: "Rust", Level: "intermediate", Years: 1}}, nil
}

const pdfBody = `{
  "lang": "fr",
  "title": "Dossier de conception",
  "subtitle": ["Pour Inès Morel, CTO chez Tessera.", "Préparé en 0:37 par cinq agents, pour 0,096 $."],
  "labels": {"project":"Projet","status":"Statut","lines":"Lignes","tests":"Tests","demo":"Démo","video":"vidéo","site":"le site",
    "none":"aucune publique","also":"Aussi retenus : {names}.","contact":"Écrire à Alexis",
    "replayNote":"Exemple rejoué : un scénario enregistré, sans appel IA.","statuses":{"production":"En production"}},
  "sections": [
    {"id":"need","title":"Le besoin, reformulé","blocks":[{"type":"lead","text":"Traiter 1 400 tickets par jour à six."}]},
    {"id":"architecture","title":"L’architecture proposée","blocks":[{"type":"diagram","diagram":{
      "label":"Chaîne proposée","journal":"Journal des décisions","width":200,"height":76,"margin":2,"scale":1,
      "journalY":52,"journalH":24,"boxH":40,
      "boxes":[{"x":0,"y":0,"w":80,"title":"Ticket","subtitle":"entrant","human":false},{"x":112,"y":0,"w":88,"title":"Trieur","subtitle":"modèle léger","human":true}],
      "edges":[{"d":"M80,20H106","head":"M112,20l-6,-3.5v7z","dashed":false}],
      "steps":[{"n":1,"x":96,"y":20}]}}]},
    {"id":"proofs","title":"Les projets qui le prouvent","blocks":[{"type":"proof","projectId":"envoia","why":"Un contenu externe ne peut pas infléchir l’action."}]},
    {"id":"next","title":"Prochaine étape","blocks":[{"type":"paragraph","text":"Un appel de 30 minutes."},{"type":"contact"}]}
  ]
}`

func pdfApp(facts services.DossierFactsSource) *fiber.App {
	app := fiber.New()
	h := NewDossierPDFHandler(services.NewDossierPDFService(facts), noDossiers{}, dvSecret, dvKey)
	h.RegisterPublicRoutes(app.Group("/api/v1"))
	h.RegisterOwnerRoutes(app.Group("/api/v1"))
	return app
}

// noDossiers — aucun dossier de run gardé.
type noDossiers struct{}

func (noDossiers) Load(context.Context, string) (*services.DossierRecord, error) {
	return nil, services.ErrDossierNotFound
}

// memRedis — un Redis en mémoire, fermé en fin de test.
func memRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

// pdfAppCapturing — les routes du PDF, sur un vrai DossierStore (Redis en mémoire), l'impression
// remplacée par une capture du HTML (cf. dossier_vivant_pdf_forge_test.go).
func pdfAppCapturing(t *testing.T) (*fiber.App, *capture) {
	t.Helper()
	c := &capture{}
	app := fiber.New()
	h := NewDossierPDFHandler(services.NewDossierPDFService(&countingFacts{}).WithRenderer(c.render), services.NewDossierStore(memRedis(t)), dvSecret, dvKey)
	h.RegisterPublicRoutes(app.Group("/api/v1"))
	h.RegisterOwnerRoutes(app.Group("/api/v1"))
	return app, c
}

func pdfRequest(path, body string, headers map[string]string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}

func TestDossierPDF_RouteOwnerRefuseToutNonOwner(t *testing.T) {
	cases := map[string]map[string]string{
		"sans authentification": nil,
		"mauvaise clé owner":    {"X-Owner-Key": "pas-la-bonne"},
		"cookie admin forgé":    {"Cookie": "maicivy_admin=admin:1.x"},
	}
	for name, headers := range cases {
		t.Run(name, func(t *testing.T) {
			facts := &countingFacts{}
			resp, err := pdfApp(facts).Test(pdfRequest("/api/v1/admin/dossier-vivant/pdf", pdfBody, headers), -1)
			require.NoError(t, err)
			assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
			assert.Zero(t, facts.calls, "refusé avant toute lecture et tout rendu")
		})
	}
}

// La route owner reçoit encore un dossier en corps : il reste validé contre le contrat.
func TestDossierPDF_RequeteHorsContrat400(t *testing.T) {
	resp, err := pdfApp(&countingFacts{}).Test(pdfRequest("/api/v1/admin/dossier-vivant/pdf", `{"lang":"fr","title":"x","sections":[{"id":"evil"}]}`, map[string]string{"X-Owner-Key": dvKey}), -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// La route de l'écran : mêmes sections que le PDF, même sélection — et la porte d'entrée publique
// refuse une requête hors contrat avant toute lecture.
func TestDossierSkills_RouteDeLEcran(t *testing.T) {
	body := `{"lang":"en","sections":[{"id":"proofs","title":"Proofs","blocks":[{"type":"proof","projectId":"envoia","why":"x"}]}]}`
	resp, err := pdfApp(&countingFacts{}).Test(pdfRequest("/api/v1/dossier-vivant/skills", body, nil), -1)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	var got services.DossierSkills
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	require.Len(t, got.Skills, 1, "Rust n'est mobilisé par aucune preuve")
	assert.Equal(t, "Automation", got.Skills[0].Name, "nom localisé, apparié sur le nom canonique des tags")
	assert.Equal(t, []string{"envoia"}, got.Skills[0].Projects)

	facts := &countingFacts{}
	resp, err = pdfApp(facts).Test(pdfRequest("/api/v1/dossier-vivant/skills", `{"lang":"xx","sections":[]}`, nil), -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.Zero(t, facts.calls)
}

func TestDossierPDF_PDFValide(t *testing.T) {
	routes := map[string]struct {
		path, body string
		headers    map[string]string
	}{
		"rejeu (public)":  {"/api/v1/dossier-vivant/pdf", `{"lang":"fr"}`, nil},
		"dossier (owner)": {"/api/v1/admin/dossier-vivant/pdf", pdfBody, map[string]string{"X-Owner-Key": dvKey}},
	}
	for name, r := range routes {
		t.Run(name, func(t *testing.T) {
			resp, err := pdfApp(&countingFacts{}).Test(pdfRequest(r.path, r.body, r.headers), 60_000)
			require.NoError(t, err)
			if resp.StatusCode == fiber.StatusInternalServerError && os.Getenv("DOSSIER_PDF_REQUIRE_CHROME") == "" {
				t.Skip("Chromium indisponible : rendu PDF non exécutable sur cette machine")
			}
			require.Equal(t, fiber.StatusOK, resp.StatusCode)
			assert.Equal(t, "application/pdf", resp.Header.Get("Content-Type"))
			assert.Contains(t, resp.Header.Get("Content-Disposition"), ".pdf")
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.True(t, bytes.HasPrefix(body, []byte("%PDF")), "un vrai PDF, pas une page HTML")
			assert.Greater(t, len(body), 2000)
		})
	}
}

// Le rejeu se dit sur le document : sa route imprime la mention, dans la langue demandée, sans que
// le navigateur ait à la fournir ; une langue hors liste est refusée avant toute lecture.
func TestDossierPDF_MentionDuRejeu(t *testing.T) {
	app, printed := pdfAppCapturing(t)
	resp, err := app.Test(pdfRequest("/api/v1/dossier-vivant/pdf", `{"lang":"en"}`, nil), -1)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	require.Len(t, printed.pages, 1)
	assert.Contains(t, printed.pages[0], "Replayed example: a recorded scenario, no AI call.")
	assert.Contains(t, printed.pages[0], "Inès Morel", "le visiteur du scénario, tel que le serveur le connaît")

	facts := &countingFacts{}
	resp, err = pdfApp(facts).Test(pdfRequest("/api/v1/dossier-vivant/pdf", `{"lang":"xx"}`, nil), -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.Zero(t, facts.calls, "refusé avant toute lecture")
}
