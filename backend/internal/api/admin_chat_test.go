package api

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// Verrouille le gate owner-only des routes de conversation. Le 401 est retourné AVANT tout accès db,
// donc testable avec db nil. Le CRUD réel (avec db) est vérifié en prod.
func newAdminChatApp(secret string) *fiber.App {
	app := fiber.New()
	NewAdminChatHandler(nil, secret).RegisterRoutes(app.Group("/api/v1"))
	return app
}

func TestAdminChat_RequiresOwnerCookie(t *testing.T) {
	app := newAdminChatApp("secret-hmac")

	cases := []struct {
		method, path string
	}{
		{"GET", "/api/v1/admin/chat/conversations"},
		{"POST", "/api/v1/admin/chat/conversations"},
		{"GET", "/api/v1/admin/chat/conversations/abc"},
		{"PUT", "/api/v1/admin/chat/conversations/abc"},
		{"DELETE", "/api/v1/admin/chat/conversations/abc"},
	}
	for _, tc := range cases {
		resp, _ := app.Test(httptest.NewRequest(tc.method, tc.path, nil))
		if resp.StatusCode != 401 {
			t.Fatalf("%s %s sans cookie : attendu 401, got %d", tc.method, tc.path, resp.StatusCode)
		}
	}
}
