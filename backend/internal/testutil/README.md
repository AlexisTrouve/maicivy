# Package testutil - Utilitaires de Test Backend

Package centralisé pour les helpers et fixtures de test.

## Utilisation

### Setup Database + Redis Mock

```go
import "maicivy/internal/testutil"

func TestSomething(t *testing.T) {
    // Setup DB + Redis mock
    db, redis, cleanup := testutil.SetupTestDB(t)
    defer cleanup()

    // Utiliser db et redis dans les tests
    // ...
}
```

### Setup Database Uniquement

```go
func TestDatabaseOnly(t *testing.T) {
    db, cleanup := testutil.SetupTestDBOnly(t)
    defer cleanup()

    // Utiliser db
    // ...
}
```

### Setup Redis Mock Uniquement

```go
func TestRedisOnly(t *testing.T) {
    redis, cleanup := testutil.SetupMiniredis(t)
    defer cleanup()

    // Utiliser redis
    // ...
}
```

## Fixtures

### Créer des Entités de Test

```go
import "maicivy/internal/testutil"

func TestWithFixtures(t *testing.T) {
    db, _, cleanup := testutil.SetupTestDB(t)
    defer cleanup()

    // Créer expérience de test
    exp := testutil.CreateTestExperience()
    db.Create(exp)

    // Créer skill de test
    skill := testutil.CreateTestSkill()
    db.Create(skill)

    // Créer projet de test
    project := testutil.CreateTestProject()
    db.Create(project)

    // Créer visiteur de test
    visitor := testutil.CreateTestVisitor("session-123")
    db.Create(visitor)

    // ...
}
```

## Fonctions Disponibles

### db.go

| Fonction | Description | Retour |
|----------|-------------|--------|
| `SetupTestDB(t)` | DB SQLite + miniredis | `*gorm.DB, *redis.Client, func()` |
| `SetupTestDBOnly(t)` | DB SQLite uniquement | `*gorm.DB, func()` |
| `SetupMiniredis(t)` | Redis mock uniquement | `*redis.Client, func()` |

### fixtures.go

| Fonction | Description | Retour |
|----------|-------------|--------|
| `CreateTestExperience()` | Expérience backend | `*models.Experience` |
| `CreateTestSkill()` | Compétence Go | `*models.Skill` |
| `CreateTestProject()` | Projet maicivy | `*models.Project` |
| `CreateTestVisitor(sessionID)` | Visiteur | `*models.Visitor` |
| `CreateTestGeneratedLetter(sessionID, company)` | Lettre | `*models.GeneratedLetter` |
| `CreateTestGitHubToken(sessionID)` | Token GitHub | `*models.GitHubToken` |
| `CreateTestGitHubProfile(sessionID)` | Profil GitHub | `*models.GitHubProfile` |

## Avantages

✅ **Pas de dépendances externes** - Utilise SQLite en mémoire + miniredis
✅ **Cleanup automatique** - Pas de fuites de ressources
✅ **Fixtures réutilisables** - Données cohérentes entre tests
✅ **Isolation** - Chaque test a sa propre DB/Redis
✅ **Rapide** - Tout en mémoire

## Exemples

### Test Handler avec DB + Redis

```go
func TestMyHandler(t *testing.T) {
    db, redis, cleanup := testutil.SetupTestDB(t)
    defer cleanup()

    // Créer fixtures
    visitor := testutil.CreateTestVisitor("test-session")
    db.Create(visitor)

    // Créer handler
    handler := NewMyHandler(db, redis)

    // Setup Fiber app
    app := fiber.New()
    app.Get("/test", handler.MyEndpoint)

    // Test request
    req := httptest.NewRequest("GET", "/test", nil)
    resp, err := app.Test(req)

    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
}
```

### Test Service avec Fixtures

```go
func TestMyService(t *testing.T) {
    db, _, cleanup := testutil.SetupTestDB(t)
    defer cleanup()

    // Seed data
    exp1 := testutil.CreateTestExperience()
    exp1.Company = "Company A"
    db.Create(exp1)

    exp2 := testutil.CreateTestExperience()
    exp2.Company = "Company B"
    db.Create(exp2)

    // Test service
    service := NewMyService(db)
    experiences, err := service.GetAllExperiences()

    assert.NoError(t, err)
    assert.Len(t, experiences, 2)
}
```

### Test avec miniredis Uniquement

```go
func TestCacheService(t *testing.T) {
    redis, cleanup := testutil.SetupMiniredis(t)
    defer cleanup()

    // Test cache
    cache := NewCacheService(redis)
    err := cache.Set("key", "value")

    assert.NoError(t, err)

    val, err := cache.Get("key")
    assert.NoError(t, err)
    assert.Equal(t, "value", val)
}
```

## Notes

- Tous les fichiers testutil ont le build tag `//go:build testing`
- SQLite est utilisé au lieu de PostgreSQL pour les tests (plus rapide)
- Miniredis simule Redis sans processus externe
- Cleanup est automatique avec `defer cleanup()`
