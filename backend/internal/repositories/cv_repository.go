package repositories

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// CVRepository handles optimized database queries for CV-related data
type CVRepository struct {
	db *gorm.DB
}

// NewCVRepository creates a new CV repository
func NewCVRepository(db *gorm.DB) *CVRepository {
	return &CVRepository{db: db}
}

// Experience represents the experiences table model
type Experience struct {
	ID           uint     `gorm:"primaryKey" json:"id"`
	Title        string   `json:"title"`
	Company      string   `json:"company"`
	Description  string   `json:"description"`
	StartDate    string   `json:"startDate"`
	EndDate      *string  `json:"endDate"`
	Technologies []string `gorm:"type:text[]" json:"technologies"`
	Tags         []string `gorm:"type:text[]" json:"tags"`
	Category     string   `json:"category"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
}

// Skill represents the skills table model
type Skill struct {
	ID              uint     `gorm:"primaryKey" json:"id"`
	Name            string   `json:"name"`
	Level           int      `json:"level"`
	Category        string   `json:"category"`
	Tags            []string `gorm:"type:text[]" json:"tags"`
	YearsExperience int      `json:"yearsExperience"`
	CreatedAt       string   `json:"createdAt"`
	UpdatedAt       string   `json:"updatedAt"`
}

// Project represents the projects table model
type Project struct {
	ID           uint     `gorm:"primaryKey" json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	GithubURL    *string  `json:"githubUrl"`
	DemoURL      *string  `json:"demoUrl"`
	Technologies []string `gorm:"type:text[]" json:"technologies"`
	Category     string   `json:"category"`
	Tags         []string `gorm:"type:text[]" json:"tags"`
	Featured     bool     `json:"featured"`
	Stars        int      `json:"stars"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
}

// GetExperiencesByCategory retrieves experiences filtered by category with pagination
func (r *CVRepository) GetExperiencesByCategory(ctx context.Context, category string, limit, offset int) ([]Experience, int64, error) {
	var experiences []Experience
	var total int64

	// Build query
	query := r.db.WithContext(ctx).Model(&Experience{})

	// Apply category filter
	if category != "" {
		query = query.Where("category = ?", category)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count experiences: %w", err)
	}

	// Fetch with pagination and ordering
	// Uses idx_experiences_category_dates index
	if err := query.
		Order("start_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&experiences).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch experiences: %w", err)
	}

	return experiences, total, nil
}

// GetExperiencesByTags retrieves experiences containing specified tags
func (r *CVRepository) GetExperiencesByTags(ctx context.Context, tags []string, limit, offset int) ([]Experience, error) {
	var experiences []Experience

	// Uses idx_experiences_tags GIN index for fast array containment
	if err := r.db.WithContext(ctx).
		Where("tags @> ?", tags). // PostgreSQL array contains operator
		Order("start_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&experiences).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch experiences by tags: %w", err)
	}

	return experiences, nil
}

// GetActiveExperiences retrieves currently active experiences (end_date IS NULL)
func (r *CVRepository) GetActiveExperiences(ctx context.Context) ([]Experience, error) {
	var experiences []Experience

	// Uses idx_experiences_active partial index
	if err := r.db.WithContext(ctx).
		Where("end_date IS NULL").
		Order("start_date DESC").
		Find(&experiences).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch active experiences: %w", err)
	}

	return experiences, nil
}

// GetSkillsByCategory retrieves skills filtered by category
func (r *CVRepository) GetSkillsByCategory(ctx context.Context, category string, limit, offset int) ([]Skill, error) {
	var skills []Skill

	query := r.db.WithContext(ctx)

	if category != "" {
		// Uses idx_skills_category_level composite index
		query = query.Where("category = ?", category)
	}

	if err := query.
		Order("level DESC, years_experience DESC").
		Limit(limit).
		Offset(offset).
		Find(&skills).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch skills: %w", err)
	}

	return skills, nil
}

// GetTopSkills retrieves top N skills by experience
func (r *CVRepository) GetTopSkills(ctx context.Context, limit int) ([]Skill, error) {
	var skills []Skill

	// Uses idx_skills_years index
	if err := r.db.WithContext(ctx).
		Order("years_experience DESC, level DESC").
		Limit(limit).
		Find(&skills).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch top skills: %w", err)
	}

	return skills, nil
}

// GetFeaturedProjects retrieves featured projects
func (r *CVRepository) GetFeaturedProjects(ctx context.Context) ([]Project, error) {
	var projects []Project

	// Uses idx_projects_featured partial index
	if err := r.db.WithContext(ctx).
		Where("featured = ?", true).
		Order("created_at DESC").
		Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch featured projects: %w", err)
	}

	return projects, nil
}

// GetProjectsByTechnology retrieves projects using specific technologies
func (r *CVRepository) GetProjectsByTechnology(ctx context.Context, technologies []string, limit, offset int) ([]Project, error) {
	var projects []Project

	// Uses idx_projects_technologies GIN index
	if err := r.db.WithContext(ctx).
		Where("technologies @> ?", technologies).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch projects by technology: %w", err)
	}

	return projects, nil
}

// GetAllProjects retrieves all projects with pagination
func (r *CVRepository) GetAllProjects(ctx context.Context, limit, offset int) ([]Project, int64, error) {
	var projects []Project
	var total int64

	// Count total
	if err := r.db.WithContext(ctx).Model(&Project{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count projects: %w", err)
	}

	// Fetch with pagination
	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&projects).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch projects: %w", err)
	}

	return projects, total, nil
}

// GetExperienceByID retrieves a single experience by ID
func (r *CVRepository) GetExperienceByID(ctx context.Context, id uint) (*Experience, error) {
	var experience Experience

	if err := r.db.WithContext(ctx).
		First(&experience, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch experience: %w", err)
	}

	return &experience, nil
}

// SearchExperiences performs full-text search on experiences
func (r *CVRepository) SearchExperiences(ctx context.Context, searchTerm string, limit, offset int) ([]Experience, error) {
	var experiences []Experience

	// Uses PostgreSQL ILIKE for case-insensitive search
	// In production, consider using PostgreSQL full-text search or Elasticsearch
	searchPattern := "%" + searchTerm + "%"

	if err := r.db.WithContext(ctx).
		Where("title ILIKE ? OR description ILIKE ? OR company ILIKE ?",
			searchPattern, searchPattern, searchPattern).
		Order("start_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&experiences).Error; err != nil {
		return nil, fmt.Errorf("failed to search experiences: %w", err)
	}

	return experiences, nil
}

// GetCVData retrieves all CV data optimized for a specific theme
// This method uses eager loading to minimize database roundtrips
func (r *CVRepository) GetCVData(ctx context.Context, theme string) (*CVData, error) {
	cvData := &CVData{}

	// Fetch experiences
	experiences, _, err := r.GetExperiencesByCategory(ctx, theme, 50, 0)
	if err != nil {
		return nil, err
	}
	cvData.Experiences = experiences

	// Fetch skills
	skills, err := r.GetSkillsByCategory(ctx, theme, 50, 0)
	if err != nil {
		return nil, err
	}
	cvData.Skills = skills

	// Fetch projects (featured first)
	projects, _, err := r.GetAllProjects(ctx, 20, 0)
	if err != nil {
		return nil, err
	}
	cvData.Projects = projects

	return cvData, nil
}

// CVData represents complete CV data
type CVData struct {
	Experiences []Experience `json:"experiences"`
	Skills      []Skill      `json:"skills"`
	Projects    []Project    `json:"projects"`
}

// BulkInsertExperiences inserts multiple experiences efficiently
func (r *CVRepository) BulkInsertExperiences(ctx context.Context, experiences []Experience) error {
	// GORM's CreateInBatches uses a single INSERT statement with multiple VALUES
	batchSize := 100
	if err := r.db.WithContext(ctx).CreateInBatches(experiences, batchSize).Error; err != nil {
		return fmt.Errorf("failed to bulk insert experiences: %w", err)
	}
	return nil
}

// CountExperiencesByCategory counts experiences by category
func (r *CVRepository) CountExperiencesByCategory(ctx context.Context) (map[string]int64, error) {
	type Result struct {
		Category string
		Count    int64
	}

	var results []Result

	if err := r.db.WithContext(ctx).
		Model(&Experience{}).
		Select("category, COUNT(*) as count").
		Group("category").
		Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to count experiences by category: %w", err)
	}

	// Convert to map
	counts := make(map[string]int64)
	for _, r := range results {
		counts[r.Category] = r.Count
	}

	return counts, nil
}
