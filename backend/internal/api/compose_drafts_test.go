package api

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"maicivy/internal/middleware"
	"maicivy/internal/models"
)

// testAdminCookie forge un cookie owner valide (signé HMAC comme le vrai login).
func testAdminCookie(t *testing.T, secret string) string {
	t.Helper()
	return adminCookieName + "=" + middleware.SignAdminCookie(secret, time.Hour)
}

// QUOI : persistance de l'atelier de composition (annonce, draft, traduction, tarif, checklist, fil).
//
// POURQUOI : sans elle, un simple rafraîchissement de page perd le message en cours. Pour un opérateur
// qui ne relit pas le français et vient de passer plusieurs tours à faire ajuster le texte par l'agent,
// c'est le travail entier qui disparaît — et il n'a aucun moyen de le reconstituer lui-même.
//
// POURQUOI une table dédiée plutôt qu'une colonne sur chat_conversations : la boucle de migration du
// projet SAUTE les tables existantes (cf. database/migrations.go). Une colonne ajoutée à une table déjà
// déployée ne serait donc jamais créée en prod. Une table neuve, elle, est bien créée. Accessoirement
// c'est la meilleure séparation : l'artefact n'est pas une affaire de conversation.

func newDraftsApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.ComposeDraft{}))

	app := fiber.New()
	NewComposeDraftsHandler(db, "secret-de-test").RegisterRoutes(app)
	return app, db
}

// doAuthed exécute une requête en simulant un owner authentifié. POURQUOI un helper : la garde owner
// est vérifiée séparément ; ici on teste le comportement métier, pas l'authentification.
func doAuthed(t *testing.T, app *fiber.App, method, path, body string) (int, []byte) {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", testAdminCookie(t, "secret-de-test"))
	resp, err := app.Test(req, 5000)
	require.NoError(t, err)
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, out
}

func TestComposeDrafts_GardeOwner(t *testing.T) {
	app, _ := newDraftsApp(t)

	for _, tc := range []struct{ method, path string }{
		{"GET", "/admin/compose/drafts"},
		{"POST", "/admin/compose/drafts"},
		{"PUT", "/admin/compose/drafts/whatever"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, 5000)
			require.NoError(t, err)
			assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
		})
	}
}

func TestComposeDrafts_CreerPuisRelire(t *testing.T) {
	app, _ := newDraftsApp(t)

	status, body := doAuthed(t, app, "POST", "/admin/compose/drafts",
		`{"announce":"Recherche freelance IA","draft":"Bonjour,\n\nAlexis","rate":300}`)
	require.Equal(t, fiber.StatusOK, status, string(body))

	var created struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(body, &created))
	require.NotEmpty(t, created.ID, "l'id est ce qui permet au front de sauvegarder les tours suivants")

	// La liste identifie et situe l'atelier ; le CONTENU se relit à l'unité (cf.
	// composeDraftSummary — le fil embarque les captures base64, il n'a rien à faire dans une liste).
	status, body = doAuthed(t, app, "GET", "/admin/compose/drafts", "")
	require.Equal(t, fiber.StatusOK, status)

	var list struct {
		Drafts []struct {
			ID   string `json:"id"`
			Rate int    `json:"rate"`
		} `json:"drafts"`
	}
	require.NoError(t, json.Unmarshal(body, &list))
	require.Len(t, list.Drafts, 1)
	assert.Equal(t, created.ID, list.Drafts[0].ID)
	assert.Equal(t, 300, list.Drafts[0].Rate)

	status, body = doAuthed(t, app, "GET", "/admin/compose/drafts/"+created.ID, "")
	require.Equal(t, fiber.StatusOK, status)
	var full struct {
		Announce string `json:"announce"`
		Draft    string `json:"draft"`
	}
	require.NoError(t, json.Unmarshal(body, &full))
	assert.Equal(t, "Recherche freelance IA", full.Announce)
	assert.Equal(t, "Bonjour,\n\nAlexis", full.Draft)
}

// C'est le cas qui répare le bug : après plusieurs tours, le dernier état doit être celui qu'on
// retrouve au rafraîchissement.
func TestComposeDrafts_SauvegardeEcraseLEtatPrecedent(t *testing.T) {
	app, _ := newDraftsApp(t)

	_, body := doAuthed(t, app, "POST", "/admin/compose/drafts", `{"draft":"version 1"}`)
	var created struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(body, &created))

	status, _ := doAuthed(t, app, "PUT", "/admin/compose/drafts/"+created.ID,
		`{"draft":"version 2","translation":"version 2 EN","rate":350,"rate_rationale":"mission longue"}`)
	require.Equal(t, fiber.StatusOK, status)

	_, body = doAuthed(t, app, "GET", "/admin/compose/drafts", "")
	var list struct {
		Drafts []struct{ ID string } `json:"drafts"`
	}
	require.NoError(t, json.Unmarshal(body, &list))
	require.Len(t, list.Drafts, 1, "la sauvegarde doit mettre à jour, pas créer une seconde ligne")

	_, body = doAuthed(t, app, "GET", "/admin/compose/drafts/"+created.ID, "")
	var full struct {
		Draft         string `json:"draft"`
		Translation   string `json:"translation"`
		Rate          int    `json:"rate"`
		RateRationale string `json:"rate_rationale"`
	}
	require.NoError(t, json.Unmarshal(body, &full))
	assert.Equal(t, "version 2", full.Draft)
	assert.Equal(t, "version 2 EN", full.Translation)
	assert.Equal(t, 350, full.Rate)
	assert.Equal(t, "mission longue", full.RateRationale)
}

// Le fil de conversation et la checklist voyagent en JSON opaque : le backend n'a pas à connaître leur
// forme, seul le front les interprète. Ils doivent revenir intacts.
func TestComposeDrafts_ConserveFilEtChecklist(t *testing.T) {
	app, _ := newDraftsApp(t)

	payload := `{"draft":"d","messages":[{"role":"user","content":"salut"}],` +
		`"checks":{"mechanical":[{"id":"remote","status":"pass"}]}}`
	status, body := doAuthed(t, app, "POST", "/admin/compose/drafts", payload)
	require.Equal(t, fiber.StatusOK, status, string(body))
	var created struct{ ID string }
	require.NoError(t, json.Unmarshal(body, &created))

	// Relecture à l'unité : c'est là que le fil circule, jamais dans la liste.
	_, body = doAuthed(t, app, "GET", "/admin/compose/drafts/"+created.ID, "")
	var full struct {
		Messages json.RawMessage `json:"messages"`
		Checks   json.RawMessage `json:"checks"`
	}
	require.NoError(t, json.Unmarshal(body, &full))
	assert.JSONEq(t, `[{"role":"user","content":"salut"}]`, string(full.Messages))
	assert.JSONEq(t, `{"mechanical":[{"id":"remote","status":"pass"}]}`, string(full.Checks))
}

// Le plus récent d'abord : au rafraîchissement le front restaure l'atelier en cours, pas un vieux.
func TestComposeDrafts_PlusRecentEnPremier(t *testing.T) {
	app, db := newDraftsApp(t)

	// On distingue les deux par leur TITRE : c'est ce que la liste transporte désormais, et c'est
	// aussi ce que l'opératrice lit pour choisir.
	doAuthed(t, app, "POST", "/admin/compose/drafts", `{"title":"ancien","draft":"d1"}`)
	doAuthed(t, app, "POST", "/admin/compose/drafts", `{"title":"recent","draft":"d2"}`)

	// sqlite peut horodater deux insertions dans la même milliseconde : on force l'écart plutôt que
	// de faire dépendre le test d'une course.
	require.NoError(t, db.Exec(
		`UPDATE compose_drafts SET updated_at = datetime('now','-1 hour') WHERE title = 'ancien'`).Error)

	_, body := doAuthed(t, app, "GET", "/admin/compose/drafts", "")
	var list struct {
		Drafts []struct {
			Title string `json:"title"`
		} `json:"drafts"`
	}
	require.NoError(t, json.Unmarshal(body, &list))
	require.Len(t, list.Drafts, 2)
	assert.Equal(t, "recent", list.Drafts[0].Title)
}

// --- Historique : liste légère, archivage, marquage envoyé ------------------

// LE point de performance de cette fonctionnalité. La liste ne doit PAS transporter le contenu des
// ateliers.
//
// Mesuré en production avant correction : 3 ateliers = 468 kB, dont un seul à 455 kB — le fil de
// conversation contient les CAPTURES D'ÉCRAN en base64. L'écran téléchargeait donc l'intégralité des
// 10 derniers ateliers, captures comprises, à chaque ouverture, pour n'en restaurer qu'un. Garder
// l'historique aggrave mécaniquement ce défaut : plus elle en garde, plus l'écran est lourd à ouvrir.
func TestComposeDrafts_ListeNeTransportePasLeContenu(t *testing.T) {
	app, _ := newDraftsApp(t)

	lourd := `{"announce":"annonce","draft":"le message","translation":"the message",` +
		`"messages":[{"role":"user","content":"CONTENU_LOURD_NE_DOIT_PAS_ETRE_LISTE"}],"checks":{}}`
	code, _ := doAuthed(t, app, "POST", "/admin/compose/drafts", lourd)
	require.Equal(t, 200, code)

	code, body := doAuthed(t, app, "GET", "/admin/compose/drafts", "")
	require.Equal(t, 200, code)
	assert.NotContains(t, string(body), "CONTENU_LOURD_NE_DOIT_PAS_ETRE_LISTE",
		"le fil de conversation (donc les captures base64) ne doit pas voyager dans la liste")
	assert.NotContains(t, string(body), "le message", "le draft ne doit pas voyager dans la liste")
}

// L'atelier complet se récupère à l'unité, quand on l'ouvre — pendant fatal de la liste allégée.
func TestComposeDrafts_LireUnAtelierParSonID(t *testing.T) {
	app, _ := newDraftsApp(t)

	code, body := doAuthed(t, app, "POST", "/admin/compose/drafts",
		`{"announce":"annonce","draft":"le message","messages":[{"role":"user","content":"fil complet"}],"checks":{}}`)
	require.Equal(t, 200, code)
	var created struct{ ID string }
	require.NoError(t, json.Unmarshal(body, &created))

	code, body = doAuthed(t, app, "GET", "/admin/compose/drafts/"+created.ID, "")
	require.Equal(t, 200, code)
	assert.Contains(t, string(body), "le message")
	assert.Contains(t, string(body), "fil complet")
}

// Le titre voyage avec l'état (il est produit par la vérification) ET apparaît dans la liste : c'est
// le seul repère lisible pour choisir un atelier.
func TestComposeDrafts_TitreDansLaListe(t *testing.T) {
	app, _ := newDraftsApp(t)

	code, _ := doAuthed(t, app, "POST", "/admin/compose/drafts",
		`{"title":"Galadrim — AI agents","announce":"a","draft":"d","checks":{}}`)
	require.Equal(t, 200, code)

	code, body := doAuthed(t, app, "GET", "/admin/compose/drafts", "")
	require.Equal(t, 200, code)
	assert.Contains(t, string(body), "Galadrim — AI agents")
}

// ARCHIVER, JAMAIS SUPPRIMER (décision d'Alexi). Un atelier archivé sort de la liste courante mais
// reste intégralement lisible — et peut redevenir actif.
func TestComposeDrafts_ArchiverPuisReactiver(t *testing.T) {
	app, _ := newDraftsApp(t)

	code, body := doAuthed(t, app, "POST", "/admin/compose/drafts", `{"announce":"a","draft":"d","checks":{}}`)
	require.Equal(t, 200, code)
	var created struct{ ID string }
	require.NoError(t, json.Unmarshal(body, &created))

	// Archivage → disparaît de la liste courante, apparaît dans la liste des archives.
	code, _ = doAuthed(t, app, "PATCH", "/admin/compose/drafts/"+created.ID+"/meta", `{"archived":true}`)
	require.Equal(t, 200, code)

	code, body = doAuthed(t, app, "GET", "/admin/compose/drafts", "")
	require.Equal(t, 200, code)
	assert.NotContains(t, string(body), created.ID, "un atelier archivé sort de la liste courante")

	code, body = doAuthed(t, app, "GET", "/admin/compose/drafts?archived=true", "")
	require.Equal(t, 200, code)
	assert.Contains(t, string(body), created.ID, "il doit rester consultable dans les archives")

	// Le contenu n'est PAS perdu : c'est toute la différence avec une suppression.
	code, body = doAuthed(t, app, "GET", "/admin/compose/drafts/"+created.ID, "")
	require.Equal(t, 200, code)
	assert.Contains(t, string(body), `"draft":"d"`)

	// Réactivation → il revient dans la liste courante.
	code, _ = doAuthed(t, app, "PATCH", "/admin/compose/drafts/"+created.ID+"/meta", `{"archived":false}`)
	require.Equal(t, 200, code)
	code, body = doAuthed(t, app, "GET", "/admin/compose/drafts", "")
	require.Equal(t, 200, code)
	assert.Contains(t, string(body), created.ID)
}

// Renommage à la main + marquage « envoyé », par le même endpoint de métadonnées.
//
// POURQUOI un endpoint SÉPARÉ du PUT d'état : renommer ne doit pas obliger le front à renvoyer
// l'annonce, le draft, la traduction et tout le fil — ce serait à la fois absurde en volume et
// risqué, un renommage pouvant alors écraser un état plus récent.
func TestComposeDrafts_RenommerEtMarquerEnvoye(t *testing.T) {
	app, _ := newDraftsApp(t)

	code, body := doAuthed(t, app, "POST", "/admin/compose/drafts", `{"title":"auto","announce":"a","draft":"d","checks":{}}`)
	require.Equal(t, 200, code)
	var created struct{ ID string }
	require.NoError(t, json.Unmarshal(body, &created))

	code, _ = doAuthed(t, app, "PATCH", "/admin/compose/drafts/"+created.ID+"/meta",
		`{"title":"Mon nom à moi","sent":true}`)
	require.Equal(t, 200, code)

	code, body = doAuthed(t, app, "GET", "/admin/compose/drafts", "")
	require.Equal(t, 200, code)
	assert.Contains(t, string(body), "Mon nom à moi")
	assert.Contains(t, string(body), "sent_at")

	// Le draft est intact : une modification de métadonnée ne touche pas à l'état de l'atelier.
	code, body = doAuthed(t, app, "GET", "/admin/compose/drafts/"+created.ID, "")
	require.Equal(t, 200, code)
	assert.Contains(t, string(body), `"draft":"d"`)
}

// Un champ absent du PATCH ne doit RIEN changer — sinon marquer « envoyé » effacerait le titre.
// C'est ce que garantit l'usage de pointeurs côté handler.
func TestComposeDrafts_MetaNeTouchePasAuxChampsAbsents(t *testing.T) {
	app, _ := newDraftsApp(t)

	code, body := doAuthed(t, app, "POST", "/admin/compose/drafts", `{"title":"Titre initial","announce":"a","draft":"d","checks":{}}`)
	require.Equal(t, 200, code)
	var created struct{ ID string }
	require.NoError(t, json.Unmarshal(body, &created))

	code, _ = doAuthed(t, app, "PATCH", "/admin/compose/drafts/"+created.ID+"/meta", `{"sent":true}`)
	require.Equal(t, 200, code)

	code, body = doAuthed(t, app, "GET", "/admin/compose/drafts", "")
	require.Equal(t, 200, code)
	assert.Contains(t, string(body), "Titre initial", "le titre ne devait pas bouger")
}

// La garde owner couvre AUSSI les nouvelles routes — une fuite d'historique exposerait le contenu
// des messages clients.
func TestComposeDrafts_NouvellesRoutesGardeOwner(t *testing.T) {
	app, _ := newDraftsApp(t)
	for _, r := range []struct{ method, path string }{
		{"GET", "/admin/compose/drafts/nimporte"},
		{"PATCH", "/admin/compose/drafts/nimporte/meta"},
	} {
		req := httptest.NewRequest(r.method, r.path, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, 5000)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode, "%s %s doit être owner-only", r.method, r.path)
	}
}

// L'atelier relu à l'unité porte sa date d'envoi. Régression du 2026-10-12 : le bouton « Envoyé » de
// l'écran se remettait à zéro au rechargement, parce que GET /drafts/:id ne renvoyait pas sent_at
// (seule la liste le portait) — l'état était en base, l'écran ne pouvait pas le relire.
func TestComposeDrafts_AtelierRelePorteSaDateDEnvoi(t *testing.T) {
	app, _ := newDraftsApp(t)
	code, body := doAuthed(t, app, "POST", "/admin/compose/drafts", `{"announce":"a","draft":"d","checks":{}}`)
	require.Equal(t, 200, code)
	var created struct{ ID string }
	require.NoError(t, json.Unmarshal(body, &created))

	code, body = doAuthed(t, app, "GET", "/admin/compose/drafts/"+created.ID, "")
	require.Equal(t, 200, code)
	assert.NotContains(t, string(body), "sent_at", "pas encore envoyé : champ absent")

	code, _ = doAuthed(t, app, "PATCH", "/admin/compose/drafts/"+created.ID+"/meta", `{"sent":true}`)
	require.Equal(t, 200, code)
	code, body = doAuthed(t, app, "GET", "/admin/compose/drafts/"+created.ID, "")
	require.Equal(t, 200, code)
	var relu struct {
		SentAt string `json:"sent_at"`
	}
	require.NoError(t, json.Unmarshal(body, &relu))
	assert.NotEmpty(t, relu.SentAt)
}
