package services

import (
	"context"
	"fmt"
	"strings"
)

// PortfolioService — source de vérité du portfolio d'Alexi.
// Fetche les données LIVE depuis maiprofiles.etheryale.com avec cache TTL 5 min.
// Plus de stubs hardcodés — tout vient de l'API.

// StatItem représente une métrique affichable sur une fiche projet
type StatItem struct {
	Label string
	Value string
}

// PortfolioEntry représente un projet du portfolio
type PortfolioEntry struct {
	Name        string
	Title       string
	Category    string
	ShortDesc   string
	LongDesc    string     // description.portfolio (Markdown complet)
	KeyFeatures []string
	TechStack   []string
	Stats       []StatItem
	SkillsTags  []string
	Status      string
	Tags        []string
}

// ExperienceData contient la bio et les expériences professionnelles
type ExperienceData struct {
	Bio      string
	BioFull  string
	Headline string
	TJM      string
	Dispo    string
	ExperienceYears int
	Experience []ExperienceItem
	Domains   []string
}

// ExperienceItem représente une expérience professionnelle
type ExperienceItem struct {
	Company string
	Role    string
	Period  string
	Summary string
}

// SkillCategory groupe des skills par catégorie
type SkillCategory struct {
	Name   string
	Skills []string
}

// GlobalStats agrège les métriques globales du portfolio
type GlobalStats struct {
	ProjectsCount int
	TotalLOC      int
	TotalTests    int
	TopStack      []string // top 10 techs par usage
}

// PortfolioService wraps le client HTTP maiProFiles
type PortfolioService struct {
	client *MaiProFilesClient
}

func NewPortfolioService() *PortfolioService {
	return &PortfolioService{
		client: NewMaiProFilesClient(),
	}
}

// GetProject retourne les détails complets d'un projet par son nom ou slug.
// Stratégie : recherche par nom d'abord, puis match sur les résultats.
func (s *PortfolioService) GetProject(name string) (PortfolioEntry, bool) {
	ctx := context.Background()

	// Tenter avec la liste complète pour trouver le bon ID
	projects, err := s.client.ListProjects(ctx)
	if err != nil {
		return PortfolioEntry{}, false
	}

	lower := strings.ToLower(name)
	var matchedID string

	for _, p := range projects {
		if strings.ToLower(p.ID) == lower ||
			strings.ToLower(p.Name) == lower ||
			strings.Contains(strings.ToLower(p.Name), lower) ||
			strings.Contains(strings.ToLower(p.ID), lower) {
			matchedID = p.ID
			break
		}
	}

	// Fallback : recherche par mot-clé via /search
	if matchedID == "" {
		results, err := s.client.Search(ctx, name)
		if err == nil && len(results) > 0 {
			matchedID = results[0].ID
		}
	}

	if matchedID == "" {
		return PortfolioEntry{}, false
	}

	// Récupère le détail complet (avec description.portfolio)
	p, err := s.client.GetProject(ctx, matchedID)
	if err != nil {
		return PortfolioEntry{}, false
	}

	return mapProject(p), true
}

// ListProjects retourne tous les projets
func (s *PortfolioService) ListProjects() []PortfolioEntry {
	projects, err := s.client.ListProjects(context.Background())
	if err != nil {
		return nil
	}
	result := make([]PortfolioEntry, 0, len(projects))
	for _, p := range projects {
		result = append(result, mapProject(&p))
	}
	return result
}

// ListSkills retourne les skills groupés (strong / familiar / domains)
func (s *PortfolioService) ListSkills() []SkillCategory {
	profile, err := s.client.GetProfile(context.Background())
	if err != nil {
		return nil
	}

	cats := []SkillCategory{}
	if len(profile.Skills.Strong) > 0 {
		cats = append(cats, SkillCategory{Name: "Langages (maîtrisés)", Skills: profile.Skills.Strong})
	}
	if len(profile.Skills.Familiar) > 0 {
		cats = append(cats, SkillCategory{Name: "Langages (familiers)", Skills: profile.Skills.Familiar})
	}
	if len(profile.Domains) > 0 {
		cats = append(cats, SkillCategory{Name: "Domaines d'expertise", Skills: profile.Domains})
	}
	return cats
}

// GetExperience retourne la bio et l'expérience d'Alexi depuis /profile
func (s *PortfolioService) GetExperience() ExperienceData {
	profile, err := s.client.GetProfile(context.Background())
	if err != nil {
		return ExperienceData{}
	}

	return ExperienceData{
		Bio:             profile.Bio.Short,
		BioFull:         profile.Bio.Full,
		Headline:        profile.Headline,
		TJM:             "Sur demande",
		Dispo:           "Disponible",
		ExperienceYears: profile.ExperienceYears,
		Domains:         profile.Domains,
		// Les expériences pro détaillées ne sont pas dans /profile — champ vide pour l'instant
		Experience: []ExperienceItem{},
	}
}

// GetStats retourne les stats globales du portfolio
func (s *PortfolioService) GetStats() GlobalStats {
	stats, err := s.client.GetStats(context.Background())
	if err != nil {
		return GlobalStats{}
	}

	// Top 10 techs par usage
	type stackItem struct{ name string; count int }
	var top []stackItem
	for name, count := range stats.Stack {
		top = append(top, stackItem{name, count})
	}
	// Tri simple (bubble, on a 100 items max)
	for i := range top {
		for j := i + 1; j < len(top); j++ {
			if top[j].count > top[i].count {
				top[i], top[j] = top[j], top[i]
			}
		}
	}
	topNames := make([]string, 0, 10)
	for i, item := range top {
		if i >= 10 { break }
		topNames = append(topNames, fmt.Sprintf("%s (%d)", item.name, item.count))
	}

	return GlobalStats{
		ProjectsCount: stats.Projects,
		TotalLOC:      stats.TotalLOC,
		TotalTests:    stats.TotalTests,
		TopStack:      topNames,
	}
}

// SearchProjects recherche des projets par mot-clé (délègue à /search)
func (s *PortfolioService) SearchProjects(query string) []PortfolioEntry {
	results, err := s.client.Search(context.Background(), query)
	if err != nil {
		return nil
	}
	entries := make([]PortfolioEntry, 0, len(results))
	for _, p := range results {
		entries = append(entries, mapProject(&p))
	}
	return entries
}

// --- Mapping API → types internes ---

// mapProject convertit un MPFProject en PortfolioEntry
func mapProject(p *MPFProject) PortfolioEntry {
	stats := []StatItem{}
	if p.Stats.LOC > 0 {
		stats = append(stats, StatItem{Label: "Lignes de code", Value: fmt.Sprintf("%d", p.Stats.LOC)})
	}
	if p.Stats.Tests > 0 {
		stats = append(stats, StatItem{Label: "Tests", Value: fmt.Sprintf("%d", p.Stats.Tests)})
	}
	if p.Stats.Modules > 0 {
		stats = append(stats, StatItem{Label: "Modules", Value: fmt.Sprintf("%d", p.Stats.Modules)})
	}

	return PortfolioEntry{
		Name:       p.ID,
		Title:      p.Name,
		Category:   p.Category,
		ShortDesc:  p.Description.Short,
		LongDesc:   p.Description.Portfolio,
		TechStack:  p.Stack,
		Stats:      stats,
		SkillsTags: p.Tags,
		Status:     p.Status,
		Tags:       p.Tags,
	}
}

// --- helpers ---

func toLower(s string) string {
	return strings.ToLower(s)
}

// strContains vérifie si s contient sub
func strContains(s, sub string) bool {
	return strings.Contains(s, sub)
}

// strContainsAny vérifie si un élément de la liste contient sub
func strContainsAny(list []string, sub string) bool {
	for _, item := range list {
		if strings.Contains(strings.ToLower(item), sub) {
			return true
		}
	}
	return false
}
