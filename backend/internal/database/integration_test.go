//go:build integration
// +build integration

// backend/internal/database/integration_test.go
package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

type IntegrationTestSuite struct {
	suite.Suite
	pgContainer    *postgres.PostgresContainer
	redisContainer *redis.RedisContainer
	db             *DB
	cache          *Cache
}

// Setup avant toute la suite
func (suite *IntegrationTestSuite) SetupSuite() {
	ctx := context.Background()

	// Démarrer PostgreSQL container
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
	)
	suite.NoError(err)
	suite.pgContainer = pgContainer

	// Démarrer Redis container
	redisContainer, err := redis.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
	)
	suite.NoError(err)
	suite.redisContainer = redisContainer

	// Connect
	connStr, _ := pgContainer.ConnectionString(ctx)
	suite.db, err = NewDB(connStr)
	suite.NoError(err)

	redisAddr, _ := redisContainer.Endpoint(ctx, "")
	suite.cache, err = NewCache(redisAddr)
	suite.NoError(err)

	// Run migrations
	suite.db.AutoMigrate()
}

// Teardown après toute la suite
func (suite *IntegrationTestSuite) TearDownSuite() {
	ctx := context.Background()
	suite.pgContainer.Terminate(ctx)
	suite.redisContainer.Terminate(ctx)
}

// Reset DB avant chaque test
func (suite *IntegrationTestSuite) SetupTest() {
	suite.db.TruncateAll() // Helper pour vider tables
}

// Test création et récupération expérience
func (suite *IntegrationTestSuite) TestCreateAndGetExperience() {
	ctx := context.Background()

	// Create
	exp := &Experience{
		Title:       "Software Engineer",
		Company:     "Tech Corp",
		StartDate:   time.Now().AddDate(-2, 0, 0),
		Tags:        []string{"go", "postgresql"},
		Description: "Backend development",
	}

	err := suite.db.CreateExperience(ctx, exp)
	suite.NoError(err)
	suite.NotZero(exp.ID)

	// Get
	retrieved, err := suite.db.GetExperienceByID(ctx, exp.ID)
	suite.NoError(err)
	suite.Equal(exp.Title, retrieved.Title)
	suite.Equal(exp.Company, retrieved.Company)
	suite.ElementsMatch(exp.Tags, retrieved.Tags)
}

// Test cache Redis
func (suite *IntegrationTestSuite) TestRedisCache() {
	ctx := context.Background()

	key := "test:key"
	value := "test value"

	// Set
	err := suite.cache.Set(ctx, key, value, 5*time.Second)
	suite.NoError(err)

	// Get
	result, err := suite.cache.Get(ctx, key)
	suite.NoError(err)
	suite.Equal(value, result)

	// Expire
	time.Sleep(6 * time.Second)
	_, err = suite.cache.Get(ctx, key)
	suite.Error(err) // Key devrait avoir expiré
}

// Test transaction PostgreSQL
func (suite *IntegrationTestSuite) TestTransaction_Commit() {
	ctx := context.Background()

	// Begin transaction
	tx, err := suite.db.Begin(ctx)
	suite.NoError(err)

	// Create multiple records dans transaction
	exp1 := &Experience{Title: "Job 1", Company: "Company 1"}
	exp2 := &Experience{Title: "Job 2", Company: "Company 2"}

	err = tx.CreateExperience(ctx, exp1)
	suite.NoError(err)
	err = tx.CreateExperience(ctx, exp2)
	suite.NoError(err)

	// Commit
	err = tx.Commit()
	suite.NoError(err)

	// Vérifier données persistées
	experiences, err := suite.db.GetExperiences(ctx)
	suite.NoError(err)
	suite.Len(experiences, 2)
}

// Test transaction rollback
func (suite *IntegrationTestSuite) TestTransaction_Rollback() {
	ctx := context.Background()

	// Begin transaction
	tx, err := suite.db.Begin(ctx)
	suite.NoError(err)

	// Create record
	exp := &Experience{Title: "Temp Job", Company: "Temp Co"}
	err = tx.CreateExperience(ctx, exp)
	suite.NoError(err)

	// Rollback
	err = tx.Rollback()
	suite.NoError(err)

	// Vérifier données NON persistées
	experiences, err := suite.db.GetExperiences(ctx)
	suite.NoError(err)
	suite.Len(experiences, 0, "Rollback devrait annuler création")
}

// Test query avec filtres
func (suite *IntegrationTestSuite) TestQueryWithFilters() {
	ctx := context.Background()

	// Create test data
	exps := []*Experience{
		{Title: "Backend Dev", Company: "TechCo", Tags: []string{"go", "postgresql"}},
		{Title: "Frontend Dev", Company: "DesignCo", Tags: []string{"react", "typescript"}},
		{Title: "Full-Stack Dev", Company: "StartupCo", Tags: []string{"go", "react"}},
	}

	for _, exp := range exps {
		suite.db.CreateExperience(ctx, exp)
	}

	// Query avec filtre tags
	filtered, err := suite.db.GetExperiencesByTag(ctx, "go")
	suite.NoError(err)
	suite.Len(filtered, 2, "Devrait retourner 2 expériences avec tag 'go'")
}

// Test pagination
func (suite *IntegrationTestSuite) TestPagination() {
	ctx := context.Background()

	// Create 15 experiences
	for i := 1; i <= 15; i++ {
		exp := &Experience{
			Title:   fmt.Sprintf("Job %d", i),
			Company: "Company",
		}
		suite.db.CreateExperience(ctx, exp)
	}

	// Page 1 (10 items)
	page1, err := suite.db.GetExperiencesPaginated(ctx, 1, 10)
	suite.NoError(err)
	suite.Len(page1, 10)

	// Page 2 (5 items)
	page2, err := suite.db.GetExperiencesPaginated(ctx, 2, 10)
	suite.NoError(err)
	suite.Len(page2, 5)
}

// Test indexes performance
func (suite *IntegrationTestSuite) TestIndexPerformance() {
	ctx := context.Background()

	// Create 1000 experiences
	for i := 1; i <= 1000; i++ {
		exp := &Experience{
			Title:   fmt.Sprintf("Job %d", i),
			Company: "BigCorp",
			Tags:    []string{"go", "react", "docker"},
		}
		suite.db.CreateExperience(ctx, exp)
	}

	// Query avec index (devrait être rapide)
	start := time.Now()
	_, err := suite.db.GetExperiencesByTag(ctx, "go")
	duration := time.Since(start)

	suite.NoError(err)
	suite.Less(duration, 100*time.Millisecond, "Query avec index devrait être < 100ms")
}

// Test Redis rate limiting
func (suite *IntegrationTestSuite) TestRedisRateLimiting() {
	ctx := context.Background()

	key := "ratelimit:test:session"
	limit := 5

	// Tente 10 increments
	var successCount int
	for i := 0; i < 10; i++ {
		count, err := suite.cache.Incr(ctx, key)
		suite.NoError(err)

		if count <= int64(limit) {
			successCount++
		}
	}

	suite.Equal(limit, successCount, "Devrait autoriser exactement 5 requêtes")
}

// Test Redis pub/sub (analytics temps réel)
func (suite *IntegrationTestSuite) TestRedisPubSub() {
	ctx := context.Background()

	channel := "analytics:events"
	message := "visitor_connected"

	// Subscribe
	sub := suite.cache.Subscribe(ctx, channel)
	defer sub.Close()

	// Publish
	err := suite.cache.Publish(ctx, channel, message)
	suite.NoError(err)

	// Receive
	select {
	case msg := <-sub.Channel():
		suite.Equal(message, msg.Payload)
	case <-time.After(2 * time.Second):
		suite.Fail("Timeout attente message pub/sub")
	}
}

func TestIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(IntegrationTestSuite))
}
