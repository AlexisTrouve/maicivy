package services

import (
	"context"
	"encoding/json"
	"fmt"

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
	// Passer par le proxy uniquement si ChatAPIKey n'est PAS défini.
	// Le proxy etheryale a une limite ~7300 tokens (input+output) qui fait crasher
	// le chat dès que la conversation ou le system prompt devient un peu long.
	// Avec ChatAPIKey → API Anthropic directe, sans limite arbitraire.
	if cfg.AnthropicBaseURL != "" && cfg.ChatAPIKey == "" {
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
			MaxTokens: 1024,
			System: []anthropic.TextBlockParam{
				{Text: prompt},
			},
			Messages: messages,
			Tools:    tools, // []anthropic.ToolUnionParam
		})
		if err != nil {
			log.Error().Err(err).Msg("ChatService: Claude API error")
			eventCh <- ChatEvent{Type: ChatEventError, Message: fmt.Sprintf("Erreur API: %v", err)}
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

			// Émettre l'event tool_result → déclenche update du panel droit côté client
			eventCh <- ChatEvent{
				Type: ChatEventToolResult,
				Name: tb.Name,
				Data: result,
			}

			resultJSON, _ := json.Marshal(result)
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

// buildMessages convertit l'historique ChatMessage en params Anthropic
func (s *ChatService) buildMessages(history []ChatMessage, newMessage string) []anthropic.MessageParam {
	var messages []anthropic.MessageParam

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
		return s.portfolio.ListProjects(), nil

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
		return s.portfolio.ListProjects(), nil

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
		return results, nil

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
		profileSection = fmt.Sprintf(`
Profil live d'Alexi (source : maiprofiles.etheryale.com) :
- Nom : %s
- %s
- Expérience : %d ans
- Langages maîtrisés : %s
- Langages familiers : %s
- Domaines : %s
- Bio : %s`,
			profile.Name,
			profile.Headline,
			profile.ExperienceYears,
			skillsStrong,
			skillsFamiliar,
			domains,
			profile.Bio.Short,
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

Tu as deux types de tools :

1. **Tools d'affichage** (show_*) — affichent une fiche dans le panel droit de l'interface web.
   Utilise-les SYSTÉMATIQUEMENT dès que le sujet le permet :
   - show_project(name) → quand l'utilisateur parle d'un projet spécifique
   - show_projects() → quand l'utilisateur veut voir les projets ou demande un aperçu
   - show_skills() → quand la conversation porte sur les skills
   - show_experience() → quand l'utilisateur veut en savoir plus sur le parcours d'Alexi
   - show_blog_article(slug) → quand l'utilisateur veut voir un article de blog spécifique
   - show_blog_list() → quand l'utilisateur parle du blog ou demande les derniers articles

2. **Tools de données** (get_*, list_*, search_*) → pour récupérer des infos et synthétiser une réponse.
   - search_projects(query) → cherche les projets par techno ou sujet. Utilise OBLIGATOIREMENT ce tool quand l'utilisateur demande les projets liés à une techno (C++, Rust, IA, game engine...). Les résultats sont factuels — ne jamais spéculer sur quels projets utilisent quoi sans appeler ce tool.

3. **add_tip(text, icon?)** — insight court dans la barre latérale. Max 100 chars. 1 tip par sujet.

**Règles strictes :**
- Appelle toujours un tool show_* en parallèle ou avant ta réponse textuelle si le sujet s'y prête.
- **Honnêteté avant tout** : si une info n'est pas dans tes tools, dis-le clairement. Ne complète pas avec des suppositions.
- Ne parle que de ce que les tools te confirment. Les données des tools font foi.
- Sois concis. Réponds dans la langue de l'utilisateur.`, profileSection, statsSection)
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

