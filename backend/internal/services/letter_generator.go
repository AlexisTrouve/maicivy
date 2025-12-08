package services

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/rs/zerolog/log"

	"maicivy/internal/models"
)

type LetterGenerator struct {
	aiService     *AIService
	scraper       *CompanyScraper
	pdfService    *PDFLetterService
	promptBuilder *PromptBuilder
}

func NewLetterGenerator(
	ai *AIService,
	scraper *CompanyScraper,
	pdf *PDFLetterService,
	profile models.UserProfile,
) *LetterGenerator {
	return &LetterGenerator{
		aiService:     ai,
		scraper:       scraper,
		pdfService:    pdf,
		promptBuilder: NewPromptBuilder(profile),
	}
}

// GenerateLetter : génère une lettre complète (IA)
func (lg *LetterGenerator) GenerateLetter(ctx context.Context, req models.LetterRequest) (*models.LetterResponse, error) {
	log.Info().
		Str("company", req.CompanyName).
		Str("type", string(req.LetterType)).
		Msg("Starting letter generation")

	// 1. Get company info via scraper
	companyInfo, err := lg.scraper.GetCompanyInfo(ctx, req.CompanyName)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get company info, using minimal data")
		// Fallback avec données minimales
		companyInfo = &models.CompanyInfo{
			Name:        req.CompanyName,
			Description: fmt.Sprintf("Entreprise %s", req.CompanyName),
		}
	}

	// 2. Build prompt based on type
	var prompt string
	switch req.LetterType {
	case models.LetterTypeMotivation:
		prompt = lg.promptBuilder.BuildMotivationPrompt(*companyInfo)
	case models.LetterTypeAntiMotivation:
		prompt = lg.promptBuilder.BuildAntiMotivationPrompt(*companyInfo)
	default:
		return nil, fmt.Errorf("unknown letter type: %s", req.LetterType)
	}

	// 3. Generate text via AI
	content, metrics, err := lg.aiService.GenerateText(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	// 4. Build response
	response := &models.LetterResponse{
		Content:       content,
		Type:          req.LetterType,
		CompanyInfo:   *companyInfo,
		GeneratedAt:   time.Now(),
		Provider:      metrics.Provider,
		TokensUsed:    metrics.TotalTokens,
		EstimatedCost: metrics.EstimatedCost,
	}

	log.Info().
		Str("company", req.CompanyName).
		Str("type", string(req.LetterType)).
		Int("tokens", metrics.TotalTokens).
		Float64("cost", metrics.EstimatedCost).
		Msg("Letter generated successfully")

	return response, nil
}

// GenerateDualLetters : génère les 2 lettres en parallèle
func (lg *LetterGenerator) GenerateDualLetters(ctx context.Context, companyName string) (*models.LetterResponse, *models.LetterResponse, error) {
	type result struct {
		letter *models.LetterResponse
		err    error
	}

	motivationChan := make(chan result, 1)
	antiMotivationChan := make(chan result, 1)

	// Generate motivation letter
	go func() {
		letter, err := lg.GenerateLetter(ctx, models.LetterRequest{
			CompanyName: companyName,
			LetterType:  models.LetterTypeMotivation,
		})
		motivationChan <- result{letter, err}
	}()

	// Generate anti-motivation letter
	go func() {
		letter, err := lg.GenerateLetter(ctx, models.LetterRequest{
			CompanyName: companyName,
			LetterType:  models.LetterTypeAntiMotivation,
		})
		antiMotivationChan <- result{letter, err}
	}()

	// Wait for both
	motivationResult := <-motivationChan
	antiMotivationResult := <-antiMotivationChan

	if motivationResult.err != nil {
		return nil, nil, fmt.Errorf("motivation letter failed: %w", motivationResult.err)
	}
	if antiMotivationResult.err != nil {
		return nil, nil, fmt.Errorf("anti-motivation letter failed: %w", antiMotivationResult.err)
	}

	return motivationResult.letter, antiMotivationResult.letter, nil
}

// GenerateLetterPDF : génère le PDF d'une lettre
func (lg *LetterGenerator) GenerateLetterPDF(ctx context.Context, letter models.LetterResponse, writer io.Writer) error {
	return lg.pdfService.GeneratePDF(ctx, letter, writer)
}
