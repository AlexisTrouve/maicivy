package api

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"maicivy/internal/models"
	"maicivy/internal/services"
)

// BlogHandler gère les endpoints API du blog.
// Depuis la migration vers maiProFiles, les opérations CRUD passent par mpfClient
// (appels HTTP vers maiprofiles.etheryale.com). Seule la génération IA (GeneratePost)
// reste dans blogGenerator, qui écrit ensuite dans maiProFiles via mpfClient.
type BlogHandler struct {
	mpfClient     *services.MaiProFilesClient   // CRUD blog → maiProFiles
	blogGenerator *services.BlogGeneratorService // Génération IA markdown
	ownerAPIKey   string
}

// NewBlogHandler crée une nouvelle instance avec les deux dépendances.
func NewBlogHandler(mpfClient *services.MaiProFilesClient, blogGenerator *services.BlogGeneratorService, ownerAPIKey string) *BlogHandler {
	return &BlogHandler{
		mpfClient:     mpfClient,
		blogGenerator: blogGenerator,
		ownerAPIKey:   ownerAPIKey,
	}
}

// ownerOnly middleware qui vérifie la clé owner (X-Owner-Key header)
func (h *BlogHandler) ownerOnly(c *fiber.Ctx) error {
	if h.ownerAPIKey == "" || c.Get("X-Owner-Key") != h.ownerAPIKey {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "Valid X-Owner-Key header required",
		})
	}
	return c.Next()
}

// RegisterRoutes enregistre les routes du blog
func (h *BlogHandler) RegisterRoutes(router fiber.Router) {
	blog := router.Group("/blog")

	// Public routes
	blog.Get("/posts", h.ListPosts)
	blog.Get("/posts/:slug", h.GetPost)
	blog.Get("/feed.xml", h.GetRSSFeed)

	// Admin routes — protégées par X-Owner-Key
	admin := blog.Group("", h.ownerOnly)
	admin.Post("/posts", h.CreatePost)
	admin.Put("/posts/:id", h.UpdatePost)
	admin.Post("/generate", h.GeneratePost)
	admin.Post("/posts/:id/publish", h.PublishPost)
	admin.Post("/posts/:id/unpublish", h.UnpublishPost)
	admin.Delete("/posts/:id", h.DeletePost)
}

// ListPosts retourne la liste des articles publiés depuis maiProFiles.
// GET /api/v1/blog/posts?page=1&per_page=10
// Note: le paramètre "all" n'est plus supporté — maiProFiles ne sert que les posts publiés.
func (h *BlogHandler) ListPosts(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "10"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 50 {
		perPage = 10
	}

	response, err := h.mpfClient.GetBlogPosts(c.Context(), page, perPage)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "failed_to_list_posts",
			"message": err.Error(),
		})
	}

	return c.JSON(response)
}

// GetPost retourne un article par son slug depuis maiProFiles.
// GET /api/v1/blog/posts/:slug
func (h *BlogHandler) GetPost(c *fiber.Ctx) error {
	postSlug := c.Params("slug")
	if postSlug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "slug_required",
		})
	}

	post, err := h.mpfClient.GetBlogPost(c.Context(), postSlug)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "post_not_found",
			"message": err.Error(),
		})
	}

	return c.JSON(post)
}

// CreatePost crée un article directement dans maiProFiles depuis du contenu fourni.
// POST /api/v1/blog/posts
// Body: {"title": "...", "summary": "...", "content": "markdown...", "tags": [...], "publish": true}
func (h *BlogHandler) CreatePost(c *fiber.Ctx) error {
	var req models.BlogCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": err.Error(),
		})
	}

	if req.Title == "" || req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "title and content are required",
		})
	}

	// Construire un BlogPost depuis la request pour le passer au client MPF
	post := &models.BlogPost{
		Title:         req.Title,
		Summary:       req.Summary,
		Content:       req.Content,
		ProjectName:   req.ProjectName,
		Tags:          req.Tags,
		CoverImageURL: req.CoverImageURL,
		Published:     req.Publish,
	}

	created, err := h.mpfClient.CreateBlogPost(c.Context(), post)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "create_failed",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"post":    created,
	})
}

// UpdatePost met à jour un article existant dans maiProFiles.
// PUT /api/v1/blog/posts/:id
func (h *BlogHandler) UpdatePost(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid_id",
		})
	}

	var req models.BlogUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": err.Error(),
		})
	}

	// Construire un BlogPost partiel depuis la request de mise à jour
	post := &models.BlogPost{}
	if req.Title != nil {
		post.Title = *req.Title
	}
	if req.Summary != nil {
		post.Summary = *req.Summary
	}
	if req.Content != nil {
		post.Content = *req.Content
	}
	if len(req.Tags) > 0 {
		post.Tags = req.Tags
	}
	if req.CoverImageURL != nil {
		post.CoverImageURL = *req.CoverImageURL
	}

	updated, err := h.mpfClient.UpdateBlogPost(c.Context(), int(id), post)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "update_failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"post":    updated,
	})
}

// GeneratePost génère un nouvel article depuis les commits via IA (blogGenerator),
// puis le sauvegarde dans maiProFiles via mpfClient.CreateBlogPost.
// POST /api/v1/blog/generate
// Body: {"project_name": "maicivy", "auto_select": true}
func (h *BlogHandler) GeneratePost(c *fiber.Ctx) error {
	var req models.BlogGenerateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid_request",
		})
	}

	if req.ProjectName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project_name_required",
		})
	}

	// Générer le contenu markdown via IA (blogGenerator utilise encore son propre pipeline)
	// mais on NE sauvegarde plus dans PostgreSQL — on passe par mpfClient ensuite.
	var generatedPost *models.BlogPost
	var err error

	if req.AutoSelect || len(req.CommitSHAs) == 0 {
		// Générer depuis l'activité récente du repo
		generatedPost, err = h.blogGenerator.GenerateFromRecentActivity(c.Context(), req.ProjectName)
	} else {
		// Générer depuis des commits spécifiques fournis par l'appelant
		var commits []models.CommitRef
		for _, sha := range req.CommitSHAs {
			commits = append(commits, models.CommitRef{
				SHA:     sha,
				Message: sha, // Le message sera enrichi par le service si disponible
				Date:    time.Now().Format(time.RFC3339),
				Project: req.ProjectName,
			})
		}
		generatedPost, err = h.blogGenerator.GenerateFromCommits(c.Context(), req.ProjectName, commits)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "generation_failed",
			"message": err.Error(),
		})
	}

	// Sauvegarder dans maiProFiles (au lieu de PostgreSQL)
	// Le post généré est en draft (Published: false) par défaut — review manuelle requise.
	saved, err := h.mpfClient.CreateBlogPost(c.Context(), generatedPost)
	if err != nil {
		// Log l'erreur mais retourner quand même le post généré pour ne pas perdre le contenu
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "save_to_maiprofiles_failed",
			"message": err.Error(),
			"post":    generatedPost, // Le contenu généré est retourné pour récupération manuelle
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"post":    saved,
		"message": "Article généré et sauvegardé dans maiProFiles. Vérifiez et publiez manuellement.",
	})
}

// PublishPost publie un article dans maiProFiles.
// POST /api/v1/blog/posts/:id/publish
func (h *BlogHandler) PublishPost(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid_id",
		})
	}

	if err := h.mpfClient.PublishBlogPost(c.Context(), int(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "publish_failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Article publié",
	})
}

// UnpublishPost dépublie un article dans maiProFiles.
// POST /api/v1/blog/posts/:id/unpublish
func (h *BlogHandler) UnpublishPost(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid_id",
		})
	}

	if err := h.mpfClient.UnpublishBlogPost(c.Context(), int(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "unpublish_failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Article dépublié",
	})
}

// DeletePost supprime un article dans maiProFiles.
// DELETE /api/v1/blog/posts/:id
func (h *BlogHandler) DeletePost(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid_id",
		})
	}

	if err := h.mpfClient.DeleteBlogPost(c.Context(), int(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "delete_failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Article supprimé",
	})
}

// GetRSSFeed retourne le flux RSS du blog en fetchant les 20 derniers posts depuis maiProFiles.
// GET /api/v1/blog/feed.xml
func (h *BlogHandler) GetRSSFeed(c *fiber.Ctx) error {
	posts, err := h.mpfClient.GetBlogPosts(c.Context(), 1, 20)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed_to_generate_feed",
		})
	}

	// Générer le RSS XML
	rss := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
<channel>
  <title>maicivy Blog</title>
  <link>https://maicivy.com/blog</link>
  <description>Actualités développement et projets</description>
  <language>fr-FR</language>
  <atom:link href="https://maicivy.com/api/v1/blog/feed.xml" rel="self" type="application/rss+xml"/>
`

	for _, post := range posts.Posts {
		pubDate := ""
		if post.PublishedAt != nil {
			pubDate = post.PublishedAt.Format("Mon, 02 Jan 2006 15:04:05 -0700")
		}

		rss += `  <item>
    <title>` + escapeXML(post.Title) + `</title>
    <link>https://maicivy.com/blog/` + post.Slug + `</link>
    <description>` + escapeXML(post.Summary) + `</description>
    <pubDate>` + pubDate + `</pubDate>
    <guid>https://maicivy.com/blog/` + post.Slug + `</guid>
  </item>
`
	}

	rss += `</channel>
</rss>`

	c.Set("Content-Type", "application/rss+xml; charset=utf-8")
	return c.SendString(rss)
}

// escapeXML échappe les caractères spéciaux XML
func escapeXML(s string) string {
	replacer := map[string]string{
		"&":  "&amp;",
		"<":  "&lt;",
		">":  "&gt;",
		"\"": "&quot;",
		"'":  "&apos;",
	}
	for old, new := range replacer {
		for {
			if idx := indexOf(s, old); idx >= 0 {
				s = s[:idx] + new + s[idx+len(old):]
			} else {
				break
			}
		}
	}
	return s
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
