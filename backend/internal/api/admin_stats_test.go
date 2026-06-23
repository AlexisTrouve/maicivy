package api

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// Verrouille le gate owner-only de GET /admin/stats. Le 401 est retourné AVANT tout accès db/redis,
// donc testable avec des deps nil. Le chemin 200 (vraies données) est vérifié en prod.
func newAdminStatsApp(secret string) *fiber.App {
	app := fiber.New()
	NewAdminStatsHandler(nil, nil, secret).RegisterRoutes(app.Group("/api/v1"))
	return app
}

func TestAdminStats_RequiresOwnerCookie(t *testing.T) {
	app := newAdminStatsApp("secret-hmac")

	// Sans cookie → 401
	resp, _ := app.Test(httptest.NewRequest("GET", "/api/v1/admin/stats", nil))
	if resp.StatusCode != 401 {
		t.Fatalf("sans cookie : attendu 401, got %d", resp.StatusCode)
	}

	// Cookie forgé → 401
	req := httptest.NewRequest("GET", "/api/v1/admin/stats", nil)
	req.Header.Set("Cookie", "maicivy_admin=admin:9999999999.signaturebidon")
	resp2, _ := app.Test(req)
	if resp2.StatusCode != 401 {
		t.Fatalf("cookie forgé : attendu 401, got %d", resp2.StatusCode)
	}
}
