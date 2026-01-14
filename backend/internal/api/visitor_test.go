package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"maicivy/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// VisitorHandlerTestSuite regroupe les tests du VisitorHandler
type VisitorHandlerTestSuite struct {
	suite.Suite
	db      *gorm.DB
	redis   *redis.Client
	handler *VisitorHandler
	app     *fiber.App
}

func (suite *VisitorHandlerTestSuite) SetupTest() {
	// Setup SQLite en mémoire
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(suite.T(), err)

	// Migrer models
	db.AutoMigrate(&models.Visitor{})

	// Setup Redis mock
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // DB 15 pour tests
	})

	suite.db = db
	suite.redis = redisClient
	suite.handler = NewVisitorHandler(db, redisClient)

	// Setup Fiber app
	suite.app = fiber.New()
	suite.app.Get("/api/v1/visitor/status", suite.handler.GetVisitorStatus)
}

func (suite *VisitorHandlerTestSuite) TearDownTest() {
	if suite.db != nil {
		sqlDB, _ := suite.db.DB()
		sqlDB.Close()
	}
	if suite.redis != nil {
		suite.redis.Close()
	}
}

// Test GET /api/v1/visitor/status - visiteur avec 1 visite
func (suite *VisitorHandlerTestSuite) TestGetVisitorStatus_FirstVisit() {
	// Créer visitor dans DB
	visitor := &models.Visitor{
		SessionID:       "session_first_visit",
		VisitCount:      1,
		FirstVisit:      time.Now(),
		LastVisit:       time.Now(),
		ProfileDetected: models.ProfileTypeUnknown,
	}
	suite.db.Create(visitor)

	// Request avec session_id dans context
	req := httptest.NewRequest("GET", "/api/v1/visitor/status", nil)

	// Simuler middleware qui set session_id dans context
	suite.app.Use(func(c *fiber.Ctx) error {
		c.Locals("session_id", "session_first_visit")
		return c.Next()
	})

	resp, err := suite.app.Test(req)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	var result VisitorStatusResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(suite.T(), "session_first_visit", result.SessionID)
	assert.Equal(suite.T(), 1, result.VisitCount)
	assert.Equal(suite.T(), "unknown", result.ProfileDetected)
	assert.False(suite.T(), result.HasAccessToAI) // Seulement 1 visite
	assert.False(suite.T(), result.IsTargetProfile)
}

// Test GET /api/v1/visitor/status - visiteur avec 3+ visites
func (suite *VisitorHandlerTestSuite) TestGetVisitorStatus_HasAccess() {
	// Créer visitor avec 3+ visites
	visitor := &models.Visitor{
		SessionID:       "session_access",
		VisitCount:      3,
		FirstVisit:      time.Now().AddDate(0, 0, -7), // Il y a 7 jours
		LastVisit:       time.Now(),
		ProfileDetected: models.ProfileTypeUnknown,
	}
	suite.db.Create(visitor)

	// Setup middleware pour ce test
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session_id", "session_access")
		return c.Next()
	})
	app.Get("/api/v1/visitor/status", suite.handler.GetVisitorStatus)

	req := httptest.NewRequest("GET", "/api/v1/visitor/status", nil)
	resp, err := app.Test(req)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	var result VisitorStatusResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(suite.T(), 3, result.VisitCount)
	assert.True(suite.T(), result.HasAccessToAI) // 3 visites = accès
	assert.False(suite.T(), result.IsTargetProfile) // Unknown != cible
}

// Test GET /api/v1/visitor/status - recruteur (profil cible)
func (suite *VisitorHandlerTestSuite) TestGetVisitorStatus_Recruiter() {
	// Créer visitor recruteur
	visitor := &models.Visitor{
		SessionID:           "session_recruiter",
		VisitCount:          1,
		FirstVisit:          time.Now(),
		LastVisit:           time.Now(),
		ProfileDetected:     models.ProfileTypeRecruiter,
		ProfileType:         "recruiter",
		DetectionConfidence: 95,
		CompanyName:         "TechCorp",
	}
	suite.db.Create(visitor)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session_id", "session_recruiter")
		return c.Next()
	})
	app.Get("/api/v1/visitor/status", suite.handler.GetVisitorStatus)

	req := httptest.NewRequest("GET", "/api/v1/visitor/status", nil)
	resp, err := app.Test(req)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	var result VisitorStatusResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(suite.T(), "recruiter", result.ProfileDetected)
	assert.True(suite.T(), result.HasAccessToAI) // Recruteur = accès immédiat
	assert.True(suite.T(), result.IsTargetProfile) // Recruteur = cible
}

// Test GET /api/v1/visitor/status - sans session
func (suite *VisitorHandlerTestSuite) TestGetVisitorStatus_NoSession() {
	app := fiber.New()
	// Pas de middleware pour set session_id
	app.Get("/api/v1/visitor/status", suite.handler.GetVisitorStatus)

	req := httptest.NewRequest("GET", "/api/v1/visitor/status", nil)
	resp, err := app.Test(req)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 404, resp.StatusCode)

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Contains(suite.T(), result["error"], "No session")
}

// Test GET /api/v1/visitor/status - session introuvable en DB
func (suite *VisitorHandlerTestSuite) TestGetVisitorStatus_NotFoundInDB() {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session_id", "non_existing_session")
		return c.Next()
	})
	app.Get("/api/v1/visitor/status", suite.handler.GetVisitorStatus)

	req := httptest.NewRequest("GET", "/api/v1/visitor/status", nil)
	resp, err := app.Test(req)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 404, resp.StatusCode)

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Contains(suite.T(), result["error"], "Visitor not found")
}

// Test GET /api/v1/visitor/status - CTO (profil cible)
func (suite *VisitorHandlerTestSuite) TestGetVisitorStatus_CTO() {
	visitor := &models.Visitor{
		SessionID:           "session_cto",
		VisitCount:          1,
		FirstVisit:          time.Now(),
		LastVisit:           time.Now(),
		ProfileDetected:     models.ProfileTypeCTO,
		DetectionConfidence: 80,
		CompanyName:         "StartupCo",
	}
	suite.db.Create(visitor)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session_id", "session_cto")
		return c.Next()
	})
	app.Get("/api/v1/visitor/status", suite.handler.GetVisitorStatus)

	req := httptest.NewRequest("GET", "/api/v1/visitor/status", nil)
	resp, err := app.Test(req)

	assert.NoError(suite.T(), err)

	var result VisitorStatusResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(suite.T(), "cto", result.ProfileDetected)
	assert.True(suite.T(), result.HasAccessToAI)
	assert.True(suite.T(), result.IsTargetProfile)
}

// Test GET /api/v1/visitor/status - Developer (pas cible)
func (suite *VisitorHandlerTestSuite) TestGetVisitorStatus_Developer() {
	visitor := &models.Visitor{
		SessionID:       "session_dev",
		VisitCount:      2,
		FirstVisit:      time.Now(),
		LastVisit:       time.Now(),
		ProfileDetected: models.ProfileTypeDeveloper,
	}
	suite.db.Create(visitor)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session_id", "session_dev")
		return c.Next()
	})
	app.Get("/api/v1/visitor/status", suite.handler.GetVisitorStatus)

	req := httptest.NewRequest("GET", "/api/v1/visitor/status", nil)
	resp, err := app.Test(req)

	assert.NoError(suite.T(), err)

	var result VisitorStatusResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(suite.T(), "developer", result.ProfileDetected)
	assert.False(suite.T(), result.HasAccessToAI) // Seulement 2 visites
	assert.False(suite.T(), result.IsTargetProfile) // Developer != cible
}

// Test NewVisitorHandler
func (suite *VisitorHandlerTestSuite) TestNewVisitorHandler() {
	handler := NewVisitorHandler(suite.db, suite.redis)

	assert.NotNil(suite.T(), handler)
	assert.Equal(suite.T(), suite.db, handler.db)
	assert.Equal(suite.T(), suite.redis, handler.redis)
}

// Run test suite
func TestVisitorHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(VisitorHandlerTestSuite))
}

// Benchmark GetVisitorStatus
func BenchmarkGetVisitorStatus(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.Visitor{})

	visitor := &models.Visitor{
		SessionID:       "bench_session",
		VisitCount:      3,
		FirstVisit:      time.Now(),
		LastVisit:       time.Now(),
		ProfileDetected: models.ProfileTypeRecruiter,
	}
	db.Create(visitor)

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})
	handler := NewVisitorHandler(db, redisClient)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session_id", "bench_session")
		return c.Next()
	})
	app.Get("/api/v1/visitor/status", handler.GetVisitorStatus)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/visitor/status", nil)
		app.Test(req)
	}
}

// Test avec UUID valide
func TestVisitorStatus_WithRealUUID(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.Visitor{})

	visitor := &models.Visitor{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		SessionID:       "session_uuid",
		VisitCount:      5,
		FirstVisit:      time.Now(),
		LastVisit:       time.Now(),
		ProfileDetected: models.ProfileTypeRecruiter,
	}
	db.Create(visitor)

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})
	handler := NewVisitorHandler(db, redisClient)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session_id", "session_uuid")
		return c.Next()
	})
	app.Get("/api/v1/visitor/status", handler.GetVisitorStatus)

	req := httptest.NewRequest("GET", "/api/v1/visitor/status", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result VisitorStatusResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, 5, result.VisitCount)
	assert.True(t, result.HasAccessToAI)
}
