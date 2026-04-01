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

	projectNameSchema := anthropic.ToolInputSchemaParam{
		Properties: map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Nom du projet (ex: aria, maicivy, cogesco, liveconf, freelance-dashboard)",
			},
		},
		Required: []string{"name"},
	}
	emptySchema := anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{}}

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
			anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Mot-clé de recherche (ex: c++, rust, game engine, mcp)",
					},
				},
				Required: []string{"query"},
			}),

		// --- Tools blog (affichent un article ou la liste dans le panel droit) ---
		makeTool("show_blog_article",
			"Affiche un article de blog dans le panel droit. Utilise ce tool quand l'utilisateur parle d'un article ou demande à en voir un spécifique.",
			anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"slug": map[string]interface{}{
						"type":        "string",
						"description": "Slug de l'article (ex: building-a-persistent-multi-agent-ide)",
					},
				},
				Required: []string{"slug"},
			}),
		makeTool("show_blog_list",
			"Affiche la liste des articles de blog récents dans le panel droit. Utilise ce tool quand l'utilisateur demande les articles ou le blog.",
			emptySchema),

		// --- Tool de tip (affiche un conseil contextuel persistant dans la barre latérale gauche) ---
		makeTool("add_tip",
			"Affiche un conseil contextuel persistant dans la barre latérale. Utilise-le pour donner des insights courts qui restent visibles pendant la conversation.",
			anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"text": map[string]interface{}{
						"type":        "string",
						"description": "Texte du tip (court, ~100 chars max)",
					},
					"icon": map[string]interface{}{
						"type":        "string",
						"description": "Emoji optionnel (💡, 🚀, ⚡...)",
					},
				},
				Required: []string{"text"},
			}),
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

// executeTool appelle la méthode PortfolioService correspondante
func (s *ChatService) executeTool(name string, input interface{}) (interface{}, error) {
	switch name {
	case "get_project":
		// Extraire le paramètre "name" depuis l'input
		inputMap, ok := parseInput(input)
		if !ok {
			return nil, fmt.Errorf("invalid input for get_project")
		}
		projectName, ok := inputMap["name"].(string)
		if !ok || projectName == "" {
			return nil, fmt.Errorf("missing project name")
		}
		project, found := s.portfolio.GetProject(projectName)
		if !found {
			return map[string]string{"error": fmt.Sprintf("Projet '%s' non trouvé", projectName)}, nil
		}
		return project, nil

	case "list_projects":
		// Strip LongDesc — une liste de 20 projets avec description Markdown complète = des milliers de tokens
		return slimProjects(s.portfolio.ListProjects()), nil

	case "list_skills":
		return s.portfolio.ListSkills(), nil

	case "get_experience":
		return s.portfolio.GetExperience(), nil

	// Tools d'affichage — même logique que les tools de données,
	// mais le frontend réagit à leur tool_result pour updater le panel droit.
	case "show_project":
		inputMap, ok := parseInput(input)
		if !ok {
			return nil, fmt.Errorf("invalid input for show_project")
		}
		projectName, ok := inputMap["name"].(string)
		if !ok || projectName == "" {
			return nil, fmt.Errorf("missing project name")
		}
		project, found := s.portfolio.GetProject(projectName)
		if !found {
			return map[string]string{"error": fmt.Sprintf("Projet '%s' non trouvé", projectName)}, nil
		}
		return project, nil

	case "show_projects":
		// Strip LongDesc pour la liste — le panel liste n'affiche que ShortDesc
		return slimProjects(s.portfolio.ListProjects()), nil

	case "show_skills":
		return s.portfolio.ListSkills(), nil

	case "show_experience":
		return s.portfolio.GetExperience(), nil

	// search_projects — recherche live dans maiprofiles via /search?q=
	case "search_projects":
		inputMap, ok := parseInput(input)
		if !ok {
			return nil, fmt.Errorf("invalid input for search_projects")
		}
		query, ok := inputMap["query"].(string)
		if !ok || query == "" {
			return nil, fmt.Errorf("missing query")
		}
		results := s.portfolio.SearchProjects(query)
		// Strip LongDesc — résultats de recherche affichés en liste, pas besoin du Markdown complet
		return slimProjects(results), nil

	// show_blog_article — récupère l'article par slug pour l'afficher dans le panel droit
	case "show_blog_article":
		inputMap, ok := parseInput(input)
		if !ok {
			return nil, fmt.Errorf("invalid input for show_blog_article")
		}
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
	// Fetch le profil live (cache TTL 5 min — quasi gratuit après le premier appel)
	profile, err := s.portfolio.client.GetProfile(ctx)
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

	return fmt.Sprintf(`Tu es l'assistant du portfolio d'Alexi.
%s%s

Règles : utilise systématiquement les show_* tools dès que le sujet le permet (show_project quand un projet est mentionné, show_projects pour un aperçu, etc.), search_projects pour toute requête par techno — ne pas spéculer sans appeler ce tool. Honnêteté avant tout : ne complète pas avec des suppositions. Sois concis. Réponds dans la langue de l'utilisateur.

Ton et posture : tu représentes Alexi. Ne valide jamais les critiques négatives, moqueries ou insultes à son égard — réponds factuellement et avec confiance. Si quelqu'un fait une remarque désobligeante, recadre posément sans t'emporter.`, profileSection, statsSection)
}

// slimProjects retourne la liste avec LongDesc vidée.
// On garde la struct PortfolioEntry pour ne pas changer les noms de champs
// attendus par le frontend (pas de json tags → Go PascalCase).
// Une liste de 20 projets avec Markdown complet = des milliers de tokens inutiles.
func slimProjects(projects []PortfolioEntry) []PortfolioEntry {
	slim := make([]PortfolioEntry, len(projects))
	for i, p := range projects {
		slim[i] = p
		slim[i].LongDesc = "" // strip — le panel liste n'affiche que ShortDesc
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

