package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// RepoScanner scanne les repos git locaux et génère le feed d'activité
type RepoScanner struct {
	redis    *redis.Client
	reposDir string
	cacheTTL time.Duration
}

// NewRepoScanner crée une nouvelle instance
func NewRepoScanner(redis *redis.Client, reposDir string) *RepoScanner {
	return &RepoScanner{
		redis:    redis,
		reposDir: reposDir,
		cacheTTL: 5 * time.Minute,
	}
}

// ScanResult représente le résultat du scan
type ScanResult struct {
	LastUpdated string          `json:"last_updated"`
	Projects    []ScannedProject `json:"projects"`
	Stats       ScanStats        `json:"stats"`
}

// ScannedProject représente un projet scanné
type ScannedProject struct {
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	RepoURL       string         `json:"repo_url"`
	Category      string         `json:"category"`
	Showcase      bool           `json:"showcase"`
	Languages     []string       `json:"languages"`
	Commits7d     int            `json:"commits_7d"`
	Commits30d    int            `json:"commits_30d"`
	RecentCommits []ScannedCommit `json:"recent_commits"`
	LastActivity  string         `json:"last_activity"`
}

// ScannedCommit représente un commit scanné
type ScannedCommit struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
	Date    string `json:"date"`
	Author  string `json:"author"`
}

// ScanStats représente les statistiques globales
type ScanStats struct {
	TotalCommits30d int      `json:"total_commits_30d"`
	ActiveProjects  int      `json:"active_projects"`
	TopLanguages    []string `json:"top_languages"`
}

// showcaseProjects définit les projets à mettre en avant
var showcaseProjects = map[string]bool{
	"maicivy":                 true,
	"ProjectTracker":          true,
	"videotomp3transcriptor":  true,
}

// projectDescriptions contient les descriptions des projets
var projectDescriptions = map[string]string{
	"maicivy":                 "Portfolio/CV automatisé avec sync GitHub",
	"ProjectTracker":          "Outil de suivi de projets multi-repos",
	"videotomp3transcriptor":  "Convertisseur vidéo vers MP3 avec transcription",
	"confluent":               "Projet Confluent",
}

// Scan effectue le scan de tous les repos
func (s *RepoScanner) Scan(ctx context.Context) (*ScanResult, error) {
	// Check cache
	cacheKey := "activity:scan"
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var result ScanResult
		if err := json.Unmarshal([]byte(cached), &result); err == nil {
			log.Debug().Msg("Returning cached scan result")
			return &result, nil
		}
	}

	log.Info().Str("dir", s.reposDir).Msg("Scanning repositories")

	// Lister les dossiers dans reposDir
	entries, err := os.ReadDir(s.reposDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read repos directory: %w", err)
	}

	var projects []ScannedProject
	languageCount := make(map[string]int)
	totalCommits30d := 0
	activeProjects := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		repoPath := filepath.Join(s.reposDir, entry.Name())
		gitDir := filepath.Join(repoPath, ".git")

		// Vérifier si c'est un repo git
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			continue
		}

		project, err := s.scanRepo(ctx, repoPath, entry.Name())
		if err != nil {
			log.Warn().Err(err).Str("repo", entry.Name()).Msg("Failed to scan repo")
			continue
		}

		if project.Commits30d > 0 {
			activeProjects++
		}
		totalCommits30d += project.Commits30d

		for _, lang := range project.Languages {
			languageCount[lang]++
		}

		projects = append(projects, *project)
	}

	// Trier par activité
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Commits30d > projects[j].Commits30d
	})

	// Top languages
	topLangs := getTopLanguages(languageCount, 6)

	result := &ScanResult{
		LastUpdated: time.Now().Format(time.RFC3339),
		Projects:    projects,
		Stats: ScanStats{
			TotalCommits30d: totalCommits30d,
			ActiveProjects:  activeProjects,
			TopLanguages:    topLangs,
		},
	}

	// Cache result
	data, _ := json.Marshal(result)
	s.redis.Set(ctx, cacheKey, string(data), s.cacheTTL)

	log.Info().Int("projects", len(projects)).Int("commits", totalCommits30d).Msg("Scan completed")
	return result, nil
}

func (s *RepoScanner) scanRepo(ctx context.Context, repoPath, name string) (*ScannedProject, error) {
	// Commits 7 jours
	commits7d, err := s.countCommits(repoPath, 7)
	if err != nil {
		return nil, err
	}

	// Commits 30 jours
	commits30d, err := s.countCommits(repoPath, 30)
	if err != nil {
		return nil, err
	}

	// Recent commits
	recentCommits, err := s.getRecentCommits(repoPath, 5)
	if err != nil {
		return nil, err
	}

	// Languages
	languages, err := s.detectLanguages(repoPath)
	if err != nil {
		languages = []string{}
	}

	// Remote URL
	repoURL := s.getRemoteURL(repoPath)

	// Last activity
	lastActivity := time.Now().Format(time.RFC3339)
	if len(recentCommits) > 0 {
		lastActivity = recentCommits[0].Date
	}

	// Category
	category := "WIP"
	if commits30d > 10 {
		category = "Production"
	} else if commits30d == 0 {
		category = "Archive"
	}

	description := projectDescriptions[name]
	if description == "" {
		description = fmt.Sprintf("Repository %s", name)
	}

	return &ScannedProject{
		Name:          name,
		Description:   description,
		RepoURL:       repoURL,
		Category:      category,
		Showcase:      showcaseProjects[name],
		Languages:     languages,
		Commits7d:     commits7d,
		Commits30d:    commits30d,
		RecentCommits: recentCommits,
		LastActivity:  lastActivity,
	}, nil
}

func (s *RepoScanner) countCommits(repoPath string, days int) (int, error) {
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	cmd := exec.Command("git", "-C", repoPath, "rev-list", "--count", "--since="+since, "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var count int
	fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &count)
	return count, nil
}

func (s *RepoScanner) getRecentCommits(repoPath string, limit int) ([]ScannedCommit, error) {
	format := "%H|%s|%aI|%an"
	cmd := exec.Command("git", "-C", repoPath, "log", fmt.Sprintf("-%d", limit), fmt.Sprintf("--format=%s", format))
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var commits []ScannedCommit
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		commits = append(commits, ScannedCommit{
			SHA:     parts[0][:7], // Short SHA
			Message: truncateMessage(parts[1], 80),
			Date:    parts[2],
			Author:  parts[3],
		})
	}
	return commits, nil
}

func (s *RepoScanner) detectLanguages(repoPath string) ([]string, error) {
	langMap := map[string]string{
		".go":    "Go",
		".ts":    "TypeScript",
		".tsx":   "TypeScript",
		".js":    "JavaScript",
		".jsx":   "JavaScript",
		".py":    "Python",
		".rs":    "Rust",
		".cpp":   "C++",
		".c":     "C",
		".java":  "Java",
		".rb":    "Ruby",
		".php":   "PHP",
		".swift": "Swift",
		".kt":    "Kotlin",
		".vue":   "Vue",
		".svelte": "Svelte",
	}

	detected := make(map[string]bool)

	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			// Skip common non-source directories
			base := filepath.Base(path)
			if base == "node_modules" || base == "vendor" || base == ".git" || base == "dist" || base == "build" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if lang, ok := langMap[ext]; ok {
			detected[lang] = true
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	var languages []string
	for lang := range detected {
		languages = append(languages, lang)
	}
	sort.Strings(languages)
	return languages, nil
}

func (s *RepoScanner) getRemoteURL(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	url := strings.TrimSpace(string(output))

	// Convert SSH to HTTPS
	if strings.HasPrefix(url, "git@") {
		re := regexp.MustCompile(`git@([^:]+):(.+)\.git`)
		url = re.ReplaceAllString(url, "https://$1/$2")
	}
	return url
}

func truncateMessage(msg string, maxLen int) string {
	// Remove newlines
	msg = strings.ReplaceAll(msg, "\n", " ")
	if len(msg) > maxLen {
		return msg[:maxLen-3] + "..."
	}
	return msg
}

func getTopLanguages(counts map[string]int, limit int) []string {
	type langCount struct {
		lang  string
		count int
	}
	var sorted []langCount
	for lang, count := range counts {
		sorted = append(sorted, langCount{lang, count})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	var result []string
	for i, lc := range sorted {
		if i >= limit {
			break
		}
		result = append(result, lc.lang)
	}
	return result
}

// InvalidateCache invalide le cache
func (s *RepoScanner) InvalidateCache(ctx context.Context) {
	s.redis.Del(ctx, "activity:scan")
}
