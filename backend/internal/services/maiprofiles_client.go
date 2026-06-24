package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"maicivy/internal/models"
)

// maiProFilesBaseURL — API de profil live
const maiProFilesBaseURL = "https://maiprofiles.etheryale.com"

// cacheTTL — durée de vie d'une entrée en cache (5 minutes)
const cacheTTL = 5 * time.Minute

// validLangs liste les langues pour lesquelles maiProFiles a réellement du contenu (fr/en + ka géorgien).
// Toute autre langue est repliée sur l'anglais (cf. langSuffix / fallbackLang).
var validLangs = map[string]bool{
	"fr": true,
	"en": true,
	"ka": true,
}

// fallbackLang — langue de repli quand la langue demandée n'est pas servie par maiProFiles.
// POURQUOI "en" et NON le défaut FR de l'API : le front expose de/it/zh, mais maiProFiles ne contient
// du contenu QUE pour fr/en/ka. Sans repli explicite, un visiteur de/it/zh recevait des fiches en
// FRANÇAIS (défaut silencieux de l'API) — incohérent avec une UI et une voix de chat allemandes.
// On sert l'ANGLAIS à la place (neutre, international). La voix du LLM reste, elle, dans la langue de
// l'utilisateur (géré côté chat_service, indépendant de ce repli données).
const fallbackLang = "en"

// --- Types API ---

// MPFProfile représente la réponse de GET /profile
type MPFProfile struct {
	Name            string         `json:"name"`
	Headline        string         `json:"headline"`
	Location        string         `json:"location"`
	ExperienceYears int            `json:"experience_years"`
	Contact         MPFContact     `json:"contact"`
	Bio             MPFBio         `json:"bio"`
	Skills          MPFSkills      `json:"skills"`
	Domains         []string       `json:"domains"`
	Links           MPFLinks       `json:"links"`
}

type MPFContact struct {
	Email    string `json:"email"`
	LinkedIn string `json:"linkedin"`
	GitHub   string `json:"github"`
	Gitea    string `json:"gitea"`
}

type MPFBio struct {
	Short string `json:"short"`
	Full  string `json:"full"`
}

type MPFSkills struct {
	Strong   []string `json:"strong"`
	Familiar []string `json:"familiar"`
}

type MPFLinks struct {
	Portfolio string `json:"portfolio"`
	GitHub    string `json:"github"`
	Gitea     string `json:"gitea"`
}

// MPFProject représente un projet retourné par GET /projects ou GET /projects/{id}
type MPFProject struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Category    string            `json:"category"`
	Status      string            `json:"status"`
	Featured    bool              `json:"featured"` // curé dans maiprofiles → projet vedette sur le CV
	Stack       []string          `json:"stack"`
	Stats       MPFStats          `json:"stats"`
	Description MPFDescription    `json:"description"`
	Tags        []string          `json:"tags"`
	Screenshots []MPFScreenshot   `json:"screenshots"`
	Links       map[string]string `json:"links"`
}

type MPFStats struct {
	LOC     int `json:"loc"`
	Tests   int `json:"tests"`
	Modules int `json:"modules"`
}

type MPFDescription struct {
	Short     string `json:"short"`
	Portfolio string `json:"portfolio"`
}

type MPFScreenshot struct {
	Filename string `json:"filename"`
	Section  string `json:"section"`
}

// MPFGlobalStats représente GET /stats
type MPFGlobalStats struct {
	Projects  int            `json:"projects"`
	TotalLOC  int            `json:"total_loc"`
	TotalTests int           `json:"total_tests"`
	Stack     map[string]int `json:"stack"`
}

// --- Cache générique ---

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

func (e *cacheEntry) isExpired() bool {
	return time.Now().After(e.expiresAt)
}

// MaiProFilesClient — client HTTP avec cache in-memory TTL 5min
type MaiProFilesClient struct {
	http    *http.Client
	baseURL string
	cache   sync.Map // clé string → *cacheEntry
}

func NewMaiProFilesClient() *MaiProFilesClient {
	return &MaiProFilesClient{
		http:    &http.Client{Timeout: 8 * time.Second},
		baseURL: maiProFilesBaseURL,
	}
}

// langSuffix retourne "&lang=<lang>" ou "?lang=<lang>" selon si la path contient déjà un "?".
// Permet d'ajouter le paramètre lang à un path qui a déjà d'autres query params (ex: /blog/posts?page=1).
//
// Repli (décision B) :
//   - lang servie par maiProFiles (fr/en/ka) → envoyée telle quelle ;
//   - lang vide → AUCUN param (défaut API = FR). Réservé aux appels internes non liés à une locale
//     visiteur (stats numériques, profil du prompt système) — on ne veut pas les basculer en EN ;
//   - toute autre lang (de/it/zh…) → repli ANGLAIS (fallbackLang), pour ne JAMAIS servir du français
//     à un visiteur de/it/zh.
func langSuffix(path, lang string) string {
	// lang vide = « défaut API » historique (FR) pour les appels internes : on n'ajoute pas de param.
	if lang == "" {
		return path
	}
	// Langue non servie par maiProFiles → repli sur l'anglais (et non le défaut FR silencieux).
	if !validLangs[lang] {
		lang = fallbackLang
	}
	// Détermine le séparateur selon la présence de query params existants
	for _, ch := range path {
		if ch == '?' {
			return path + "&lang=" + lang
		}
	}
	return path + "?lang=" + lang
}

// get fait un GET sur path et décode le JSON dans out.
// Si lang est une valeur valide (fr/en/ka), l'ajoute comme query param ?lang=.
// Cherche d'abord dans le cache (clé = path+lang), sinon fetch et met en cache.
func (c *MaiProFilesClient) get(ctx context.Context, path string, lang string, out interface{}) error {
	// Construire le path final avec le paramètre de langue si valide.
	// La clé de cache inclut le lang pour éviter des collisions inter-langues.
	fullPath := langSuffix(path, lang)

	// Vérifier le cache (clé = path localisé)
	if cached, ok := c.cache.Load(fullPath); ok {
		entry := cached.(*cacheEntry)
		if !entry.isExpired() {
			// Ré-encoder/décoder pour copier dans out (types différents selon l'appelant)
			b, err := json.Marshal(entry.value)
			if err == nil {
				if err = json.Unmarshal(b, out); err == nil {
					return nil
				}
			}
		}
	}

	// Fetch l'API avec le path localisé
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+fullPath, nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("maiprofiles fetch %s: %w", fullPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("not found: %s", fullPath)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("maiprofiles %s: HTTP %d", fullPath, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Décoder dans une interface{} générique pour le cache
	var raw interface{}
	if err = json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("json decode %s: %w", fullPath, err)
	}

	// Mettre en cache avec la clé localisée
	c.cache.Store(fullPath, &cacheEntry{value: raw, expiresAt: time.Now().Add(cacheTTL)})

	// Décoder dans le type cible
	return json.Unmarshal(body, out)
}

// GetProfile retourne le profil complet dans la langue demandée.
// lang : "fr" | "en" | "ka" servies tel quel ; de/it/zh repliées sur l'ANGLAIS ; "" → défaut API (FR).
func (c *MaiProFilesClient) GetProfile(ctx context.Context, lang string) (*MPFProfile, error) {
	var p MPFProfile
	err := c.get(ctx, "/profile", lang, &p)
	return &p, err
}

// ListProjects retourne tous les projets dans la langue demandée (sans description.portfolio).
// lang : "fr" | "en" | "ka" servies tel quel ; de/it/zh repliées sur l'ANGLAIS ; "" → défaut API (FR).
func (c *MaiProFilesClient) ListProjects(ctx context.Context, lang string) ([]MPFProject, error) {
	var projects []MPFProject
	err := c.get(ctx, "/projects", lang, &projects)
	return projects, err
}

// GetProject retourne les détails complets d'un projet par ID dans la langue demandée.
// lang : "fr" | "en" | "ka" servies tel quel ; de/it/zh repliées sur l'ANGLAIS ; "" → défaut API (FR).
func (c *MaiProFilesClient) GetProject(ctx context.Context, id string, lang string) (*MPFProject, error) {
	var p MPFProject
	err := c.get(ctx, "/projects/"+id, lang, &p)
	return &p, err
}

// GetStats retourne les stats globales (pas de traduction — données numériques uniquement).
func (c *MaiProFilesClient) GetStats(ctx context.Context) (*MPFGlobalStats, error) {
	var s MPFGlobalStats
	// Les stats sont purement numériques — pas besoin de lang
	err := c.get(ctx, "/stats", "", &s)
	return &s, err
}

// Search recherche des projets par mot-clé dans la langue demandée.
// lang : "fr" | "en" | "ka" servies tel quel ; de/it/zh repliées sur l'ANGLAIS ; "" → défaut API (FR).
func (c *MaiProFilesClient) Search(ctx context.Context, query string, lang string) ([]MPFProject, error) {
	var results []MPFProject
	path := "/search?q=" + url.QueryEscape(query)
	err := c.get(ctx, path, lang, &results)
	return results, err
}

// --- Expériences & skills (source du CV — endpoints maiProFiles /experiences et /skills) ---

// MPFLink représente un lien {name, url, icon} d'une expérience.
type MPFLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Icon string `json:"icon"`
}

// MPFExpDescription : les 3 niveaux de description d'une expérience.
type MPFExpDescription struct {
	Short      string `json:"short"`
	Technical  string `json:"technical"`
	Functional string `json:"functional"`
}

// MPFExperience représente une expérience retournée par GET /experiences.
type MPFExperience struct {
	ID           string            `json:"id"`
	Title        string            `json:"title"`
	Company      string            `json:"company"`
	Category     string            `json:"category"`
	StartDate    string            `json:"start_date"` // "YYYY-MM"
	EndDate      *string           `json:"end_date"`   // null = poste en cours
	Technologies []string          `json:"technologies"`
	Tags         []string          `json:"tags"`
	Featured     bool              `json:"featured"`
	Catchphrase  string            `json:"catchphrase"`
	Links        []MPFLink         `json:"links"`
	Description  MPFExpDescription `json:"description"`
}

// MPFSkill représente une compétence riche retournée par GET /skills.
type MPFSkill struct {
	Name        string   `json:"name"`
	Level       string   `json:"level"` // expert | advanced | intermediate | beginner
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	Years       int      `json:"years"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	Featured    bool     `json:"featured"`
}

// GetExperiences retourne les expériences pro dans la langue demandée.
func (c *MaiProFilesClient) GetExperiences(ctx context.Context, lang string) ([]MPFExperience, error) {
	var exps []MPFExperience
	err := c.get(ctx, "/experiences", lang, &exps)
	return exps, err
}

// GetSkills retourne les skills riches (niveau, tags, années...) dans la langue demandée.
func (c *MaiProFilesClient) GetSkills(ctx context.Context, lang string) ([]MPFSkill, error) {
	var skills []MPFSkill
	err := c.get(ctx, "/skills", lang, &skills)
	return skills, err
}

// InvalidateCache vide l'entrée en cache pour un path donné (utile après PATCH).
// Note : le path peut être avec ou sans suffix de langue — invalide uniquement la clé exacte.
func (c *MaiProFilesClient) InvalidateCache(path string) {
	c.cache.Delete(path)
}

// --- Méthodes Blog ---

// maiProFilesAPIKey retourne la clé d'auth Bearer pour les endpoints d'écriture.
// Lue depuis la variable d'env MAIPROFILES_API_KEY — non cachée dans la struct
// pour permettre la rotation sans restart.
func maiProFilesAPIKey() string {
	return os.Getenv("MAIPROFILES_API_KEY")
}

// doWriteRequest exécute une requête mutante (POST/PUT/DELETE) avec Bearer auth.
// statusExpected est le code HTTP attendu en succès (200, 201, 204...).
// Si body est nil, aucun payload n'est envoyé (ex: DELETE, publish/unpublish).
func (c *MaiProFilesClient) doWriteRequest(ctx context.Context, method, path string, body interface{}, out interface{}, statusExpected int) error {
	var bodyReader io.Reader

	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("json encode %s %s: %w", method, path, err)
		}
		bodyReader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	// Ajouter le Bearer token sur tous les endpoints d'écriture
	if key := maiProFilesAPIKey(); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("maiprofiles %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("not found: %s", path)
	}
	if resp.StatusCode != statusExpected {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("maiprofiles %s %s: HTTP %d — %s", method, path, resp.StatusCode, string(respBody))
	}

	// Certaines réponses sont vides (204 No Content)
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(respBody, out)
}

// MPFBlogPost représente un post tel que retourné par maiProFiles.
// Champs identiques à models.BlogPost mais sans les tags GORM ni les types SQL.
// Utilisé pour désérialiser la réponse avant de convertir vers models.BlogPost.
type MPFBlogPost struct {
	ID                   int              `json:"id"`
	Slug                 string           `json:"slug"`
	Title                string           `json:"title"`
	Summary              string           `json:"summary"`
	Content              string           `json:"content"`
	ContentHTML          *string          `json:"content_html"`
	ProjectName          string           `json:"project_name"`
	Tags                 []string         `json:"tags"`
	GeneratedFromCommits []models.CommitRef `json:"generated_from_commits"`
	CoverImageURL        string           `json:"cover_image_url,omitempty"`
	ReadingTimeMinutes   int              `json:"reading_time_minutes"`
	Published            bool             `json:"published"`
	PublishedAt          *string          `json:"published_at,omitempty"` // RFC3339 string depuis l'API
	CreatedAt            string           `json:"created_at"`
	UpdatedAt            string           `json:"updated_at"`
}

// MPFBlogListResponse représente la réponse paginée de GET /blog/posts
type MPFBlogListResponse struct {
	Posts   []MPFBlogPost `json:"posts"`
	Total   int64         `json:"total"`
	Page    int           `json:"page"`
	PerPage int           `json:"per_page"`
}

// toModel convertit un MPFBlogPost (types API) vers models.BlogPost (types internes).
// Les timestamps sont parsés depuis les strings RFC3339 retournées par maiProFiles.
func (p *MPFBlogPost) toModel() *models.BlogPost {
	m := &models.BlogPost{
		ID:                   uint(p.ID),
		Slug:                 p.Slug,
		Title:                p.Title,
		Summary:              p.Summary,
		Content:              p.Content,
		ProjectName:          p.ProjectName,
		Tags:                 p.Tags,
		GeneratedFromCommits: p.GeneratedFromCommits,
		CoverImageURL:        p.CoverImageURL,
		ReadingTimeMinutes:   p.ReadingTimeMinutes,
		Published:            p.Published,
	}

	// content_html peut être null dans la réponse API (non généré côté maiProFiles)
	if p.ContentHTML != nil {
		m.ContentHTML = *p.ContentHTML
	}

	// Parser les timestamps RFC3339
	if t, err := parseRFC3339Lax(p.CreatedAt); err == nil {
		m.CreatedAt = t
	}
	if t, err := parseRFC3339Lax(p.UpdatedAt); err == nil {
		m.UpdatedAt = t
	}
	if p.PublishedAt != nil {
		if t, err := parseRFC3339Lax(*p.PublishedAt); err == nil {
			m.PublishedAt = &t
		}
	}

	return m
}

// parseRFC3339Lax tente de parser un timestamp RFC3339 avec ou sans offset.
// maiProFiles peut retourner "+00:00" au lieu de "Z" — Go accepte les deux.
func parseRFC3339Lax(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// MPFBlogCreateRequest payload pour POST /blog/posts
type MPFBlogCreateRequest struct {
	Slug                 string             `json:"slug,omitempty"`
	Title                string             `json:"title"`
	Summary              string             `json:"summary"`
	Content              string             `json:"content"`
	ContentHTML          string             `json:"content_html,omitempty"`
	ProjectName          string             `json:"project_name"`
	Tags                 []string           `json:"tags"`
	GeneratedFromCommits []models.CommitRef `json:"generated_from_commits,omitempty"`
	CoverImageURL        string             `json:"cover_image_url,omitempty"`
	ReadingTimeMinutes   int                `json:"reading_time_minutes,omitempty"`
	Published            bool               `json:"published"`
}

// MPFBlogUpdateRequest payload pour PUT /blog/posts/{id}
type MPFBlogUpdateRequest struct {
	Title         *string  `json:"title,omitempty"`
	Summary       *string  `json:"summary,omitempty"`
	Content       *string  `json:"content,omitempty"`
	ContentHTML   *string  `json:"content_html,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	CoverImageURL *string  `json:"cover_image_url,omitempty"`
}

// GetBlogPosts retourne la liste des posts publiés depuis maiProFiles (paginée) dans la langue demandée.
// lang : "fr" | "en" | "ka" servies tel quel ; de/it/zh repliées sur EN ; "" → défaut API. (Le blog
// n'est pas localisé côté maiProFiles → le param n'a en pratique pas d'effet, mais on reste cohérent.)
// Résultats mis en cache TTL 5min avec clé par langue (invalidés explicitement après écriture).
func (c *MaiProFilesClient) GetBlogPosts(ctx context.Context, page, perPage int, lang string) (*models.BlogPostListResponse, error) {
	// Le path de base contient déjà des query params (page, per_page).
	// get() appellera langSuffix() pour ajouter "&lang=<lang>" correctement.
	path := fmt.Sprintf("/blog/posts?page=%d&per_page=%d", page, perPage)

	var raw MPFBlogListResponse
	if err := c.get(ctx, path, lang, &raw); err != nil {
		return nil, fmt.Errorf("GetBlogPosts: %w", err)
	}

	// Convertir chaque post vers le type interne
	posts := make([]models.BlogPost, len(raw.Posts))
	for i, p := range raw.Posts {
		posts[i] = *p.toModel()
	}

	totalPages := int(raw.Total) / raw.PerPage
	if raw.PerPage > 0 && int(raw.Total)%raw.PerPage > 0 {
		totalPages++
	}

	return &models.BlogPostListResponse{
		Posts:      posts,
		Total:      raw.Total,
		Page:       raw.Page,
		PerPage:    raw.PerPage,
		TotalPages: totalPages,
	}, nil
}

// GetBlogPost retourne un post complet par son slug (avec content markdown) dans la langue demandée.
// lang : "fr" | "en" | "ka" servies tel quel ; de/it/zh repliées sur l'ANGLAIS ; "" → défaut API (FR).
func (c *MaiProFilesClient) GetBlogPost(ctx context.Context, postSlug string, lang string) (*models.BlogPost, error) {
	path := "/blog/posts/" + url.PathEscape(postSlug)

	var raw MPFBlogPost
	if err := c.get(ctx, path, lang, &raw); err != nil {
		return nil, fmt.Errorf("GetBlogPost(%s): %w", postSlug, err)
	}

	return raw.toModel(), nil
}

// CreateBlogPost crée un post dans maiProFiles via POST /blog/posts.
// Nécessite MAIPROFILES_API_KEY. Invalide le cache de liste après création.
func (c *MaiProFilesClient) CreateBlogPost(ctx context.Context, post *models.BlogPost) (*models.BlogPost, error) {
	// Garantir un slug : maiProFiles n'en génère PAS, et le handler API ne propage aucun slug
	// client (BlogCreateRequest n'a pas de champ slug). Sans slug → URL publique cassée
	// (/blog/) + anti-doublon WanMira cassé (post_exists sur slug vide). On le dérive du titre
	// ICI, point de passage unique de TOUTES les créations (CreatePost handler + GeneratePost).
	if post.Slug == "" {
		post.Slug = toSlug(post.Title)
	}
	// Plafonner à 80 chars (comme WanMira slugify + les slugs existants). POURQUOI : cohérence
	// d'URL ET surtout d'anti-doublon — WanMira post_exists() interroge GET /blog/posts/{slug}
	// avec slugify(title) tronqué à 80 ; un slug serveur plus long ne matcherait jamais → doublons.
	if len(post.Slug) > 80 {
		post.Slug = strings.TrimRight(post.Slug[:80], "-")
	}
	// Pré-rendre le markdown en HTML (goldmark : GFM + coloration syntaxique). POURQUOI : le
	// frontend affiche content_html ; sans lui il tombe sur le markdown BRUT (moche). maiProFiles
	// ne rend rien — c'est ici que ça doit se faire (comme GeneratePost le fait déjà).
	if post.ContentHTML == "" {
		post.ContentHTML = markdownToHTML(post.Content)
	}

	payload := MPFBlogCreateRequest{
		Slug:                 post.Slug,
		Title:                post.Title,
		Summary:              post.Summary,
		Content:              post.Content,
		ContentHTML:          post.ContentHTML,
		ProjectName:          post.ProjectName,
		Tags:                 post.Tags,
		GeneratedFromCommits: post.GeneratedFromCommits,
		CoverImageURL:        post.CoverImageURL,
		ReadingTimeMinutes:   post.ReadingTimeMinutes,
		Published:            post.Published,
	}

	var raw MPFBlogPost
	if err := c.doWriteRequest(ctx, http.MethodPost, "/blog/posts", payload, &raw, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("CreateBlogPost: %w", err)
	}

	// Invalider le cache de liste (les pages 1..N sont désormais stales)
	c.invalidateBlogListCache()

	return raw.toModel(), nil
}

// UpdateBlogPost met à jour un post dans maiProFiles via PUT /blog/posts/{id}.
// Nécessite MAIPROFILES_API_KEY. Invalide le cache du post et de la liste.
func (c *MaiProFilesClient) UpdateBlogPost(ctx context.Context, id int, post *models.BlogPost) (*models.BlogPost, error) {
	payload := MPFBlogUpdateRequest{
		Title:   &post.Title,
		Summary: &post.Summary,
		Content: &post.Content,
		Tags:    post.Tags,
	}
	// Si le contenu change, re-rendre content_html (goldmark) pour garder le HTML synchro.
	if post.Content != "" {
		h := markdownToHTML(post.Content)
		payload.ContentHTML = &h
	}
	if post.CoverImageURL != "" {
		payload.CoverImageURL = &post.CoverImageURL
	}

	var raw MPFBlogPost
	path := fmt.Sprintf("/blog/posts/%d", id)
	if err := c.doWriteRequest(ctx, http.MethodPut, path, payload, &raw, http.StatusOK); err != nil {
		return nil, fmt.Errorf("UpdateBlogPost(%d): %w", id, err)
	}

	// Invalider le cache du post individuel et la liste (toutes les langues connues)
	for lang := range validLangs {
		c.InvalidateCache("/blog/posts/" + raw.Slug + "?lang=" + lang)
	}
	c.InvalidateCache("/blog/posts/" + raw.Slug)
	c.invalidateBlogListCache()

	return raw.toModel(), nil
}

// DeleteBlogPost supprime un post via DELETE /blog/posts/{id}.
// Nécessite MAIPROFILES_API_KEY.
func (c *MaiProFilesClient) DeleteBlogPost(ctx context.Context, id int) error {
	path := fmt.Sprintf("/blog/posts/%d", id)
	// maiProFiles renvoie 200 (dict {"success": true}) sur DELETE, pas 204.
	if err := c.doWriteRequest(ctx, http.MethodDelete, path, nil, nil, http.StatusOK); err != nil {
		return fmt.Errorf("DeleteBlogPost(%d): %w", id, err)
	}
	// Invalider toute la liste (on ne connaît pas le slug depuis l'id seul)
	c.invalidateBlogListCache()
	return nil
}

// PublishBlogPost publie un post via POST /blog/posts/{id}/publish.
// Nécessite MAIPROFILES_API_KEY.
func (c *MaiProFilesClient) PublishBlogPost(ctx context.Context, id int) error {
	path := fmt.Sprintf("/blog/posts/%d/publish", id)
	if err := c.doWriteRequest(ctx, http.MethodPost, path, nil, nil, http.StatusOK); err != nil {
		return fmt.Errorf("PublishBlogPost(%d): %w", id, err)
	}
	c.invalidateBlogListCache()
	return nil
}

// UnpublishBlogPost dépublie un post via POST /blog/posts/{id}/unpublish.
// Nécessite MAIPROFILES_API_KEY.
func (c *MaiProFilesClient) UnpublishBlogPost(ctx context.Context, id int) error {
	path := fmt.Sprintf("/blog/posts/%d/unpublish", id)
	if err := c.doWriteRequest(ctx, http.MethodPost, path, nil, nil, http.StatusOK); err != nil {
		return fmt.Errorf("UnpublishBlogPost(%d): %w", id, err)
	}
	c.invalidateBlogListCache()
	return nil
}

// invalidateBlogListCache supprime toutes les entrées de cache /blog/posts?...
// On invalide les pages courantes les plus communes (page 1 à 5, per_page 10 et 20),
// pour chaque langue supportée + sans langue (fallback FR).
// Cette approche est simple et suffisante pour le volume de posts attendu.
func (c *MaiProFilesClient) invalidateBlogListCache() {
	langs := []string{"", "fr", "en", "ka"}
	for page := 1; page <= 5; page++ {
		for _, pp := range []int{10, 20} {
			for _, lang := range langs {
				base := fmt.Sprintf("/blog/posts?page=%d&per_page=%d", page, pp)
				if lang != "" {
					c.InvalidateCache(base + "&lang=" + lang)
				} else {
					c.InvalidateCache(base)
				}
			}
		}
	}
}
