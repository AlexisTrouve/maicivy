package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func newAdminApp(password, secret string) *fiber.App {
	app := fiber.New()
	NewAdminHandler(password, secret).RegisterRoutes(app.Group("/api/v1"))
	return app
}

func postLogin(app *fiber.App, password string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/api/v1/admin/login", strings.NewReader(`{"password":"`+password+`"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	rec := httptest.NewRecorder()
	rec.Code = resp.StatusCode
	rec.Header().Set("Set-Cookie", resp.Header.Get("Set-Cookie"))
	return rec
}

func TestAdminLogin_CorrectPassword(t *testing.T) {
	app := newAdminApp("hunter2", "secret-hmac")
	rec := postLogin(app, "hunter2")
	if rec.Code != 200 {
		t.Fatalf("attendu 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Set-Cookie"), adminCookieName+"=admin:") {
		t.Fatalf("cookie admin signé non posé: %q", rec.Header().Get("Set-Cookie"))
	}
}

func TestAdminLogin_WrongPassword(t *testing.T) {
	app := newAdminApp("hunter2", "secret-hmac")
	rec := postLogin(app, "nope")
	if rec.Code != 401 {
		t.Fatalf("attendu 401 sur mauvais mot de passe, got %d", rec.Code)
	}
	if strings.Contains(rec.Header().Get("Set-Cookie"), adminCookieName+"=admin:") {
		t.Fatal("aucun cookie ne doit être posé sur échec")
	}
}

func TestAdminLogin_DisabledWhenUnconfigured(t *testing.T) {
	// ADMIN_PASSWORD vide → login désactivé, 401 systématique (pas de porte ouverte).
	app := newAdminApp("", "secret-hmac")
	rec := postLogin(app, "")
	if rec.Code != 401 {
		t.Fatalf("login doit être désactivé (401) si ADMIN_PASSWORD vide, got %d", rec.Code)
	}
}

func TestAdminMe_RequiresValidCookie(t *testing.T) {
	app := newAdminApp("hunter2", "secret-hmac")

	// 1. login → récupère le cookie (1er segment "name=value", sans les attributs)
	login := postLogin(app, "hunter2")
	cookie := strings.Split(login.Header().Get("Set-Cookie"), ";")[0]

	// 2. /me AVEC cookie → 200
	meReq := httptest.NewRequest("GET", "/api/v1/admin/me", nil)
	meReq.Header.Set("Cookie", cookie)
	meResp, _ := app.Test(meReq)
	if meResp.StatusCode != 200 {
		t.Fatalf("/me avec cookie valide → attendu 200, got %d", meResp.StatusCode)
	}

	// 3. /me SANS cookie → 401
	bareResp, _ := app.Test(httptest.NewRequest("GET", "/api/v1/admin/me", nil))
	if bareResp.StatusCode != 401 {
		t.Fatalf("/me sans cookie → attendu 401, got %d", bareResp.StatusCode)
	}
}
