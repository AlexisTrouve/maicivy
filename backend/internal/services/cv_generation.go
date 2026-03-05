package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"maicivy/internal/config"
	"maicivy/internal/content"
	"maicivy/internal/models"
)

// CVGenerationService génère un CV dynamique tailoré à une offre d'emploi réelle.
// Contrairement aux thèmes statiques (tag-weight), il analyse l'offre en entier
// et retourne scores + catchphrases réécrites par le LLM pour matcher le vocabulaire exact.
// Peut aussi générer un PDF avec couche stealth ATS intégrée via GenerateDynamicPDF.
type CVGenerationService struct {
	contentLoader *content.Loader
	baseURL       string
	apiKey        string
	httpClient    *http.Client
	l10n          *LocalizationHelper
	pdfService    *PDFService // pour la génération PDF avec stealth ATS
}

// NewCVGenerationService crée une instance. Retourne nil si baseURL ou apiKey vides.
func NewCVGenerationService(contentLoader *content.Loader, baseURL, apiKey string, pdfService *PDFService) *CVGenerationService {
	if baseURL == "" || apiKey == "" {
		return nil
	}
	return &CVGenerationService{
		contentLoader: contentLoader,
		baseURL:       strings.TrimRight(baseURL, "/"),
		apiKey:        apiKey,
		httpClient:    &http.Client{Timeout: 90 * time.Second}, // Sonnet + CoT = plus lent que Haiku
		l10n:          NewLocalizationHelper(),
		pdfService:    pdfService,
	}
}

// --- Structures pour parser la réponse JSON du LLM ---

// llmGenerationResponse est le format attendu en sortie du LLM
type llmGenerationResponse struct {
	JobTitle    string               `json:"job_title"`
	Summary     string               `json:"summary"`  // 2-3 phrases, style "builder", few-shot enforced
	Location    string               `json:"location"` // ville extraite de l'offre (pour stealth ATS)
	Experiences []llmExpScore        `json:"experiences"`
	Projects    []llmProjScore       `json:"projects"`
	Skills      []llmSkillScoreEntry `json:"skills"`
}

// llmExpScore : score + catchphrase réécrit pour une expérience
type llmExpScore struct {
	Slug        string `json:"slug"`
	Score       int    `json:"score"`
	Catchphrase string `json:"catchphrase,omitempty"` // réécriture optionnelle (top 3 seulement)
}

// llmProjScore : score pour un projet
type llmProjScore struct {
	Slug  string `json:"slug"`
	Score int    `json:"score"`
}

// llmSkillScoreEntry : score pour une compétence (matchée par nom)
type llmSkillScoreEntry struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

// GenerateDynamicCV génère un CV adapté à l'offre fournie (texte brut ou URL).
// Auto-détection : si offer commence par "http" → fetch + strip HTML.
// Retourne un AdaptiveCVResponse avec thème "dynamic" → PDF renderer inchangé.
func (s *CVGenerationService) GenerateDynamicCV(ctx context.Context, offer, lang string) (*AdaptiveCVResponse, error) {
	// Normaliser la langue — même logique que CVService
	lang = s.l10n.NormalizeLanguage(lang)

	// Auto-détection URL : fetch + strip HTML si nécessaire
	offerText := offer
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(offer)), "http") {
		fetched, err := s.fetchURLContent(offer)
		if err == nil {
			offerText = fetched
		}
		// Erreur de fetch → on continue avec l'URL brute plutôt que bloquer
	}

	// Charger toutes les données CV depuis le contentLoader (source de vérité markdown)
	experiences := s.contentLoader.GetExperiences()
	projects := s.contentLoader.GetProjects()
	skills := s.contentLoader.GetSkills()

	// Appel LLM : analyse de l'offre + scoring de tout le contenu CV en une seule passe
	prompt := s.buildGenerationPrompt(offerText, experiences, projects, skills, lang)
	raw, err := s.callClaude(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	llmResp, err := s.parseLLMResponse(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// --- Construire les maps slug→score pour accès O(1) lors du merge ---

	// Experiences : indexées par slug (toSlug(company + "-" + title))
	expScores := make(map[string]llmExpScore, len(llmResp.Experiences))
	for _, e := range llmResp.Experiences {
		expScores[e.Slug] = e
	}

	// Projects : indexés par toSlug(title)
	projScores := make(map[string]int, len(llmResp.Projects))
	for _, p := range llmResp.Projects {
		projScores[p.Slug] = p.Score
	}

	// Skills : indexés par nom en lowercase (matching insensible à la casse)
	skillScores := make(map[string]int, len(llmResp.Skills))
	for _, sk := range llmResp.Skills {
		skillScores[strings.ToLower(sk.Name)] = sk.Score
	}

	// --- Expériences scorées + catchphrases réécrites ---
	scoredExps := make([]ScoredExperienceResponse, 0, len(experiences))
	for _, exp := range experiences {
		// Localiser d'abord pour obtenir la bonne version linguistique
		localExp := s.l10n.LocalizeExperience(exp, lang)

		// Slug identique à celui utilisé dans le prompt pour garantir la correspondance
		slug := toSlug(exp.Company + "-" + exp.Title)

		score := 50 // score par défaut si le LLM n'a pas scoré cet item
		if llmData, ok := expScores[slug]; ok {
			score = llmData.Score
			// Appliquer la réécriture du catchphrase si fournie par le LLM
			if llmData.Catchphrase != "" {
				localExp.Catchphrase = llmData.Catchphrase
			}
		}

		scoredExps = append(scoredExps, ScoredExperienceResponse{
			Experience: localExp,
			Score:      float64(score) / 100.0,
		})
	}
	// Tri par score DESC (même ordre que CVService)
	sort.Slice(scoredExps, func(i, j int) bool {
		return scoredExps[i].Score > scoredExps[j].Score
	})

	// --- Projets scorés ---
	scoredProjects := make([]ScoredProjectResponse, 0, len(projects))
	for _, proj := range projects {
		localProj := s.l10n.LocalizeProject(proj, lang)
		slug := toSlug(proj.Title) // même slug que llm_scoring.go
		score := 50
		if s, ok := projScores[slug]; ok {
			score = s
		}
		scoredProjects = append(scoredProjects, ScoredProjectResponse{
			Project: localProj,
			Score:   float64(score) / 100.0,
		})
	}
	sort.Slice(scoredProjects, func(i, j int) bool {
		return scoredProjects[i].Score > scoredProjects[j].Score
	})

	// --- Skills scorées ---
	scoredSkills := make([]ScoredSkillResponse, 0, len(skills))
	for _, skill := range skills {
		localSkill := s.l10n.LocalizeSkill(skill, lang)
		score := 50
		if s, ok := skillScores[strings.ToLower(skill.Name)]; ok {
			score = s
		}
		scoredSkills = append(scoredSkills, ScoredSkillResponse{
			Skill: localSkill,
			Score: float64(score) / 100.0,
		})
	}
	sort.Slice(scoredSkills, func(i, j int) bool {
		return scoredSkills[i].Score > scoredSkills[j].Score
	})

	// --- Thème dynamique créé à la volée ---
	// Le nom du thème = titre du poste extrait par le LLM
	jobTitle := llmResp.JobTitle
	if jobTitle == "" {
		jobTitle = "CV Dynamique"
	}
	// Description = aperçu de l'offre (max 100 chars) pour référence
	offerPreview := offerText
	if len(offerPreview) > 100 {
		offerPreview = offerPreview[:100] + "..."
	}
	theme := config.CVTheme{
		ID:          "dynamic",
		Name:        jobTitle,
		Description: offerPreview,
		Icon:        "🎯",
	}

	// Post-processing summary : les " — " LLM deviennent "." (plus ATS-safe, meilleur rendu)
	summary := strings.ReplaceAll(llmResp.Summary, " — ", ". ")
	summary = strings.ReplaceAll(summary, " —", ".")

	return &AdaptiveCVResponse{
		Theme:       theme,
		Summary:     summary,
		Location:    llmResp.Location,
		Experiences: scoredExps,
		Skills:      scoredSkills,
		Projects:    scoredProjects,
		GeneratedAt: time.Now(),
	}, nil
}

// GenerateDynamicPDF combine les deux optimisations en un seul appel :
//  1. LLM score + réécrit les catchphrases pour l'offre (build)
//  2. Injection couche stealth ATS à partir des top skills scorés (ultraopti)
//
// La couche stealth est quasi-invisible visuellement (7px, #e8e8e8) mais présente
// dans le flux texte du PDF — lue par les parseurs ATS et les LLMs de screening.
func (s *CVGenerationService) GenerateDynamicPDF(ctx context.Context, offer, lang string) ([]byte, error) {
	if s.pdfService == nil {
		return nil, fmt.Errorf("PDF service not available")
	}

	// Étape 1 : générer le CV optimisé (LLM scoring + catchphrases réécrites)
	cv, err := s.GenerateDynamicCV(ctx, offer, lang)
	if err != nil {
		return nil, fmt.Errorf("generation failed: %w", err)
	}

	// Étape 2 : extraire les top skills pour la couche stealth ATS
	// On prend les skills avec score >= 60% (pertinents pour l'offre), max 15
	topSkillNames := make([]string, 0, 15)
	for _, skill := range cv.Skills {
		if skill.Score >= 0.60 && len(topSkillNames) < 15 {
			topSkillNames = append(topSkillNames, skill.Name)
		}
	}

	// Étape 3 : construire et injecter la couche stealth ATS
	// Si le LLM a extrait une ville depuis l'offre, on l'inclut dans les keywords stealth
	// → l'ATS trouve "Paris", "Lyon" etc. même si c'est pas dans le CV visible
	if cv.Location != "" {
		topSkillNames = append(topSkillNames, cv.Location, "France", cv.Location+" France")
	}
	stealthHTML := buildStealthHTML(topSkillNames, lang)

	return s.pdfService.GenerateTailoredPDF(cv, lang, stealthHTML)
}

// fetchURLContent récupère le texte brut d'une URL en strippant les balises HTML.
// Timeout : 15s pour ne pas bloquer le pipeline de génération.
func (s *CVGenerationService) fetchURLContent(url string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	// User-Agent standard pour éviter les blocages simples
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; CVGenerator/1.0)")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body failed: %w", err)
	}

	// Strip HTML : regex simple — suffit pour des pages d'offre standard
	htmlTagRe := regexp.MustCompile(`<[^>]+>`)
	text := htmlTagRe.ReplaceAllString(string(body), " ")

	// Nettoyer les espaces multiples issus du stripping
	spaceRe := regexp.MustCompile(`\s+`)
	text = spaceRe.ReplaceAllString(text, " ")

	return strings.TrimSpace(text), nil
}

// buildGenerationPrompt construit le prompt LLM avec toutes les données CV + l'offre.
// Le LLM reçoit les slugs exacts pour garantir la correspondance lors du parsing.
func (s *CVGenerationService) buildGenerationPrompt(
	offer string,
	experiences []models.Experience,
	projects []models.Project,
	skills []models.Skill,
	lang string,
) string {
	var sb strings.Builder

	sb.WriteString(`Tu es un chasseur de têtes senior spécialisé tech. Mission : trouver le meilleur angle pour vendre CE candidat à CETTE offre — pas en inventant des compétences, mais en identifiant l'angle inattendu qui fait qu'il est plus fort que les candidats évidents.

Règle d'or : un recruteur intelligent préfère "développeur qui peut construire les outils de ce domaine" à "quelqu'un qui prétend être expert du domaine". Trouve l'angle qui tient en entretien.

STYLE DU RÉSUMÉ — few shots obligatoires à respecter :
Le summary doit sonner comme quelqu'un qui build des systèmes ambitieux et le sait.
Pas de "passionné", "motivé", "dynamique", "résultats-driven". Jamais.
Pattern : "Je construis [X concret] — [preuve chiffrée ou technique]. [Angle taillé pour cette offre]."

Exemples (NE PAS copier, s'en inspirer pour le ton) :
- Offre Rust backend → "Je construis des systèmes distribués qui tiennent — 43 microservices Rust en prod, architecture Actix-web à l'échelle. Votre stack backend, c'est mon terrain."
- Offre AI/LLM → "Je construis des pipelines LLM qui vont en prod, pas en démo — de l'intégration Claude API au scoring adaptatif multi-modèle. L'IA générative comme brique d'architecture, pas comme gadget."
- Offre robotique → "Je construis les systèmes qui font tourner les robots — vision 3D temps réel en C++, pipelines de détection, architectures embarquées. Pas l'opérateur : l'ingénieur derrière."
- Offre hors domaine → rester honnête mais trouver l'angle transfert le plus fort, même style.

`)


	// Tronquer l'offre à 3000 chars — garde le prompt sous ~4K tokens
	sb.WriteString("OFFRE D'EMPLOI:\n")
	offerTrunc := offer
	if len(offerTrunc) > 3000 {
		offerTrunc = offerTrunc[:3000] + "..."
	}
	sb.WriteString(offerTrunc)

	sb.WriteString("\n\nDONNÉES CV BRUTES:\nExpériences (slug | titre | société | techs | catchphrase):\n")
	for _, exp := range experiences {
		// Slug construit avec company + title pour l'unicité entre expériences
		slug := toSlug(exp.Company + "-" + exp.Title)
		techs := strings.Join(exp.Technologies, ", ")
		catchphrase := exp.Catchphrase
		if catchphrase == "" {
			// Fallback sur description tronquée si pas de catchphrase
			catchphrase = exp.Description
			if len(catchphrase) > 80 {
				catchphrase = catchphrase[:80]
			}
		}
		sb.WriteString(fmt.Sprintf("- %s | %s | %s | %s | %s\n",
			slug, exp.Title, exp.Company, techs, catchphrase))
	}

	sb.WriteString("\nProjets (slug | titre | techs | description):\n")
	for _, proj := range projects {
		slug := toSlug(proj.Title) // même format que llm_scoring.go
		techs := strings.Join(proj.Technologies, ", ")
		desc := proj.Description
		if len(desc) > 120 {
			desc = desc[:120]
		}
		sb.WriteString(fmt.Sprintf("- %s | %s | %s | %s\n", slug, proj.Title, techs, desc))
	}

	sb.WriteString("\nCompétences (nom | niveau | catégorie):\n")
	for _, skill := range skills {
		sb.WriteString(fmt.Sprintf("- %s | %s | %s\n", skill.Name, string(skill.Level), skill.Category))
	}

	sb.WriteString(`
INSTRUCTIONS — raisonne d'abord, score ensuite :

<analysis>
1. DÉCODE L'OFFRE : Au-delà des mots-clés, qu'est-ce que cette boîte cherche VRAIMENT ?
   (ex: "robotique agricole" = peut-être surtout besoin de quelqu'un qui code des systèmes embarqués fiables)

2. TROUVE L'ANGLE FORT : Quelle est la vraie valeur ajoutée de ce candidat pour CE contexte ?
   - Compétences directement applicables (score 70-100)
   - Compétences transférables légitimement défendables en entretien (score 40-69)
   - Hors sujet même avec bonne volonté (score 1-39)

3. STRATÉGIE CATCHPHRASE : Pour les 3 meilleures expériences, comment reformuler pour résonner
   avec le vocabulaire de l'offre SANS mentir ? L'angle "je construis les outils" > "je fais le métier".

4. TITRE JUSTE : Quel titre reflète honnêtement ce que ce candidat apporte à CE poste ?
</analysis>

Donne UNIQUEMENT ce JSON après ton analyse (sans markdown, sans explication):
{
  "job_title": "...",
  "summary": "2-3 phrases max, style builder, few-shot ci-dessus",
  "location": "ville mentionnée dans l'offre, ou vide si non précisée",
  "experiences": [{"slug":"...", "score": 90, "catchphrase":"...max 80 chars, top 3 seulement"},...],
  "projects": [{"slug":"...", "score": 85},...],
  "skills": [{"name":"...", "score": 95},...]
}`)

	return sb.String()
}

// callClaude envoie le prompt à Anthropic et retourne le texte brut de la réponse.
// Haiku avec CoT inline : max_tokens 4000 pour raisonnement + JSON de sortie.
func (s *CVGenerationService) callClaude(ctx context.Context, prompt string) (string, error) {
	body := map[string]interface{}{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 4000, // CoT (~2K) + JSON (~1.5K) + marge
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBytes))
	}

	// Structure de réponse Anthropic : content[0].text
	var apiResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse API response: %w", err)
	}
	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("empty API response")
	}

	return apiResp.Content[0].Text, nil
}

// parseLLMResponse parse le JSON retourné par le LLM.
// Avec CoT, le modèle produit <analysis>...</analysis> PUIS le JSON.
// On prend le dernier bloc {...} pour ignorer tout raisonnement préalable.
func (s *CVGenerationService) parseLLMResponse(text string) (*llmGenerationResponse, error) {
	text = strings.TrimSpace(text)

	// Chercher "job_title" pour trouver le début du JSON de sortie
	// (plus fiable que le dernier "{" qui peut apparaître dans l'analyse)
	if idx := strings.LastIndex(text, `"job_title"`); idx >= 0 {
		// Reculer pour trouver le "{" qui précède job_title
		if start := strings.LastIndex(text[:idx], "{"); start >= 0 {
			text = text[start:]
		}
	} else if idx := strings.LastIndex(text, "{"); idx >= 0 {
		// Fallback : dernier "{" si job_title introuvable
		text = text[idx:]
	}
	if idx := strings.LastIndex(text, "}"); idx >= 0 {
		text = text[:idx+1]
	}

	var result llmGenerationResponse
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w\nraw: %s", err, text)
	}

	// Clamp les scores pour garantir la validité des valeurs (1-100)
	for i := range result.Experiences {
		result.Experiences[i].Score = clampScore(result.Experiences[i].Score)
	}
	for i := range result.Projects {
		result.Projects[i].Score = clampScore(result.Projects[i].Score)
	}
	for i := range result.Skills {
		result.Skills[i].Score = clampScore(result.Skills[i].Score)
	}

	return &result, nil
}

// clampScore borne un score entre 1 et 100 pour éviter les valeurs aberrantes du LLM
func clampScore(score int) int {
	if score < 1 {
		return 1
	}
	if score > 100 {
		return 100
	}
	return score
}
