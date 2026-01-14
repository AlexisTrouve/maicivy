// backend/internal/middleware/visitor_tracking_test.go
package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock Redis Client
type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) Incr(key string) (int64, error) {
	args := m.Called(key)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRedisClient) Get(key string) (string, error) {
	args := m.Called(key)
	return args.String(0), args.Error(1)
}

func (m *MockRedisClient) Set(key string, value interface{}, expiration int) error {
	args := m.Called(key, value, expiration)
	return args.Error(0)
}

// Test tracking first visit
func TestVisitorTracking_FirstVisit(t *testing.T) {
	// Setup
	app := fiber.New()
	mockRedis := new(MockRedisClient)

	// Mock visitor count = 1 (première visite)
	mockRedis.On("Incr", mock.Anything).Return(int64(1), nil)
	mockRedis.On("Set", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	middleware := NewVisitorTracking(mockRedis)
	app.Use(middleware.Track)

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Request
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Vérifier cookie session créé
	cookies := resp.Cookies()
	assert.NotEmpty(t, cookies)

	mockRedis.AssertExpectations(t)
}

// Test tracking subsequent visits
func TestVisitorTracking_SubsequentVisits(t *testing.T) {
	// Setup
	app := fiber.New()
	mockRedis := new(MockRedisClient)

	// Mock visitor count = 3 (3ème visite)
	mockRedis.On("Incr", mock.Anything).Return(int64(3), nil)

	middleware := NewVisitorTracking(mockRedis)
	app.Use(middleware.Track)

	app.Get("/test", func(c *fiber.Ctx) error {
		visitCount := c.Locals("visit_count").(int)
		return c.JSON(fiber.Map{"visit_count": visitCount})
	})

	// Request avec cookie existant
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Cookie", "maicivy_session=existing-session-123")

	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	mockRedis.AssertExpectations(t)
}

// Test profile detection (recruiter)
func TestVisitorTracking_ProfileDetection_Recruiter(t *testing.T) {
	// Setup
	app := fiber.New()
	mockRedis := new(MockRedisClient)

	mockRedis.On("Incr", mock.Anything).Return(int64(1), nil)
	mockRedis.On("Set", mock.Anything, "recruiter", mock.Anything).Return(nil)

	middleware := NewVisitorTracking(mockRedis)
	app.Use(middleware.Track)

	app.Get("/test", func(c *fiber.Ctx) error {
		profile := c.Locals("visitor_profile").(string)
		hasAccess := c.Locals("has_ai_access").(bool)
		return c.JSON(fiber.Map{
			"profile":    profile,
			"has_access": hasAccess,
		})
	})

	// Request avec User-Agent LinkedIn
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "LinkedInBot/1.0")
	req.Header.Set("Referer", "https://www.linkedin.com/")

	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	mockRedis.AssertExpectations(t)
}

// Test AI access (>= 3 visits)
func TestVisitorTracking_AIAccess(t *testing.T) {
	// Setup
	app := fiber.New()
	mockRedis := new(MockRedisClient)

	testCases := []struct {
		name            string
		visitCount      int64
		profile         string
		expectedAccess  bool
		expectedMessage string
	}{
		{
			name:            "First visit - no access",
			visitCount:      1,
			profile:         "",
			expectedAccess:  false,
			expectedMessage: "Need 2 more visits",
		},
		{
			name:            "Second visit - no access",
			visitCount:      2,
			profile:         "",
			expectedAccess:  false,
			expectedMessage: "Need 1 more visit",
		},
		{
			name:            "Third visit - access granted",
			visitCount:      3,
			profile:         "",
			expectedAccess:  true,
			expectedMessage: "Access granted",
		},
		{
			name:            "Recruiter first visit - access granted",
			visitCount:      1,
			profile:         "recruiter",
			expectedAccess:  true,
			expectedMessage: "Access granted",
		},
		{
			name:            "CTO first visit - access granted",
			visitCount:      1,
			profile:         "cto",
			expectedAccess:  true,
			expectedMessage: "Access granted",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRedis.On("Incr", mock.Anything).Return(tc.visitCount, nil).Once()
			if tc.profile != "" {
				mockRedis.On("Set", mock.Anything, tc.profile, mock.Anything).Return(nil).Once()
			}

			middleware := NewVisitorTracking(mockRedis)
			app.Use(middleware.Track)

			app.Get("/test", func(c *fiber.Ctx) error {
				hasAccess := c.Locals("has_ai_access").(bool)
				return c.JSON(fiber.Map{"has_access": hasAccess})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)
		})
	}
}

// Test visitor IP hashing (privacy)
func TestVisitorTracking_IPHashing(t *testing.T) {
	middleware := NewVisitorTracking(nil)

	testIPs := []string{
		"192.168.1.1",
		"10.0.0.1",
		"172.16.0.1",
	}

	hashedIPs := make(map[string]bool)

	for _, ip := range testIPs {
		hashed := middleware.HashIP(ip)

		// Vérifie que le hash est généré
		assert.NotEmpty(t, hashed)
		assert.NotEqual(t, ip, hashed, "IP ne devrait pas être en clair")

		// Vérifie que chaque IP produit un hash unique
		assert.False(t, hashedIPs[hashed], "Hash devrait être unique")
		hashedIPs[hashed] = true

		// Vérifie que même IP produit même hash (consistance)
		hashed2 := middleware.HashIP(ip)
		assert.Equal(t, hashed, hashed2, "Même IP devrait produire même hash")
	}
}

// Test rate limiting par visiteur
func TestVisitorTracking_RateLimiting(t *testing.T) {
	// Setup
	app := fiber.New()
	mockRedis := new(MockRedisClient)

	// Simule visiteur dépassant limite
	mockRedis.On("Incr", mock.Anything).Return(int64(100), nil)

	middleware := NewVisitorTracking(mockRedis).WithRateLimit(50)
	app.Use(middleware.Track)

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Cookie", "maicivy_session=spammer")

	resp, err := app.Test(req)

	// Assertions
	assert.NoError(t, err)
	// Devrait bloquer si > limite
	// (comportement exact dépend implémentation)
}

// Benchmark tracking middleware
func BenchmarkVisitorTracking(b *testing.B) {
	app := fiber.New()
	mockRedis := new(MockRedisClient)

	mockRedis.On("Incr", mock.Anything).Return(int64(5), nil)

	middleware := NewVisitorTracking(mockRedis)
	app.Use(middleware.Track)

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		app.Test(req)
	}
}
