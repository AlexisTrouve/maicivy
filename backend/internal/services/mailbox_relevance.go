package services

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/rs/zerolog/log"
)

// MailboxRelevanceVerdict — jugement de l'agent sur un mail Malt capté. Reason/CoT sont toujours en
// anglais (demandé explicitement — contenu technique interne au panel admin, pas de l'UI i18n).
type MailboxRelevanceVerdict struct {
	IsOpportunity bool   // vrai UNIQUEMENT si c'est une proposition concrète de mission/job
	Score         int    // 0-100, pertinence vs. profil — significatif seulement si IsOpportunity
	Reason        string // explication courte (EN)
	CoT           string // raisonnement pas-à-pas de l'agent (EN) — ce qu'il a vérifié et pourquoi
	Link          string // URL principale (voir/postuler à la mission) extraite du corps, "" si absente
}

// MailboxRelevanceEvaluator juge si un mail Malt capté est une vraie opportunité de mission et, si
// oui, si elle correspond au profil d'Alexi — interface pour permettre un fake en test
// (mailbox_service_test.go), sans appel réseau réel. MailboxRelevanceService (ci-dessous) est la
// vraie implémentation (agent Anthropic multi-tours, cf. relevanceSystemPrompt).
type MailboxRelevanceEvaluator interface {
	// Evaluate juge un mail. err != nil → MailboxService considère ça un échec technique et
	// transfère quand même (fail-open : ne jamais perdre une opportunité à cause d'une panne LLM).
	Evaluate(ctx context.Context, subject, body string) (MailboxRelevanceVerdict, error)
	// Threshold retourne le score minimal pour qu'une opportunité soit jugée pertinente.
	Threshold() int
}

// maxRelevanceBodyChars plafonne le corps (déjà nettoyé du bruit HTML, cf. StripHTMLNoise) envoyé à
// l'agent — coût/bruit, le corps stocké peut faire jusqu'à 200KB (cf. MailboxEmail), très au-delà de
// ce qu'il faut pour juger une offre de mission. 8000 (pas 4000) : marge de sécurité après un vrai
// backtest où un bloc CSS de ~3000 caractères (mail Malt sans partie text/plain, fallback HTML brut —
// cf. ImapFetcher.extractPlainText) repoussait la description de mission hors de la troncature avant
// le nettoyage — l'agent concluait à tort "email tronqué/incomplet".
const maxRelevanceBodyChars = 8000

var (
	styleScriptRe = regexp.MustCompile(`(?is)<(style|script)[^>]*>.*?</(style|script)>`)
	anchorRe      = regexp.MustCompile(`(?is)<a\s+[^>]*href=["']([^"']+)["'][^>]*>(.*?)</a>`)
	htmlTagRe     = regexp.MustCompile(`(?is)<[^>]+>`)
	blankLinesRe  = regexp.MustCompile(`\n{3,}`)
)

// StripHTMLNoise nettoie un corps de mail — retire CSS/JS et balises HTML tout en PRÉSERVANT les URLs
// des liens (converties en "texte (URL)", jamais perdues). Utilisé pour TROIS consommateurs : l'input
// de l'agent de pertinence, l'input du traducteur (cf. MailboxTranslationService.Translate), et
// l'affichage panel admin (cf. MailboxHandler.detail) — un mail sans partie text/plain (fallback
// HTML brut, cf. ImapFetcher.extractPlainText) ne doit JAMAIS montrer de balises/CSS bruts à qui que
// ce soit qui le consulte, humain ou agent.
//
// POURQUOI : le stockage (MailboxEmail.BodyText) garde le corps BRUT tel qu'IMAP le renvoie — un mail
// sans partie text/plain peut commencer par un gros bloc <style> AVANT le contenu utile. Sans ce
// nettoyage : (1) une troncature à maxRelevanceBodyChars coupe le mail avant même d'atteindre la
// description de mission (vécu en backtest réel : l'agent a conclu "email tronqué/incomplet" sur une
// VRAIE opportunité, uniquement à cause du bruit CSS en tête) ; (2) le panel admin affiche un pavé de
// balises/CSS illisible (retour utilisateur réel après le backfill des 28 mails historiques).
// Le nettoyage est appliqué À LA CONSULTATION (jamais persisté) — MailboxEmail.BodyText reste brut en
// base, source de vérité.
//
// COMMENT : 1. retire <style>/<script> ; 2. transforme "<a href=URL>texte</a>" en "texte (URL)" AVANT
// de retirer les balises génériquement — sinon le lien de mission disparaît avec le reste des tags
// (bug constaté dans le même backtest : Link revenait vide après un strip naïf de toutes les balises).
func StripHTMLNoise(body string) string {
	body = styleScriptRe.ReplaceAllString(body, "")
	body = anchorRe.ReplaceAllString(body, "$2 ($1)")
	body = htmlTagRe.ReplaceAllString(body, " ")
	body = blankLinesRe.ReplaceAllString(body, "\n\n")
	return strings.TrimSpace(body)
}

// maxRelevanceTurns borne la boucle agentic (comme ChatService.Chat). Le DERNIER tour force
// ToolChoice=submit_verdict (cf. Evaluate) — la boucle converge donc TOUJOURS vers un verdict
// structuré en au plus maxRelevanceTurns appels, jamais un "no verdict" en pratique.
const maxRelevanceTurns = 5

// relevanceMaxTokens — budget de sortie par tour. 1024 s'est révélé TROP BAS lors d'un backtest réel :
// le tool call submit_verdict (is_opportunity + score + reason + cot + link) coupait en plein milieu
// (stop_reason="max_tokens"), le modèle laissait cot/link vides pour tenir dans le budget, et le
// verdict lui-même devenait instable d'un run à l'autre sur le MÊME mail (artefact de troncature, pas
// un vrai désaccord de jugement). 4096 laisse une marge large — coût négligeable vu le volume
// (quelques mails Malt/semaine) et le modèle (Haiku).
const relevanceMaxTokens = 4096

// relevanceModel — Haiku : volume faible (quelques mails Malt/semaine), coût non significatif.
const relevanceModel = "claude-haiku-4-5-20251001"

// relevanceSystemPrompt — en anglais : c'est la langue de travail de l'agent (raisonnement + verdict),
// indépendamment de la langue du mail capté (Malt notifie en français).
const relevanceSystemPrompt = `You are a freelance mission triage agent for Alexi, a software developer. You receive an email captured from Malt (a French freelance platform) and must judge it.

Your task:
1. Determine if this email is a CONCRETE freelance mission/job proposal from a client — NOT a newsletter, community digest, verification code, account notification, or generic marketing content.
2. If it is a real mission proposal, judge how well it matches Alexi's ACTUAL skills and experience. Use the get_profile, get_experience, list_skills, and search_projects tools as needed to verify specifics before deciding — e.g. if the mission mentions a particular technology, domain, or seniority level, check whether Alexi genuinely has relevant experience rather than guessing from the headline alone.
3. Extract the primary URL from the email body that lets Alexi view or apply to the mission, if one is present — ALWAYS do this regardless of how relevant the mission is, even a poor-fit mission may still be worth Alexi seeing directly. The email body has been stripped of HTML tags but URLs remain as plain text, often in parentheses after a link's anchor text (e.g. "Respond to client (https://...)") and often near the end of the message. They may contain raw HTML entities like &amp; or &#x3D; instead of & and = — normalize these when extracting (decode &amp;→&, &#x3D;→=, &quot;→", etc.) so the link field contains a clean, directly-usable URL. Look for it deliberately; do not report it as absent just because the main mission description doesn't mention it.

Think step by step, and verify claims against the tools rather than assuming. Once you are confident in your judgment, call submit_verdict EXACTLY ONCE with your final answer. All text you produce (reason, cot) must be in English, regardless of the email's own language.`

// MailboxRelevanceService — implémentation réelle de MailboxRelevanceEvaluator : agent multi-tours
// (même pattern que ChatService.Chat — SDK Anthropic direct, boucle tool_use) plutôt qu'un simple
// appel HTTP JSON in/out. L'agent peut interroger le profil/expérience/skills/projets d'Alexi via
// PortfolioService avant de conclure, au lieu de se fier à un résumé statique.
type MailboxRelevanceService struct {
	client    *anthropic.Client
	portfolio *PortfolioService
	threshold int
}

var _ MailboxRelevanceEvaluator = (*MailboxRelevanceService)(nil)

// NewMailboxRelevanceService — nil si baseURL/apiKey absents. POURQUOI : feature optionnelle, même
// convention que le reste de maicivy — credentials absents = service désactivé proprement,
// MailboxService avec relevance=nil transfère tout sans jugement.
func NewMailboxRelevanceService(baseURL, apiKey string, portfolio *PortfolioService, threshold int) *MailboxRelevanceService {
	if baseURL == "" || apiKey == "" {
		return nil
	}
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	client := anthropic.NewClient(opts...)
	return &MailboxRelevanceService{
		client:    &client,
		portfolio: portfolio,
		threshold: threshold,
	}
}

// Threshold expose le seuil configuré.
func (s *MailboxRelevanceService) Threshold() int { return s.threshold }

// Evaluate exécute la boucle agentic : l'agent peut appeler des tools de lecture (profil/expérience/
// skills/projets) avant de rendre son verdict via le tool terminal submit_verdict. Le dernier tour
// force ToolChoice=submit_verdict pour garantir la convergence (jamais de sortie sans verdict, sauf
// échec de l'appel API lui-même — qui remonte en erreur, géré fail-open par MailboxService).
func (s *MailboxRelevanceService) Evaluate(ctx context.Context, subject, body string) (MailboxRelevanceVerdict, error) {
	body = StripHTMLNoise(body)
	if len(body) > maxRelevanceBodyChars {
		body = body[:maxRelevanceBodyChars] + "..."
	}

	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(fmt.Sprintf("Subject: %s\n\nBody:\n%s", subject, body))),
	}
	tools := buildRelevanceTools()

	for turn := 0; turn < maxRelevanceTurns; turn++ {
		params := anthropic.MessageNewParams{
			Model:     anthropic.Model(relevanceModel),
			MaxTokens: relevanceMaxTokens,
			System:    []anthropic.TextBlockParam{{Text: relevanceSystemPrompt}},
			Messages:  messages,
			Tools:     tools,
		}
		if turn == maxRelevanceTurns-1 {
			params.ToolChoice = anthropic.ToolChoiceParamOfTool("submit_verdict")
		}

		resp, err := s.client.Messages.New(ctx, params)
		if err != nil {
			return MailboxRelevanceVerdict{}, fmt.Errorf("mailbox relevance agent call: %w", err)
		}
		// Garde-fou : si ça arrive quand même malgré relevanceMaxTokens généreux (cf. sa doc), un
		// verdict tronqué peut avoir des champs vides (cot/link) voire un jugement instable — on ne
		// bloque pas dessus (le verdict partiel peut rester exploitable), mais on le rend VISIBLE
		// plutôt que de laisser passer silencieusement (déjà vécu une fois en backtest réel).
		if resp.StopReason == "max_tokens" {
			log.Warn().Int("turn", turn).Msg("mailbox relevance: réponse coupée par max_tokens — verdict potentiellement incomplet")
		}

		var assistantContent []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			switch b := block.AsAny().(type) {
			case anthropic.TextBlock:
				assistantContent = append(assistantContent, anthropic.NewTextBlock(b.Text))
			case anthropic.ToolUseBlock:
				assistantContent = append(assistantContent, anthropic.NewToolUseBlock(b.ID, b.Input, b.Name))
				if b.Name == "submit_verdict" {
					return parseVerdictToolInput(b.Input)
				}
			}
		}
		messages = append(messages, anthropic.NewAssistantMessage(assistantContent...))

		var toolResults []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			tb, ok := block.AsAny().(anthropic.ToolUseBlock)
			if !ok {
				continue
			}
			result, toolErr := s.executeRelevanceTool(tb.Name, tb.Input)
			if toolErr != nil {
				result = map[string]string{"error": toolErr.Error()}
			}
			resultJSON, _ := json.Marshal(result)
			toolResults = append(toolResults, anthropic.NewToolResultBlock(tb.ID, string(resultJSON), toolErr != nil))
		}
		if len(toolResults) > 0 {
			messages = append(messages, anthropic.NewUserMessage(toolResults...))
		} else {
			// Pas de tool_use (texte seul, pas de verdict) → relance en insistant.
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(
				"Please call submit_verdict with your final judgment.")))
		}
	}

	return MailboxRelevanceVerdict{}, fmt.Errorf("mailbox relevance agent: no verdict after %d turns", maxRelevanceTurns)
}

// buildRelevanceTools définit les tools disponibles pour l'agent — lecture seule (profil/expérience/
// skills/projets, réutilise PortfolioService comme ChatService) + le tool terminal submit_verdict qui
// force une sortie structurée (pas de parsing JSON fragile sur du texte libre).
func buildRelevanceTools() []anthropic.ToolUnionParam {
	makeTool := func(name, desc string, schema anthropic.ToolInputSchemaParam) anthropic.ToolUnionParam {
		p := anthropic.ToolParam{Name: name, Description: anthropic.String(desc), InputSchema: schema}
		return anthropic.ToolUnionParam{OfTool: &p}
	}
	empty := anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{}}

	return []anthropic.ToolUnionParam{
		makeTool("get_profile",
			"Get Alexi's profile: headline, bio, years of experience, domains, strong/familiar skills.",
			empty),
		makeTool("get_experience",
			"Get Alexi's full work history: companies, dates, technologies used per role.",
			empty),
		makeTool("list_skills",
			"Get Alexi's skills grouped by category, each with level and years of experience.",
			empty),
		makeTool("search_projects",
			"Search Alexi's past projects by keyword (technology, domain) to verify hands-on experience with something specific mentioned in the mission.",
			anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Keyword to search for (e.g. 'kubernetes', 'fintech', 'react native')",
					},
				},
				Required: []string{"query"},
			}),
		makeTool("submit_verdict",
			"Submit your final judgment on this email. MUST be called exactly once, as your final action.",
			anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"is_opportunity": map[string]interface{}{
						"type":        "boolean",
						"description": "True ONLY if this is a concrete freelance mission/job proposal from a client. False for newsletters, community digests, verification codes, account notifications, or generic marketing.",
					},
					"score": map[string]interface{}{
						"type":        "integer",
						"description": "0-100 relevance of the mission vs. Alexi's profile (100 = perfect match, 0 = no relation). Leave 0 if is_opportunity is false.",
					},
					"reason": map[string]interface{}{
						"type":        "string",
						"description": "One short sentence (English) explaining the score, or why this isn't an opportunity.",
					},
					"cot": map[string]interface{}{
						"type":        "string",
						"description": "Your step-by-step reasoning (English): what you checked, what you found, how it led to the score.",
					},
					"link": map[string]interface{}{
						"type":        "string",
						"description": "The primary URL to view or apply to this mission, extracted from the email body. Empty string if none found.",
					},
				},
				Required: []string{"is_opportunity", "score", "reason", "cot", "link"},
			}),
	}
}

// executeRelevanceTool exécute un tool de lecture (jamais submit_verdict, intercepté avant dans
// Evaluate). Réutilise parseInput/slimProjects déjà définis dans chat_service.go (même package).
func (s *MailboxRelevanceService) executeRelevanceTool(name string, input interface{}) (interface{}, error) {
	switch name {
	case "get_profile":
		return s.portfolio.GetProfile("en"), nil
	case "get_experience":
		return s.portfolio.GetExperience("en"), nil
	case "list_skills":
		return s.portfolio.ListSkills("en"), nil
	case "search_projects":
		inputMap, ok := parseInput(input)
		if !ok {
			return nil, fmt.Errorf("invalid input for search_projects")
		}
		query, _ := inputMap["query"].(string)
		if query == "" {
			return nil, fmt.Errorf("missing query")
		}
		return slimProjects(s.portfolio.SearchProjects(query, "en")), nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// parseVerdictToolInput décode l'input du tool submit_verdict en verdict, en clampant le score.
func parseVerdictToolInput(input interface{}) (MailboxRelevanceVerdict, error) {
	m, ok := parseInput(input)
	if !ok {
		return MailboxRelevanceVerdict{}, fmt.Errorf("invalid submit_verdict input")
	}
	isOpportunity, _ := m["is_opportunity"].(bool)
	reason, _ := m["reason"].(string)
	cot, _ := m["cot"].(string)
	link, _ := m["link"].(string)
	scoreF, _ := m["score"].(float64) // les nombres JSON décodent en float64
	score := int(scoreF)
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return MailboxRelevanceVerdict{
		IsOpportunity: isOpportunity,
		Score:         score,
		Reason:        reason,
		CoT:           cot,
		Link:          link,
	}, nil
}
