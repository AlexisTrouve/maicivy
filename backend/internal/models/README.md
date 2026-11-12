# Models - GORM

Ce dossier contient tous les models GORM pour le projet maicivy.

## Structure

- `base.go` - BaseModel avec champs communs (ID, timestamps, soft delete)
- `experience.go` - Expériences professionnelles
- `skill.go` - Compétences techniques
- `project.go` - Projets réalisés
- `visitor.go` - Tracking visiteurs
- `generated_letter.go` - Lettres IA générées
- `analytics_event.go` - Événements analytics

## Utilisation

### Créer un nouvel enregistrement

```go
import "maicivy/internal/models"

// Créer une expérience
exp := models.Experience{
    Title:        "Senior Developer",
    Company:      "TechCorp",
    StartDate:    time.Now(),
    Category:     "backend",
    Technologies: pq.StringArray{"Go", "PostgreSQL"},
}

db.Create(&exp)
```

### Lire des enregistrements

```go
// Trouver par ID
var exp models.Experience
db.First(&exp, "id = ?", id)

// Filtrer par catégorie
var experiences []models.Experience
db.Where("category = ?", "backend").Find(&experiences)

// Avec relations (preload)
var visitor models.Visitor
db.Preload("GeneratedLetters").First(&visitor, "id = ?", id)
```

### Mettre à jour

```go
// Mise à jour partielle
db.Model(&exp).Update("title", "New Title")

// Mise à jour complète
exp.Title = "New Title"
db.Save(&exp)
```

### Supprimer (Soft Delete)

```go
// Soft delete (met deleted_at)
db.Delete(&exp)

// Trouver avec soft deleted
db.Unscoped().Find(&experiences)

// Hard delete (suppression définitive)
db.Unscoped().Delete(&exp)
```

## Features Communes

### BaseModel

Tous les models héritent de `BaseModel` qui fournit :

- `ID` (UUID v4)
- `CreatedAt` (timestamp auto)
- `UpdatedAt` (timestamp auto via trigger)
- `DeletedAt` (soft delete)

### Validation

Les models utilisent les tags de validation :

```go
Title string `validate:"required,min=3,max=255"`
```

Pour valider :

```go
import "github.com/go-playground/validator/v10"

validate := validator.New()
err := validate.Struct(exp)
```

### PostgreSQL Arrays

Pour les champs array (technologies, tags) :

```go
import "github.com/lib/pq"

Technologies: pq.StringArray{"Go", "PostgreSQL", "Redis"}
```

### JSONB

Pour les champs JSONB (company_info, event_data) :

```go
// Stocker
companyInfo := `{"name": "TechCorp", "size": 100}`
letter.CompanyInfo = companyInfo

// Parser
var data map[string]interface{}
json.Unmarshal([]byte(letter.CompanyInfo), &data)
```

## Enums

### SkillLevel

- `beginner`
- `intermediate`
- `advanced`
- `expert`

Utilisation :

```go
skill.Level = models.SkillLevelExpert
```

### ProfileType

- `unknown`
- `recruiter`
- `tech_lead`
- `cto`
- `ceo`
- `developer`

### LetterType

- `motivation`
- `anti_motivation`

### EventType

- `page_view`
- `cv_theme_change`
- `letter_generate`
- `pdf_download`
- `button_click`
- `link_click`
- `form_submit`

## Helper Methods

### Visitor

```go
// Vérifier accès IA
if visitor.HasAccessToAI() {
    // Autoriser génération lettre
}

// Vérifier profil cible
if visitor.IsTargetProfile() {
    // Afficher fonctionnalités premium
}

// Incrémenter visites
visitor.IncrementVisit()
db.Save(&visitor)
```

### Experience

```go
// Vérifier emploi actuel
if exp.IsCurrentJob() {
    // Afficher "Présent"
}

// Calculer durée
duration := exp.Duration()
years := duration.Hours() / 24 / 365
```

### Skill

```go
// Obtenir score numérique
score := skill.LevelScore() // 1-4
```

### Project

```go
// Vérifier GitHub
if project.HasGithub() {
    // Afficher lien GitHub
}

// Vérifier démo
if project.HasDemo() {
    // Afficher lien démo
}
```

### GeneratedLetter

```go
// Vérifier type
if letter.IsMotivation() {
    // Logique motivation
}

// Calculer coût
cost := letter.EstimatedCost() // USD
```

### AnalyticsEvent

```go
// Vérifier page view
if event.IsPageView() {
    // Incrémenter compteur
}

// Vérifier conversion
if event.IsConversion() {
    // Tracking conversion
}
```

## Tests

Voir `visitor_test.go` pour exemples de tests unitaires.

Pour tester les models :

```bash
cd backend
go test ./internal/models -v
```

## Notes Importantes

1. **UUID Generation** : Automatique via PostgreSQL `gen_random_uuid()`
2. **Timezone** : Toujours UTC
3. **Soft Deletes** : Utiliser `Unscoped()` pour voir les enregistrements supprimés
4. **Arrays PostgreSQL** : Utiliser `pq.StringArray`, pas `[]string` natif Go
5. **JSONB** : Stocker en string, parser avec `json.Unmarshal`
6. **Foreign Keys** : GORM gère automatiquement les relations
7. **Validation** : Utiliser validator/v10 pour valider avant `Create/Update`
