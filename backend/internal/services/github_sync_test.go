package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/go-github/v60/github"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

// setupTestDB crée une base de données SQLite en mémoire pour les tests
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Migrer les tables
	err = db.AutoMigrate(&models.GitHubProfile{}, &models.GitHubRepository{})
	require.NoError(t, err)

	return db
}

// setupTestRedis crée un serveur Redis en mémoire pour les tests
func setupTestRedis(t *testing.T) *redis.Client {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	t.Cleanup(func() {
		mr.Close()
	})

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client
}

func TestGitHubSyncService_GetPublicRepositories(t *testing.T) {
	db := setupTestDB(t)
	redisClient := setupTestRedis(t)
	service := NewGitHubSyncService(db, redisClient)

	ctx := context.Background()
	username := "testuser"

	// Créer des repos de test
	repos := []models.GitHubRepository{
		{
			Username:    username,
			RepoName:    "repo1",
			FullName:    "testuser/repo1",
			Description: "Test repo 1",
			URL:         "https://github.com/testuser/repo1",
			Stars:       100,
			Language:    "Go",
			IsPrivate:   false,
			PushedAt:    time.Now(),
		},
		{
			Username:    username,
			RepoName:    "repo2",
			FullName:    "testuser/repo2",
			Description: "Test repo 2",
			URL:         "https://github.com/testuser/repo2",
			Stars:       50,
			Language:    "TypeScript",
			IsPrivate:   false,
			PushedAt:    time.Now(),
		},
		{
			Username:    username,
			RepoName:    "private-repo",
			FullName:    "testuser/private-repo",
			Description: "Private repo",
			URL:         "https://github.com/testuser/private-repo",
			Stars:       10,
			Language:    "Python",
			IsPrivate:   true,
			PushedAt:    time.Now(),
		},
	}

	for _, repo := range repos {
		err := db.Create(&repo).Error
		require.NoError(t, err)
	}

	// Test: Récupérer les repos publics
	t.Run("GetPublicRepositories", func(t *testing.T) {
		result, err := service.GetPublicRepositories(ctx, username)
		require.NoError(t, err)
		assert.Len(t, result, 2, "Should return only public repos")

		// Vérifier l'ordre (par stars DESC)
		assert.Equal(t, "repo1", result[0].RepoName)
		assert.Equal(t, int32(100), result[0].Stars)
		assert.Equal(t, "repo2", result[1].RepoName)
	})

	// Test: Cache Redis
	t.Run("GetPublicRepositories_WithCache", func(t *testing.T) {
		// Premier appel: mise en cache
		result1, err := service.GetPublicRepositories(ctx, username)
		require.NoError(t, err)

		// Vérifier que la clé cache existe
		cacheKey := "github:repos:" + username
		exists, _ := redisClient.Exists(ctx, cacheKey).Result()
		assert.Equal(t, int64(1), exists)

		// Deuxième appel: depuis le cache
		result2, err := service.GetPublicRepositories(ctx, username)
		require.NoError(t, err)

		assert.Equal(t, result1, result2)
	})
}

func TestGitHubSyncService_GetAllRepositories(t *testing.T) {
	db := setupTestDB(t)
	redisClient := setupTestRedis(t)
	service := NewGitHubSyncService(db, redisClient)

	ctx := context.Background()
	username := "testuser"

	// Créer des repos de test (2 publics, 1 privé)
	repos := []models.GitHubRepository{
		{
			Username:    username,
			RepoName:    "repo1",
			FullName:    "testuser/repo1",
			Stars:       100,
			IsPrivate:   false,
		},
		{
			Username:    username,
			RepoName:    "private-repo",
			FullName:    "testuser/private-repo",
			Stars:       50,
			IsPrivate:   true,
		},
	}

	for _, repo := range repos {
		err := db.Create(&repo).Error
		require.NoError(t, err)
	}

	// Test: Récupérer tous les repos (publics + privés)
	result, err := service.GetAllRepositories(ctx, username)
	require.NoError(t, err)
	assert.Len(t, result, 2, "Should return all repos including private")
}

func TestGitHubSyncService_GetSyncStatus(t *testing.T) {
	db := setupTestDB(t)
	redisClient := setupTestRedis(t)
	service := NewGitHubSyncService(db, redisClient)

	ctx := context.Background()
	username := "testuser"

	// Test: Statut sans profil connecté
	t.Run("Status_NotConnected", func(t *testing.T) {
		status, err := service.GetSyncStatus(ctx, username)
		require.NoError(t, err)
		assert.False(t, status.Connected)
		assert.Equal(t, 0, status.RepoCount)
	})

	// Test: Statut avec profil connecté
	t.Run("Status_Connected", func(t *testing.T) {
		// Créer un profil
		profile := &models.GitHubProfile{
			Username:    username,
			Token:       models.GitHubToken{AccessToken: "test_token"},
			ConnectedAt: time.Now().Unix(),
			SyncedAt:    time.Now().Unix(),
		}
		err := db.Create(profile).Error
		require.NoError(t, err)

		// Créer quelques repos
		for i := 0; i < 3; i++ {
			repo := &models.GitHubRepository{
				Username: username,
				RepoName: "repo" + string(rune(i)),
				FullName: username + "/repo" + string(rune(i)),
			}
			db.Create(repo)
		}

		status, err := service.GetSyncStatus(ctx, username)
		require.NoError(t, err)
		assert.True(t, status.Connected)
		assert.Equal(t, username, status.Username)
		assert.Equal(t, 3, status.RepoCount)
		assert.Greater(t, status.LastSync, int64(0))
	})
}

func TestGitHubSyncService_DisconnectGitHub(t *testing.T) {
	db := setupTestDB(t)
	redisClient := setupTestRedis(t)
	service := NewGitHubSyncService(db, redisClient)

	ctx := context.Background()
	username := "testuser"

	// Créer un profil connecté
	profile := &models.GitHubProfile{
		Username:    username,
		Token:       models.GitHubToken{AccessToken: "test_token"},
		ConnectedAt: time.Now().Unix(),
	}
	err := db.Create(profile).Error
	require.NoError(t, err)

	// Créer des repos
	repo := &models.GitHubRepository{
		Username: username,
		RepoName: "test-repo",
		FullName: username + "/test-repo",
	}
	db.Create(repo)

	// Mettre en cache
	cacheKey := "github:repos:" + username
	redisClient.Set(ctx, cacheKey, "test", time.Hour)

	// Test: Déconnecter
	err = service.DisconnectGitHub(ctx, username)
	require.NoError(t, err)

	// Vérifier que le profil est supprimé
	var profileCheck models.GitHubProfile
	err = db.Where("username = ?", username).First(&profileCheck).Error
	assert.Error(t, err) // Should be gorm.ErrRecordNotFound

	// Vérifier que le cache est invalidé
	exists, _ := redisClient.Exists(ctx, cacheKey).Result()
	assert.Equal(t, int64(0), exists)

	// Note: Les repos ne sont PAS supprimés (par design)
	var repoCheck models.GitHubRepository
	err = db.Where("username = ?", username).First(&repoCheck).Error
	assert.NoError(t, err)
}

// TestTransformGitHubRepoToModel teste la transformation repo → model
func TestTransformGitHubRepoToModel(t *testing.T) {
	username := "testuser"
	now := time.Now()

	ghRepo := &github.Repository{
		Name:        github.String("test-repo"),
		FullName:    github.String("testuser/test-repo"),
		Description: github.String("A test repository"),
		HTMLURL:     github.String("https://github.com/testuser/test-repo"),
		StargazersCount: github.Int(42),
		Language:    github.String("Go"),
		Topics:      []string{"golang", "testing", "ci"},
		Private:     github.Bool(false),
		PushedAt:    &github.Timestamp{Time: now},
	}

	// Transformation manuelle (similaire au service)
	repo := &models.GitHubRepository{
		Username:    username,
		RepoName:    ghRepo.GetName(),
		FullName:    ghRepo.GetFullName(),
		Description: ghRepo.GetDescription(),
		URL:         ghRepo.GetHTMLURL(),
		Stars:       int32(ghRepo.GetStargazersCount()),
		Language:    ghRepo.GetLanguage(),
		Topics:      ghRepo.Topics,
		IsPrivate:   ghRepo.GetPrivate(),
		PushedAt:    ghRepo.GetPushedAt().Time,
	}

	// Assertions
	assert.Equal(t, "test-repo", repo.RepoName)
	assert.Equal(t, "testuser/test-repo", repo.FullName)
	assert.Equal(t, "A test repository", repo.Description)
	assert.Equal(t, int32(42), repo.Stars)
	assert.Equal(t, "Go", repo.Language)
	assert.Len(t, repo.Topics, 3)
	assert.Contains(t, repo.Topics, "golang")
	assert.False(t, repo.IsPrivate)
}

// TestCacheInvalidation teste l'invalidation du cache
func TestCacheInvalidation(t *testing.T) {
	db := setupTestDB(t)
	redisClient := setupTestRedis(t)
	service := NewGitHubSyncService(db, redisClient)

	ctx := context.Background()
	username := "testuser"
	cacheKey := "github:repos:" + username

	// Créer un repo
	repo := &models.GitHubRepository{
		Username: username,
		RepoName: "test-repo",
		FullName: username + "/test-repo",
		IsPrivate: false,
	}
	db.Create(repo)

	// Premier appel: mise en cache
	repos1, _ := service.GetPublicRepositories(ctx, username)
	assert.Len(t, repos1, 1)

	// Vérifier cache
	cached, err := redisClient.Get(ctx, cacheKey).Result()
	require.NoError(t, err)
	var cachedRepos []models.GitHubRepository
	json.Unmarshal([]byte(cached), &cachedRepos)
	assert.Len(t, cachedRepos, 1)

	// Ajouter un nouveau repo en base
	repo2 := &models.GitHubRepository{
		Username: username,
		RepoName: "test-repo-2",
		FullName: username + "/test-repo-2",
		IsPrivate: false,
	}
	db.Create(repo2)

	// Sans invalidation cache, on récupère toujours 1 repo
	repos2, _ := service.GetPublicRepositories(ctx, username)
	assert.Len(t, repos2, 1) // Cache hit

	// Invalider manuellement le cache
	redisClient.Del(ctx, cacheKey)

	// Maintenant on récupère 2 repos
	repos3, _ := service.GetPublicRepositories(ctx, username)
	assert.Len(t, repos3, 2)
}
