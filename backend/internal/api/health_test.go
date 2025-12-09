package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// HealthHandlerTestSuite regroupe les tests du HealthHandler
type HealthHandlerTestSuite struct {
	suite.Suite
	db      *gorm.DB
	redis   *redis.Client
	handler *HealthHandler
	app     *fiber.App
}

func (suite *HealthHandlerTestSuite) SetupTest() {
	// Setup SQLite en mémoire
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(suite.T(), err)

	// Setup Redis mock (miniredis serait idéal, mais utilisons redis réel si disponible)
	// Pour les tests, on peut utiliser un client Redis pointant vers localhost
	// Si Redis n'est pas disponible, les tests deep health échoueront gracieusement
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // Utiliser DB 15 pour tests
	})

	suite.db = db
	suite.redis = redisClient
	suite.handler = NewHealthHandler(db, redisClient)

	// Setup Fiber app
	suite.app = fiber.New()
	suite.app.Get("/health", suite.handler.Health)
	suite.app.Get("/health/deep", suite.handler.HealthDeep)
}

func (suite *HealthHandlerTestSuite) TearDownTest() {
	// Cleanup
	if suite.db != nil {
		sqlDB, _ := suite.db.DB()
		sqlDB.Close()
	}
	if suite.redis != nil {
		suite.redis.Close()
	}
}

// Test GET /health - shallow health check
func (suite *HealthHandlerTestSuite) TestHealth_BasicCheck() {
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := suite.app.Test(req)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	var result HealthResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(suite.T(), "ok", result.Status)
	assert.Equal(suite.T(), "up", result.Services["api"])
}

// Test GET /health - toujours rapide
func (suite *HealthHandlerTestSuite) TestHealth_AlwaysReturnsOK() {
	// Même si DB/Redis down, health shallow devrait retourner OK
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := suite.app.Test(req)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	var result HealthResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(suite.T(), "ok", result.Status)
	assert.NotNil(suite.T(), result.Services)
}

// Test GET /health/deep - avec PostgreSQL up
func (suite *HealthHandlerTestSuite) TestHealthDeep_AllServicesUp() {
	// Note: Ce test peut échouer si Redis n'est pas disponible
	// C'est attendu dans un environnement de test sans Redis

	req := httptest.NewRequest("GET", "/health/deep", nil)
	resp, err := suite.app.Test(req, 5000) // 5s timeout

	assert.NoError(suite.T(), err)

	var result HealthResponse
	json.NewDecoder(resp.Body).Decode(&result)

	// API devrait toujours être up
	assert.Equal(suite.T(), "up", result.Services["api"])

	// PostgreSQL (SQLite en test) devrait être up
	assert.Equal(suite.T(), "up", result.Services["postgres"])

	// Redis peut être up ou down selon environnement
	if result.Services["redis"] == "up" {
		assert.Equal(suite.T(), "ok", result.Status)
		assert.Equal(suite.T(), 200, resp.StatusCode)
	} else {
		// Redis down = status degraded
		assert.Equal(suite.T(), "degraded", result.Status)
		assert.Equal(suite.T(), 503, resp.StatusCode)
	}
}

// Test GET /health/deep - avec PostgreSQL down
func (suite *HealthHandlerTestSuite) TestHealthDeep_PostgreSQLDown() {
	// Fermer la connexion DB pour simuler une panne
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()

	req := httptest.NewRequest("GET", "/health/deep", nil)
	resp, err := suite.app.Test(req, 5000)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 503, resp.StatusCode)

	var result HealthResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(suite.T(), "degraded", result.Status)
	assert.Equal(suite.T(), "down", result.Services["postgres"])
	assert.Equal(suite.T(), "up", result.Services["api"])
}

// Test GET /health/deep - timeout protection
func (suite *HealthHandlerTestSuite) TestHealthDeep_TimeoutProtection() {
	// Le handler a un timeout de 2s pour chaque check
	// Ce test vérifie que ça ne bloque pas indéfiniment

	start := time.Now()
	req := httptest.NewRequest("GET", "/health/deep", nil)
	resp, err := suite.app.Test(req, 10000) // 10s timeout global

	duration := time.Since(start)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)

	// Le handler devrait répondre en moins de 5 secondes
	// (2s timeout DB + 2s timeout Redis + overhead)
	assert.Less(suite.T(), duration.Seconds(), 5.0)
}

// Test GET /health/deep - response structure
func (suite *HealthHandlerTestSuite) TestHealthDeep_ResponseStructure() {
	req := httptest.NewRequest("GET", "/health/deep", nil)
	resp, err := suite.app.Test(req, 5000)

	assert.NoError(suite.T(), err)

	var result HealthResponse
	json.NewDecoder(resp.Body).Decode(&result)

	// Vérifier structure
	assert.NotEmpty(suite.T(), result.Status)
	assert.NotNil(suite.T(), result.Services)
	assert.Contains(suite.T(), result.Services, "api")
	assert.Contains(suite.T(), result.Services, "postgres")
	assert.Contains(suite.T(), result.Services, "redis")

	// Chaque service devrait être "up" ou "down"
	for service, status := range result.Services {
		assert.Contains(suite.T(), []string{"up", "down"}, status,
			"Service %s has invalid status: %s", service, status)
	}

	// Status global devrait être "ok" ou "degraded"
	assert.Contains(suite.T(), []string{"ok", "degraded"}, result.Status)
}

// Test NewHealthHandler
func (suite *HealthHandlerTestSuite) TestNewHealthHandler() {
	handler := NewHealthHandler(suite.db, suite.redis)

	assert.NotNil(suite.T(), handler)
	assert.Equal(suite.T(), suite.db, handler.db)
	assert.Equal(suite.T(), suite.redis, handler.redis)
}

// Run test suite
func TestHealthHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HealthHandlerTestSuite))
}

// Benchmark Health endpoint
func BenchmarkHealth(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})

	handler := NewHealthHandler(db, redisClient)
	app := fiber.New()
	app.Get("/health", handler.Health)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		app.Test(req)
	}
}

// Benchmark HealthDeep endpoint
func BenchmarkHealthDeep(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})

	handler := NewHealthHandler(db, redisClient)
	app := fiber.New()
	app.Get("/health/deep", handler.HealthDeep)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health/deep", nil)
		app.Test(req, 5000)
	}
}

// Test helper - mock Redis client qui échoue toujours
type FailingRedisClient struct {
	*redis.Client
}

func (f *FailingRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	cmd.SetErr(assert.AnError)
	return cmd
}

// Test GET /health/deep - avec Redis down spécifiquement
func TestHealthDeep_RedisDown(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:99999"}) // Port invalide

	handler := NewHealthHandler(db, redisClient)
	app := fiber.New()
	app.Get("/health/deep", handler.HealthDeep)

	req := httptest.NewRequest("GET", "/health/deep", nil)
	resp, err := app.Test(req, 5000)

	assert.NoError(t, err)
	assert.Equal(t, 503, resp.StatusCode)

	var result HealthResponse
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "degraded", result.Status)
	assert.Equal(t, "down", result.Services["redis"])
}
