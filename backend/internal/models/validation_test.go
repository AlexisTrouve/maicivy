package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ValidationTestSuite regroupe les tests de validation des models GORM
type ValidationTestSuite struct {
	suite.Suite
	db *gorm.DB
}

func (suite *ValidationTestSuite) SetupTest() {
	// Utiliser SQLite en mémoire pour les tests
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(suite.T(), err)

	// Migrer tous les models
	err = db.AutoMigrate(
		&Experience{},
		&Skill{},
		&Project{},
		&Visitor{},
		&GeneratedLetter{},
		&AnalyticsEvent{},
	)
	assert.NoError(suite.T(), err)

	suite.db = db
}

func (suite *ValidationTestSuite) TearDownTest() {
	// Nettoyer la DB
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()
}

// ========== Tests Experience ==========

func (suite *ValidationTestSuite) TestExperience_ValidCreation() {
	exp := &Experience{
		Title:        "Backend Developer",
		Company:      "TechCorp",
		Description:  "Building scalable APIs",
		StartDate:    time.Now().AddDate(-2, 0, 0),
		Technologies: pq.StringArray{"go", "postgresql"},
		Tags:         pq.StringArray{"backend", "api"},
		Category:     "backend",
		Featured:     true,
	}

	result := suite.db.Create(exp)

	assert.NoError(suite.T(), result.Error)
	assert.NotEqual(suite.T(), uuid.Nil, exp.ID)
	assert.Equal(suite.T(), "Backend Developer", exp.Title)
	assert.True(suite.T(), exp.IsCurrentJob()) // EndDate est nil
}

func (suite *ValidationTestSuite) TestExperience_RequiredFields() {
	// Test sans Title (requis)
	exp := &Experience{
		Company:   "Corp",
		StartDate: time.Now(),
		Category:  "backend",
	}

	result := suite.db.Create(exp)

	// GORM devrait échouer car Title est NOT NULL
	assert.Error(suite.T(), result.Error)
}

func (suite *ValidationTestSuite) TestExperience_InvalidCategory() {
	exp := &Experience{
		Title:     "Dev",
		Company:   "Corp",
		StartDate: time.Now(),
		Category:  "invalid_category", // Pas dans enum
	}

	result := suite.db.Create(exp)

	// Note: GORM ne valide pas les enums par défaut
	// Il faut utiliser un validateur comme go-playground/validator
	// Ici on vérifie juste que la création fonctionne
	assert.NoError(suite.T(), result.Error)
}

func (suite *ValidationTestSuite) TestExperience_DurationCalculation() {
	start := time.Now().AddDate(-2, -3, -15) // Il y a 2 ans, 3 mois, 15 jours
	exp := &Experience{
		Title:     "Dev",
		Company:   "Corp",
		StartDate: start,
		Category:  "backend",
	}

	duration := exp.Duration()

	assert.Greater(suite.T(), duration.Hours(), float64(24*365*2)) // Plus de 2 ans
}

func (suite *ValidationTestSuite) TestExperience_WithEndDate() {
	start := time.Now().AddDate(-1, 0, 0)
	end := time.Now()
	exp := &Experience{
		Title:     "Dev",
		Company:   "Corp",
		StartDate: start,
		EndDate:   &end,
		Category:  "backend",
	}

	suite.db.Create(exp)

	assert.False(suite.T(), exp.IsCurrentJob())
	assert.Less(suite.T(), exp.Duration().Hours(), float64(24*365*2)) // Moins de 2 ans
}

// ========== Tests Skill ==========

func (suite *ValidationTestSuite) TestSkill_ValidCreation() {
	skill := &Skill{
		Name:            "Go",
		Level:           SkillLevelExpert,
		Category:        "backend",
		Tags:            pq.StringArray{"programming", "backend"},
		YearsExperience: 5,
		Description:     "Expert in Go programming",
		Featured:        true,
		Icon:            "golang",
	}

	result := suite.db.Create(skill)

	assert.NoError(suite.T(), result.Error)
	assert.NotEqual(suite.T(), uuid.Nil, skill.ID)
	assert.Equal(suite.T(), 4, skill.LevelScore()) // Expert = 4
}

func (suite *ValidationTestSuite) TestSkill_RequiredFields() {
	// Test sans Name (requis + unique)
	skill := &Skill{
		Level:    SkillLevelBeginner,
		Category: "backend",
	}

	result := suite.db.Create(skill)

	assert.Error(suite.T(), result.Error)
}

func (suite *ValidationTestSuite) TestSkill_UniqueConstraint() {
	skill1 := &Skill{
		Name:     "React",
		Level:    SkillLevelAdvanced,
		Category: "frontend",
	}

	skill2 := &Skill{
		Name:     "React", // Même nom
		Level:    SkillLevelExpert,
		Category: "frontend",
	}

	suite.db.Create(skill1)
	result := suite.db.Create(skill2)

	// Devrait échouer car Name est unique
	assert.Error(suite.T(), result.Error)
}

func (suite *ValidationTestSuite) TestSkill_LevelScores() {
	tests := []struct {
		level    SkillLevel
		expected int
	}{
		{SkillLevelExpert, 4},
		{SkillLevelAdvanced, 3},
		{SkillLevelIntermediate, 2},
		{SkillLevelBeginner, 1},
		{SkillLevel("invalid"), 0},
	}

	for _, tt := range tests {
		skill := &Skill{Level: tt.level}
		assert.Equal(suite.T(), tt.expected, skill.LevelScore())
	}
}

// ========== Tests Project ==========

func (suite *ValidationTestSuite) TestProject_ValidCreation() {
	project := &Project{
		Title:          "maicivy",
		Description:    "Interactive CV with AI",
		GithubURL:      "https://github.com/user/maicivy",
		DemoURL:        "https://maicivy.com",
		ImageURL:       "https://maicivy.com/image.png",
		Technologies:   pq.StringArray{"go", "react", "typescript"},
		Category:       "fullstack",
		GithubStars:    42,
		GithubForks:    7,
		GithubLanguage: "Go",
		Featured:       true,
		InProgress:     false,
	}

	result := suite.db.Create(project)

	assert.NoError(suite.T(), result.Error)
	assert.NotEqual(suite.T(), uuid.Nil, project.ID)
	assert.True(suite.T(), project.HasGithub())
	assert.True(suite.T(), project.HasDemo())
}

func (suite *ValidationTestSuite) TestProject_RequiredFields() {
	// Test sans Title (requis)
	project := &Project{
		Description: "Test",
		Category:    "backend",
	}

	result := suite.db.Create(project)

	assert.Error(suite.T(), result.Error)
}

func (suite *ValidationTestSuite) TestProject_NoGithubURL() {
	project := &Project{
		Title:       "Private Project",
		Description: "No public repo",
		Category:    "backend",
	}

	suite.db.Create(project)

	assert.False(suite.T(), project.HasGithub())
	assert.False(suite.T(), project.HasDemo())
}

// ========== Tests Visitor ==========

func (suite *ValidationTestSuite) TestVisitor_ValidCreation() {
	visitor := &Visitor{
		SessionID:           "session_123",
		IPHash:              "hash_abc",
		UserAgent:           "Mozilla/5.0",
		Browser:             "Chrome",
		OS:                  "Linux",
		Device:              "desktop",
		VisitCount:          1,
		FirstVisit:          time.Now(),
		LastVisit:           time.Now(),
		ProfileDetected:     ProfileTypeRecruiter,
		ProfileType:         "recruiter",
		DetectionConfidence: 85,
		CompanyName:         "TechCorp",
		Country:             "France",
		City:                "Paris",
		Timezone:            "Europe/Paris",
	}

	result := suite.db.Create(visitor)

	assert.NoError(suite.T(), result.Error)
	assert.NotEqual(suite.T(), uuid.Nil, visitor.ID)
	assert.False(suite.T(), visitor.HasAccessToAI()) // Seulement 1 visite
	assert.True(suite.T(), visitor.IsTargetProfile()) // Recruiter = cible
}

func (suite *ValidationTestSuite) TestVisitor_UniqueSessionID() {
	visitor1 := &Visitor{
		SessionID:  "session_duplicate",
		VisitCount: 1,
		FirstVisit: time.Now(),
		LastVisit:  time.Now(),
	}

	visitor2 := &Visitor{
		SessionID:  "session_duplicate", // Même session
		VisitCount: 1,
		FirstVisit: time.Now(),
		LastVisit:  time.Now(),
	}

	suite.db.Create(visitor1)
	result := suite.db.Create(visitor2)

	// Devrait échouer car SessionID est unique
	assert.Error(suite.T(), result.Error)
}

func (suite *ValidationTestSuite) TestVisitor_HasAccessToAI() {
	tests := []struct {
		name        string
		visitCount  int
		profile     ProfileType
		expectedAI  bool
	}{
		{"1 visit unknown", 1, ProfileTypeUnknown, false},
		{"2 visits unknown", 2, ProfileTypeUnknown, false},
		{"3 visits unknown", 3, ProfileTypeUnknown, true},
		{"1 visit recruiter", 1, ProfileTypeRecruiter, true},
		{"1 visit CTO", 1, ProfileTypeCTO, true},
		{"1 visit developer", 1, ProfileTypeDeveloper, false},
		{"5 visits developer", 5, ProfileTypeDeveloper, true},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			visitor := &Visitor{
				VisitCount:      tt.visitCount,
				ProfileDetected: tt.profile,
			}
			assert.Equal(t, tt.expectedAI, visitor.HasAccessToAI())
		})
	}
}

func (suite *ValidationTestSuite) TestVisitor_IncrementVisit() {
	visitor := &Visitor{
		SessionID:  "session_increment",
		VisitCount: 1,
		FirstVisit: time.Now(),
		LastVisit:  time.Now().Add(-1 * time.Hour),
	}

	suite.db.Create(visitor)

	oldLastVisit := visitor.LastVisit
	visitor.IncrementVisit()

	assert.Equal(suite.T(), 2, visitor.VisitCount)
	assert.True(suite.T(), visitor.LastVisit.After(oldLastVisit))
}

// ========== Tests GeneratedLetter ==========

func (suite *ValidationTestSuite) TestGeneratedLetter_ValidCreation() {
	// Créer visitor d'abord
	visitor := &Visitor{
		SessionID:  "session_letter",
		VisitCount: 3,
		FirstVisit: time.Now(),
		LastVisit:  time.Now(),
	}
	suite.db.Create(visitor)

	letter := &GeneratedLetter{
		VisitorID:    visitor.ID,
		CompanyName:  "TechCorp",
		LetterType:   LetterTypeMotivation,
		Content:      "Je suis très motivé...",
		AIModel:      "claude-3-sonnet",
		TokensUsed:   1500,
		GenerationMS: 2500,
		CompanyInfo:  `{"name":"TechCorp","domain":"techcorp.com"}`,
		Downloaded:   false,
	}

	result := suite.db.Create(letter)

	assert.NoError(suite.T(), result.Error)
	assert.NotEqual(suite.T(), uuid.Nil, letter.ID)
	assert.True(suite.T(), letter.IsMotivation())
	assert.Greater(suite.T(), letter.EstimatedCost(), 0.0)
}

func (suite *ValidationTestSuite) TestGeneratedLetter_RequiredFields() {
	letter := &GeneratedLetter{
		CompanyName: "Corp",
		// Manque VisitorID, LetterType, Content
	}

	result := suite.db.Create(letter)

	assert.Error(suite.T(), result.Error)
}

func (suite *ValidationTestSuite) TestGeneratedLetter_Types() {
	visitor := &Visitor{
		SessionID:  "session_types",
		VisitCount: 1,
		FirstVisit: time.Now(),
		LastVisit:  time.Now(),
	}
	suite.db.Create(visitor)

	motivationLetter := &GeneratedLetter{
		VisitorID:   visitor.ID,
		CompanyName: "Corp1",
		LetterType:  LetterTypeMotivation,
		Content:     "Motivation",
	}

	antiMotivationLetter := &GeneratedLetter{
		VisitorID:   visitor.ID,
		CompanyName: "Corp2",
		LetterType:  LetterTypeAntiMotivation,
		Content:     "Anti-motivation",
	}

	suite.db.Create(motivationLetter)
	suite.db.Create(antiMotivationLetter)

	assert.True(suite.T(), motivationLetter.IsMotivation())
	assert.False(suite.T(), antiMotivationLetter.IsMotivation())
}

func (suite *ValidationTestSuite) TestGeneratedLetter_EstimatedCost() {
	letter := &GeneratedLetter{
		TokensUsed: 10000, // 10k tokens
	}

	cost := letter.EstimatedCost()

	// 10000 * 0.00001 = 0.1
	assert.InDelta(suite.T(), 0.1, cost, 0.001)
}

// ========== Tests AnalyticsEvent ==========

func (suite *ValidationTestSuite) TestAnalyticsEvent_ValidCreation() {
	// Créer visitor d'abord
	visitor := &Visitor{
		SessionID:  "session_analytics",
		VisitCount: 1,
		FirstVisit: time.Now(),
		LastVisit:  time.Now(),
	}
	suite.db.Create(visitor)

	event := &AnalyticsEvent{
		VisitorID:       visitor.ID,
		EventType:       EventTypePageView,
		EventData:       `{"page":"home"}`,
		PageURL:         "/",
		Referrer:        "https://google.com",
		SessionDuration: 120,
	}

	result := suite.db.Create(event)

	assert.NoError(suite.T(), result.Error)
	assert.NotEqual(suite.T(), uuid.Nil, event.ID)
	assert.True(suite.T(), event.IsPageView())
	assert.False(suite.T(), event.IsConversion())
}

func (suite *ValidationTestSuite) TestAnalyticsEvent_ConversionTypes() {
	visitor := &Visitor{
		SessionID:  "session_conversion",
		VisitCount: 1,
		FirstVisit: time.Now(),
		LastVisit:  time.Now(),
	}
	suite.db.Create(visitor)

	tests := []struct {
		eventType    EventType
		isConversion bool
	}{
		{EventTypePageView, false},
		{EventTypeCVThemeChange, false},
		{EventTypeLetterGenerate, true},
		{EventTypePDFDownload, true},
		{EventTypeButtonClick, false},
	}

	for _, tt := range tests {
		suite.T().Run(string(tt.eventType), func(t *testing.T) {
			event := &AnalyticsEvent{
				VisitorID: visitor.ID,
				EventType: tt.eventType,
			}
			assert.Equal(t, tt.isConversion, event.IsConversion())
		})
	}
}

// ========== Tests BaseModel ==========

func (suite *ValidationTestSuite) TestBaseModel_UUIDGeneration() {
	exp := &Experience{
		Title:     "Test",
		Company:   "Test",
		StartDate: time.Now(),
		Category:  "backend",
	}

	// Avant création, ID devrait être Nil
	assert.Equal(suite.T(), uuid.Nil, exp.ID)

	suite.db.Create(exp)

	// Après création, UUID devrait être généré
	assert.NotEqual(suite.T(), uuid.Nil, exp.ID)
}

func (suite *ValidationTestSuite) TestBaseModel_Timestamps() {
	before := time.Now()

	exp := &Experience{
		Title:     "Test",
		Company:   "Test",
		StartDate: time.Now(),
		Category:  "backend",
	}

	suite.db.Create(exp)

	// CreatedAt et UpdatedAt devraient être définis
	assert.True(suite.T(), exp.CreatedAt.After(before.Add(-1*time.Second)))
	assert.True(suite.T(), exp.UpdatedAt.After(before.Add(-1*time.Second)))
}

func (suite *ValidationTestSuite) TestBaseModel_SoftDelete() {
	exp := &Experience{
		Title:     "Test",
		Company:   "Test",
		StartDate: time.Now(),
		Category:  "backend",
	}

	suite.db.Create(exp)
	expID := exp.ID

	// Soft delete
	suite.db.Delete(exp)

	// Ne devrait plus être trouvé par défaut
	var found Experience
	result := suite.db.First(&found, "id = ?", expID)
	assert.Error(suite.T(), result.Error)

	// Devrait être trouvé avec Unscoped
	result = suite.db.Unscoped().First(&found, "id = ?", expID)
	assert.NoError(suite.T(), result.Error)
	assert.NotNil(suite.T(), found.DeletedAt.Time)
}

// Run test suite
func TestValidationTestSuite(t *testing.T) {
	suite.Run(t, new(ValidationTestSuite))
}
