package content

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"

	"maicivy/internal/models"
)

// Loader loads and caches all CV content from markdown files.
type Loader struct {
	contentDir  string
	mu          sync.RWMutex
	experiences []models.Experience
	skills      []models.Skill
	projects    []models.Project
}

// NewLoader creates a content loader for the given directory.
func NewLoader(contentDir string) *Loader {
	return &Loader{contentDir: contentDir}
}

// Load reads all markdown files and populates the in-memory cache.
func (l *Loader) Load() error {
	experiences, err := l.loadExperiences()
	if err != nil {
		return fmt.Errorf("loading experiences: %w", err)
	}

	skills, err := l.loadSkills()
	if err != nil {
		return fmt.Errorf("loading skills: %w", err)
	}

	projects, err := l.loadProjects()
	if err != nil {
		return fmt.Errorf("loading projects: %w", err)
	}

	l.mu.Lock()
	l.experiences = experiences
	l.skills = skills
	l.projects = projects
	l.mu.Unlock()

	log.Info().
		Int("experiences", len(experiences)).
		Int("skills", len(skills)).
		Int("projects", len(projects)).
		Str("dir", l.contentDir).
		Msg("Content loaded from markdown")

	return nil
}

// Reload reloads all content from disk (hot reload).
func (l *Loader) Reload() error {
	return l.Load()
}

// GetExperiences returns all experiences sorted by start_date DESC.
func (l *Loader) GetExperiences() []models.Experience {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]models.Experience, len(l.experiences))
	copy(result, l.experiences)
	return result
}

// GetSkills returns all skills sorted by years_experience DESC.
func (l *Loader) GetSkills() []models.Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]models.Skill, len(l.skills))
	copy(result, l.skills)
	return result
}

// GetProjects returns all projects sorted by featured DESC, then title.
func (l *Loader) GetProjects() []models.Project {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]models.Project, len(l.projects))
	copy(result, l.projects)
	return result
}

func (l *Loader) loadExperiences() ([]models.Experience, error) {
	dir := filepath.Join(l.contentDir, "experiences")
	files, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, err
	}

	var experiences []models.Experience
	for _, f := range files {
		// Skip i18n files for now (e.g., *.en.md)
		base := filepath.Base(f)
		if strings.Contains(base, ".en.") {
			continue
		}

		data, err := os.ReadFile(f)
		if err != nil {
			log.Warn().Err(err).Str("file", f).Msg("Failed to read experience file")
			continue
		}

		exp, err := ParseExperience(string(data))
		if err != nil {
			log.Warn().Err(err).Str("file", f).Msg("Failed to parse experience file")
			continue
		}

		// Try to load English translation
		enFile := strings.TrimSuffix(f, ".md") + ".en.md"
		if enData, err := os.ReadFile(enFile); err == nil {
			if enExp, err := ParseExperience(string(enData)); err == nil {
				exp.TitleEn = enExp.Title
				exp.DescriptionEn = enExp.Description
				exp.CatchphraseEn = enExp.Catchphrase
				exp.TechnicalDescriptionEn = enExp.TechnicalDescription
				exp.FunctionalDescriptionEn = enExp.FunctionalDescription
			}
		}

		experiences = append(experiences, exp)
	}

	// Sort by start_date DESC
	sort.Slice(experiences, func(i, j int) bool {
		return experiences[i].StartDate.After(experiences[j].StartDate)
	})

	return experiences, nil
}

func (l *Loader) loadSkills() ([]models.Skill, error) {
	file := filepath.Join(l.contentDir, "skills.md")
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("reading skills.md: %w", err)
	}

	skills, err := ParseSkills(string(data))
	if err != nil {
		return nil, err
	}

	// Sort by years_experience DESC
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].YearsExperience > skills[j].YearsExperience
	})

	return skills, nil
}

func (l *Loader) loadProjects() ([]models.Project, error) {
	dir := filepath.Join(l.contentDir, "projects")
	files, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, err
	}

	var projects []models.Project
	for _, f := range files {
		base := filepath.Base(f)
		if strings.Contains(base, ".en.") {
			continue
		}

		data, err := os.ReadFile(f)
		if err != nil {
			log.Warn().Err(err).Str("file", f).Msg("Failed to read project file")
			continue
		}

		proj, err := ParseProject(string(data))
		if err != nil {
			log.Warn().Err(err).Str("file", f).Msg("Failed to parse project file")
			continue
		}

		// Try to load English translation
		enFile := strings.TrimSuffix(f, ".md") + ".en.md"
		if enData, err := os.ReadFile(enFile); err == nil {
			if enProj, err := ParseProject(string(enData)); err == nil {
				proj.TitleEn = enProj.Title
				proj.DescriptionEn = enProj.Description
				proj.CatchphraseEn = enProj.Catchphrase
				proj.TechnicalDescriptionEn = enProj.TechnicalDescription
				proj.FunctionalDescriptionEn = enProj.FunctionalDescription
			}
		}

		projects = append(projects, proj)
	}

	// Sort: featured first, then by title
	sort.Slice(projects, func(i, j int) bool {
		if projects[i].Featured != projects[j].Featured {
			return projects[i].Featured
		}
		return projects[i].Title < projects[j].Title
	})

	return projects, nil
}
