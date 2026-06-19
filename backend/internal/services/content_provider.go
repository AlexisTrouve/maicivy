package services

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"

	"maicivy/internal/models"
)

// ContentProvider — source des données CV (expériences, skills, projets) sous forme de
// models internes. Implémenté par MaiProFilesContentProvider.
//
// QUOI : remplace l'ancien content.Loader (qui lisait des .md locaux). Même API sync
//   (méthodes sans ctx/lang, renvoient des slices) → drop-in : CVService, CVGenerationService
//   et LetterGenerator l'utilisent sans changement de leur logique interne.
// POURQUOI : maiprofiles est désormais la SOURCE DE VÉRITÉ UNIQUE. Le contenu local = supprimé.
// COMMENT : fetch maiprofiles en FR (primaire) + EN (best-effort), mappe vers models.* avec le
//   FR en champs principaux et l'EN dans les champs *En → la LocalizationHelper existante
//   (qui choisit FR/EN sur ces champs) continue de fonctionner telle quelle.
type ContentProvider interface {
	GetExperiences(lang string) []models.Experience
	GetSkills(lang string) []models.Skill
	GetProjects(lang string) []models.Project
}

// providerLangs — langues réellement servies (fr = source exp/skills, en = traductions seedées ;
// les projets sont en anglais dans les deux). Les autres locales retombent sur en via resolveLang.
var providerLangs = []string{"fr", "en"}

// resolveLang mappe une locale frontend vers une langue servie : fr→fr, tout le reste→en
// (le portfolio est anglophone ; de/it/zh non seedés retombent proprement sur l'anglais).
func resolveLang(lang string) string {
	if lang == "fr" {
		return "fr"
	}
	return "en"
}

// contentProviderTTL — durée de cache des données mappées en mémoire.
// Aligné sur le cache du client maiProFiles (5 min) : au-delà, on re-fetch.
const contentProviderTTL = 5 * time.Minute

// MaiProFilesContentProvider implémente ContentProvider via l'API maiProFiles.
type MaiProFilesContentProvider struct {
	client     *MaiProFilesClient
	giteaStats *GiteaStatsService // pour le scoring vedette auto (commits par repo) — peut être nil

	mu sync.RWMutex
	// Caches per-langue (clé = "fr"|"en"). Le contenu diffère par langue (exp/skills traduits),
	// donc un cache par langue plutôt qu'un seul snapshot.
	experiences map[string][]models.Experience
	skills      map[string][]models.Skill
	projects    map[string][]models.Project
	loaded      bool
	expiresAt   time.Time
}

// NewMaiProFilesContentProvider crée le provider (le 1er Get déclenche le 1er fetch).
// giteaStats (optionnel, peut être nil) alimente la vedette auto (top par activité Gitea).
func NewMaiProFilesContentProvider(giteaStats *GiteaStatsService) *MaiProFilesContentProvider {
	return &MaiProFilesContentProvider{
		client:      NewMaiProFilesClient(),
		giteaStats:  giteaStats,
		experiences: map[string][]models.Experience{},
		skills:      map[string][]models.Skill{},
		projects:    map[string][]models.Project{},
	}
}

// ensureFresh recharge depuis maiprofiles si le cache est vide ou expiré.
func (p *MaiProFilesContentProvider) ensureFresh() {
	p.mu.RLock()
	fresh := p.loaded && time.Now().Before(p.expiresAt)
	p.mu.RUnlock()
	if fresh {
		return
	}
	p.refresh()
}

// refresh fetch FR + EN et met à jour le cache. Sur erreur d'une entité, on CONSERVE la
// dernière valeur connue (pas d'écrasement par du vide) — échec franc loggué, pas de fallback muet.
func (p *MaiProFilesContentProvider) refresh() {
	ctx := context.Background()

	// i18n : on fetch CHAQUE langue servie (cf. providerLangs). maiprofiles cache translations.yaml
	// en mémoire (lookups O(1), <0.2s) → ?lang=en rapide, plus de timeout. fr sert la source
	// (exp/skills en français, projets en anglais), en sert les traductions seedées (tout anglais).
	exp := make(map[string][]models.Experience, len(providerLangs))
	sk := make(map[string][]models.Skill, len(providerLangs))
	proj := make(map[string][]models.Project, len(providerLangs))
	var autoFeatured map[string]bool
	allOK := true

	for _, lang := range providerLangs {
		e, errE := p.client.GetExperiences(ctx, lang)
		s, errS := p.client.GetSkills(ctx, lang)
		pr, errP := p.client.ListProjects(ctx, lang)
		if errE != nil || errS != nil || errP != nil {
			allOK = false
			log.Warn().Str("lang", lang).AnErr("exp", errE).AnErr("skills", errS).AnErr("projects", errP).
				Msg("[content] fetch maiprofiles partiel — langue ignorée ce cycle (données précédentes conservées)")
			continue
		}
		// Vedette AUTO : les links.repo (clé du scoring) sont identiques en fr/en → calcul une fois.
		if autoFeatured == nil {
			autoFeatured = p.autoFeaturedSet(ctx, pr)
		}
		exp[lang] = mapExperiences(e, nil)
		sk[lang] = mapSkills(s, nil)
		proj[lang] = mapProjects(pr, nil, autoFeatured)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Merge uniquement les langues effectivement récupérées (pas d'écrasement par du vide).
	for lang := range exp {
		p.experiences[lang] = exp[lang]
		p.skills[lang] = sk[lang]
		p.projects[lang] = proj[lang]
	}
	if len(exp) > 0 {
		p.loaded = true
	}

	// Sur erreur : re-essayer vite (transitoire) plutôt que d'attendre le TTL plein de 5 min.
	if !allOK {
		p.expiresAt = time.Now().Add(30 * time.Second)
	} else {
		p.expiresAt = time.Now().Add(contentProviderTTL)
	}
}

func (p *MaiProFilesContentProvider) GetExperiences(lang string) []models.Experience {
	p.ensureFresh()
	p.mu.RLock()
	defer p.mu.RUnlock()
	src := p.experiences[resolveLang(lang)]
	out := make([]models.Experience, len(src))
	copy(out, src)
	return out
}

func (p *MaiProFilesContentProvider) GetSkills(lang string) []models.Skill {
	p.ensureFresh()
	p.mu.RLock()
	defer p.mu.RUnlock()
	src := p.skills[resolveLang(lang)]
	out := make([]models.Skill, len(src))
	copy(out, src)
	return out
}

func (p *MaiProFilesContentProvider) GetProjects(lang string) []models.Project {
	p.ensureFresh()
	p.mu.RLock()
	defer p.mu.RUnlock()
	src := p.projects[resolveLang(lang)]
	out := make([]models.Project, len(src))
	copy(out, src)
	return out
}

// --- Mapping maiProFiles → models internes ---

// parseExpDate parse "YYYY-MM" (ou "YYYY-MM-DD") en time.Time.
func parseExpDate(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{"2006-01", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// nonNil garantit une slice JSON non-nulle (`[]` au lieu de `null`).
// POURQUOI : le frontend fait `.technologies.slice(...)` / `.tags.map(...)` sans garde ;
// une slice nil → JSON `null` → crash "can't access property slice of null". On émet toujours [].
func nonNil(a []string) pq.StringArray {
	if a == nil {
		return pq.StringArray{}
	}
	return pq.StringArray(a)
}

func mapExpLinks(links []MPFLink) models.LinksJSON {
	if len(links) == 0 {
		return nil
	}
	out := make(models.LinksJSON, 0, len(links))
	for _, l := range links {
		out = append(out, models.LinkData{Name: l.Name, URL: l.URL, Icon: l.Icon})
	}
	return out
}

func mapExperiences(fr, en []MPFExperience) []models.Experience {
	enByID := make(map[string]MPFExperience, len(en))
	for _, e := range en {
		enByID[e.ID] = e
	}

	out := make([]models.Experience, 0, len(fr))
	for _, e := range fr {
		start, _ := parseExpDate(e.StartDate)
		var end *time.Time
		if e.EndDate != nil {
			if t, ok := parseExpDate(*e.EndDate); ok {
				end = &t
			}
		}
		exp := models.Experience{
			Title:                 e.Title,
			Company:               e.Company,
			Category:              e.Category,
			StartDate:             start,
			EndDate:               end,
			Technologies:          nonNil(e.Technologies),
			Tags:                  nonNil(e.Tags),
			Featured:              e.Featured,
			Catchphrase:           e.Catchphrase,
			Links:                 mapExpLinks(e.Links),
			Description:           e.Description.Short,
			TechnicalDescription:  e.Description.Technical,
			FunctionalDescription: e.Description.Functional,
		}
		exp.ID = uuid.New()
		// EN seulement si réellement différent du FR (sinon c'est juste le fallback FR de l'API).
		if t, ok := enByID[e.ID]; ok {
			if t.Title != e.Title {
				exp.TitleEn = t.Title
			}
			if t.Description.Short != e.Description.Short {
				exp.DescriptionEn = t.Description.Short
			}
			if t.Catchphrase != e.Catchphrase {
				exp.CatchphraseEn = t.Catchphrase
			}
			if t.Description.Technical != e.Description.Technical {
				exp.TechnicalDescriptionEn = t.Description.Technical
			}
			if t.Description.Functional != e.Description.Functional {
				exp.FunctionalDescriptionEn = t.Description.Functional
			}
		}
		out = append(out, exp)
	}
	return out
}

func mapSkills(fr, en []MPFSkill) []models.Skill {
	enByName := make(map[string]MPFSkill, len(en))
	for _, s := range en {
		enByName[s.Name] = s
	}

	out := make([]models.Skill, 0, len(fr))
	for _, s := range fr {
		sk := models.Skill{
			Name:            s.Name,
			Level:           models.SkillLevel(s.Level),
			Category:        s.Category,
			Tags:            nonNil(s.Tags),
			YearsExperience: s.Years,
			Description:     s.Description,
			Featured:        s.Featured,
			Icon:            s.Icon,
		}
		sk.ID = uuid.New()
		if t, ok := enByName[s.Name]; ok && t.Description != s.Description {
			sk.DescriptionEn = t.Description
		}
		out = append(out, sk)
	}
	return out
}

// ensureScheme préfixe https:// si l'URL n'a pas de schéma. maiProFiles stocke souvent les liens
// nus ("git.etheryale.com/...", "github.com/...") → href relatif cassé sans ça.
func ensureScheme(u string) string {
	u = strings.TrimSpace(u)
	if u == "" || strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u
	}
	return "https://" + u
}

// isGithubURL : un lien repo n'est exposé sur le CV public QUE s'il pointe vers github.com. Les repos
// Gitea de StillHammer sont TOUS privés (0 public) → lien mort (mur de login) pour les visiteurs → masqué.
func isGithubURL(u string) bool {
	return strings.Contains(strings.ToLower(u), "github.com")
}

// isRepoLinkKey : clés maiProFiles désignant un lien de dépôt (géré via Project.GithubURL, pas ici).
func isRepoLinkKey(k string) bool {
	switch strings.ToLower(k) {
	case "repo", "github", "gitea", "git":
		return true
	}
	return false
}

// isDemoLinkKey : clés désignant la démo live (gérée via Project.DemoURL, pas dans la liste de liens).
func isDemoLinkKey(k string) bool {
	switch strings.ToLower(k) {
	case "demo", "live", "site", "url":
		return true
	}
	return false
}

func mapProjectLinks(links map[string]string) models.LinksJSON {
	if len(links) == 0 {
		return nil
	}
	out := make(models.LinksJSON, 0, len(links))
	for k, v := range links {
		// Le dépôt (→ GithubURL, filtré github-only) et la démo (→ DemoURL) ont leurs propres boutons.
		// On ne les ré-émet pas ici → évite le doublon ET l'exposition des Gitea privés. Restent les
		// liens "extra" (website, linkedin, docs…) pour la section Links du modal.
		if isRepoLinkKey(k) || isDemoLinkKey(k) {
			continue
		}
		out = append(out, models.LinkData{Name: k, URL: ensureScheme(v)})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// repoFromLinks extrait le nom du repo Gitea depuis links.repo (git.etheryale.com/StillHammer/<repo>).
func repoFromLinks(links map[string]string) string {
	const marker = "/StillHammer/"
	repo := links["repo"]
	i := strings.Index(repo, marker)
	if i < 0 {
		return ""
	}
	name := repo[i+len(marker):]
	if j := strings.IndexAny(name, "/?#"); j >= 0 {
		name = name[:j]
	}
	return strings.TrimSuffix(name, ".git")
}

// Constantes du scoring vedette AUTO — échelles FIXES (score stable, pas de dérive quand le
// portefeuille évolue). Calibrées sur la data réelle (cf. l'historique de design).
const (
	featAgeCapDays    = 180.0 // âge plafonné à 6 mois : au-delà, plus de bonus de longévité
	featActiveDaysCap = 30.0  // 30 jours de commit distincts = régularité "pleine" (1.0)
	featRecencyWindow = 90.0  // récence : décroît linéairement jusqu'à 0 à 90j sans commit

	// Sélection : on vise featTargetCount vedettes au total (pins + auto). L'auto comble les
	// places restantes avec les meilleurs non-pins PAR SCORE, mais n'ajoute JAMAIS un projet
	// abandonné — dernier commit > featDormantCutoffDays → exclu, même si ça laisse moins de 12.
	featTargetCount       = 12 // total vedettes visé
	featDormantCutoffDays = 60 // au-delà, un non-pin est "abandonné" et jamais comblé
)

// autoFeaturedSet — vedette AUTO par SCORE multiplicatif, qui comble jusqu'à featTargetCount au
// total avec les pins manuels (cf. mapProjects). Un projet score d'autant plus haut qu'il est à la
// fois ANCIEN, RÉGULIÈREMENT mis à jour, et ENCORE actif :
//
//	score = âge_norm × régularité_norm × récence
//
// POURQUOI multiplicatif (et non additif) : c'est le "ET". Un gros âge ne RACHÈTE PAS une
// régularité nulle (un repo créé il y a 5 mois mais touché 2 jours ≈ 0) ni un abandon : la récence
// (1 − jours_depuis_dernier_commit/90) écrase le score d'un projet dormant. Tous les signaux
// dérivent de RepoStat.CommitDays (jours de commit branche par défaut) → mesure le CODE, pas
// UpdatedAt repo (bumpé par un tag/branche/setting).
//
// SÉLECTION : pins toujours pris (et comptés dans le total) ; on classe les non-pins par score et
// on comble les places restantes jusqu'à featTargetCount — SAUF les abandonnés (dernier commit
// > featDormantCutoffDays), jamais comblés même si ça laisse moins de 12. Anti-"vieux projet mort".
func (p *MaiProFilesContentProvider) autoFeaturedSet(ctx context.Context, fr []MPFProject) map[string]bool {
	set := map[string]bool{}
	if p.giteaStats == nil {
		return set
	}
	stats, err := p.giteaStats.GetStats(ctx, false)
	if err != nil || stats == nil {
		return set
	}
	// Index repo (minuscule) → jours de commit, pour matcher chaque fiche via repoFromLinks.
	daysByRepo := make(map[string][]string, len(stats.Repos))
	for _, r := range stats.Repos {
		daysByRepo[strings.ToLower(r.Name)] = r.CommitDays
	}
	now := time.Now()

	// Pins comptés à part (toujours affichés via mapProjects) ; non-pins classés par score.
	type scored struct {
		id    string
		score float64
	}
	pins := 0
	cands := make([]scored, 0, len(fr))
	for _, proj := range fr {
		if proj.Featured {
			pins++ // pin manuel : compte dans le total des 12, sélectionné en dehors du score
			continue
		}
		repo := repoFromLinks(proj.Links)
		if repo == "" {
			continue
		}
		days := daysByRepo[strings.ToLower(repo)]
		// Plancher anti-dormant : un projet sans code depuis > cutoff n'est JAMAIS comblé.
		if daysSinceLastCommit(days, now) > featDormantCutoffDays {
			continue
		}
		s := featuredScore(days, now)
		if s <= 0 {
			continue
		}
		cands = append(cands, scored{proj.ID, s})
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].score > cands[j].score })

	// Combler les places restantes (12 − pins) avec les meilleurs candidats.
	need := featTargetCount - pins
	for i := 0; i < need && i < len(cands); i++ {
		set[cands[i].id] = true
	}
	return set
}

// daysSinceLastCommit = nb de jours depuis le dernier commit (CommitDays trié croissant).
// Renvoie +Inf si aucun commit → traité comme dormant (jamais comblé).
func daysSinceLastCommit(commitDays []string, now time.Time) float64 {
	if len(commitDays) == 0 {
		return math.Inf(1)
	}
	last, err := time.Parse("2006-01-02", commitDays[len(commitDays)-1])
	if err != nil {
		return math.Inf(1)
	}
	return now.Sub(last).Hours() / 24
}

// featuredScore = âge_norm × régularité_norm × récence, à partir des jours de commit (triés croissant).
// Renvoie 0 sans commit → jamais vedette auto. Échelles fixes via les constantes feat*.
func featuredScore(commitDays []string, now time.Time) float64 {
	if len(commitDays) == 0 {
		return 0
	}
	const dayFmt = "2006-01-02"
	first, err1 := time.Parse(dayFmt, commitDays[0])
	last, err2 := time.Parse(dayFmt, commitDays[len(commitDays)-1])
	if err1 != nil || err2 != nil {
		return 0
	}
	ageDays := now.Sub(first).Hours() / 24  // longévité (1er commit dans la fenêtre)
	lastDays := now.Sub(last).Hours() / 24  // récence (dernier commit de code)

	ageNorm := math.Min(1, ageDays/featAgeCapDays)
	regNorm := math.Min(1, float64(len(commitDays))/featActiveDaysCap)
	recency := math.Max(0, 1-lastDays/featRecencyWindow)
	return ageNorm * regNorm * recency
}

func mapProjects(fr, en []MPFProject, autoFeatured map[string]bool) []models.Project {
	enByID := make(map[string]MPFProject, len(en))
	for _, p := range en {
		enByID[p.ID] = p
	}

	out := make([]models.Project, 0, len(fr))
	for _, p := range fr {
		// Ignore les entrées malformées (ex: ancien wrapper {projects:[...]} sans id/nom).
		if p.ID == "" && p.Name == "" {
			continue
		}
		status := strings.ToLower(p.Status)
		proj := models.Project{
			Title:        p.Name,
			Description:  p.Description.Short,
			Category:     p.Category,
			Technologies: nonNil(p.Stack),
			Tags:         nonNil(p.Tags), // catégorisation (flags-concept inclus) → matching skill, pas affichage
			Featured:     p.Featured || autoFeatured[p.ID], // pin curé OU top activité Gitea (mix)
			InProgress:   status == "wip" || status == "in_progress" || status == "beta",
			Links:        mapProjectLinks(p.Links),
		}
		proj.ID = uuid.New()
		// Dériver github/demo depuis les links (clés conventionnelles maiprofiles).
		// GithubURL : UNIQUEMENT si le lien repo est public (github.com) — les Gitea privés sont masqués.
		for k, v := range p.Links {
			if proj.GithubURL == "" && isRepoLinkKey(k) && isGithubURL(v) {
				proj.GithubURL = ensureScheme(v)
			}
			if proj.DemoURL == "" && isDemoLinkKey(k) {
				proj.DemoURL = ensureScheme(v)
			}
		}
		if t, ok := enByID[p.ID]; ok {
			if t.Name != p.Name {
				proj.TitleEn = t.Name
			}
			if t.Description.Short != p.Description.Short {
				proj.DescriptionEn = t.Description.Short
			}
		}
		out = append(out, proj)
	}
	return out
}
