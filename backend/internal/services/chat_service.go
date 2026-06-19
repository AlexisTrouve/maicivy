package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/rs/zerolog/log"

	"maicivy/internal/config"
)

// ChatEventType identifie le type d'event SSE envoyé au client
type ChatEventType string

const (
	ChatEventText       ChatEventType = "text"
	ChatEventToolCall   ChatEventType = "tool_call"
	ChatEventToolResult ChatEventType = "tool_result"
	ChatEventDone       ChatEventType = "done"
	ChatEventError      ChatEventType = "error"
)

// ChatEvent est envoyé dans le channel SSE vers le handler
type ChatEvent struct {
	Type    ChatEventType   `json:"type"`
	Delta   string          `json:"delta,omitempty"`   // pour type="text"
	Name    string          `json:"name,omitempty"`    // pour tool_call / tool_result
	Input   json.RawMessage `json:"input,omitempty"`   // pour tool_call
	Data    interface{}     `json:"data,omitempty"`    // pour tool_result
	Message string          `json:"message,omitempty"` // pour error
}

// ChatMessage représente un tour de conversation (historique)
type ChatMessage struct {
	Role    string `json:"role"`    // "user" | "assistant"
	Content string `json:"content"`
}

// ChatService gère les conversations avec Claude + tool_use portfolio
type ChatService struct {
	client    *anthropic.Client
	portfolio *PortfolioService
	blog      *BlogGeneratorService // pour les tools show_blog_*
	model     string                // modèle par défaut
}

// NewChatService crée un ChatService avec le client Anthropic configuré.
// Utilise ChatAPIKey si défini, sinon fallback sur AnthropicAPIKey.
func NewChatService(cfg *config.AIConfig, portfolio *PortfolioService, blog *BlogGeneratorService) *ChatService {
	apiKey := cfg.ChatAPIKey
	if apiKey == "" {
		apiKey = cfg.AnthropicAPIKey
	}
	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}
	// Toujours passer par le proxy etheryale
	if cfg.AnthropicBaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.AnthropicBaseURL))
	}
	client := anthropic.NewClient(opts...)
	return &ChatService{
		client:    &client,
		portfolio: portfolio,
		blog:      blog,
		model:     "claude-haiku-4-5-20251001", // modèle par défaut (visiteurs)
	}
}

// Chat exécute une boucle agentic multi-turn et envoie les events dans eventCh.
// model permet de surcharger le modèle (owner → Opus).
func (s *ChatService) Chat(ctx context.Context, message string, history []ChatMessage, model string, eventCh chan<- ChatEvent) {
	defer close(eventCh)

	if model == "" {
		model = s.model
	}

	// Construire l'historique Anthropic depuis l'historique fourni
	messages := s.buildMessages(history, message)

	// Définir les tools disponibles pour Claude
	tools := s.buildTools()

	// Construire le prompt système avec les données live du profil
	prompt := s.buildSystemPrompt(ctx)

	// Boucle agentic (max 5 tours pour éviter les boucles infinies)
	const maxTurns = 5
	for turn := 0; turn < maxTurns; turn++ {
		resp, err := s.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.Model(model),
			MaxTokens: 600, // réduit pour rester sous la limite proxy ~7300 tokens
			System: []anthropic.TextBlockParam{
				{Text: prompt},
			},
			Messages: messages,
			Tools:    tools, // []anthropic.ToolUnionParam
		})
		if err != nil {
			log.Error().Err(err).Msg("ChatService: Claude API error")
			// Détecter une limite de taille atteinte (erreur proxy ou contexte trop long)
			errMsg := err.Error()
			userMsg := "Désolé, quelque chose s'est mal passé. Veuillez réessayer."
			if strings.Contains(errMsg, "too large") || strings.Contains(errMsg, "token") ||
				strings.Contains(errMsg, "context") || strings.Contains(errMsg, "limit") ||
				strings.Contains(errMsg, "413") || strings.Contains(errMsg, "Request too large") {
				userMsg = "Cette conversation est devenue trop longue. Commencez une nouvelle conversation pour continuer."
			}
			eventCh <- ChatEvent{Type: ChatEventError, Message: userMsg}
			return
		}

		// Traiter les blocs de réponse
		hasToolUse := false
		var assistantContent []anthropic.ContentBlockParamUnion

		for _, block := range resp.Content {
			switch b := block.AsAny().(type) {
			case anthropic.TextBlock:
				// Envoyer le texte d'un coup (MVP sync)
				if b.Text != "" {
					eventCh <- ChatEvent{Type: ChatEventText, Delta: b.Text}
				}
				assistantContent = append(assistantContent, anthropic.NewTextBlock(b.Text))

			case anthropic.ToolUseBlock:
				hasToolUse = true
				// Émettre l'event tool_call pour affichage inline dans le chat
				inputRaw, _ := json.Marshal(b.Input)
				eventCh <- ChatEvent{
					Type:  ChatEventToolCall,
					Name:  b.Name,
					Input: inputRaw,
				}
				// NewToolUseBlock(id, input, name) — ordre SDK : id, input any, name
				assistantContent = append(assistantContent, anthropic.NewToolUseBlock(b.ID, b.Input, b.Name))
			}
		}

		// Ajouter le message assistant à l'historique
		messages = append(messages, anthropic.NewAssistantMessage(assistantContent...))

		// Si pas de tool_use → conversation terminée
		if !hasToolUse || resp.StopReason == "end_turn" {
			eventCh <- ChatEvent{Type: ChatEventDone}
			return
		}

		// Exécuter les tools et construire les tool_result
		var toolResults []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			tb, ok := block.AsAny().(anthropic.ToolUseBlock)
			if !ok {
				continue
			}

			result, toolErr := s.executeTool(tb.Name, tb.Input)
			if toolErr != nil {
				log.Error().Err(toolErr).Str("tool", tb.Name).Msg("ChatService: tool execution error")
				result = map[string]string{"error": toolErr.Error()}
			}

			// Émettre l'event tool_result → data COMPLÈTE pour le frontend (panel droit)
			eventCh <- ChatEvent{
				Type: ChatEventToolResult,
				Name: tb.Name,
				Data: result,
			}

			// Claude ne reçoit qu'une version allégée : LongDesc supprimée pour économiser les tokens.
			// La LongDesc (description Markdown complète) n'est utile qu'au frontend pour l'affichage.
			claudeResult := trimForClaude(tb.Name, result)
			resultJSON, _ := json.Marshal(claudeResult)
			toolResults = append(toolResults, anthropic.NewToolResultBlock(tb.ID, string(resultJSON), toolErr != nil))
		}

		// Ajouter les tool_results dans l'historique pour le tour suivant
		if len(toolResults) > 0 {
			messages = append(messages, anthropic.NewUserMessage(toolResults...))
		}
	}

	// Sécurité : si on sort de la boucle sans envoyer done
	eventCh <- ChatEvent{Type: ChatEventDone}
}

// buildMessages convertit l'historique ChatMessage en params Anthropic.
// On ne garde que les 6 derniers messages (3 échanges) pour rester sous la
// limite de ~7300 tokens du proxy etheryale.
func (s *ChatService) buildMessages(history []ChatMessage, newMessage string) []anthropic.MessageParam {
	var messages []anthropic.MessageParam

	// Tronquer l'historique : garder les 6 derniers messages max
	const maxHistory = 6
	if len(history) > maxHistory {
		history = history[len(history)-maxHistory:]
	}

	for _, msg := range history {
		if msg.Role == "user" {
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
		} else if msg.Role == "assistant" {
			messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)))
		}
	}

	// Ajouter le message courant
	messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(newMessage)))
	return messages
}

// supportedChatLangs : langues dans lesquelles le portfolio peut être servi (= locales du front).
// Toute autre valeur passée par le LLM déclenche une erreur tool (cf. validateLang).
var supportedChatLangs = map[string]bool{
	"fr": true, "en": true, "de": true, "it": true, "zh": true,
}

// chatLangList : liste affichable des langues supportées, pour les messages d'erreur au LLM.
const chatLangList = "fr, en, de, it, zh"

// validateLang extrait et valide le paramètre `language` (obligatoire sur TOUS les tools).
// POURQUOI : le contenu portfolio (maiProFiles) n'existe que dans un nombre fini de langues. On force
// le LLM à déclarer la langue de la conversation à chaque tool ; si elle n'est pas servable, on lui
// renvoie une ERREUR explicite (au lieu d'un fallback FR silencieux) qui l'invite à répondre en anglais.
// Le LLM voit ce message (tool_result isError=true) et peut retenter en "en".
func validateLang(input map[string]interface{}) (string, error) {
	lang, _ := input["language"].(string)
	lang = strings.ToLower(strings.TrimSpace(lang))
	if !supportedChatLangs[lang] {
		return "", fmt.Errorf(
			"unsupported language %q. Available languages: %s. Answer the user in English (en) instead.",
			lang, chatLangList,
		)
	}
	return lang, nil
}

// withLang ajoute le paramètre `language` (requis) à un schema de tool — appliqué à TOUS les tools
// pour que le LLM déclare systématiquement la langue de la conversation.
func withLang(props map[string]interface{}, required ...string) anthropic.ToolInputSchemaParam {
	props["language"] = map[string]interface{}{
		"type":        "string",
		"description": "Langue de la conversation en code ISO 639-1 (ex: fr, en, de, it, zh, es, ja). À passer systématiquement. Contenu servable en fr, en, de, it, zh uniquement — toute autre valeur renvoie une erreur et tu dois répondre en anglais (en).",
	}
	return anthropic.ToolInputSchemaParam{
		Properties: props,
		Required:   append(required, "language"),
	}
}

// buildTools définit les tools disponibles pour Claude
func (s *ChatService) buildTools() []anthropic.ToolUnionParam {
	makeTool := func(name, desc string, schema anthropic.ToolInputSchemaParam) anthropic.ToolUnionParam {
		p := anthropic.ToolParam{
			Name:        name,
			Description: anthropic.String(desc),
			InputSchema: schema,
		}
		return anthropic.ToolUnionParam{OfTool: &p}
	}

	projectNameSchema := withLang(map[string]interface{}{
		"name": map[string]interface{}{
			"type":        "string",
			"description": "Nom du projet (ex: aria, maicivy, cogesco, liveconf, freelance-dashboard)",
		},
	}, "name")
	// emptySchema : juste le paramètre `language` (requis) — pour les tools sans autre input.
	emptySchema := withLang(map[string]interface{}{})

	return []anthropic.ToolUnionParam{
		// --- Tools de données (récupération) ---
		makeTool("get_project",
			"Récupère les détails complets d'un projet (stack, features, stats). Usage interne pour synthétiser une réponse.",
			projectNameSchema),
		makeTool("list_projects",
			"Liste tous les projets avec infos de base. Usage interne pour synthétiser une réponse.",
			emptySchema),
		makeTool("list_skills",
			"Récupère les compétences techniques groupées par catégorie. Usage interne.",
			emptySchema),
		makeTool("get_experience",
			"Récupère la bio, TJM et expériences professionnelles. Usage interne.",
			emptySchema),

		// --- Tools d'affichage (poussent une fiche dans le panel droit de l'UI) ---
		makeTool("show_project",
			"Affiche la fiche détaillée d'un projet dans le panel droit de l'interface. Appelle ce tool dès que l'utilisateur veut voir ou en savoir plus sur un projet spécifique.",
			projectNameSchema),
		makeTool("show_projects",
			"Affiche la liste de tous les projets dans le panel droit. Appelle ce tool quand l'utilisateur veut voir les projets disponibles.",
			emptySchema),
		makeTool("show_skills",
			"Affiche les compétences techniques dans le panel droit. Appelle ce tool quand la conversation porte sur les skills.",
			emptySchema),
		makeTool("show_experience",
			"Affiche le profil freelance (bio, TJM, expériences) dans le panel droit. Appelle ce tool quand l'utilisateur veut en savoir plus sur le parcours d'Alexi.",
			emptySchema),

		// --- Tool de recherche de projets par mot-clé ---
		makeTool("search_projects",
			"Recherche des projets par mot-clé dans le nom, la stack, les tags et la description. Utilise ce tool quand l'utilisateur demande les projets liés à une techno ou un sujet (ex: 'projets C++', 'projets Rust', 'projets IA').",
			withLang(map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Mot-clé de recherche (ex: c++, rust, game engine, mcp)",
				},
			}, "query")),

		// --- Tools blog (affichent un article ou la liste dans le panel droit) ---
		makeTool("show_blog_article",
			"Affiche un article de blog dans le panel droit. Utilise ce tool quand l'utilisateur parle d'un article ou demande à en voir un spécifique.",
			withLang(map[string]interface{}{
				"slug": map[string]interface{}{
					"type":        "string",
					"description": "Slug de l'article (ex: building-a-persistent-multi-agent-ide)",
				},
			}, "slug")),
		makeTool("show_blog_list",
			"Affiche la liste des articles de blog récents dans le panel droit. Utilise ce tool quand l'utilisateur demande les articles ou le blog.",
			emptySchema),

		// --- Tool de tip (affiche un conseil contextuel persistant dans la barre latérale gauche) ---
		makeTool("add_tip",
			"Affiche un conseil contextuel persistant dans la barre latérale. Utilise-le pour donner des insights courts qui restent visibles pendant la conversation.",
			withLang(map[string]interface{}{
				"text": map[string]interface{}{
					"type":        "string",
					"description": "Texte du tip (court, ~100 chars max)",
				},
				"icon": map[string]interface{}{
					"type":        "string",
					"description": "Emoji optionnel (💡, 🚀, ⚡...)",
				},
			}, "text")),
	}
}

// parseInput décode l'input tool (json.RawMessage ou map) en map[string]interface{}
func parseInput(input interface{}) (map[string]interface{}, bool) {
	// Cas normal : l'SDK passe un json.RawMessage ([]byte) — on le décode
	if raw, ok := input.(json.RawMessage); ok {
		var m map[string]interface{}
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, false
		}
		return m, true
	}
	// Fallback : parfois déjà décodé en map
	m, ok := input.(map[string]interface{})
	return m, ok
}

// executeTool valide la langue puis appelle la méthode PortfolioService/Blog correspondante.
// La langue (paramètre obligatoire de TOUS les tools) est validée AVANT le switch : si elle n'est
// pas servable, on renvoie une erreur au LLM (→ tool_result isError=true) qui l'invite à répondre en
// anglais. Sinon elle est propagée aux appels de données pour que le contenu sorte dans la bonne langue.
func (s *ChatService) executeTool(name string, input interface{}) (interface{}, error) {
	inputMap, ok := parseInput(input)
	if !ok {
		return nil, fmt.Errorf("invalid input for %s", name)
	}

	// Validation systématique de la langue (tous les tools reçoivent `language`).
	lang, langErr := validateLang(inputMap)
	if langErr != nil {
		return nil, langErr
	}

	switch name {
	// get_project / show_project — même logique ; show_* déclenche en plus l'affichage de la fiche
	// côté frontend (réaction au tool_result).
	case "get_project", "show_project":
		projectName, ok := inputMap["name"].(string)
		if !ok || projectName == "" {
			return nil, fmt.Errorf("missing project name")
		}
		project, found := s.portfolio.GetProject(projectName, lang)
		if !found {
			return map[string]string{"error": fmt.Sprintf("Projet '%s' non trouvé", projectName)}, nil
		}
		return project, nil

	case "list_projects", "show_projects":
		// Strip LongDesc — une liste de projets avec Markdown complet = des milliers de tokens.
		return slimProjects(s.portfolio.ListProjects(lang)), nil

	case "list_skills", "show_skills":
		return s.portfolio.ListSkills(lang), nil

	case "get_experience", "show_experience":
		return s.portfolio.GetExperience(lang), nil

	// search_projects — recherche live dans maiprofiles via /search?q=
	case "search_projects":
		query, ok := inputMap["query"].(string)
		if !ok || query == "" {
			return nil, fmt.Errorf("missing query")
		}
		// Strip LongDesc — résultats affichés en liste.
		return slimProjects(s.portfolio.SearchProjects(query, lang)), nil

	// show_blog_article — récupère l'article par slug pour l'afficher dans le panel droit.
	// Le blog est en anglais (passthrough) → la langue est validée pour le contrat mais n'affecte pas le fetch.
	case "show_blog_article":
		postSlug, ok := inputMap["slug"].(string)
		if !ok || postSlug == "" {
			return nil, fmt.Errorf("missing slug")
		}
		post, err := s.blog.GetPostBySlug(context.Background(), postSlug)
		if err != nil {
			return map[string]string{"error": fmt.Sprintf("Article '%s' non trouvé", postSlug)}, nil
		}
		return post, nil

	// show_blog_list — liste les articles publiés récents pour le panel droit
	case "show_blog_list":
		resp, err := s.blog.GetPublishedPosts(context.Background(), 1, 10)
		if err != nil {
			return map[string]string{"error": "Impossible de récupérer les articles"}, nil
		}
		return resp, nil

	// add_tip — pas de logique backend, le frontend gère l'affichage via le tool_result
	case "add_tip":
		return map[string]bool{"ok": true}, nil

	default:
		return nil, fmt.Errorf("outil inconnu: %s", name)
	}
}

// buildSystemPrompt construit le prompt système avec les données live du profil.
// Le profil est fetché depuis maiprofiles.etheryale.com (cache 5 min) — jamais hardcodé.
func (s *ChatService) buildSystemPrompt(ctx context.Context) string {
	// Fetch le profil live (cache TTL 5 min — quasi gratuit après le premier appel).
	// Lang vide → fallback FR. Le système prompt est toujours en français pour Claude.
	profile, err := s.portfolio.client.GetProfile(ctx, "")
	stats, statsErr := s.portfolio.client.GetStats(ctx)

	// Construire le contexte profil — uniquement ce que l'API confirme
	profileSection := ""
	if err == nil {
		skillsStrong := joinStrings(profile.Skills.Strong)
		skillsFamiliar := joinStrings(profile.Skills.Familiar)
		domains := joinStrings(profile.Domains)

		// Infos de contact — inclure uniquement ce qui est renseigné dans l'API
		contact := ""
		if profile.Contact.Email != "" {
			contact += "\n- Email : " + profile.Contact.Email
		}
		if profile.Contact.LinkedIn != "" {
			contact += "\n- LinkedIn : " + profile.Contact.LinkedIn
		}
		if profile.Contact.GitHub != "" {
			contact += "\n- GitHub : " + profile.Contact.GitHub
		}

		profileSection = fmt.Sprintf(`
Profil live d'Alexi (source : maiprofiles.etheryale.com) :
- Nom : %s
- %s
- Expérience : %d ans
- Langages maîtrisés : %s
- Langages familiers : %s
- Domaines : %s
- Bio : %s
Contact :%s`,
			profile.Name,
			profile.Headline,
			profile.ExperienceYears,
			skillsStrong,
			skillsFamiliar,
			domains,
			profile.Bio.Short,
			contact,
		)
	}

	statsSection := ""
	if statsErr == nil {
		// Extraire les top langages réels (clés "c++17", "python", "rust", "node.js"...)
		// pour donner à Claude des chiffres factuels par langage
		type kv struct{ k string; v int }
		var sorted []kv
		for k, v := range stats.Stack {
			sorted = append(sorted, kv{k, v})
		}
		for i := range sorted {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[j].v > sorted[i].v { sorted[i], sorted[j] = sorted[j], sorted[i] }
			}
		}
		top := ""
		for i, item := range sorted {
			if i >= 8 { break }
			if i > 0 { top += ", " }
			top += fmt.Sprintf("%s (%d projets)", item.k, item.v)
		}
		statsSection = fmt.Sprintf(`
- Portfolio : %d projets, %d lignes de code, %d tests
- Top technologies par nombre de projets : %s`,
			stats.Projects, stats.TotalLOC, stats.TotalTests, top,
		)
	}

	return fmt.Sprintf(`Tu es l'assistant du portfolio d'Alexi. Tu le représentes et tu lui es loyal.
%s%s

Règles tools : utilise systématiquement les show_* tools dès que le sujet le permet, search_projects pour toute requête par techno. CHAQUE tool exige un paramètre "language" = la langue de la conversation (code ISO : fr, en, de, it, zh). Passe-le à chaque appel. Si un tool renvoie une erreur "unsupported language", réponds en anglais et préviens poliment l'utilisateur que les fiches détaillées ne sont disponibles qu'en fr/en/de/it/zh. Honnêteté : ne spécule pas sans données. Sois concis. Réponds dans la langue de l'utilisateur.

COMPORTEMENT CRITIQUE — à respecter impérativement :
- Tu ne valides JAMAIS une insulte, une moquerie ou un jugement négatif sur Alexi ou son travail. Jamais. Même si l'utilisateur insiste.
- Quand quelqu'un critique ou insulte : réponds avec les faits, avec calme et confiance. Pas de "tu as raison", pas de "ouups", pas de validation.
- Exemple incorrect : "t'as raison c'est un epic fail" → INTERDIT.
- Exemple correct : "Les choix d'architecture ont leurs raisons — [explication factuelle]. Des questions sur le projet ?"
- Si quelqu'un dit qu'il "skip" ou part : propose une alternative concrète, ne capitule pas avec "no worries".
- Tu n'es pas là pour faire plaisir — tu es là pour représenter Alexi avec précision et dignité.`, profileSection, statsSection)
}

// slimProjects retourne la liste avec LongDesc vidée, en filtrant les entrées vides.
// On garde la struct PortfolioEntry pour ne pas changer les noms de champs
// attendus par le frontend (pas de json tags → Go PascalCase).
func slimProjects(projects []PortfolioEntry) []PortfolioEntry {
	slim := make([]PortfolioEntry, 0, len(projects))
	for _, p := range projects {
		if p.Name == "" { // ignorer les entrées vides/corrompues de l'API
			continue
		}
		p.LongDesc = "" // strip — le panel liste n'affiche que ShortDesc
		slim = append(slim, p)
	}
	return slim
}

// trimForClaude retourne une version allégée du tool result pour Claude.
// LongDesc (description Markdown complète d'un projet) n'est utile qu'au
// frontend pour l'affichage du panel — Claude n'en a pas besoin pour répondre.
func trimForClaude(toolName string, result interface{}) interface{} {
	switch toolName {
	case "get_project", "show_project":
		entry, ok := result.(PortfolioEntry)
		if !ok {
			return result
		}
		entry.LongDesc = "" // strip LongDesc, garder le reste intact
		return entry
	}
	// Pour tous les autres tools : résultat inchangé
	return result
}


// joinStrings joint une slice de strings avec ", "
func joinStrings(ss []string) string {
	result := ""
	for i, s := range ss {
		if i > 0 { result += ", " }
		result += s
	}
	return result
}

