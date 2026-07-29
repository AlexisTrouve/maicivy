package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"maicivy/internal/models"
)

// GitHubOAuthService gère l'authentification OAuth GitHub
type GitHubOAuthService struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewGitHubOAuthService crée une nouvelle instance
func NewGitHubOAuthService(db *gorm.DB, redis *redis.Client) *GitHubOAuthService {
	return &GitHubOAuthService{
		db:    db,
		redis: redis,
	}
}

// GitHubUser représente les infos utilisateur GitHub
type GitHubUser struct {
	Login     string `json:"login"`
	ID        int64  `json:"id"`
	AvatarURL string `json:"avatar_url"`
	Name      string `json:"name"`
	Email     string `json:"email"`
}

// GenerateAuthURL génère l'URL d'authentification GitHub avec state CSRF
func (s *GitHubOAuthService) GenerateAuthURL(ctx context.Context) (string, error) {
	state := generateRandomState()

	// Stocker state en Redis (TTL 10 min) pour protection CSRF
	err := s.redis.Set(ctx, "github:oauth:state:"+state, "true", 10*time.Minute).Err()
	if err != nil {
		return "", fmt.Errorf("failed to store state: %w", err)
	}

	clientID := os.Getenv("GITHUB_CLIENT_ID")
	redirectURI := os.Getenv("GITHUB_REDIRECT_URI")

	if clientID == "" || redirectURI == "" {
		return "", fmt.Errorf("missing GITHUB_CLIENT_ID or GITHUB_REDIRECT_URI env vars")
	}

	url := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		clientID,
		redirectURI,
		"user:email,public_repo",
		state,
	)

	return url, nil
}

// HandleCallback traite la réponse OAuth et échange le code contre un token
func (s *GitHubOAuthService) HandleCallback(ctx context.Context, code, state string) (*models.GitHubProfile, error) {
	// Vérifier state CSRF
	key := "github:oauth:state:" + state
	exists, err := s.redis.Exists(ctx, key).Result()
	if err != nil || exists == 0 {
		return nil, fmt.Errorf("invalid state: CSRF protection")
	}

	// Supprimer state (usage unique)
	s.redis.Del(ctx, key)

	// Échanger code contre access token
	token, err := s.exchangeCodeForToken(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	// Récupérer infos utilisateur GitHub
	githubUser, err := s.getGitHubUser(ctx, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GitHub user: %w", err)
	}

	// Sauvegarder/mettre à jour profil en DB
	profile := &models.GitHubProfile{
		Username:    githubUser.Login,
		Token:       *token,
		ConnectedAt: time.Now().Unix(),
		SyncedAt:    0, // Pas encore synchronisé
	}

	// Upsert (create or update)
	result := s.db.Where("username = ?", profile.Username).FirstOrCreate(profile)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to save profile: %w", result.Error)
	}

	// Si profil existe déjà, mettre à jour le token
	if result.RowsAffected == 0 {
		s.db.Model(profile).Where("username = ?", profile.Username).Updates(map[string]interface{}{
			"token":        profile.Token,
			"connected_at": profile.ConnectedAt,
		})
	}

	return profile, nil
}

// RefreshToken rafraîchit le token GitHub si expiré
func (s *GitHubOAuthService) RefreshToken(ctx context.Context, username string) error {
	// Note: GitHub OAuth tokens ne nécessitent pas de refresh explicite
	// Ils n'expirent pas par défaut, sauf révocation manuelle
	// Cette méthode est un placeholder pour compatibilité future

	var profile models.GitHubProfile
	if err := s.db.Where("username = ?", username).First(&profile).Error; err != nil {
		return fmt.Errorf("profile not found: %w", err)
	}

	// Vérifier si token expiré
	if profile.Token.ExpiresAt > 0 && time.Now().Unix() > profile.Token.ExpiresAt {
		return fmt.Errorf("token expired - user must re-authenticate")
	}

	return nil
}

// exchangeCodeForToken échange le code OAuth contre un access token
func (s *GitHubOAuthService) exchangeCodeForToken(ctx context.Context, code string) (*models.GitHubToken, error) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("missing GITHUB_CLIENT_ID or GITHUB_CLIENT_SECRET")
	}

	client := resty.New()
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetFormData(map[string]string{
			"client_id":     clientID,
			"client_secret": clientSecret,
			"code":          code,
		}).
		Post("https://github.com/login/oauth/access_token")

	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("GitHub API error: %s", resp.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Vérifier erreur dans la réponse
	if errMsg, exists := result["error"]; exists {
		return nil, fmt.Errorf("GitHub OAuth error: %v", errMsg)
	}

	accessToken, ok := result["access_token"].(string)
	if !ok || accessToken == "" {
		return nil, fmt.Errorf("no access token in response")
	}

	tokenType, _ := result["token_type"].(string)
	if tokenType == "" {
		tokenType = "bearer"
	}

	token := &models.GitHubToken{
		AccessToken: accessToken,
		TokenType:   tokenType,
		ExpiresAt:   0, // GitHub tokens don't expire by default
	}

	return token, nil
}

// getGitHubUser récupère les infos utilisateur depuis GitHub API
func (s *GitHubOAuthService) getGitHubUser(ctx context.Context, accessToken string) (*GitHubUser, error) {
	client := resty.New()
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+accessToken).
		SetHeader("Accept", "application/vnd.github.v3+json").
		Get("https://api.github.com/user")

	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("GitHub API error: %s", resp.String())
	}

	var user GitHubUser
	if err := json.Unmarshal(resp.Body(), &user); err != nil {
		return nil, fmt.Errorf("failed to parse user: %w", err)
	}

	return &user, nil
}

// generateRandomState génère un state random pour protection CSRF
func generateRandomState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
