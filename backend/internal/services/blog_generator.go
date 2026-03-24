package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/yuin/goldmark"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

// BlogGeneratorService génère des articles de blog depuis les commits
type BlogGeneratorService struct {
	db          *gorm.DB
	redis       *redis.Client
	aiService   *AIService
	repoScanner *RepoScanner
}

// NewBlogGeneratorService crée une nouvelle instance
func NewBlogGeneratorService(db *gorm.DB, redis *redis.Client, aiService *AIService, repoScanner *RepoScanner) *BlogGeneratorService {
	return &BlogGeneratorService{
		db:          db,
		redis:       redis,
		aiService:   aiService,
		repoScanner: repoScanner,
	}
}

// trivialCommitPatterns patterns pour filtrer les commits triviaux
var trivialCommitPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^merge\s`),
	regexp.MustCompile(`(?i)^wip\b`),
	regexp.MustCompile(`(?i)^fix\s*typo`),
	regexp.MustCompile(`(?i)^update\s+readme`),
	regexp.MustCompile(`(?i)^chore:\s*(update|bump)\s+dep`),
	regexp.MustCompile(`(?i)^style:\s*format`),
	regexp.MustCompile(`(?i)^\.\s*$`),
	regexp.MustCompile(`(?i)^temp\b`),
	regexp.MustCompile(`(?i)^test commit`),
}

// IsSignificantCommit vérifie si un commit mérite un article
func (s *BlogGeneratorService) IsSignificantCommit(message string) bool {
	message = strings.TrimSpace(message)

	// Trop court
	if len(message) < 10 {
		return false
	}

	// Pattern trivial
	for _, pattern := range trivialCommitPatterns {
		if pattern.MatchString(message) {
			return false
		}
	}

	// Commits intéressants (feat, fix important, refactor, etc.)
	significantPrefixes := []string{
		"feat:", "feat(", "feature:",
		"fix:", "fix(",
		"refactor:", "refactor(",
		"perf:", "perf(",
		"add:", "implement:",
		"create:", "build:",
	}

	messageLower := strings.ToLower(message)
	for _, prefix := range significantPrefixes {
		if strings.HasPrefix(messageLower, prefix) {
			return true
		}
	}

	// Si le message est assez long et descriptif
	return len(message) > 50
}

// GenerateFromCommits génère un article depuis une liste de commits
func (s *BlogGeneratorService) GenerateFromCommits(ctx context.Context, projectName string, commits []models.CommitRef) (*models.BlogPost, error) {
	if len(commits) == 0 {
		return nil, fmt.Errorf("no commits provided")
	}

	// Filtrer les commits significatifs
	var significantCommits []models.CommitRef
	for _, c := range commits {
		if s.IsSignificantCommit(c.Message) {
			significantCommits = append(significantCommits, c)
		}
	}

	if len(significantCommits) == 0 {
		return nil, fmt.Errorf("no significant commits found")
	}

	// Construire le prompt pour l'IA
	prompt := s.buildGenerationPrompt(projectName, significantCommits)

	// Générer avec l'IA
	if s.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}

	response, _, err := s.aiService.GenerateText(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	// Parser la réponse
	post := s.parseAIResponse(response, projectName, significantCommits)

	// Sauvegarder
	if err := s.db.Create(post).Error; err != nil {
		return nil, fmt.Errorf("failed to save blog post: %w", err)
	}

	log.Info().Str("slug", post.Slug).Str("project", projectName).Msg("Blog post generated")

	return post, nil
}

// buildGenerationPrompt construit le prompt pour la génération
func (s *BlogGeneratorService) buildGenerationPrompt(projectName string, commits []models.CommitRef) string {
	var commitList strings.Builder
	for _, c := range commits {
		commitList.WriteString(fmt.Sprintf("- %s: %s\n", c.SHA[:7], c.Message))
	}

	return fmt.Sprintf(`Tu es un rédacteur technique expert. Génère un article de blog professionnel basé sur ces commits du projet "%s":

%s

L'article doit:
1. Avoir un titre accrocheur et professionnel
2. Expliquer ce qui a été fait de manière claire
3. Mettre en valeur les compétences techniques démontrées
4. Être écrit en français
5. Faire entre 200-400 mots

Format de réponse (JSON):
{
  "title": "Titre de l'article",
  "summary": "Résumé en une phrase",
  "content": "Contenu de l'article en markdown",
  "tags": ["tag1", "tag2", "tag3"]
}`, projectName, commitList.String())
}

// aiArticleResponse structure JSON retournée par Claude
type aiArticleResponse struct {
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

// extractJSON extrait le premier bloc JSON d'une réponse (gère les blocs ```json```)
func extractJSON(response string) string {
	// Chercher un bloc ```json ... ```
	if start := strings.Index(response, "```json"); start >= 0 {
		inner := response[start+7:]
		if end := strings.Index(inner, "```"); end >= 0 {
			return strings.TrimSpace(inner[:end])
		}
	}
	// Chercher un bloc ``` ... ```
	if start := strings.Index(response, "```"); start >= 0 {
		inner := response[start+3:]
		if end := strings.Index(inner, "```"); end >= 0 {
			return strings.TrimSpace(inner[:end])
		}
	}
	// Chercher le premier { ... } englobant
	if start := strings.Index(response, "{"); start >= 0 {
		if end := strings.LastIndex(response, "}"); end > start {
			return response[start : end+1]
		}
	}
	return response
}

// markdownToHTML convertit du markdown en HTML via goldmark
func markdownToHTML(md string) string {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(md), &buf); err != nil {
		log.Warn().Err(err).Msg("Failed to convert markdown to HTML, returning raw content")
		return md
	}
	return buf.String()
}

// parseAIResponse parse la réponse de l'IA et crée un BlogPost
func (s *BlogGeneratorService) parseAIResponse(response string, projectName string, commits []models.CommitRef) *models.BlogPost {
	// Valeurs par défaut si parsing échoue
	title := fmt.Sprintf("Mise à jour %s - %s", projectName, time.Now().Format("02/01/2006"))
	summary := "Nouvelles fonctionnalités et améliorations"
	content := response
	tags := []string{projectName, "development"}

	// Parser le JSON de la réponse Claude
	jsonStr := extractJSON(response)
	var parsed aiArticleResponse
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		log.Warn().Err(err).Msg("Failed to parse AI response as JSON, using defaults")
	} else {
		if parsed.Title != "" {
			title = parsed.Title
		}
		if parsed.Summary != "" {
			summary = parsed.Summary
		}
		if parsed.Content != "" {
			content = parsed.Content
		}
		if len(parsed.Tags) > 0 {
			tags = parsed.Tags
		}
	}

	// Convertir markdown → HTML
	contentHTML := markdownToHTML(content)

	// Générer le slug
	postSlug := slug.Make(title)
	if postSlug == "" {
		postSlug = fmt.Sprintf("%s-%d", slug.Make(projectName), time.Now().Unix())
	}

	// Calculer le temps de lecture (environ 200 mots/minute)
	wordCount := len(strings.Fields(content))
	readingTime := wordCount / 200
	if readingTime < 1 {
		readingTime = 1
	}

	now := time.Now()

	return &models.BlogPost{
		Slug:                 postSlug,
		Title:                title,
		Summary:              summary,
		Content:              content,
		ContentHTML:          contentHTML,
		ProjectName:          projectName,
		Tags:                 tags,
		GeneratedFromCommits: commits,
		ReadingTimeMinutes:   readingTime,
		Published:            false, // Pas publié par défaut, review manuelle
		PublishedAt:          &now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

// GenerateFromRecentActivity génère un article depuis l'activité récente d'un projet
func (s *BlogGeneratorService) GenerateFromRecentActivity(ctx context.Context, projectName string) (*models.BlogPost, error) {
	// Récupérer les projets via le scanner
	if s.repoScanner == nil {
		return nil, fmt.Errorf("repo scanner not available")
	}

	scanResult, err := s.repoScanner.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scan repos: %w", err)
	}

	var targetProject *ScannedProject
	for i := range scanResult.Projects {
		if scanResult.Projects[i].Name == projectName {
			targetProject = &scanResult.Projects[i]
			break
		}
	}

	if targetProject == nil {
		return nil, fmt.Errorf("project %s not found", projectName)
	}

	// Convertir les commits récents en CommitRef
	var commits []models.CommitRef
	for _, c := range targetProject.RecentCommits {
		commits = append(commits, models.CommitRef{
			SHA:     c.SHA,
			Message: c.Message,
			Date:    c.Date,
			Project: projectName,
		})
	}

	return s.GenerateFromCommits(ctx, projectName, commits)
}

// GetPublishedPosts récupère les articles publiés
func (s *BlogGeneratorService) GetPublishedPosts(ctx context.Context, page, perPage int) (*models.BlogPostListResponse, error) {
	var posts []models.BlogPost
	var total int64

	// Compter le total
	if err := s.db.Model(&models.BlogPost{}).Where("published = true").Count(&total).Error; err != nil {
		return nil, err
	}

	// Pagination
	offset := (page - 1) * perPage
	if err := s.db.Where("published = true").
		Order("published_at DESC").
		Offset(offset).
		Limit(perPage).
		Find(&posts).Error; err != nil {
		return nil, err
	}

	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	return &models.BlogPostListResponse{
		Posts:      posts,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

// GetAllPosts récupère tous les articles (admin)
func (s *BlogGeneratorService) GetAllPosts(ctx context.Context, page, perPage int) (*models.BlogPostListResponse, error) {
	var posts []models.BlogPost
	var total int64

	if err := s.db.Model(&models.BlogPost{}).Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (page - 1) * perPage
	if err := s.db.Order("created_at DESC").
		Offset(offset).
		Limit(perPage).
		Find(&posts).Error; err != nil {
		return nil, err
	}

	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	return &models.BlogPostListResponse{
		Posts:      posts,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

// GetPostBySlug récupère un article par son slug
func (s *BlogGeneratorService) GetPostBySlug(ctx context.Context, postSlug string) (*models.BlogPost, error) {
	var post models.BlogPost
	if err := s.db.Where("slug = ?", postSlug).First(&post).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

// PublishPost publie un article
func (s *BlogGeneratorService) PublishPost(ctx context.Context, postID uint) error {
	now := time.Now()
	return s.db.Model(&models.BlogPost{}).
		Where("id = ?", postID).
		Updates(map[string]interface{}{
			"published":    true,
			"published_at": now,
		}).Error
}

// UnpublishPost dépublie un article
func (s *BlogGeneratorService) UnpublishPost(ctx context.Context, postID uint) error {
	return s.db.Model(&models.BlogPost{}).
		Where("id = ?", postID).
		Update("published", false).Error
}

// DeletePost supprime un article
func (s *BlogGeneratorService) DeletePost(ctx context.Context, postID uint) error {
	return s.db.Delete(&models.BlogPost{}, postID).Error
}
