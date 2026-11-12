package api

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"maicivy/internal/middleware"
	"maicivy/internal/models"
)

// QUOI : rôle prospecteur (apporteur d'affaires) — défaut-refus sur /admin, révocation immédiate,
// invitations à usage unique, cloisonnement des ateliers.
//
// POURQUOI ces tests passent par RegisterAdminSurface (la fonction qu'appelle main) : c'est ce qui
// permet d'énumérer les routes RÉELLEMENT enregistrées. Une route owner ajoutée demain sans être
// inscrite dans ProspectAllowedAdminRoutes fait échouer TestProspect_DefautRefusSurToutAdmin si le
// garde ne la couvre pas.

const prospectTestSecret = "secret-de-test"

func newProspectApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Prospect{}, &models.ProspectEvent{}, &models.ComposeDraft{}, &models.ChatConversation{}))
	app := fiber.New()
	RegisterAdminSurface(app.Group("/api/v1"), AdminSurfaceDeps{
		DB: db, SessionSecret: prospectTestSecret, AdminPassword: "pw-owner",
	})
	return app, db
}

// seedProspect crée un compte ACTIF et renvoie son cookie de session.
func seedProspect(t *testing.T, db *gorm.DB, name string) (models.Prospect, string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("motdepasse-solide"), bcrypt.MinCost)
	require.NoError(t, err)
	p := models.Prospect{ID: "id-" + name, Name: name, Ref: "r" + name, PasswordHash: string(hash), Active: true}
	require.NoError(t, db.Create(&p).Error)
	return p, middleware.ProspectCookieName + "=" + middleware.SignProspectCookie(prospectTestSecret, p.ID, time.Hour)
}

func call(t *testing.T, app *fiber.App, method, path, cookie, body string) (int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	resp, err := app.Test(req, 10000)
	require.NoError(t, err)
	raw, _ := io.ReadAll(resp.Body)
	out := map[string]interface{}{}
	_ = json.Unmarshal(raw, &out)
	return resp.StatusCode, out
}

func ownerCookie() string {
	return adminCookieName + "=" + middleware.SignAdminCookie(prospectTestSecret, time.Hour)
}

var paramSeg = regexp.MustCompile(`:[A-Za-z_]+\??`)

// Défaut-refus : CHAQUE route /admin enregistrée et non autorisée répond 403 à un prospecteur.
func TestProspect_DefautRefusSurToutAdmin(t *testing.T) {
	app, db := newProspectApp(t)
	_, cookie := seedProspect(t, db, "alice")

	allowed := func(method, path string) bool {
		for _, r := range ProspectAllowedAdminRoutes {
			if r.Method == method && strings.TrimSuffix(r.Path, "/") == strings.TrimSuffix(path, "/") {
				return true
			}
		}
		return false
	}

	tested := 0
	for _, r := range app.GetRoutes(true) {
		if !strings.HasPrefix(r.Path, "/api/v1/admin/") || r.Method == "HEAD" || r.Method == "USE" {
			continue
		}
		if allowed(r.Method, r.Path) {
			continue
		}
		path := paramSeg.ReplaceAllString(r.Path, "x")
		status, _ := call(t, app, r.Method, path, cookie, "{}")
		assert.Equal(t, fiber.StatusForbidden, status, "%s %s doit être interdit au prospecteur", r.Method, r.Path)
		tested++
	}
	// Garde-fou du test lui-même : s'il n'énumère plus rien (préfixe changé…), il ne prouve rien.
	// 13 depuis l'ouverture de l'onglet Agent (5 routes de conversations passées dans la liste blanche).
	assert.GreaterOrEqual(t, tested, 12, "trop peu de routes énumérées — le test ne couvre plus la surface")

	// Et chaque route autorisée existe réellement (une entrée morte dans la liste serait un trou futur).
	registered := map[string]bool{}
	for _, r := range app.GetRoutes(true) {
		registered[r.Method+" "+strings.TrimSuffix(r.Path, "/")] = true
	}
	for _, r := range ProspectAllowedAdminRoutes {
		if r.Path == "/api/v1/admin/compose/verify" {
			continue // enregistrée seulement avec un service IA, absent ici
		}
		assert.True(t, registered[r.Method+" "+strings.TrimSuffix(r.Path, "/")], "route autorisée inexistante : %s %s", r.Method, r.Path)
	}
}

// La révocation coupe l'accès dès la requête SUIVANTE, avec le même cookie (toujours bien signé).
func TestProspect_RevocationImmediate(t *testing.T) {
	app, db := newProspectApp(t)
	p, cookie := seedProspect(t, db, "bob")

	status, me := call(t, app, "GET", "/api/v1/admin/me", cookie, "")
	require.Equal(t, 200, status)
	assert.Equal(t, "prospect", me["role"])
	status, _ = call(t, app, "GET", "/api/v1/admin/compose/drafts", cookie, "")
	require.Equal(t, 200, status)

	status, _ = call(t, app, "POST", "/api/v1/admin/prospects/"+p.ID+"/revoke", ownerCookie(), "")
	require.Equal(t, 200, status)

	status, _ = call(t, app, "GET", "/api/v1/admin/compose/drafts", cookie, "")
	assert.Equal(t, fiber.StatusUnauthorized, status)
	status, _ = call(t, app, "GET", "/api/v1/admin/me", cookie, "")
	assert.Equal(t, fiber.StatusUnauthorized, status)
	status, _ = call(t, app, "POST", "/api/v1/admin/prospect/login", "", `{"login":"bob","password":"motdepasse-solide"}`)
	assert.Equal(t, fiber.StatusUnauthorized, status, "un compte révoqué ne se reconnecte pas")

	// Réactivation : l'accès revient.
	status, _ = call(t, app, "POST", "/api/v1/admin/prospects/"+p.ID+"/activate", ownerCookie(), "")
	require.Equal(t, 200, status)
	status, _ = call(t, app, "GET", "/api/v1/admin/compose/drafts", cookie, "")
	assert.Equal(t, 200, status)
}

// Invitation : usage unique, puis login par nom ET par email ; une invitation expirée est refusée.
func TestProspect_InvitationUsageUniqueEtExpiration(t *testing.T) {
	app, db := newProspectApp(t)

	status, _ := call(t, app, "POST", "/api/v1/admin/prospects/invitations", "", `{"name":"Carla"}`)
	require.Equal(t, fiber.StatusUnauthorized, status, "inviter est réservé à l'owner")

	status, inv := call(t, app, "POST", "/api/v1/admin/prospects/invitations", ownerCookie(), `{"name":"Carla","email":"Carla@Ex.com"}`)
	require.Equal(t, 200, status)
	token := inv["token"].(string)
	assert.Len(t, inv["ref"], 6)

	status, info := call(t, app, "GET", "/api/v1/admin/invite/"+token, "", "")
	require.Equal(t, 200, status)
	assert.Equal(t, "Carla", info["name"])

	status, _ = call(t, app, "POST", "/api/v1/admin/invite/"+token, "", `{"password":"court"}`)
	assert.Equal(t, fiber.StatusBadRequest, status)
	status, _ = call(t, app, "POST", "/api/v1/admin/invite/"+token, "", `{"password":"un-mot-de-passe-long"}`)
	require.Equal(t, 200, status)
	status, _ = call(t, app, "POST", "/api/v1/admin/invite/"+token, "", `{"password":"un-autre-mot-de-passe"}`)
	assert.Equal(t, fiber.StatusGone, status, "le lien ne sert qu'une fois")

	status, _ = call(t, app, "POST", "/api/v1/admin/prospect/login", "", `{"login":"Carla","password":"un-mot-de-passe-long"}`)
	assert.Equal(t, 200, status)
	status, _ = call(t, app, "POST", "/api/v1/admin/prospect/login", "", `{"login":"carla@ex.com","password":"un-mot-de-passe-long"}`)
	assert.Equal(t, 200, status, "login par email")
	status, _ = call(t, app, "POST", "/api/v1/admin/prospect/login", "", `{"login":"Carla","password":"faux-mot-de-passe"}`)
	assert.Equal(t, fiber.StatusUnauthorized, status)

	// Expirée.
	status, inv = call(t, app, "POST", "/api/v1/admin/prospects/invitations", ownerCookie(), `{"name":"Dan"}`)
	require.Equal(t, 200, status)
	past := time.Now().Add(-time.Minute)
	require.NoError(t, db.Model(&models.Prospect{}).Where("name = ?", "Dan").Update("invite_expires_at", &past).Error)
	status, _ = call(t, app, "POST", "/api/v1/admin/invite/"+inv["token"].(string), "", `{"password":"un-mot-de-passe-long"}`)
	assert.Equal(t, fiber.StatusGone, status)
}

// Cloisonnement : A ne lit ni ne modifie l'atelier de B, ni ceux de l'owner ; l'owner ne voit pas
// ceux des prospecteurs dans SA liste.
func TestProspect_AteliersCloisonnes(t *testing.T) {
	app, db := newProspectApp(t)
	_, cookieA := seedProspect(t, db, "anna")
	_, cookieB := seedProspect(t, db, "boris")

	_, resB := call(t, app, "POST", "/api/v1/admin/compose/drafts", cookieB, `{"title":"B"}`)
	idB := resB["id"].(string)
	_, resO := call(t, app, "POST", "/api/v1/admin/compose/drafts", ownerCookie(), `{"title":"O"}`)
	idO := resO["id"].(string)
	_, resA := call(t, app, "POST", "/api/v1/admin/compose/drafts", cookieA, `{"title":"A"}`)
	idA := resA["id"].(string)

	for _, id := range []string{idB, idO} {
		status, _ := call(t, app, "GET", "/api/v1/admin/compose/drafts/"+id, cookieA, "")
		assert.Equal(t, fiber.StatusNotFound, status)
		status, _ = call(t, app, "PUT", "/api/v1/admin/compose/drafts/"+id, cookieA, `{"draft":"pirate"}`)
		assert.Equal(t, fiber.StatusNotFound, status)
		status, _ = call(t, app, "PATCH", "/api/v1/admin/compose/drafts/"+id+"/meta", cookieA, `{"title":"pirate"}`)
		assert.Equal(t, fiber.StatusNotFound, status)
	}
	var b models.ComposeDraft
	require.NoError(t, db.First(&b, "id = ?", idB).Error)
	assert.Equal(t, "B", b.Title, "l'atelier de B n'a pas bougé")

	status, list := call(t, app, "GET", "/api/v1/admin/compose/drafts", cookieA, "")
	require.Equal(t, 200, status)
	drafts := list["drafts"].([]interface{})
	require.Len(t, drafts, 1)
	assert.Equal(t, idA, drafts[0].(map[string]interface{})["id"])

	_, list = call(t, app, "GET", "/api/v1/admin/compose/drafts", ownerCookie(), "")
	drafts = list["drafts"].([]interface{})
	require.Len(t, drafts, 1)
	assert.Equal(t, idO, drafts[0].(map[string]interface{})["id"])
}

// Attribution : une visite ?ref= est comptée une fois par visiteur ; un CV généré est compté.
func TestProspect_Attribution(t *testing.T) {
	app, db := newProspectApp(t)
	p, _ := seedProspect(t, db, "eve")
	h := NewProspectsHandler(db, prospectTestSecret)

	for i := 0; i < 3; i++ {
		status, _ := call(t, app, "POST", "/api/v1/ref", "", `{"ref":"reve"}`)
		require.Equal(t, fiber.StatusNoContent, status)
	}
	status, _ := call(t, app, "POST", "/api/v1/ref", "", `{"ref":"inconnu"}`)
	assert.Equal(t, fiber.StatusNoContent, status, "un code inconnu ne se distingue pas")
	h.RecordCV(p.ID)

	status, list := call(t, app, "GET", "/api/v1/admin/prospects", ownerCookie(), "")
	require.Equal(t, 200, status)
	row := list["prospects"].([]interface{})[0].(map[string]interface{})
	assert.Equal(t, float64(1), row["visits"])
	assert.Equal(t, float64(1), row["cvs"])
}

// Onglet Agent : les conversations sont cloisonnées EXACTEMENT comme les ateliers. A ne lit, ne
// modifie ni ne supprime la conversation de B ni celle de l'owner (404 : on ne révèle pas qu'elle
// existe) ; chacun ne voit que les siennes dans sa liste, l'owner compris.
func TestProspect_ConversationsCloisonnees(t *testing.T) {
	app, db := newProspectApp(t)
	_, cookieA := seedProspect(t, db, "anna")
	_, cookieB := seedProspect(t, db, "boris")

	status, resB := call(t, app, "POST", "/api/v1/admin/chat/conversations", cookieB, "")
	require.Equal(t, 200, status, "le prospecteur doit pouvoir créer une conversation")
	idB := resB["id"].(string)
	_, resO := call(t, app, "POST", "/api/v1/admin/chat/conversations", ownerCookie(), "")
	idO := resO["id"].(string)
	_, resA := call(t, app, "POST", "/api/v1/admin/chat/conversations", cookieA, "")
	idA := resA["id"].(string)

	for _, id := range []string{idB, idO} {
		status, _ := call(t, app, "GET", "/api/v1/admin/chat/conversations/"+id, cookieA, "")
		assert.Equal(t, fiber.StatusNotFound, status)
		status, _ = call(t, app, "PUT", "/api/v1/admin/chat/conversations/"+id, cookieA, `{"title":"pirate"}`)
		assert.Equal(t, fiber.StatusNotFound, status)
		status, _ = call(t, app, "DELETE", "/api/v1/admin/chat/conversations/"+id, cookieA, "")
		assert.Equal(t, fiber.StatusNotFound, status)
	}
	var o models.ChatConversation
	require.NoError(t, db.First(&o, "id = ?", idO).Error, "la conversation owner survit")
	assert.NotEqual(t, "pirate", o.Title)

	// A lit, renomme et supprime la sienne.
	status, _ = call(t, app, "PUT", "/api/v1/admin/chat/conversations/"+idA, cookieA, `{"title":"mienne","messages":[]}`)
	assert.Equal(t, 200, status)
	status, got := call(t, app, "GET", "/api/v1/admin/chat/conversations/"+idA, cookieA, "")
	require.Equal(t, 200, status)
	assert.Equal(t, "mienne", got["title"])

	list := func(cookie string) []interface{} {
		status, l := call(t, app, "GET", "/api/v1/admin/chat/conversations", cookie, "")
		require.Equal(t, 200, status)
		return l["conversations"].([]interface{})
	}
	la := list(cookieA)
	require.Len(t, la, 1)
	assert.Equal(t, idA, la[0].(map[string]interface{})["id"])
	lo := list(ownerCookie())
	require.Len(t, lo, 1)
	assert.Equal(t, idO, lo[0].(map[string]interface{})["id"])

	status, _ = call(t, app, "DELETE", "/api/v1/admin/chat/conversations/"+idA, cookieA, "")
	assert.Equal(t, 200, status)
	assert.Len(t, list(cookieA), 0)
}
