# 03. DATABASE SCHEMA

## 📋 Métadonnées

- **Phase:** 1
- **Priorité:** CRITIQUE
- **Complexité:** ⭐⭐⭐⭐ (4/5)
- **Prérequis:** 02. BACKEND_FOUNDATION.md
- **Temps estimé:** 3-4 jours
- **Status:** 🔲 À faire

---

## 🎯 Objectif

Définir et implémenter le schéma de base de données complet pour maicivy, incluant :
- Models GORM pour les 6 tables PostgreSQL définies dans PROJECT_SPEC.md
- Relations et associations entre entités
- Migrations SQL avec golang-migrate
- Indexes pour optimiser les performances
- Constraints pour garantir l'intégrité des données
- Seed data pour le développement et les tests
- Stratégie de versioning du schéma

---

## 🏗️ Architecture

### Vue d'Ensemble

```
PostgreSQL Database: maicivy
│
├── experiences         # Parcours professionnel
├── skills             # Compétences techniques
├── projects           # Projets réalisés
├── generated_letters  # Historique lettres IA
├── visitors           # Tracking visiteurs
└── analytics_events   # Événements analytics
```

### Design Decisions

1. **GORM comme ORM** : Facilite les opérations CRUD et gère automatiquement les migrations
2. **UUID pour les IDs** : Meilleure sécurité et distribution pour scaling futur
3. **JSONB pour données flexibles** : Arrays PostgreSQL natifs pour tags/technologies
4. **Soft deletes** : Conservation historique via `deleted_at`
5. **Timestamps automatiques** : `created_at`, `updated_at` via GORM hooks
6. **Indexes stratégiques** : Optimisation des requêtes fréquentes (filtrage par tags, dates)

---

## 📦 Dépendances

### Bibliothèques Go

```bash
# ORM et driver PostgreSQL
go get -u gorm.io/gorm
go get -u gorm.io/driver/postgres

# Migrations
go get -u github.com/golang-migrate/migrate/v4
go get -u github.com/golang-migrate/migrate/v4/database/postgres
go get -u github.com/golang-migrate/migrate/v4/source/file

# UUID
go get -u github.com/google/uuid

# Validation
go get -u github.com/go-playground/validator/v10
```

### Services Externes

- PostgreSQL 15+ (support avancé JSONB et indexes)

---

## 🔨 Implémentation

### Étape 1: Structure des Models GORM

**Description:** Créer les models Go correspondant aux tables PostgreSQL définies dans PROJECT_SPEC.md

**Fichier:** `backend/internal/models/base.go`

```go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel contient les champs communs à tous les models
type BaseModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// BeforeCreate hook pour générer UUID si non fourni
func (base *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
	return nil
}
```

**Explications:**
- `BaseModel` évite la duplication de code pour ID et timestamps
- `gorm.DeletedAt` active le soft delete automatique
- UUID v4 par défaut via PostgreSQL `gen_random_uuid()`
- Hook `BeforeCreate` comme fallback si UUID non généré côté DB

---

### Étape 2: Model Experience

**Fichier:** `backend/internal/models/experience.go`

```go
package models

import (
	"time"

	"github.com/lib/pq"
)

// Experience représente une expérience professionnelle
type Experience struct {
	BaseModel

	// Informations principales
	Title       string    `gorm:"type:varchar(255);not null" json:"title" validate:"required,min=3,max=255"`
	Company     string    `gorm:"type:varchar(255);not null" json:"company" validate:"required,min=2,max=255"`
	Description string    `gorm:"type:text" json:"description" validate:"max=5000"`
	StartDate   time.Time `gorm:"not null" json:"start_date" validate:"required"`
	EndDate     *time.Time `json:"end_date"` // Nullable pour emploi actuel

	// Catégorisation et filtrage
	Technologies pq.StringArray `gorm:"type:text[]" json:"technologies"` // PostgreSQL array
	Tags         pq.StringArray `gorm:"type:text[]" json:"tags"`
	Category     string         `gorm:"type:varchar(100);index" json:"category" validate:"required,oneof=backend frontend fullstack devops data ai mobile other"`

	// Métadonnées
	Featured bool `gorm:"default:false" json:"featured"` // Pour mise en avant
}

// TableName override le nom de table par défaut
func (Experience) TableName() string {
	return "experiences"
}

// IsCurrentJob vérifie si c'est l'emploi actuel
func (e *Experience) IsCurrentJob() bool {
	return e.EndDate == nil
}

// Duration calcule la durée de l'expérience
func (e *Experience) Duration() time.Duration {
	endDate := time.Now()
	if e.EndDate != nil {
		endDate = *e.EndDate
	}
	return endDate.Sub(e.StartDate)
}
```

**Explications:**
- `pq.StringArray` : Type PostgreSQL natif pour arrays de strings
- `*time.Time` pour `EndDate` : Permet NULL pour emplois actuels
- `Category` avec validation enum pour cohérence
- `Featured` pour marquer expériences clés
- Méthodes helper pour logique métier

---

### Étape 3: Model Skill

**Fichier:** `backend/internal/models/skill.go`

```go
package models

import (
	"github.com/lib/pq"
)

// SkillLevel représente le niveau de maîtrise
type SkillLevel string

const (
	SkillLevelBeginner     SkillLevel = "beginner"
	SkillLevelIntermediate SkillLevel = "intermediate"
	SkillLevelAdvanced     SkillLevel = "advanced"
	SkillLevelExpert       SkillLevel = "expert"
)

// Skill représente une compétence technique
type Skill struct {
	BaseModel

	// Informations principales
	Name            string         `gorm:"type:varchar(100);not null;uniqueIndex" json:"name" validate:"required,min=2,max=100"`
	Level           SkillLevel     `gorm:"type:varchar(20);not null" json:"level" validate:"required,oneof=beginner intermediate advanced expert"`
	Category        string         `gorm:"type:varchar(100);index" json:"category" validate:"required"`
	Tags            pq.StringArray `gorm:"type:text[]" json:"tags"`
	YearsExperience int            `gorm:"default:0" json:"years_experience" validate:"min=0,max=50"`

	// Métadonnées
	Description string `gorm:"type:text" json:"description" validate:"max=500"`
	Featured    bool   `gorm:"default:false" json:"featured"`
	Icon        string `gorm:"type:varchar(100)" json:"icon"` // Icon name (ex: "golang", "react")
}

// TableName override le nom de table par défaut
func (Skill) TableName() string {
	return "skills"
}

// LevelScore retourne un score numérique pour le niveau
func (s *Skill) LevelScore() int {
	switch s.Level {
	case SkillLevelExpert:
		return 4
	case SkillLevelAdvanced:
		return 3
	case SkillLevelIntermediate:
		return 2
	case SkillLevelBeginner:
		return 1
	default:
		return 0
	}
}
```

**Explications:**
- Enum `SkillLevel` avec constantes pour type-safety
- `uniqueIndex` sur `Name` : Pas de doublons de compétences
- `Category` pour regroupement (Languages, Frameworks, Tools, etc.)
- `Icon` pour affichage visuel avec icônes de logos
- Méthode `LevelScore()` pour algorithme de scoring CV

---

### Étape 4: Model Project

**Fichier:** `backend/internal/models/project.go`

```go
package models

import (
	"github.com/lib/pq"
)

// Project représente un projet réalisé
type Project struct {
	BaseModel

	// Informations principales
	Title       string `gorm:"type:varchar(255);not null" json:"title" validate:"required,min=3,max=255"`
	Description string `gorm:"type:text" json:"description" validate:"max=5000"`

	// URLs
	GithubURL string `gorm:"type:varchar(500)" json:"github_url" validate:"omitempty,url"`
	DemoURL   string `gorm:"type:varchar(500)" json:"demo_url" validate:"omitempty,url"`
	ImageURL  string `gorm:"type:varchar(500)" json:"image_url" validate:"omitempty,url"`

	// Catégorisation
	Technologies pq.StringArray `gorm:"type:text[]" json:"technologies"`
	Category     string         `gorm:"type:varchar(100);index" json:"category" validate:"required"`

	// Métadonnées GitHub (synced automatiquement)
	GithubStars    int    `gorm:"default:0" json:"github_stars"`
	GithubForks    int    `gorm:"default:0" json:"github_forks"`
	GithubLanguage string `gorm:"type:varchar(50)" json:"github_language"`

	// Flags
	Featured   bool `gorm:"default:false" json:"featured"`
	InProgress bool `gorm:"default:false" json:"in_progress"`
}

// TableName override le nom de table par défaut
func (Project) TableName() string {
	return "projects"
}

// HasGithub vérifie si le projet a un repo GitHub
func (p *Project) HasGithub() bool {
	return p.GithubURL != ""
}

// HasDemo vérifie si le projet a une démo live
func (p *Project) HasDemo() bool {
	return p.DemoURL != ""
}
```

**Explications:**
- Champs GitHub pour import automatique via API GitHub
- `Featured` pour projets à mettre en avant
- `InProgress` pour projets en cours de développement
- Validation URL avec tag `url`
- Helper methods pour checks booléens

---

### Étape 5: Model GeneratedLetter

**Fichier:** `backend/internal/models/generated_letter.go`

```go
package models

import (
	"github.com/google/uuid"
)

// LetterType représente le type de lettre générée
type LetterType string

const (
	LetterTypeMotivation     LetterType = "motivation"
	LetterTypeAntiMotivation LetterType = "anti_motivation"
)

// GeneratedLetter représente une lettre générée par IA
type GeneratedLetter struct {
	BaseModel

	// Relation avec visiteur
	VisitorID uuid.UUID `gorm:"type:uuid;not null;index" json:"visitor_id"`
	Visitor   *Visitor  `gorm:"foreignKey:VisitorID" json:"visitor,omitempty"`

	// Informations lettre
	CompanyName string     `gorm:"type:varchar(255);not null" json:"company_name" validate:"required,min=2,max=255"`
	LetterType  LetterType `gorm:"type:varchar(20);not null" json:"letter_type" validate:"required,oneof=motivation anti_motivation"`
	Content     string     `gorm:"type:text;not null" json:"content" validate:"required"`

	// Métadonnées génération
	AIModel      string `gorm:"type:varchar(50)" json:"ai_model"`       // "claude-3" ou "gpt-4"
	TokensUsed   int    `gorm:"default:0" json:"tokens_used"`           // Tracking coûts
	GenerationMS int    `gorm:"default:0" json:"generation_ms"`         // Temps de génération en ms
	CompanyInfo  string `gorm:"type:jsonb" json:"company_info"`         // Données entreprise scrapées (JSON)

	// Flags
	Downloaded bool `gorm:"default:false" json:"downloaded"` // Tracking PDF téléchargé
}

// TableName override le nom de table par défaut
func (GeneratedLetter) TableName() string {
	return "generated_letters"
}

// IsMotivation vérifie si c'est une lettre de motivation
func (gl *GeneratedLetter) IsMotivation() bool {
	return gl.LetterType == LetterTypeMotivation
}

// EstimatedCost calcule le coût estimé en USD
func (gl *GeneratedLetter) EstimatedCost() float64 {
	// Exemple: Claude Sonnet ~$3 / 1M input tokens, ~$15 / 1M output tokens
	// Estimation simplifiée : $0.01 par 1000 tokens
	return float64(gl.TokensUsed) * 0.00001
}
```

**Explications:**
- Foreign key vers `Visitor` pour tracking
- `LetterType` enum pour les 2 types de lettres
- `CompanyInfo` en JSONB pour flexibilité des données scrapées
- Metrics de performance (`GenerationMS`) et coûts (`TokensUsed`)
- `Downloaded` pour analytics utilisation PDF

---

### Étape 6: Model Visitor

**Fichier:** `backend/internal/models/visitor.go`

```go
package models

import (
	"time"
)

// ProfileType représente le type de profil détecté
type ProfileType string

const (
	ProfileTypeUnknown   ProfileType = "unknown"
	ProfileTypeRecruiter ProfileType = "recruiter"
	ProfileTypeTechLead  ProfileType = "tech_lead"
	ProfileTypeCTO       ProfileType = "cto"
	ProfileTypeCEO       ProfileType = "ceo"
	ProfileTypeDeveloper ProfileType = "developer"
)

// Visitor représente un visiteur unique du site
type Visitor struct {
	BaseModel

	// Identification
	SessionID string `gorm:"type:varchar(255);uniqueIndex;not null" json:"session_id" validate:"required"`
	IPHash    string `gorm:"type:varchar(64);index" json:"ip_hash"` // Hash SHA256 de l'IP (RGPD)

	// User Agent parsing
	UserAgent string `gorm:"type:text" json:"user_agent"`
	Browser   string `gorm:"type:varchar(100)" json:"browser"`
	OS        string `gorm:"type:varchar(100)" json:"os"`
	Device    string `gorm:"type:varchar(50)" json:"device"` // mobile, tablet, desktop

	// Tracking visites
	VisitCount int       `gorm:"default:1" json:"visit_count"`
	FirstVisit time.Time `gorm:"not null" json:"first_visit"`
	LastVisit  time.Time `gorm:"not null" json:"last_visit"`

	// Détection profil
	ProfileDetected ProfileType `gorm:"type:varchar(50);default:'unknown'" json:"profile_detected"`
	CompanyName     string      `gorm:"type:varchar(255)" json:"company_name"` // Via IP lookup ou LinkedIn
	LinkedInURL     string      `gorm:"type:varchar(500)" json:"linkedin_url"` // Si détecté via referrer

	// Géolocalisation (via IP)
	Country     string `gorm:"type:varchar(100)" json:"country"`
	City        string `gorm:"type:varchar(100)" json:"city"`
	Timezone    string `gorm:"type:varchar(50)" json:"timezone"`

	// Relations
	GeneratedLetters []GeneratedLetter `gorm:"foreignKey:VisitorID" json:"generated_letters,omitempty"`
	AnalyticsEvents  []AnalyticsEvent  `gorm:"foreignKey:VisitorID" json:"analytics_events,omitempty"`
}

// TableName override le nom de table par défaut
func (Visitor) TableName() string {
	return "visitors"
}

// HasAccessToAI vérifie si le visiteur a accès aux fonctionnalités IA
func (v *Visitor) HasAccessToAI() bool {
	// Règle: 3+ visites OU profil cible détecté
	if v.VisitCount >= 3 {
		return true
	}

	targetProfiles := []ProfileType{
		ProfileTypeRecruiter,
		ProfileTypeTechLead,
		ProfileTypeCTO,
		ProfileTypeCEO,
	}

	for _, profile := range targetProfiles {
		if v.ProfileDetected == profile {
			return true
		}
	}

	return false
}

// IsTargetProfile vérifie si c'est un profil cible (recruteur, etc.)
func (v *Visitor) IsTargetProfile() bool {
	return v.ProfileDetected != ProfileTypeUnknown && v.ProfileDetected != ProfileTypeDeveloper
}

// IncrementVisit incrémente le compteur de visites
func (v *Visitor) IncrementVisit() {
	v.VisitCount++
	v.LastVisit = time.Now()
}
```

**Explications:**
- `SessionID` unique index pour lookup rapide
- `IPHash` au lieu de stocker IP brute (RGPD compliance)
- Détection User-Agent parsée en Browser/OS/Device
- `ProfileDetected` avec enum pour profils cibles
- Géolocalisation basique via IP lookup
- Relations `has_many` avec letters et events
- **Méthode clé:** `HasAccessToAI()` implémente la logique d'accès IA

---

### Étape 7: Model AnalyticsEvent

**Fichier:** `backend/internal/models/analytics_event.go`

```go
package models

import (
	"github.com/google/uuid"
)

// EventType représente les types d'événements trackés
type EventType string

const (
	EventTypePageView       EventType = "page_view"
	EventTypeCVThemeChange  EventType = "cv_theme_change"
	EventTypeLetterGenerate EventType = "letter_generate"
	EventTypePDFDownload    EventType = "pdf_download"
	EventTypeButtonClick    EventType = "button_click"
	EventTypeLinkClick      EventType = "link_click"
	EventTypeFormSubmit     EventType = "form_submit"
)

// AnalyticsEvent représente un événement analytics
type AnalyticsEvent struct {
	BaseModel

	// Relation avec visiteur
	VisitorID uuid.UUID `gorm:"type:uuid;not null;index" json:"visitor_id"`
	Visitor   *Visitor  `gorm:"foreignKey:VisitorID" json:"visitor,omitempty"`

	// Type d'événement
	EventType EventType `gorm:"type:varchar(50);not null;index" json:"event_type" validate:"required"`

	// Données événement (flexible JSONB)
	EventData string `gorm:"type:jsonb" json:"event_data"`

	// Contexte page
	PageURL  string `gorm:"type:varchar(500)" json:"page_url"`
	Referrer string `gorm:"type:varchar(500)" json:"referrer"`

	// Métadonnées
	SessionDuration int `gorm:"default:0" json:"session_duration"` // En secondes
}

// TableName override le nom de table par défaut
func (AnalyticsEvent) TableName() string {
	return "analytics_events"
}

// IsPageView vérifie si c'est un page view
func (ae *AnalyticsEvent) IsPageView() bool {
	return ae.EventType == EventTypePageView
}

// IsConversion vérifie si c'est un événement de conversion
func (ae *AnalyticsEvent) IsConversion() bool {
	conversionEvents := []EventType{
		EventTypeLetterGenerate,
		EventTypePDFDownload,
	}

	for _, event := range conversionEvents {
		if ae.EventType == event {
			return true
		}
	}

	return false
}
```

**Explications:**
- `EventType` enum pour catégoriser événements
- `EventData` en JSONB pour flexibilité (ex: `{"theme": "backend"}`)
- Index sur `EventType` et `VisitorID` pour agrégations rapides
- `SessionDuration` pour analytics engagement
- Méthodes helper pour filtrage événements

---

### Étape 8: Database Connection & Auto-Migration

**Fichier:** `backend/internal/database/postgres.go`

```go
package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"maicivy/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// ConnectPostgres initialise la connexion PostgreSQL
func ConnectPostgres() error {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SSLMODE"),
	)

	// Configuration GORM
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	// Connexion
	db, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configuration connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = db
	log.Println("✅ PostgreSQL connected successfully")

	return nil
}

// AutoMigrate exécute les migrations automatiques GORM
func AutoMigrate() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	// Liste de tous les models à migrer
	models := []interface{}{
		&models.Experience{},
		&models.Skill{},
		&models.Project{},
		&models.Visitor{},
		&models.GeneratedLetter{},
		&models.AnalyticsEvent{},
	}

	// Auto-migration
	if err := DB.AutoMigrate(models...); err != nil {
		return fmt.Errorf("auto-migration failed: %w", err)
	}

	log.Println("✅ Database auto-migration completed")
	return nil
}

// Close ferme la connexion database
func Close() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}
```

**Explications:**
- Lecture config depuis variables d'environnement
- Connection pooling configuré (10 idle, 100 max)
- Logger GORM en mode Info pour dev (à ajuster en prod)
- `AutoMigrate()` crée/modifie tables automatiquement
- Timezone UTC forcée pour cohérence

---

### Étape 9: Migrations SQL Manuelles

**Description:** Créer des migrations SQL pour contrôle fin (alternative/complément à AutoMigrate)

**Fichier:** `backend/migrations/000001_init_schema.up.sql`

```sql
-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Table: experiences
CREATE TABLE IF NOT EXISTS experiences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    company VARCHAR(255) NOT NULL,
    description TEXT,
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP,
    technologies TEXT[],
    tags TEXT[],
    category VARCHAR(100) NOT NULL,
    featured BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Index sur category pour filtrage
CREATE INDEX idx_experiences_category ON experiences(category);
CREATE INDEX idx_experiences_deleted_at ON experiences(deleted_at);

-- Table: skills
CREATE TABLE IF NOT EXISTS skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    level VARCHAR(20) NOT NULL CHECK (level IN ('beginner', 'intermediate', 'advanced', 'expert')),
    category VARCHAR(100) NOT NULL,
    tags TEXT[],
    years_experience INTEGER DEFAULT 0,
    description TEXT,
    featured BOOLEAN DEFAULT FALSE,
    icon VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Index sur category et level
CREATE INDEX idx_skills_category ON skills(category);
CREATE INDEX idx_skills_level ON skills(level);
CREATE INDEX idx_skills_deleted_at ON skills(deleted_at);

-- Table: projects
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    github_url VARCHAR(500),
    demo_url VARCHAR(500),
    image_url VARCHAR(500),
    technologies TEXT[],
    category VARCHAR(100) NOT NULL,
    github_stars INTEGER DEFAULT 0,
    github_forks INTEGER DEFAULT 0,
    github_language VARCHAR(50),
    featured BOOLEAN DEFAULT FALSE,
    in_progress BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Index sur category et featured
CREATE INDEX idx_projects_category ON projects(category);
CREATE INDEX idx_projects_featured ON projects(featured);
CREATE INDEX idx_projects_deleted_at ON projects(deleted_at);

-- Table: visitors
CREATE TABLE IF NOT EXISTS visitors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id VARCHAR(255) NOT NULL UNIQUE,
    ip_hash VARCHAR(64),
    user_agent TEXT,
    browser VARCHAR(100),
    os VARCHAR(100),
    device VARCHAR(50),
    visit_count INTEGER DEFAULT 1,
    first_visit TIMESTAMP NOT NULL,
    last_visit TIMESTAMP NOT NULL,
    profile_detected VARCHAR(50) DEFAULT 'unknown',
    company_name VARCHAR(255),
    linkedin_url VARCHAR(500),
    country VARCHAR(100),
    city VARCHAR(100),
    timezone VARCHAR(50),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Index sur session_id (lookup primaire), ip_hash, profile
CREATE UNIQUE INDEX idx_visitors_session_id ON visitors(session_id);
CREATE INDEX idx_visitors_ip_hash ON visitors(ip_hash);
CREATE INDEX idx_visitors_profile_detected ON visitors(profile_detected);
CREATE INDEX idx_visitors_deleted_at ON visitors(deleted_at);

-- Table: generated_letters
CREATE TABLE IF NOT EXISTS generated_letters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    visitor_id UUID NOT NULL REFERENCES visitors(id) ON DELETE CASCADE,
    company_name VARCHAR(255) NOT NULL,
    letter_type VARCHAR(20) NOT NULL CHECK (letter_type IN ('motivation', 'anti_motivation')),
    content TEXT NOT NULL,
    ai_model VARCHAR(50),
    tokens_used INTEGER DEFAULT 0,
    generation_ms INTEGER DEFAULT 0,
    company_info JSONB,
    downloaded BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Index sur visitor_id et letter_type
CREATE INDEX idx_generated_letters_visitor_id ON generated_letters(visitor_id);
CREATE INDEX idx_generated_letters_letter_type ON generated_letters(letter_type);
CREATE INDEX idx_generated_letters_created_at ON generated_letters(created_at DESC);
CREATE INDEX idx_generated_letters_deleted_at ON generated_letters(deleted_at);

-- Table: analytics_events
CREATE TABLE IF NOT EXISTS analytics_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    visitor_id UUID NOT NULL REFERENCES visitors(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB,
    page_url VARCHAR(500),
    referrer VARCHAR(500),
    session_duration INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Index sur visitor_id, event_type et created_at (analytics queries)
CREATE INDEX idx_analytics_events_visitor_id ON analytics_events(visitor_id);
CREATE INDEX idx_analytics_events_event_type ON analytics_events(event_type);
CREATE INDEX idx_analytics_events_created_at ON analytics_events(created_at DESC);
CREATE INDEX idx_analytics_events_deleted_at ON analytics_events(deleted_at);

-- Index composite pour requêtes fréquentes
CREATE INDEX idx_analytics_events_type_date ON analytics_events(event_type, created_at DESC);

-- Trigger pour updated_at automatique
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_experiences_updated_at BEFORE UPDATE ON experiences FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_skills_updated_at BEFORE UPDATE ON skills FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON projects FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_visitors_updated_at BEFORE UPDATE ON visitors FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_generated_letters_updated_at BEFORE UPDATE ON generated_letters FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_analytics_events_updated_at BEFORE UPDATE ON analytics_events FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

**Fichier:** `backend/migrations/000001_init_schema.down.sql`

```sql
-- Rollback de la migration initiale
DROP TRIGGER IF EXISTS update_analytics_events_updated_at ON analytics_events;
DROP TRIGGER IF EXISTS update_generated_letters_updated_at ON generated_letters;
DROP TRIGGER IF EXISTS update_visitors_updated_at ON visitors;
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS update_skills_updated_at ON skills;
DROP TRIGGER IF EXISTS update_experiences_updated_at ON experiences;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS analytics_events;
DROP TABLE IF EXISTS generated_letters;
DROP TABLE IF EXISTS visitors;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS skills;
DROP TABLE IF EXISTS experiences;

DROP EXTENSION IF EXISTS "uuid-ossp";
```

**Explications:**
- UUID extension pour `gen_random_uuid()`
- Contraintes CHECK sur enums (`level`, `letter_type`)
- Foreign keys avec `ON DELETE CASCADE` pour intégrité référentielle
- Trigger PostgreSQL pour `updated_at` automatique
- Indexes stratégiques sur colonnes de filtrage/tri
- Migration down pour rollback propre

---

### Étape 10: Seed Data

**Fichier:** `backend/scripts/seed.go`

```go
package main

import (
	"log"
	"time"

	"maicivy/internal/database"
	"maicivy/internal/models"

	"github.com/lib/pq"
)

func main() {
	// Connexion DB
	if err := database.ConnectPostgres(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	log.Println("🌱 Starting database seeding...")

	seedExperiences()
	seedSkills()
	seedProjects()

	log.Println("✅ Database seeding completed!")
}

func seedExperiences() {
	experiences := []models.Experience{
		{
			Title:        "Senior Backend Developer",
			Company:      "TechCorp Inc.",
			Description:  "Led backend architecture migration from monolith to microservices using Go and Kubernetes.",
			StartDate:    time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:      nil, // Emploi actuel
			Technologies: pq.StringArray{"Go", "PostgreSQL", "Redis", "Kubernetes", "Docker"},
			Tags:         pq.StringArray{"backend", "microservices", "devops"},
			Category:     "backend",
			Featured:     true,
		},
		{
			Title:        "Full-Stack Developer",
			Company:      "StartupXYZ",
			Description:  "Developed complete SaaS platform with React frontend and Node.js backend.",
			StartDate:    time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC),
			EndDate:      ptrTime(time.Date(2021, 12, 31, 0, 0, 0, 0, time.UTC)),
			Technologies: pq.StringArray{"React", "Node.js", "TypeScript", "MongoDB", "AWS"},
			Tags:         pq.StringArray{"fullstack", "saas", "cloud"},
			Category:     "fullstack",
			Featured:     false,
		},
		{
			Title:        "C++ Developer",
			Company:      "GameDev Studios",
			Description:  "Implemented game engine features and optimized rendering pipeline.",
			StartDate:    time.Date(2018, 3, 1, 0, 0, 0, 0, time.UTC),
			EndDate:      ptrTime(time.Date(2020, 5, 31, 0, 0, 0, 0, time.UTC)),
			Technologies: pq.StringArray{"C++", "OpenGL", "Vulkan", "CMake"},
			Tags:         pq.StringArray{"cpp", "gamedev", "graphics"},
			Category:     "other",
			Featured:     false,
		},
	}

	for _, exp := range experiences {
		if err := database.DB.Create(&exp).Error; err != nil {
			log.Printf("⚠️  Failed to seed experience %s: %v", exp.Title, err)
		} else {
			log.Printf("✅ Seeded experience: %s at %s", exp.Title, exp.Company)
		}
	}
}

func seedSkills() {
	skills := []models.Skill{
		{Name: "Go", Level: models.SkillLevelExpert, Category: "Languages", Tags: pq.StringArray{"backend", "performance"}, YearsExperience: 5, Icon: "golang", Featured: true},
		{Name: "PostgreSQL", Level: models.SkillLevelAdvanced, Category: "Databases", Tags: pq.StringArray{"backend", "sql"}, YearsExperience: 6, Icon: "postgresql", Featured: true},
		{Name: "Redis", Level: models.SkillLevelAdvanced, Category: "Databases", Tags: pq.StringArray{"backend", "cache"}, YearsExperience: 4, Icon: "redis", Featured: true},
		{Name: "Docker", Level: models.SkillLevelAdvanced, Category: "DevOps", Tags: pq.StringArray{"devops", "containers"}, YearsExperience: 5, Icon: "docker", Featured: true},
		{Name: "Kubernetes", Level: models.SkillLevelIntermediate, Category: "DevOps", Tags: pq.StringArray{"devops", "orchestration"}, YearsExperience: 3, Icon: "kubernetes", Featured: false},
		{Name: "TypeScript", Level: models.SkillLevelAdvanced, Category: "Languages", Tags: pq.StringArray{"frontend", "backend"}, YearsExperience: 5, Icon: "typescript", Featured: true},
		{Name: "React", Level: models.SkillLevelAdvanced, Category: "Frameworks", Tags: pq.StringArray{"frontend"}, YearsExperience: 5, Icon: "react", Featured: true},
		{Name: "Next.js", Level: models.SkillLevelAdvanced, Category: "Frameworks", Tags: pq.StringArray{"frontend", "fullstack"}, YearsExperience: 3, Icon: "nextjs", Featured: true},
		{Name: "C++", Level: models.SkillLevelIntermediate, Category: "Languages", Tags: pq.StringArray{"systems", "performance"}, YearsExperience: 4, Icon: "cplusplus", Featured: false},
		{Name: "Python", Level: models.SkillLevelIntermediate, Category: "Languages", Tags: pq.StringArray{"backend", "ai", "scripting"}, YearsExperience: 4, Icon: "python", Featured: false},
	}

	for _, skill := range skills {
		if err := database.DB.Create(&skill).Error; err != nil {
			log.Printf("⚠️  Failed to seed skill %s: %v", skill.Name, err)
		} else {
			log.Printf("✅ Seeded skill: %s (%s)", skill.Name, skill.Level)
		}
	}
}

func seedProjects() {
	projects := []models.Project{
		{
			Title:          "maicivy - AI-Powered CV",
			Description:    "Interactive CV with AI-generated motivation/anti-motivation letters. Built with Go, Next.js, PostgreSQL, and Redis.",
			GithubURL:      "https://github.com/yourusername/maicivy",
			DemoURL:        "https://maicivy.example.com",
			Technologies:   pq.StringArray{"Go", "Next.js", "PostgreSQL", "Redis", "Docker", "AI"},
			Category:       "fullstack",
			GithubStars:    42,
			GithubLanguage: "Go",
			Featured:       true,
			InProgress:     true,
		},
		{
			Title:          "Microservices E-Commerce",
			Description:    "Scalable e-commerce platform with microservices architecture, event-driven communication, and CQRS pattern.",
			GithubURL:      "https://github.com/yourusername/ecommerce-microservices",
			Technologies:   pq.StringArray{"Go", "gRPC", "Kafka", "PostgreSQL", "Kubernetes"},
			Category:       "backend",
			GithubStars:    156,
			GithubLanguage: "Go",
			Featured:       true,
			InProgress:     false,
		},
		{
			Title:          "Real-Time Chat Application",
			Description:    "WebSocket-based chat with rooms, private messages, typing indicators, and message history.",
			GithubURL:      "https://github.com/yourusername/realtime-chat",
			DemoURL:        "https://chat.example.com",
			Technologies:   pq.StringArray{"Node.js", "Socket.io", "React", "MongoDB", "Redis"},
			Category:       "fullstack",
			GithubStars:    89,
			GithubLanguage: "TypeScript",
			Featured:       false,
			InProgress:     false,
		},
	}

	for _, project := range projects {
		if err := database.DB.Create(&project).Error; err != nil {
			log.Printf("⚠️  Failed to seed project %s: %v", project.Title, err)
		} else {
			log.Printf("✅ Seeded project: %s", project.Title)
		}
	}
}

// Helper pour créer *time.Time
func ptrTime(t time.Time) *time.Time {
	return &t
}
```

**Explications:**
- Script standalone pour peupler DB avec données de test
- Exemples réalistes d'expériences, skills, projets
- Utilisation de `pq.StringArray` pour arrays PostgreSQL
- Helper `ptrTime()` pour nullable `EndDate`
- Logs détaillés pour suivi du seeding

---

### Étape 11: Migration Runner

**Fichier:** `backend/scripts/migrate.go`

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run migrate.go [up|down|version|force <version>]")
	}

	command := os.Args[1]

	// Database URL
	dbURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
	)

	// Migrations path
	migrationsPath := "file://./migrations"

	// Create migrate instance
	m, err := migrate.New(migrationsPath, dbURL)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer m.Close()

	// Execute command
	switch command {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration up failed: %v", err)
		}
		log.Println("✅ Migrations applied successfully")

	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration down failed: %v", err)
		}
		log.Println("✅ Migrations rolled back successfully")

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		log.Printf("Current version: %d (dirty: %v)", version, dirty)

	case "force":
		if len(os.Args) < 3 {
			log.Fatal("Usage: go run migrate.go force <version>")
		}
		var version int
		fmt.Sscanf(os.Args[2], "%d", &version)
		if err := m.Force(version); err != nil {
			log.Fatalf("Force version failed: %v", err)
		}
		log.Printf("✅ Forced version to %d", version)

	default:
		log.Fatalf("Unknown command: %s", command)
	}
}
```

**Explications:**
- Wrapper pour `golang-migrate` CLI
- Commandes: `up`, `down`, `version`, `force`
- Lecture config depuis env vars
- Gestion erreur `ErrNoChange` (pas une vraie erreur)

---

## 🧪 Tests

### Tests Unitaires Models

**Fichier:** `backend/internal/models/visitor_test.go`

```go
package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestVisitor_HasAccessToAI(t *testing.T) {
	tests := []struct {
		name     string
		visitor  Visitor
		expected bool
	}{
		{
			name: "3+ visits grants access",
			visitor: Visitor{
				VisitCount:      3,
				ProfileDetected: ProfileTypeUnknown,
			},
			expected: true,
		},
		{
			name: "Recruiter profile grants access immediately",
			visitor: Visitor{
				VisitCount:      1,
				ProfileDetected: ProfileTypeRecruiter,
			},
			expected: true,
		},
		{
			name: "CTO profile grants access immediately",
			visitor: Visitor{
				VisitCount:      1,
				ProfileDetected: ProfileTypeCTO,
			},
			expected: true,
		},
		{
			name: "Developer with 2 visits does not have access",
			visitor: Visitor{
				VisitCount:      2,
				ProfileDetected: ProfileTypeDeveloper,
			},
			expected: false,
		},
		{
			name: "Unknown profile with 1 visit does not have access",
			visitor: Visitor{
				VisitCount:      1,
				ProfileDetected: ProfileTypeUnknown,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.visitor.HasAccessToAI()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVisitor_IncrementVisit(t *testing.T) {
	visitor := Visitor{
		VisitCount: 1,
		LastVisit:  time.Now().Add(-24 * time.Hour),
	}

	visitor.IncrementVisit()

	assert.Equal(t, 2, visitor.VisitCount)
	assert.WithinDuration(t, time.Now(), visitor.LastVisit, 1*time.Second)
}

func TestVisitor_IsTargetProfile(t *testing.T) {
	tests := []struct {
		name     string
		profile  ProfileType
		expected bool
	}{
		{"Recruiter is target", ProfileTypeRecruiter, true},
		{"CTO is target", ProfileTypeCTO, true},
		{"Developer is not target", ProfileTypeDeveloper, false},
		{"Unknown is not target", ProfileTypeUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visitor := Visitor{ProfileDetected: tt.profile}
			assert.Equal(t, tt.expected, visitor.IsTargetProfile())
		})
	}
}
```

### Tests Integration Database

**Fichier:** `backend/internal/database/postgres_test.go`

```go
package database

import (
	"os"
	"testing"

	"maicivy/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Setup test database
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_USER", "test")
	os.Setenv("DB_PASSWORD", "test")
	os.Setenv("DB_NAME", "maicivy_test")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_SSLMODE", "disable")

	code := m.Run()
	os.Exit(code)
}

func TestConnectPostgres(t *testing.T) {
	err := ConnectPostgres()
	require.NoError(t, err)
	require.NotNil(t, DB)

	defer Close()

	// Test connection avec simple query
	var result int
	err = DB.Raw("SELECT 1").Scan(&result).Error
	assert.NoError(t, err)
	assert.Equal(t, 1, result)
}

func TestAutoMigrate(t *testing.T) {
	err := ConnectPostgres()
	require.NoError(t, err)
	defer Close()

	err = AutoMigrate()
	assert.NoError(t, err)

	// Vérifier que les tables existent
	tables := []string{
		"experiences",
		"skills",
		"projects",
		"visitors",
		"generated_letters",
		"analytics_events",
	}

	for _, table := range tables {
		var exists bool
		err := DB.Raw(
			"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = ?)",
			table,
		).Scan(&exists).Error

		assert.NoError(t, err)
		assert.True(t, exists, "Table %s should exist", table)
	}
}

func TestCRUD_Experience(t *testing.T) {
	err := ConnectPostgres()
	require.NoError(t, err)
	defer Close()

	err = AutoMigrate()
	require.NoError(t, err)

	// Create
	exp := models.Experience{
		Title:       "Test Engineer",
		Company:     "Test Corp",
		Description: "Testing database operations",
		StartDate:   time.Now(),
		Category:    "backend",
	}

	result := DB.Create(&exp)
	assert.NoError(t, result.Error)
	assert.NotEqual(t, uuid.Nil, exp.ID)

	// Read
	var retrieved models.Experience
	err = DB.First(&retrieved, "id = ?", exp.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, exp.Title, retrieved.Title)

	// Update
	retrieved.Title = "Updated Title"
	err = DB.Save(&retrieved).Error
	assert.NoError(t, err)

	var updated models.Experience
	err = DB.First(&updated, "id = ?", exp.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "Updated Title", updated.Title)

	// Delete (soft delete)
	err = DB.Delete(&updated).Error
	assert.NoError(t, err)

	// Verify soft delete
	var deleted models.Experience
	err = DB.First(&deleted, "id = ?", exp.ID).Error
	assert.Error(t, err) // Should not find (soft deleted)

	// Find with Unscoped (includes soft deleted)
	err = DB.Unscoped().First(&deleted, "id = ?", exp.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, deleted.DeletedAt)

	// Cleanup (hard delete)
	DB.Unscoped().Delete(&deleted)
}
```

### Commandes

```bash
# Tests unitaires models
go test ./internal/models -v

# Tests integration database
go test ./internal/database -v

# Coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Linting
golangci-lint run
```

---

## ⚠️ Points d'Attention

### Pièges à Éviter

- **PostgreSQL Arrays avec GORM**: Toujours utiliser `pq.StringArray` (package `github.com/lib/pq`), pas de slices Go natifs
- **UUID Default**: S'assurer que l'extension `uuid-ossp` est installée dans PostgreSQL
- **Soft Deletes**: Utiliser `db.Unscoped()` pour requêtes incluant les soft deleted
- **Timezone**: Toujours travailler en UTC, conversion frontend si besoin
- **Connection Pool**: Ajuster `MaxOpenConns` selon charge (100 est un bon début)
- **JSONB vs JSON**: Préférer JSONB pour indexation et performance
- **Foreign Keys**: Ne pas oublier `ON DELETE CASCADE` pour éviter orphelins
- **Migrations**: Toujours tester rollback (`down` migration) avant production

### Edge Cases

- **EndDate NULL**: Vérifier `IsCurrentJob()` avant d'afficher dates
- **Empty Arrays**: PostgreSQL array vide != NULL, gérer les deux cas
- **Concurrent Updates**: Utiliser GORM optimistic locking si nécessaire (`Version` field)
- **Long Descriptions**: Limiter taille côté validation (5000 chars) pour éviter overflow
- **Session ID Collisions**: UUID pratiquement impossible, mais gérer erreur unique constraint

### Astuces d'Optimisation

- **Indexes Composites**: Créer pour requêtes multi-colonnes fréquentes (`event_type + created_at`)
- **EXPLAIN ANALYZE**: Tester performance requêtes avec `EXPLAIN ANALYZE`
- **Preloading Relations**: Utiliser GORM `Preload()` pour éviter N+1 queries
- **Batch Inserts**: Utiliser `DB.CreateInBatches()` pour gros volumes
- **Partial Indexes**: PostgreSQL supporte indexes partiels (ex: `WHERE deleted_at IS NULL`)

---

## 📚 Ressources

### Documentation Officielle

- [GORM Documentation](https://gorm.io/docs/)
- [PostgreSQL 15 Documentation](https://www.postgresql.org/docs/15/)
- [golang-migrate Guide](https://github.com/golang-migrate/migrate)
- [pq Package (PostgreSQL arrays)](https://github.com/lib/pq)

### Tutoriels

- [GORM Best Practices](https://gorm.io/docs/performance.html)
- [PostgreSQL Indexing Strategies](https://www.postgresql.org/docs/current/indexes.html)
- [Database Migration Strategies](https://www.openmymind.net/Database-Migrations/)

### Articles Recommandés

- [UUID vs Auto-Increment](https://tomharrisonjr.com/uuid-or-guid-as-primary-keys-be-careful-7b2aa3dcb439)
- [JSONB Performance in PostgreSQL](https://www.postgresql.org/docs/current/datatype-json.html)
- [Soft Deletes: Pros and Cons](https://brandur.org/soft-deletion)

---

## ✅ Checklist de Complétion

- [ ] Code implémenté
  - [ ] Tous les models GORM créés (6 tables)
  - [ ] Relations foreign keys configurées
  - [ ] Validators ajoutés sur champs critiques
  - [ ] Helper methods implémentées
- [ ] Migrations SQL
  - [ ] `000001_init_schema.up.sql` créée
  - [ ] `000001_init_schema.down.sql` créée (rollback)
  - [ ] Triggers `updated_at` configurés
  - [ ] Indexes créés sur colonnes clés
- [ ] Database Connection
  - [ ] `postgres.go` avec ConnectPostgres()
  - [ ] Connection pooling configuré
  - [ ] AutoMigrate() fonctionnel
- [ ] Seed Data
  - [ ] Script `seed.go` avec fixtures
  - [ ] Données réalistes pour dev/test
- [ ] Tests écrits et passants
  - [ ] Tests unitaires models (HasAccessToAI, etc.)
  - [ ] Tests integration database (CRUD)
  - [ ] Coverage > 80%
- [ ] Migration Runner
  - [ ] Script `migrate.go` opérationnel
  - [ ] Commandes up/down/version testées
- [ ] Documentation code
  - [ ] Commentaires sur types complexes
  - [ ] Godoc sur fonctions publiques
- [ ] Review sécurité
  - [ ] Validation inputs (struct tags)
  - [ ] IPHash au lieu d'IP brute (RGPD)
  - [ ] Foreign keys avec ON DELETE CASCADE
- [ ] Review performance
  - [ ] Indexes sur colonnes filtrées
  - [ ] JSONB pour données flexibles
  - [ ] Connection pool optimisé
- [ ] Commit & Push
  - [ ] Git commit avec message clair
  - [ ] Push vers repository

---

**Dernière mise à jour:** 2025-12-08
**Auteur:** Alexi
