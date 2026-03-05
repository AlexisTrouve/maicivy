package services

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"maicivy/internal/content"
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
	contentLoader *content.Loader, // pour injecter les projets dans le prompt
) *LetterGenerator {
	projects := []models.Project{}
	if contentLoader != nil {
		projects = contentLoader.GetProjects()
	}
	return &LetterGenerator{
		aiService:     ai,
		scraper:       scraper,
		pdfService:    pdf,
		promptBuilder: NewPromptBuilder(profile, projects),
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
	// Default lang to fr if empty
	lang := req.Lang
	if lang == "" {
		lang = "fr"
	}

	var prompt string
	switch req.LetterType {
	case models.LetterTypeMotivation:
		prompt = lg.promptBuilder.BuildMotivationPrompt(*companyInfo, lang, req.JobOffer)
	case models.LetterTypeAntiMotivation:
		prompt = lg.promptBuilder.BuildAntiMotivationPrompt(*companyInfo, lang)
	default:
		return nil, fmt.Errorf("unknown letter type: %s", req.LetterType)
	}

	// 3. Generate text via AI
	content, metrics, err := lg.aiService.GenerateText(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	// 3.5. Post-traitement : nettoyer les artefacts LLM typiques
	content = cleanLetterContent(content)

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
// jobOffer est optionnel — si fourni, la lettre de motivation est tailorée pour l'offre
func (lg *LetterGenerator) GenerateDualLetters(ctx context.Context, companyName string, lang string, jobOffer ...string) (*models.LetterResponse, *models.LetterResponse, error) {
	// Default lang to fr if empty
	if lang == "" {
		lang = "fr"
	}

	offer := ""
	if len(jobOffer) > 0 {
		offer = jobOffer[0]
	}

	type result struct {
		letter *models.LetterResponse
		err    error
	}

	motivationChan := make(chan result, 1)
	antiMotivationChan := make(chan result, 1)

	// Generate motivation letter (tailorée si offre fournie)
	go func() {
		letter, err := lg.GenerateLetter(ctx, models.LetterRequest{
			CompanyName: companyName,
			LetterType:  models.LetterTypeMotivation,
			Lang:        lang,
			JobOffer:    offer,
		})
		motivationChan <- result{letter, err}
	}()

	// Generate anti-motivation letter (humour — pas besoin de l'offre)
	go func() {
		letter, err := lg.GenerateLetter(ctx, models.LetterRequest{
			CompanyName: companyName,
			LetterType:  models.LetterTypeAntiMotivation,
			Lang:        lang,
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

// cleanLetterContent nettoie les artefacts LLM typiques du contenu généré
// Les LLMs ont tendance à utiliser " - " comme séparateur en-tête, et d'autres patterns
// mécaniques qui trahissent une génération automatique.
func cleanLetterContent(content string) string {
	lines := strings.Split(content, "\n")
	cleaned := make([]string, 0, len(lines))

	for _, line := range lines {
		// Dans l'en-tête : "Nom - Email - Téléphone" sur une ligne → séparer sur des lignes distinctes
		// On détecte les lignes d'en-tête avec plusieurs " - " (pattern LLM classique)
		if strings.Count(line, " - ") >= 2 && !strings.HasPrefix(strings.TrimSpace(line), "Objet") {
			parts := strings.Split(line, " - ")
			for _, part := range parts {
				p := strings.TrimSpace(part)
				if p != "" {
					cleaned = append(cleaned, p)
				}
			}
			continue
		}

		// Lignes "Objet : Candidature - Poste" → garder tel quel (le " - " est voulu)
		// Ligne avec un seul " - " dans l'en-tête → remplacer par saut de ligne
		// Heuristique : ligne courte (< 80 chars) avec exactement un " - " = probablement en-tête
		trimmed := strings.TrimSpace(line)
		if len(trimmed) < 80 && strings.Count(trimmed, " - ") == 1 &&
			!strings.HasPrefix(trimmed, "Objet") &&
			!strings.Contains(trimmed, "mission") &&
			!strings.Contains(trimmed, "poste") {
			parts := strings.SplitN(trimmed, " - ", 2)
			cleaned = append(cleaned, strings.TrimSpace(parts[0]))
			cleaned = append(cleaned, strings.TrimSpace(parts[1]))
			continue
		}

		cleaned = append(cleaned, line)
	}

	return strings.Join(cleaned, "\n")
}
