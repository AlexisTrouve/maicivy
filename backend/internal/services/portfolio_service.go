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
	LongDesc    string // description.portfolio (Markdown complet)
	KeyFeatures []string
	TechStack   []string
	Stats       []StatItem
	SkillsTags  []string
	Status      string
	Tags        []string
	// GithubURL/DemoURL : mêmes règles que le CV (cf. content_provider.go mapProjectLinks/isGithubURL)
	// — réutilisées, pas réimplémentées. GithubURL n'est renseigné QUE si le lien repo pointe vers
	// github.com (les Gitea privés StillHammer sont TOUS privés → lien mort/mur de login masqué).
	// Alimente le lien cliquable de ProjectFiche.tsx (avant ce champ : aucun lien, incohérent avec
	// BlogFiche qui en a un depuis le début).
	GithubURL string
	DemoURL   string
}

// ExperienceData contient la bio et les expériences professionnelles
type ExperienceData struct {
	Bio             string
	BioFull         string
	Headline        string
	TJM             string
	Dispo           string
	ExperienceYears int
	Experience      []ExperienceItem
	Domains         []string
}

// ExperienceItem représente une expérience professionnelle (vue chat/agent).
// Enrichie au-delà du strict minimum : technos + accroche + catégorie permettent à l'agent de
// répondre précisément ("chez X il a utilisé Y/Z") sans noyer le contexte (on garde Summary court,
// pas les descriptions technique/fonctionnelle longues — coût tokens).
type ExperienceItem struct {
	Company      string
	Role         string
	Period       string
	Summary      string
	Technologies []string // technos du poste (ex: VBA, Access, SQL) — pour répondre "quelle stack chez X"
	Catchphrase  string   // accroche courte du poste (résumé d'une ligne)
	Category     string   // fullstack | backend | … — axe du poste
}

// SkillDetail — une compétence avec son niveau et son ancienneté. POURQUOI : l'agent doit pouvoir
// répondre "quel est son niveau en Rust" / "depuis combien de temps fait-il du Go". Level garde la
// valeur maiProFiles (expert|advanced|intermediate|beginner) NON traduite : c'est une donnée passée
// au LLM, qui la ré-exprime dans la langue de l'utilisateur (pas une string d'UI à i18n-er).
type SkillDetail struct {
	Name  string
	Level string
	Years int
}

// SkillCategory groupe des skills (avec niveau/années) par catégorie
type SkillCategory struct {
	Name   string
	Skills []SkillDetail
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
// lang est passé à l'API maiProFiles pour retourner les textes dans la bonne langue.
// Stratégie : recherche par nom d'abord sur la liste, puis match sur les résultats.
func (s *PortfolioService) GetProject(name string, lang string) (PortfolioEntry, bool) {
	ctx := context.Background()

	// Tenter avec la liste complète pour trouver le bon ID
	// On passe lang pour que les descriptions soient déjà dans la bonne langue
	projects, err := s.client.ListProjects(ctx, lang)
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
	// On passe lang pour que les résultats soient dans la bonne langue
	if matchedID == "" {
		results, err := s.client.Search(ctx, name, lang)
		if err == nil && len(results) > 0 {
			matchedID = results[0].ID
		}
	}

	if matchedID == "" {
		return PortfolioEntry{}, false
	}

	// Récupère le détail complet (avec description.portfolio) dans la bonne langue
	p, err := s.client.GetProject(ctx, matchedID, lang)
	if err != nil {
		return PortfolioEntry{}, false
	}

	return mapProject(p), true
}

// ListProjects retourne tous les projets dans la langue demandée.
// lang : "fr" | "en" | "ka" servies tel quel ; de/it/zh repliées sur l'ANGLAIS ; "" → défaut API (FR).
func (s *PortfolioService) ListProjects(lang string) []PortfolioEntry {
	projects, err := s.client.ListProjects(context.Background(), lang)
	if err != nil {
		return nil
	}
	result := make([]PortfolioEntry, 0, len(projects))
	for _, p := range projects {
		result = append(result, mapProject(&p))
	}
	return result
}

// ListSkills retourne les skills curés, groupés par catégorie maiProFiles.
// SOURCE = /skills (la MÊME que le CV : ~27 skills avec catégorie/niveau/années). AVANT on lisait
// /profile (≈7 noms strong/familiar) → vue appauvrie pour l'agent. Pas de fallback /profile : si
// /skills échoue on retourne nil (échec franc — doctrine anti-fallback), les deux viennent de la
// même API de toute façon.
func (s *PortfolioService) ListSkills(lang string) []SkillCategory {
	skills, err := s.client.GetSkills(context.Background(), lang)
	if err != nil {
		return nil
	}
	return groupMPFSkillsByCategory(skills)
}

// groupMPFSkillsByCategory regroupe les skills de /skills par leur catégorie maiProFiles (AI, Languages,
// Backend, Frontend, DevOps…), en préservant l'ordre d'apparition (déterministe pour l'affichage et
// les tests). Chaque skill garde son niveau + ses années (SkillDetail) pour que l'agent réponde
// précisément sur la maîtrise.
func groupMPFSkillsByCategory(skills []MPFSkill) []SkillCategory {
	order := []string{}                 // catégories dans l'ordre de 1re apparition
	byCat := map[string][]SkillDetail{} // catégorie → skills détaillés
	for _, sk := range skills {
		cat := sk.Category
		if cat == "" {
			cat = "Other" // garde-fou : skill sans catégorie (rare) regroupé à part, libellé neutre
		}
		if _, seen := byCat[cat]; !seen {
			order = append(order, cat)
		}
		byCat[cat] = append(byCat[cat], SkillDetail{Name: sk.Name, Level: sk.Level, Years: sk.Years})
	}
	out := make([]SkillCategory, 0, len(order))
	for _, cat := range order {
		out = append(out, SkillCategory{Name: cat, Skills: byCat[cat]})
	}
	return out
}

// GetExperience retourne la bio (depuis /profile) ET le parcours pro détaillé (depuis /experiences).
// POURQUOI le double fetch : /profile ne contient PAS les expériences → sans l'appel à /experiences,
// l'agent ne sait rien du parcours (postes, entreprises, dates). On mappe les expériences vers une
// vue légère (ExperienceItem : poste/boîte/période/résumé). Si /experiences échoue, le parcours est
// vide mais la bio reste servie (échec franc loggué côté client, pas de fallback inventé).
// lang : "fr" | "en" | "ka" servies tel quel ; de/it/zh repliées sur l'ANGLAIS ; "" → défaut API (FR).
func (s *PortfolioService) GetExperience(lang string) ExperienceData {
	profile, err := s.client.GetProfile(context.Background(), lang)
	if err != nil {
		return ExperienceData{}
	}

	// Parcours pro : source = /experiences (curé, avec dates/résumés). Indépendant du fetch /profile.
	var items []ExperienceItem
	if exps, expErr := s.client.GetExperiences(context.Background(), lang); expErr == nil {
		items = mapExperienceItems(exps)
	}

	return ExperienceData{
		Bio:             profile.Bio.Short,
		BioFull:         profile.Bio.Full,
		Headline:        profile.Headline,
		TJM:             "Sur demande",
		Dispo:           "Disponible",
		ExperienceYears: profile.ExperienceYears,
		Domains:         profile.Domains,
		Experience:      items,
	}
}

// mapExperienceItems convertit les expériences maiProFiles (/experiences) en items légers pour le
// chat : poste, entreprise, période, résumé court. On omet technos/catchphrase/descriptions longues
// pour ne pas gonfler le contexte LLM — l'essentiel pour que l'agent parle du parcours.
func mapExperienceItems(exps []MPFExperience) []ExperienceItem {
	out := make([]ExperienceItem, 0, len(exps))
	for _, e := range exps {
		out = append(out, ExperienceItem{
			Company:      e.Company,
			Role:         e.Title,
			Period:       formatExpPeriod(e.StartDate, e.EndDate),
			Summary:      e.Description.Short,
			Technologies: e.Technologies,
			Catchphrase:  e.Catchphrase,
			Category:     e.Category,
		})
	}
	return out
}

// formatExpPeriod formate la période "AAAA-MM → AAAA-MM". end_date null (poste en cours) → plage
// ouverte "AAAA-MM → …". Le "…" est volontairement neutre (pas de mot FR/EN en dur) : c'est une
// donnée passée au LLM, qui la ré-exprime dans la langue de l'utilisateur.
func formatExpPeriod(start string, end *string) string {
	if start == "" {
		return ""
	}
	if end == nil || *end == "" {
		return start + " → …"
	}
	return start + " → " + *end
}

// GetProfile retourne le profil (identité, headline, bio, domaines, contact) depuis maiprofiles.
// Données publiques (affichées sur le site) → exposable au chat. Vide si maiprofiles indispo.
func (s *PortfolioService) GetProfile(lang string) MPFProfile {
	profile, err := s.client.GetProfile(context.Background(), lang)
	if err != nil || profile == nil {
		return MPFProfile{}
	}
	return *profile
}

// GetStats retourne les stats globales du portfolio (données numériques, pas de traduction).
func (s *PortfolioService) GetStats() GlobalStats {
	stats, err := s.client.GetStats(context.Background())
	if err != nil {
		return GlobalStats{}
	}

	// Top 10 techs par usage
	type stackItem struct {
		name  string
		count int
	}
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
		if i >= 10 {
			break
		}
		topNames = append(topNames, fmt.Sprintf("%s (%d)", item.name, item.count))
	}

	return GlobalStats{
		ProjectsCount: stats.Projects,
		TotalLOC:      stats.TotalLOC,
		TotalTests:    stats.TotalTests,
		TopStack:      topNames,
	}
}

// SearchProjects recherche des projets par mot-clé dans la langue demandée.
// lang : "fr" | "en" | "ka" servies tel quel ; de/it/zh repliées sur l'ANGLAIS ; "" → défaut API (FR).
func (s *PortfolioService) SearchProjects(query string, lang string) []PortfolioEntry {
	results, err := s.client.Search(context.Background(), query, lang)
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

	// GithubURL (filtré github.com uniquement) / DemoURL — mêmes helpers que le CV (content_provider.go).
	var githubURL, demoURL string
	for k, v := range p.Links {
		if githubURL == "" && isRepoLinkKey(k) && isGithubURL(v) {
			githubURL = ensureScheme(v)
		}
		if demoURL == "" && isDemoLinkKey(k) {
			demoURL = ensureScheme(v)
		}
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
		GithubURL:  githubURL,
		DemoURL:    demoURL,
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
