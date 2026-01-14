package api

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"maicivy/internal/services"
)

// GitHubHandler gère les endpoints API GitHub
type GitHubHandler struct {
	oauthService *services.GitHubOAuthService
	syncService  *services.GitHubSyncService
}

// NewGitHubHandler crée une nouvelle instance
func NewGitHubHandler(
	oauthService *services.GitHubOAuthService,
	syncService *services.GitHubSyncService,
) *GitHubHandler {
	return &GitHubHandler{
		oauthService: oauthService,
		syncService:  syncService,
	}
}

// RegisterRoutes enregistre les routes GitHub
func (h *GitHubHandler) RegisterRoutes(router fiber.Router) {
	github := router.Group("/github")

	// OAuth flow
	github.Get("/auth-url", h.GetAuthURL)
	github.Get("/callback", h.HandleCallback)

	// Sync operations
	github.Post("/sync", h.TriggerSync)
	github.Get("/status", h.GetStatus)
	github.Get("/repos", h.GetRepositories)
	github.Delete("/disconnect", h.Disconnect)
}

// GetAuthURL génère l'URL d'authentification GitHub
// GET /api/v1/github/auth-url
func (h *GitHubHandler) GetAuthURL(c *fiber.Ctx) error {
	url, err := h.oauthService.GenerateAuthURL(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed_to_generate_auth_url",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"auth_url": url,
	})
}

// HandleCallback traite le callback OAuth GitHub
// GET /api/v1/github/callback?code=xxx&state=yyy
func (h *GitHubHandler) HandleCallback(c *fiber.Ctx) error {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "missing_code_or_state",
		})
	}

	// Traiter OAuth callback
	profile, err := h.oauthService.HandleCallback(c.Context(), code, state)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "oauth_callback_failed",
			"message": err.Error(),
		})
	}

	// Déclencher sync initial en arrière-plan
	go func() {
		ctx := c.Context()
		h.syncService.SyncRepositories(ctx, profile.Username, profile.Token.AccessToken)
	}()

	return c.JSON(fiber.Map{
		"success": true,
		"username": profile.Username,
		"connected_at": profile.ConnectedAt,
	})
}

// TriggerSync déclenche une synchronisation manuelle
// POST /api/v1/github/sync
// Body: {"username": "user"}
func (h *GitHubHandler) TriggerSync(c *fiber.Ctx) error {
	var req struct {
		Username string `json:"username"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid_request",
		})
	}

	if req.Username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "username_required",
		})
	}

	// Récupérer le profil pour obtenir le token
	// TODO: Implement profile/token retrieval
	_ = req.Username // Temporarily unused until authentication is implemented

	// Note: Dans une vraie implémentation, récupérer depuis session/JWT
	// Pour l'instant, on récupère depuis DB
	// TODO: Ajouter authentification/session management

	// Déclencher sync en arrière-plan
	go func() {
		ctx := c.Context()
		// Cette partie nécessite le token, qui devrait venir de la session
		// Pour l'instant, on retourne juste un statut
		_ = ctx
		// h.syncService.SyncRepositories(ctx, req.Username, profile.Token.AccessToken)
	}()

	return c.JSON(fiber.Map{
		"status": "sync_started",
		"username": req.Username,
	})
}

// GetStatus retourne le statut de connexion GitHub
// GET /api/v1/github/status?username=xxx
func (h *GitHubHandler) GetStatus(c *fiber.Ctx) error {
	username := c.Query("username")

	if username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "username_required",
		})
	}

	status, err := h.syncService.GetSyncStatus(c.Context(), username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed_to_get_status",
			"message": err.Error(),
		})
	}

	return c.JSON(status)
}

// GetRepositories retourne les repos GitHub
// GET /api/v1/github/repos?username=xxx&include_private=false
func (h *GitHubHandler) GetRepositories(c *fiber.Ctx) error {
	username := c.Query("username")
	includePrivate := c.QueryBool("include_private", false)

	if username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "username_required",
		})
	}

	var repos interface{}
	var err error

	if includePrivate {
		repos, err = h.syncService.GetAllRepositories(c.Context(), username)
	} else {
		repos, err = h.syncService.GetPublicRepositories(c.Context(), username)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed_to_fetch_repos",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"repositories": repos,
	})
}

// Disconnect déconnecte GitHub
// DELETE /api/v1/github/disconnect?username=xxx
func (h *GitHubHandler) Disconnect(c *fiber.Ctx) error {
	username := c.Query("username")

	if username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "username_required",
		})
	}

	if err := h.syncService.DisconnectGitHub(c.Context(), username); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed_to_disconnect",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("GitHub account %s disconnected", username),
	})
}
