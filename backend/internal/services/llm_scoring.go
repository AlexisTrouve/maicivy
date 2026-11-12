package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"maicivy/internal/config"
	"maicivy/internal/models"
)

// Pertinence des projets pour un thème de CV, jugée par un LLM.
//
// Refonte du 2026-10-11. Ce que faisait l'ancienne version, mesuré sur le cache de prod :
//   - Haiku 4.5 épinglé, un prompt « expert en recrutement, note 1-100 », et pour chaque projet
//     le titre, les technos et 120 caractères de description. Rien sur la production, l'ampleur,
//     l'activité.
//   - Résultat : un classement par MOTS-CLÉS. Sur le thème agents, ClaudeDuo (retiré) sortait 1er,
//     un petit serveur MCP Reddit 6e, maicivy (en prod, 94 k lignes) 18e, GroveEngine 57e (vu
//     comme « moteur de jeu 2D »). Notes tassées : 34 projets sur 75 à 70 ou plus, donc un ordre
//     qui se jouait à quelques points et changeait d'un appel à l'autre.
//   - Scores indexés par un slug du TITRE : réécrire ou traduire un titre perdait son score.
//
// Ce qui change :
//   - Le modèle ne juge plus que la PERTINENCE (ce que le projet démontre pour le positionnement),
//     avec un barème ancré et une échelle imposée. La preuve (production, ampleur) et l'activité
//     sont pondérées ensuite par le CODE (cv_project_ranking.go) : déterministe, testable, et
//     jamais comptée deux fois.
//   - Il reçoit l'accroche, le statut et l'ampleur — de quoi distinguer un outil d'un système.
//   - Scores indexés par l'identifiant maiProFiles (stable).
//   - Modèle configurable et épinglé (CV_MODEL) : le classement parle au nom d'Alexis.

// Dimensionnement du budget de sortie. Une paire "id": score coûte ~10 tokens avec les
// identifiants maiProFiles ; on garde la marge mesurée sur les anciens slugs longs (32), le coût
// d'un budget trop large est nul, celui d'un budget trop court est une troncature.
const (
	tokensParEntreeScore = 32
	margeTokensScoring   = 256
	budgetScoringMin     = 512

	// couvertureMinimale : en dessous, le scoring ne discrimine plus assez pour trier quoi que
	// ce soit. On rend une erreur explicite plutot qu'un classement qui a l'air valide.
	couvertureMinimale = 0.5

	// scoreProjetNonNote : pertinence (sur 100) des projets que le modele a omis. BAS : un projet
	// ignoré ne doit pas s'insérer au milieu du classement.
	scoreProjetNonNote = 15

	// versionPromptPertinence : à incrémenter à CHAQUE modification du prompt. Elle entre dans la
	// clé de cache — sans elle, un nouveau prompt servirait l'ancien classement pendant 6 h.
	versionPromptPertinence = "v2"

	// ttlScoresPertinence : durée de vie d'un classement en cache. La clé change d'elle-même quand
	// le catalogue, le thème, le modèle ou le prompt changent ; le TTL ne borne que la dérive lente.
	ttlScoresPertinence = 6 * time.Hour
)

// budgetScoring calcule le max_tokens necessaire pour scorer n projets.
func budgetScoring(n int) int {
	budget := n*tokensParEntreeScore + margeTokensScoring
	if budget < budgetScoringMin {
		return budgetScoringMin
	}
	return budget
}

// LLMScoringService juge la pertinence des projets via un appel LLM unique par thème.
type LLMScoringService struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
	redis      *redis.Client
}

// NewLLMScoringService crée une instance. Retourne nil si baseURL ou apiKey vides.
// model : identifiant épinglé (cf. config.AIConfig.CVModel).
func NewLLMScoringService(baseURL, apiKey, model string, redisClient *redis.Client) *LLMScoringService {
	if baseURL == "" || apiKey == "" {
		return nil
	}
	return &LLMScoringService{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			// Un modèle plus capable que Haiku met plus longtemps sur ~75 projets.
			Timeout: 90 * time.Second,
		},
		redis: redisClient,
	}
}

// ScoreProjectsForTheme retourne la pertinence (0-100) de chaque projet pour le thème, indexée par
// SourceID. Un projet sans SourceID est ignoré. Un appel LLM par (thème, catalogue, modèle, prompt).
func (s *LLMScoringService) ScoreProjectsForTheme(ctx context.Context, projects []models.Project, theme *config.CVTheme) (map[string]int, error) {
	projets := projetsIdentifies(projects)
	if len(projets) == 0 {
		return nil, fmt.Errorf("aucun projet identifié à noter")
	}
	cacheKey := scoringCacheKey(theme, s.model, projets)

	if cached, err := s.redis.Get(ctx, cacheKey).Result(); err == nil {
		var scores map[string]int
		if json.Unmarshal([]byte(cached), &scores) == nil {
			return scores, nil
		}
	}

	prompt := buildProjectScoringPrompt(projets, theme)
	response, stopReason, err := s.callClaude(ctx, prompt, budgetScoring(len(projets)))
	if err != nil {
		return nil, fmt.Errorf("LLM scoring failed: %w", err)
	}

	// Troncature : le JSON est forcement incomplet. Sans ce test, l'echec se presentait comme
	// un banal "invalid JSON" et se confondait avec une reponse mal formee.
	if stopReason == "max_tokens" {
		return nil, fmt.Errorf("reponse tronquee par le budget de sortie (%d tokens pour %d projets)",
			budgetScoring(len(projets)), len(projets))
	}

	scores, err := parseScoreJSON(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM scores: %w", err)
	}

	// Couverture : un JSON valide qui ne note que 3 projets sur 75 n'est pas un succès.
	notes := 0
	for _, p := range projets {
		if _, ok := scores[p.SourceID]; ok {
			notes++
		}
	}
	if float64(notes) < couvertureMinimale*float64(len(projets)) {
		return nil, fmt.Errorf("scoring partiel : %d/%d projets notes", notes, len(projets))
	}

	// Ne garder QUE les identifiants connus (un id inventé par le modèle n'a rien à faire ici),
	// et faire tomber bas les oublis résiduels.
	propres := make(map[string]int, len(projets))
	manquants := 0
	for _, p := range projets {
		if v, ok := scores[p.SourceID]; ok {
			propres[p.SourceID] = v
		} else {
			propres[p.SourceID] = scoreProjetNonNote
			manquants++
		}
	}
	if manquants > 0 {
		log.Warn().Int("manquants", manquants).Int("total", len(projets)).
			Str("theme", theme.ID).Msg("projets non notes par le LLM, pertinence plancher appliquee")
	}

	if data, err := json.Marshal(propres); err == nil {
		s.redis.Set(ctx, cacheKey, data, ttlScoresPertinence)
	}
	return propres, nil
}

// projetsIdentifies : les projets qui ont un SourceID, triés par identifiant (prompt et empreinte
// déterministes, indépendants de l'ordre d'arrivée).
func projetsIdentifies(projects []models.Project) []models.Project {
	out := make([]models.Project, 0, len(projects))
	for _, p := range projects {
		if p.SourceID != "" {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SourceID < out[j].SourceID })
	return out
}

// scoringCacheKey : clé Redis d'un classement. Elle change d'elle-même quand l'une de ses entrées
// change — le thème (nom, description, pitch), le modèle, la version du prompt, ou le CATALOGUE
// (identifiants, accroches, statuts). Ajouter un projet ou en passer un en « retired » re-classe
// donc immédiatement, au lieu d'attendre 6 h.
func scoringCacheKey(theme *config.CVTheme, model string, projets []models.Project) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00%s\x00%s\x00%s\x00%s\x00", versionPromptPertinence, model, theme.Name, theme.Description, theme.PitchFor("fr"))
	for _, p := range projets {
		fmt.Fprintf(h, "%s|%s|%s\x00", p.SourceID, p.Status, p.Catchphrase)
	}
	return fmt.Sprintf("llm:project-scores:%s:%x", theme.ID, h.Sum(nil)[:6])
}

// buildProjectScoringPrompt construit le prompt de pertinence pour tous les projets d'un coup.
func buildProjectScoringPrompt(projets []models.Project, theme *config.CVTheme) string {
	var sb strings.Builder
	sb.WriteString("Tu classes les projets d'un développeur freelance pour la section « Projets » de son CV.\n\n")
	fmt.Fprintf(&sb, "POSITIONNEMENT VISÉ : %s\n%s\n", theme.Name, theme.Description)
	if pitch := theme.PitchFor("fr"); pitch != "" {
		fmt.Fprintf(&sb, "%s\n", pitch)
	}
	sb.WriteString(`
LECTEUR : un CTO technique qui parcourt le CV en quelques secondes et cherche des PREUVES que ce
positionnement est réel.

Pour CHAQUE projet, note de 0 à 100 ce que le projet DÉMONTRE pour ce positionnement.

Barème :
- 90-100 : démontre le CŒUR du positionnement, sous la forme d'un vrai système (architecture,
  plusieurs composants, usage réel). Au plus 8 projets dans cette tranche.
- 70-89 : démontre clairement une partie du positionnement, OU une capacité qui le rend crédible
  (par exemple construire soi-même l'infrastructure, le moteur ou l'application qui entoure le cœur).
- 40-69 : pertinent mais secondaire : petit outil, intégration ponctuelle, prototype, wrapper d'API.
- 10-39 : adjacent — compétence transférable, sans plus.
- 0-9 : hors sujet.

Règles :
- Juge le FOND, pas les mots-clés : un petit outil qui porte « agent », « IA » ou « MCP » dans son
  nom ne vaut pas un système qui fait réellement le travail. Lis l'accroche.
- Utilise l'ampleur (lignes, tests) pour distinguer un outil d'un système, mais ne note pas la
  taille pour elle-même : la mise en production, l'ampleur et l'activité récente sont pondérées
  APRÈS toi, par ailleurs. Ne les compte pas deux fois.
- Ignore le statut « retired » : il est traité par ailleurs. Note la pertinence telle quelle.
- Utilise toute l'échelle : la majorité des projets doit être SOUS 70.

Projets (id | nom | accroche | catégorie | statut | lignes | tests | technos) :
`)
	for _, p := range projets {
		techs := p.Technologies
		if len(techs) > 8 {
			techs = techs[:8]
		}
		accroche := p.Catchphrase
		if accroche == "" {
			accroche = tronquer(p.Description, 160)
		}
		fmt.Fprintf(&sb, "- %s | %s | %s | %s | %s | %d | %d | %s\n",
			p.SourceID, nomProjetCourt(p.Title), accroche, p.Category, p.Status, p.LOC, p.TestCount,
			strings.Join(techs, ", "))
	}
	sb.WriteString(`
Réponds UNIQUEMENT avec un objet JSON valide qui couvre TOUS les id ci-dessus, sans texte autour,
sans markdown : {"id_du_projet": 85, ...}`)
	return sb.String()
}

// tronquer coupe un texte à n runes (accroche de repli quand un projet n'a pas de `mini`).
func tronquer(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}

// callClaude envoie le prompt au proxy et retourne le texte de la reponse ET son stop_reason.
// Le stop_reason est ce qui distingue "le modele a fini" de "le budget a coupe la phrase" :
// sans lui, une reponse tronquee ressemble a un JSON mal forme.
// ⚠️ Aucun `thinking` : sur les modèles 5.x, `budget_tokens` à travers la passerelle rend un 200
// avec un bloc vide (cf. CLAUDE.md). Le barème suffit, on lit le premier bloc TEXTE.
func (s *LLMScoringService) callClaude(ctx context.Context, prompt string, maxTokens int) (string, string, error) {
	body := map[string]interface{}{
		"model":      s.model,
		"max_tokens": maxTokens,
		// Pas de `temperature` : les modèles 5.x la refusent (400 « may only be set to 1 when
		// thinking is enabled or in adaptive mode », mesuré le 2026-10-11 sur Sonnet 5.5). La
		// stabilité du classement vient du cache, indexé sur tout ce qui peut le changer.
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBytes))
	}

	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
	}
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return "", "", fmt.Errorf("failed to parse API response: %w", err)
	}
	for _, c := range apiResp.Content {
		if (c.Type == "" || c.Type == "text") && strings.TrimSpace(c.Text) != "" {
			return c.Text, apiResp.StopReason, nil
		}
	}
	return "", apiResp.StopReason, fmt.Errorf("empty API response")
}

// parseScoreJSON parse le JSON retourné par le LLM (tolère un bloc markdown autour) et borne
// chaque score à 0..100.
func parseScoreJSON(text string) (map[string]int, error) {
	text = strings.TrimSpace(text)
	if idx := strings.Index(text, "{"); idx >= 0 {
		text = text[idx:]
	}
	if idx := strings.LastIndex(text, "}"); idx >= 0 {
		text = text[:idx+1]
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w\ntext: %s", err, text)
	}

	scores := make(map[string]int, len(raw))
	for key, val := range raw {
		if v, ok := val.(float64); ok {
			score := int(v)
			if score < 0 {
				score = 0
			}
			if score > 100 {
				score = 100
			}
			scores[key] = score
		}
	}
	return scores, nil
}

// toSlug convertit un titre en identifiant slug (lowercase, tirets). N'indexe PLUS les scores
// (cf. SourceID) ; reste utilisé ailleurs (slugs de blog, CV taillé).
func toSlug(title string) string {
	s := strings.ToLower(title)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}
