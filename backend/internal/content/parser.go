package content

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gopkg.in/yaml.v3"

	"maicivy/internal/models"
)

// experienceFrontmatter represents the YAML frontmatter of an experience file
type experienceFrontmatter struct {
	Title        string         `yaml:"title"`
	Company      string         `yaml:"company"`
	Category     string         `yaml:"category"`
	StartDate    string         `yaml:"start_date"`
	EndDate      *string        `yaml:"end_date"`
	Technologies []string       `yaml:"technologies"`
	Tags         []string       `yaml:"tags"`
	Featured     bool           `yaml:"featured"`
	Catchphrase  string         `yaml:"catchphrase"`
	Images       []string       `yaml:"images"`
	Links        []linkDataYAML `yaml:"links"`
}

// linkDataYAML represents a link in YAML frontmatter
type linkDataYAML struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
	Icon string `yaml:"icon"`
}

// projectFrontmatter represents the YAML frontmatter of a project file
type projectFrontmatter struct {
	Title          string         `yaml:"title"`
	Category       string         `yaml:"category"`
	Technologies   []string       `yaml:"technologies"`
	Featured       bool           `yaml:"featured"`
	InProgress     bool           `yaml:"in_progress"`
	GithubURL      string         `yaml:"github_url"`
	DemoURL        string         `yaml:"demo_url"`
	ImageURL       string         `yaml:"image_url"`
	GithubLanguage string         `yaml:"github_language"`
	Catchphrase    string         `yaml:"catchphrase"`
	Images         []string       `yaml:"images"`
	Links          []linkDataYAML `yaml:"links"`
}

// toLinksJSON converts YAML link data to the model's LinksJSON type.
func toLinksJSON(links []linkDataYAML) models.LinksJSON {
	if len(links) == 0 {
		return nil
	}
	result := make(models.LinksJSON, len(links))
	for i, l := range links {
		result[i] = models.LinkData{
			Name: l.Name,
			URL:  l.URL,
			Icon: l.Icon,
		}
	}
	return result
}

// parseFrontmatter splits a markdown file into YAML frontmatter and body sections.
func parseFrontmatter(content string) (frontmatter string, body string, err error) {
	lines := strings.SplitAfter(content, "\n")

	// Find opening ---
	start := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			start = i
			break
		}
		// Allow empty lines before frontmatter
		if trimmed != "" {
			return "", content, nil // No frontmatter
		}
	}

	if start == -1 {
		return "", content, nil
	}

	// Find closing ---
	for i := start + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			frontmatter = strings.Join(lines[start+1:i], "")
			body = strings.Join(lines[i+1:], "")
			return frontmatter, body, nil
		}
	}

	return "", content, fmt.Errorf("unclosed frontmatter")
}

// extractSection extracts content under a specific ## heading from markdown body.
func extractSection(body string, heading string) string {
	scanner := bufio.NewScanner(strings.NewReader(body))
	var result strings.Builder
	inSection := false

	target := "## " + heading

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "## ") {
			if trimmed == target {
				inSection = true
				continue
			}
			if inSection {
				break // Hit next section
			}
			continue
		}

		if inSection {
			result.WriteString(line)
			result.WriteString("\n")
		}
	}

	return strings.TrimSpace(result.String())
}

// parseDate parses a YYYY-MM date string into time.Time.
func parseDate(s string) (time.Time, error) {
	// Try YYYY-MM first
	t, err := time.Parse("2006-01", s)
	if err == nil {
		return t, nil
	}
	// Try YYYY-MM-DD
	return time.Parse("2006-01-02", s)
}

// ParseExperience parses a markdown file content into an Experience model.
func ParseExperience(content string) (models.Experience, error) {
	fm, body, err := parseFrontmatter(content)
	if err != nil {
		return models.Experience{}, fmt.Errorf("parsing frontmatter: %w", err)
	}

	var meta experienceFrontmatter
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return models.Experience{}, fmt.Errorf("parsing YAML: %w", err)
	}

	startDate, err := parseDate(meta.StartDate)
	if err != nil {
		return models.Experience{}, fmt.Errorf("parsing start_date %q: %w", meta.StartDate, err)
	}

	var endDate *time.Time
	if meta.EndDate != nil && *meta.EndDate != "" && *meta.EndDate != "null" {
		t, err := parseDate(*meta.EndDate)
		if err != nil {
			return models.Experience{}, fmt.Errorf("parsing end_date %q: %w", *meta.EndDate, err)
		}
		endDate = &t
	}

	exp := models.Experience{
		Title:                 meta.Title,
		Company:               meta.Company,
		Category:              meta.Category,
		StartDate:             startDate,
		EndDate:               endDate,
		Technologies:          pq.StringArray(meta.Technologies),
		Tags:                  pq.StringArray(meta.Tags),
		Featured:              meta.Featured,
		Catchphrase:           meta.Catchphrase,
		Images:                pq.StringArray(meta.Images),
		Links:                 toLinksJSON(meta.Links),
		Description:           extractSection(body, "Description"),
		TechnicalDescription:  extractSection(body, "Description Technique"),
		FunctionalDescription: extractSection(body, "Description Fonctionnelle"),
	}
	exp.ID = uuid.New()

	return exp, nil
}

// ParseProject parses a markdown file content into a Project model.
func ParseProject(content string) (models.Project, error) {
	fm, body, err := parseFrontmatter(content)
	if err != nil {
		return models.Project{}, fmt.Errorf("parsing frontmatter: %w", err)
	}

	var meta projectFrontmatter
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return models.Project{}, fmt.Errorf("parsing YAML: %w", err)
	}

	proj := models.Project{
		Title:          meta.Title,
		Category:       meta.Category,
		Technologies:   pq.StringArray(meta.Technologies),
		Featured:       meta.Featured,
		InProgress:     meta.InProgress,
		GithubURL:      meta.GithubURL,
		DemoURL:        meta.DemoURL,
		ImageURL:       meta.ImageURL,
		GithubLanguage: meta.GithubLanguage,
		Catchphrase:    meta.Catchphrase,
		Images:         pq.StringArray(meta.Images),
		Links:          toLinksJSON(meta.Links),
		Description:    extractSection(body, "Description"),
	}
	proj.ID = uuid.New()

	// Also extract optional sections
	if tech := extractSection(body, "Description Technique"); tech != "" {
		proj.TechnicalDescription = tech
	}
	if fn := extractSection(body, "Description Fonctionnelle"); fn != "" {
		proj.FunctionalDescription = fn
	}

	return proj, nil
}

// ParseSkills parses the skills.md file into a slice of Skill models.
// Format per line: - Name | Category | tags (csv) | years | description | icon [| featured]
func ParseSkills(content string) ([]models.Skill, error) {
	_, body, err := parseFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", err)
	}
	// If no frontmatter, use full content
	if body == "" {
		body = content
	}

	var skills []models.Skill
	var currentLevel models.SkillLevel

	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Detect level headings
		if strings.HasPrefix(line, "## ") {
			levelStr := strings.ToLower(strings.TrimPrefix(line, "## "))
			switch levelStr {
			case "expert":
				currentLevel = models.SkillLevelExpert
			case "advanced":
				currentLevel = models.SkillLevelAdvanced
			case "intermediate":
				currentLevel = models.SkillLevelIntermediate
			case "beginner":
				currentLevel = models.SkillLevelBeginner
			}
			continue
		}

		// Parse skill lines: - Name | Category | tags | years | description | icon [| featured]
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		line = strings.TrimPrefix(line, "- ")

		parts := strings.Split(line, " | ")
		if len(parts) < 5 {
			continue // Skip malformed lines
		}

		name := strings.TrimSpace(parts[0])
		category := strings.TrimSpace(parts[1])

		// Parse tags
		tagStrs := strings.Split(parts[2], ",")
		tags := make(pq.StringArray, 0, len(tagStrs))
		for _, t := range tagStrs {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}

		years, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
		description := strings.TrimSpace(parts[4])

		var icon string
		if len(parts) >= 6 {
			icon = strings.TrimSpace(parts[5])
		}

		featured := false
		if len(parts) >= 7 && strings.TrimSpace(parts[6]) == "featured" {
			featured = true
		}

		skill := models.Skill{
			Name:            name,
			Level:           currentLevel,
			Category:        category,
			Tags:            tags,
			YearsExperience: years,
			Description:     description,
			Icon:            icon,
			Featured:        featured,
		}
		skill.ID = uuid.New()

		skills = append(skills, skill)
	}

	return skills, nil
}
