package services

import (
	"context"
	"maicivy/internal/config"
	"maicivy/internal/models"
)

// CVServiceInterface defines the interface for CV service operations
type CVServiceInterface interface {
	GetAdaptiveCV(ctx context.Context, themeID string) (*AdaptiveCVResponse, error)
	GetAllExperiences(ctx context.Context) ([]models.Experience, error)
	GetAllSkills(ctx context.Context) ([]models.Skill, error)
	GetAllProjects(ctx context.Context) ([]models.Project, error)
	GetAvailableThemes() []config.CVTheme
	InvalidateCache(ctx context.Context, themeID string) error
}
