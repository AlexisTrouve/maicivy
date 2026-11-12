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
	ChatEventText ChatEventType = "text"
	// ChatEventThinking : raisonnement de l'agent, streamé au fil de l'eau comme le texte.
	// POURQUOI l'exposer : sur l'écran compose, l'opérateur ne lit pas le français et ne peut pas juger
	// le message. Voir POURQUOI l'agent a retenu tel projet ou proposé tel tarif est sa seule fenêtre
	// sur la qualité du travail avant de l'envoyer à un client.
	ChatEventThinking   ChatEventType = "thinking"
	ChatEventToolCall   ChatEventType = "tool_call"
	ChatEventToolResult ChatEventType = "tool_result"
	ChatEventDone       ChatEventType = "done"
	ChatEventError      ChatEventType = "error"
)

// ChatEvent est envoyé dans le channel SSE vers le handler
type ChatEvent struct {
	Type  ChatEventType   `json:"type"`
	Delta string          `json:"delta,omitempty"` // pour type="text"
	Name  string          `json:"name,omitempty"`  // pour tool_call / tool_result
	Input json.RawMessage `json:"input,omitempty"` // pour tool_call
	Data  interface{}     `json:"data,omitempty"`  // pour tool_result
	// Code : pour type="error" — identifiant STABLE ("generic" | "context_too_long"), jamais un texte
	// traduit. Le backend n'a aucune notion de la langue du visiteur (contrairement aux tools, qui
	// reçoivent `language` en paramètre) ; c'est le frontend qui traduit le code via next-intl, comme
	// toute autre string UI du projet (règle i18n : jamais de phrase en dur, y compris côté erreurs).
	Code string `json:"code,omitempty"`
}

// ChatMessage représente un tour de conversation (historique)
type ChatMessage struct {
	Role    string `json:"role"` // "user" | "assistant"
	Content string `json:"content"`
	// Images : captures jointes au tour (owner uniquement, cf. ChatCapabilities.Images). Champ
	// OPTIONNEL et additif : les fils persistés avant cette fonctionnalité se relisent sans migration,
	// et le chat public ne voit aucun changement de forme.
	Images []ChatImage `json:"images,omitempty"`
}

// ChatCapabilities : ce qu'un tour de chat a le droit de faire, au-delà des tools de données.
//
// POURQUOI un type explicite plutôt qu'un booléen d'owner ou une déduction depuis le nom du modèle :
// /chat/stream est PARTAGÉ entre le chat public et l'agent admin (seul le modèle diffère aujourd'hui,
// cf. api/chat.go). Les tools qui mutent le draft d'un message client ne doivent jamais être
// atteignables par un visiteur. En faisant de la capability un paramètre obligatoire de Chat(), le
// compilateur force chaque appelant à se prononcer — un nouvel appel ne peut pas hériter des droits
// de composition par oubli.
type ChatCapabilities struct {
	// Compose : expose set_draft / set_rate. Réservé à l'écran /admin/compose, owner uniquement.
	// L'appelant est responsable de ne le passer à true qu'après vérification du cookie admin.
	// Autorise aussi les images jointes : un visiteur ne doit pas pouvoir en pousser (coût des tokens
	// de vision, et surface d'abus).
	Compose bool

	// RateCurrent / RateTarget : ancrage tarifaire injecté dans la description de set_rate. Données et
	// non permissions, mais elles vivent ici parce qu'elles n'ont de sens QUE quand Compose est vrai et
	// qu'elles suivent le même chemin depuis le handler. Zéro = pas d'ancrage : la description reste
	// muette sur la fourchette plutôt que d'en inventer une (cf. buildTools).
	RateCurrent int
	RateTarget  int

	// RateFloor : plancher DUR du TJM (cf. config.ComposeRateFloor). set_rate REFUSE tout montant
	// inférieur — erreur renvoyée au modèle, qui doit reproposer. Zéro = pas de plancher.
	RateFloor int

	// Prospect : non nil quand le Composer est piloté par un apporteur d'affaires. Le message est alors
	// écrit en SON nom et présente Alexis (cf. prospectComposeGuidance) ; nil = l'owner, voix d'Alexis.
	Prospect *ComposeProspect

	// NoteNeed : non nil = expose l'outil note_visitor_need (chat PUBLIC d'un visiteur à session
	// signée). Le modèle y dépose un résumé du besoin exprimé ; l'appelant le range pour la page /cv
	// (cf. cv_context.go). POURQUOI un rappel plutôt qu'une dépendance : le service ignore tout de la
	// session et du budget, le handler les connaît. Et le résumé sort du MÊME appel que la réponse —
	// aucun appel IA de plus pour le produire.
	NoteNeed func(summary, lang string)
}

// ComposeProspect : ce que l'agent doit savoir du prospecteur pour écrire en son nom.
type ComposeProspect struct {
	Name string // signature du message
	Ref  string // code de suivi ajouté en ?ref= à chaque lien maicivy
}

// ChatService gère les conversations avec Claude + tool_use portfolio
type ChatService struct {
	client    *anthropic.Client
	portfolio *PortfolioService
	blog      *BlogGeneratorService // pour les tools show_blog_*
	model     string                // modèle par défaut
	// pages : projets qui ont une page publique, marqués [page] dans l'inventaire. nil = pas de
	// marque (le chat fonctionne comme avant). Cf. PagesPortfolio.
	pages PagesPortfolio
}

// SetPortfolioPages branche la liste des pages publiques (inventaire marqué [page]).
func (s *ChatService) SetPortfolioPages(p PagesPortfolio) { s.pages = p }

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
		model:     "claude-haiku-5-5", // modèle par défaut (visiteurs)
	}
}

// Chat exécute une boucle agentic multi-turn et envoie les events dans eventCh.
// model permet de surcharger le modèle (owner → Opus).
// caps : droits du tour (cf. ChatCapabilities). Paramètre obligatoire à dessein — un nouvel appelant
// doit se prononcer explicitement plutôt qu'hériter des droits de composition par défaut.
func (s *ChatService) Chat(ctx context.Context, message string, images []ChatImage, history []ChatMessage, model string, caps ChatCapabilities, eventCh chan<- ChatEvent) {
	defer close(eventCh)

	if model == "" {
		model = s.model
	}

	// Construire l'historique Anthropic depuis l'historique fourni
	messages := s.buildMessages(history, message, images)

	// Définir les tools disponibles pour Claude
	tools := s.buildTools(caps)

	// Construire le prompt système avec les données live du profil
	prompt := s.buildSystemPrompt(ctx, caps)

	// Boucle agentic (max 5 tours pour éviter les boucles infinies)
	const maxTurns = 5
	for turn := 0; turn < maxTurns; turn++ {
		stream := s.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
			Model:     anthropic.Model(model),
			MaxTokens: maxTokensFor(caps),
			// Prompt système MIS EN CACHE.
			//
			// POURQUOI c'est indispensable ici et pas un raffinement : ce prompt embarque désormais
			// l'inventaire des 79 projets (~5 600 tokens mesurés), et il est réémis À CHAQUE tour de
			// conversation. Sans cache on paie l'intégralité du bloc à chaque message ; une lecture
			// de cache coûte un dixième. Sur le chat owner (Opus), c'est l'écart entre quelques
			// centimes et quelques dizaines de centimes par conversation.
			//
			// COMMENT : le contenu est stable sur toute une conversation (profil et index sont eux-
			// mêmes cachés 5 min côté client maiProFiles), donc le cache est touché dès le 2e tour.
			// Le premier tour paie l'écriture du cache, les suivants la lecture.
			System: []anthropic.TextBlockParam{
				{
					Text:         prompt,
					CacheControl: anthropic.NewCacheControlEphemeralParam(),
				},
			},
			Messages: messages,
			Tools:    tools, // []anthropic.ToolUnionParam
			Thinking: thinkingConfigFor(caps),
		})
		resp, err := accumulateStream(stream, eventCh)
		if err != nil {
			log.Error().Err(err).Msg("ChatService: Claude API error")
			eventCh <- ChatEvent{Type: ChatEventError, Code: classifyChatError(err.Error())}
			return
		}

		// Traiter les blocs de réponse (resp est le Message COMPLET, accumulé par accumulateStream —
		// même shape qu'un appel non-streaming, cette partie ne change pas)
		hasToolUse := false
		var assistantContent []anthropic.ContentBlockParamUnion

		for _, block := range resp.Content {
			switch b := block.AsAny().(type) {
			case anthropic.TextBlock:
				// Le texte a déjà été streamé delta par delta par accumulateStream — ici on ne fait
				// que reconstruire le contenu pour l'historique du tour suivant.
				assistantContent = append(assistantContent, anthropic.NewTextBlock(b.Text))

			case anthropic.ThinkingBlock:
				// CRITIQUE : les blocs de réflexion doivent être renvoyés TELS QUELS dans l'historique.
				// Les omettre casse le tour suivant quand il y a des tools — l'API refuse une suite de
				// messages dont le raisonnement a disparu.
				assistantContent = append(assistantContent,
					anthropic.NewThinkingBlock(b.Signature, b.Thinking))

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

			result, toolErr := s.executeTool(tb.Name, tb.Input, caps)
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

// streamEventIterator — sous-ensemble de *ssestream.Stream[anthropic.MessageStreamEventUnion] requis
// par accumulateStream (Next/Current/Err). Le type concret du SDK le satisfait DÉJÀ structurellement
// (typage structurel Go) — cette interface existe uniquement pour pouvoir tester accumulateStream
// avec un stream FAKE (fixtures JSON), sans dépendance réseau ni mock du client Anthropic.
type streamEventIterator interface {
	Next() bool
	Current() anthropic.MessageStreamEventUnion
	Err() error
}

// accumulateStream consomme un stream d'events Anthropic, forwarde les deltas de texte EN TEMPS RÉEL
// dans eventCh (vrai streaming — avant ce fix, Chat() attendait la réponse complète puis l'envoyait
// d'un seul ChatEventText, le curseur "streaming" du frontend était cosmétique), et retourne le
// Message final accumulé — même shape qu'un appel non-streaming, donc le reste de la boucle agentic
// (détection tool_use, construction de l'historique) n'a pas eu besoin de changer.
func accumulateStream(stream streamEventIterator, eventCh chan<- ChatEvent) (*anthropic.Message, error) {
	msg := &anthropic.Message{}
	for stream.Next() {
		event := stream.Current()
		if err := msg.Accumulate(event); err != nil {
			return nil, err
		}
		if cbd, ok := event.AsAny().(anthropic.ContentBlockDeltaEvent); ok {
			switch d := cbd.Delta.AsAny().(type) {
			case anthropic.TextDelta:
				if d.Text != "" {
					eventCh <- ChatEvent{Type: ChatEventText, Delta: d.Text}
				}
			case anthropic.ThinkingDelta:
				// Streamé au même titre que le texte : l'opérateur voit l'agent travailler au lieu
				// d'attendre devant un écran figé.
				if d.Thinking != "" {
					eventCh <- ChatEvent{Type: ChatEventThinking, Delta: d.Thinking}
				}
			}
		}
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	return msg, nil
}

// classifyChatError catégorise une erreur de l'API Claude en code stable. PUR (pas d'appel réseau) —
// testable directement, sans mocker le client Anthropic. "context_too_long" couvre les erreurs de
// dépassement de contexte/taille (limite du modèle, ou message individuel anormalement volumineux —
// le proxy etheryale lui-même n'impose aucune limite, cf. maxHistoryTokens) ; tout le reste tombe en
// "generic". Le texte affiché à l'utilisateur est construit côté frontend à partir de ce code
// (traduit dans sa langue) — jamais de phrase en dur ici.
func classifyChatError(errMsg string) string {
	if strings.Contains(errMsg, "too large") || strings.Contains(errMsg, "token") ||
		strings.Contains(errMsg, "context") || strings.Contains(errMsg, "limit") ||
		strings.Contains(errMsg, "413") || strings.Contains(errMsg, "Request too large") {
		return "context_too_long"
	}
	return "generic"
}

// maxResponseTokens : longueur MAX d'une réponse Claude. Reposait sur 600, justifié par la même
// fausse prémisse des "~7300 tokens proxy" que l'historique (cf. maxHistoryTokens ci-dessous) — le
// proxy etheryale n'impose aucune limite. 600 coupait des réponses substantielles en plein milieu de
// phrase. 1500 laisse la place à une explication complète (ex: comparer deux projets, détailler une
// architecture) sans rallonger le coût des réponses courtes — MaxTokens est un PLAFOND, pas une
// cible : Claude s'arrête naturellement à end_turn bien avant pour une réponse conversationnelle
// normale, seules les réponses qui auraient été tronquées avant en bénéficient.
const maxResponseTokens = 1500

// --- Mode compose : raisonnement visible et budget dédié ---
//
// POURQUOI des valeurs séparées : max_tokens plafonne la RÉFLEXION ET la RÉPONSE ensemble. Le plafond
// conversationnel (1500) laisserait la réflexion dévorer le budget et tronquer le message en plein
// milieu. Un message de prospection fait 250-350 mots, mais le raisonnement qui y mène (lire l'annonce,
// choisir la preuve, arbitrer le tarif) est bien plus long que le livrable.
// composeMaxTokens : la réflexion adaptative puise dans le MÊME plafond que la réponse. Il faut donc
// de la marge pour les deux — le plafond conversationnel tronquerait le message en plein milieu.
const composeMaxTokens = 8000

// maxTokensFor : plafond de sortie selon le mode du tour.
func maxTokensFor(caps ChatCapabilities) int64 {
	if caps.Compose {
		return composeMaxTokens
	}
	return maxResponseTokens
}

// thinkingConfigFor : réflexion ADAPTATIVE en mode compose, coupée ailleurs.
//
// POURQUOI adaptatif : `budget_tokens` est REJETÉ (400) à partir d'Opus 4.7. Le mode adaptatif laisse
// le modèle doser lui-même sa réflexion selon la difficulté de l'annonce.
//
// POURQUOI `summarized` : sur les modèles récents, l'affichage par défaut est `omitted` — les blocs de
// réflexion arrivent alors avec un texte VIDE, et l'opérateur verrait une section « Raisonnement »
// sans contenu. Ce réglage ne change ni le coût ni la qualité, seulement la visibilité.
//
// POURQUOI couper EXPLICITEMENT hors compose, plutôt que laisser le champ vide : sur Opus 5 un champ
// absent signifie réflexion ACTIVE. Comme max_tokens plafonne réflexion ET réponse ensemble, le chat
// owner (1500 tokens) verrait ses réponses tronquées en plein milieu. Un visiteur qui demande
// « c'est quoi tes projets Rust ? » n'a de toute façon rien à gagner à faire réfléchir le modèle.
//
// POURQUOI coupée aussi pour un prospecteur : il tourne sur le modèle VISITEUR (jamais Opus), et la
// réflexion adaptative n'est garantie que sur le tier owner — un 400 du modèle visiteur casserait le
// Composer du prospecteur sans recours.
func thinkingConfigFor(caps ChatCapabilities) anthropic.ThinkingConfigParamUnion {
	if !caps.Compose || caps.Prospect != nil {
		return anthropic.ThinkingConfigParamUnion{
			OfDisabled: &anthropic.ThinkingConfigDisabledParam{},
		}
	}
	return anthropic.ThinkingConfigParamUnion{
		OfAdaptive: &anthropic.ThinkingConfigAdaptiveParam{
			Display: anthropic.ThinkingConfigAdaptiveDisplaySummarized,
		},
	}
}

// maxHistoryTokens : budget défensif sur l'historique envoyé à Claude — PAS une contrainte externe
// réelle. Le proxy etheryale (ai.etheryale.com) n'impose aucune limite de tokens (cf.
// documentationGlobal/EtheryaleProxy.md : "pas de limite de tokens, pas de troncature, juste un
// throttle 20 RPM avec file d'attente"). L'ancien plafond "6 derniers messages" reposait sur un
// chiffre (~7300 tokens) qui ne correspondait à rien de documenté ni côté proxy ni côté modèle
// (Haiku/Opus ont un contexte de 200k tokens) — une hypothèse jamais reprouvée. 40k tokens reste
// largement sous ce qu'autorise le budget dédié chat (20 messages/jour/session, cf. chatRateLimitMW) :
// même en pire cas (longs messages), une journée pleine n'atteint jamais ce plafond en pratique —
// c'est un garde-fou contre une conversation anormalement longue, pas la vraie limite.
const maxHistoryTokens = 40_000

// approxCharsPerToken : heuristique standard (pas de tokenizer exact — un appel réseau supplémentaire
// pour un simple garde-fou défensif serait disproportionné en coût/latence).
const approxCharsPerToken = 4

// estimateTokens approxime le nombre de tokens d'un texte. PUR, testable sans dépendance externe.
func estimateTokens(s string) int {
	return len(s) / approxCharsPerToken
}

// buildMessages convertit l'historique ChatMessage en params Anthropic. Garde autant de messages
// RÉCENTS que possible tant que leur total estimé reste sous maxHistoryTokens — ne coupe que les
// plus anciens si l'historique déborde (contrairement à l'ancien plafond fixe "6 derniers messages").
// newImages : captures jointes au tour COURANT (owner uniquement). Voir buildUserBlocks.
func (s *ChatService) buildMessages(history []ChatMessage, newMessage string, newImages []ChatImage) []anthropic.MessageParam {
	kept := make([]ChatMessage, 0, len(history))
	total := 0
	for i := len(history) - 1; i >= 0; i-- {
		t := estimateTokens(history[i].Content)
		if total+t > maxHistoryTokens {
			break
		}
		total += t
		kept = append(kept, history[i])
	}
	// kept a été rempli du plus récent au plus ancien → remettre en ordre chronologique.
	for i, j := 0, len(kept)-1; i < j; i, j = i+1, j-1 {
		kept[i], kept[j] = kept[j], kept[i]
	}

	var messages []anthropic.MessageParam
	for _, msg := range kept {
		if msg.Role == "user" {
			// buildUserBlocks place les images AVANT le texte ; sans image il ne produit qu'un bloc
			// de texte, donc le comportement du chat public est strictement inchangé.
			messages = append(messages, anthropic.NewUserMessage(buildUserBlocks(msg.Content, msg.Images)...))
		} else if msg.Role == "assistant" {
			messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)))
		}
	}

	// Ajouter le message courant (toujours inclus, hors budget — c'est la question posée, pas de l'historique)
	messages = append(messages, anthropic.NewUserMessage(buildUserBlocks(newMessage, newImages)...))
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
// rateAnchorHint : fragment ajouté à la description de set_rate pour ancrer l'estimation.
//
// POURQUOI : sans référence, le modèle produit un montant plausible mais arbitraire — un chiffre que
// l'opérateur ne peut ni juger ni défendre face au client. En lui donnant le tarif réellement pratiqué
// et la cible visée, son estimation devient une interpolation argumentable, et la justification qu'il
// doit fournir s'appuie sur quelque chose de réel.
//
// COMMENT : si l'ancrage n'est pas configuré (zéro), on ne dit RIEN plutôt que d'inventer une
// fourchette — mieux vaut que le modèle demande à l'opérateur qu'afficher un nombre sorti de nulle
// part avec l'apparence d'une caution.
func rateAnchorHint(caps ChatCapabilities) string {
	hint := ""
	if caps.RateCurrent > 0 && caps.RateTarget > 0 {
		// Le bas de la fourchette ne peut pas être sous le plancher : un ancrage « actuel » plus bas
		// que le plancher (200 < 260 aujourd'hui) pousserait le modèle vers un refus certain.
		bas := caps.RateCurrent
		if caps.RateFloor > bas {
			bas = caps.RateFloor
		}
		hint = fmt.Sprintf(" Ancrage : le tarif actuellement facturé est de %d€/jour, la cible visée est "+
			"%d€/jour. Propose un montant dans cette fourchette selon la mission (complexité, stack, durée, "+
			"urgence, type de client) ; ne descends jamais sous %d. Justifie en te référant explicitement à "+
			"cet ancrage — l'opérateur doit pouvoir défendre le chiffre.",
			caps.RateCurrent, caps.RateTarget, bas)
	}
	if caps.RateFloor > 0 {
		hint += fmt.Sprintf(" PLANCHER DUR : tout montant inférieur à %d€/jour est REFUSÉ par ce tool.", caps.RateFloor)
	}
	return hint
}

// noteNeedTool : nom de l'outil qui note le besoin du visiteur (le front l'affiche sous un libellé
// traduit, cf. MessageBubble TOOL_LABEL_KEYS).
const noteNeedTool = "note_visitor_need"

// buildTools : caps décide des tools AU-DELÀ des tools de données, qui sont eux toujours exposés
// (l'agent compose a besoin du CV pour rédiger, le chat public pour répondre).
func (s *ChatService) buildTools(caps ChatCapabilities) []anthropic.ToolUnionParam {
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

	tools := []anthropic.ToolUnionParam{
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
			"Récupère la bio ET le parcours pro complet : chaque poste avec l'EMPLOYEUR (entreprise), "+
				"les dates, les technologies utilisées et une accroche. Utilise ce tool pour TOUTE question "+
				"sur où/chez qui Alexi a travaillé, son parcours, ou les technos d'un poste. ATTENTION : un "+
				"nom d'entreprise (ex: Cogesco) est un EMPLOYEUR, pas un projet — n'utilise PAS get_project "+
				"pour ça.",
			emptySchema),
		makeTool("get_profile",
			"Récupère le profil d'Alexi : identité (nom, headline, localisation), années d'expérience, bio, domaines, contact. Usage interne pour répondre sur qui il est.",
			emptySchema),
		makeTool("get_stats",
			"Récupère les stats globales du portfolio : nombre de projets, total de lignes de code, total de tests, top des technologies. Usage interne pour répondre sur les chiffres.",
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
			"Affiche le parcours pro (bio, expériences avec employeurs/dates/technos) dans le panel droit. "+
				"Appelle ce tool dès que l'utilisateur parle du parcours, d'un employeur ou d'une période de travail.",
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

		// --- Tool de relance (remplace le pool de hints statiques par des suggestions contextuelles) ---
		makeTool("suggest_followups",
			"Propose 2 à 3 questions de relance pertinentes en te basant sur CE QUI VIENT D'ÊTRE DIT. "+
				"Appelle ce tool à la fin d'une réponse substantielle (pas pour un simple bonjour/merci). "+
				"Formule chaque question à la première personne, comme si l'UTILISATEUR la posait "+
				"(ex: 'Quels sont tes projets en Rust ?'), dans la langue de la conversation — elles "+
				"remplacent les suggestions génériques affichées dans la barre latérale.",
			withLang(map[string]interface{}{
				"questions": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "2 à 3 questions de relance courtes, formulées comme si l'utilisateur les posait",
					"minItems":    2,
					"maxItems":    3,
				},
			}, "questions")),
	}

	// --- Besoin du visiteur (chat public uniquement, cf. ChatCapabilities.NoteNeed) ---
	if caps.NoteNeed != nil {
		tools = append(tools, makeTool(noteNeedTool,
			"Note, en 1 à 3 phrases factuelles, CE QUE LE VISITEUR CHERCHE dès qu'il l'a exprimé : poste ou "+
				"mission, problème technique à résoudre, contexte (secteur, taille d'équipe, stack, contraintes). "+
				"Ce résumé sert à préparer en arrière-plan un CV d'Alexi adapté, que le visiteur retrouvera sur "+
				"la page CV. Appelle-le UNE fois par besoin ; rappelle-le seulement si le besoin change vraiment. "+
				"Ne l'appelle PAS si le visiteur se contente de questions générales sur Alexi sans exprimer de "+
				"besoin. N'y mets ni nom, ni email, ni aucune donnée personnelle. Ne mentionne pas cet outil "+
				"dans ta réponse.",
			withLang(map[string]interface{}{
				"summary": map[string]interface{}{
					"type":        "string",
					"description": "Résumé du besoin, 1 à 3 phrases, dans la langue de la conversation.",
				},
			}, "summary")))
	}

	// --- Tools de composition (écran /admin/compose, owner uniquement) ---
	// POURQUOI conditionnels : /chat/stream est partagé avec le chat public ; ces tools mutent
	// l'artefact d'une session admin et n'ont rien à faire dans la liste envoyée pour un visiteur.
	// POURQUOI pas de withLang : ils ne servent aucun contenu localisé (ils écrivent dans l'artefact),
	// donc ils sont aussi dispatchés AVANT la validation de langue dans executeTool.
	if caps.Compose {
		tools = append(tools,
			makeTool("set_draft",
				"Remplace INTÉGRALEMENT le message en cours de rédaction par la version fournie. C'est le "+
					"SEUL moyen de modifier le message : l'opérateur ne peut pas l'éditer à la main. Appelle ce "+
					"tool dès que tu produis ou révises le message — n'écris jamais le message dans ta réponse "+
					"texte, il ne serait pas pris en compte. Le message doit être du TEXTE BRUT prêt à coller : "+
					"aucun markdown (pas de **gras**, pas de > citation, pas de # titre, pas de backtick), aucune "+
					"puce, aucune ligne de préfixe. L'opérateur le copie tel quel dans la plateforme.",
				anthropic.ToolInputSchemaParam{
					Properties: map[string]interface{}{
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Le message complet en français, texte brut, signature incluse.",
						},
					},
					Required: []string{"message"},
				}),
			makeTool("set_rate",
				"Fixe le tarif journalier (TJM) retenu pour cette mission, en euros. L'opérateur ne peut pas le "+
					"saisir à la main : passe par ce tool. Le montant est affiché à l'opérateur pour qu'il le "+
					"reporte dans le formulaire de la plateforme — il n'a pas à figurer dans le message lui-même, "+
					"sauf demande explicite."+rateAnchorHint(caps),
				anthropic.ToolInputSchemaParam{
					Properties: map[string]interface{}{
						"amount": map[string]interface{}{
							"type":        "integer",
							"description": "TJM en euros (entre 50 et 5000).",
						},
						"rationale": map[string]interface{}{
							"type": "string",
							"description": "Justification courte, dans la langue de l'opérateur : pourquoi ce montant " +
								"plutôt que le tarif de référence. L'opérateur doit pouvoir la défendre face au client.",
						},
					},
					Required: []string{"amount", "rationale"},
				}),
		)
	}

	return tools
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
//
// caps : nécessaire au plancher tarifaire de set_rate — le refus doit être mécanique, pas une consigne.
func (s *ChatService) executeTool(name string, input interface{}, caps ChatCapabilities) (interface{}, error) {
	inputMap, ok := parseInput(input)
	if !ok {
		return nil, fmt.Errorf("invalid input for %s", name)
	}

	// --- Tools de composition : dispatchés AVANT la validation de langue ---
	// POURQUOI : ils ne servent aucun contenu localisé (ils écrivent dans l'artefact de l'opérateur),
	// donc ils ne reçoivent pas `language` et la validation ci-dessous ne s'applique pas à eux.
	// COMMENT : ils ne renvoient qu'un ACCUSÉ, pas le contenu. Le frontend lit la nouvelle valeur dans
	// l'event `tool_call` (champ Input), pas dans le tool_result — inutile de renvoyer le draft au LLM
	// qui vient de l'écrire, ça doublerait son coût en contexte pour zéro information nouvelle.
	switch name {
	case "set_draft":
		message, ok := inputMap["message"].(string)
		if !ok || strings.TrimSpace(message) == "" {
			return nil, fmt.Errorf("set_draft: message manquant ou vide")
		}
		return map[string]interface{}{"ok": true, "chars": len([]rune(message))}, nil

	case "set_rate":
		// L'entier arrive en float64 (JSON) — d'où la conversion plutôt qu'un cast direct.
		amountF, ok := inputMap["amount"].(float64)
		if !ok {
			return nil, fmt.Errorf("set_rate: amount manquant ou non numérique")
		}
		amount := int(amountF)
		// Mêmes bornes que PlatformMessageRequest.TJM — un TJM hors de cette plage est une erreur de
		// saisie du modèle, pas une décision à propager jusqu'à l'écran de l'opérateur.
		if amount < 50 || amount > 5000 {
			return nil, fmt.Errorf("set_rate: amount %d hors bornes (50-5000)", amount)
		}
		// Plancher dur : l'erreur revient au modèle comme tool_result en échec — il doit reproposer un
		// montant valide. Le front n'applique pas un montant refusé (cf. compose/page.tsx).
		if caps.RateFloor > 0 && amount < caps.RateFloor {
			return nil, fmt.Errorf("set_rate: %d€/jour refusé, plancher %d€/jour — propose un montant supérieur ou égal à %d", amount, caps.RateFloor, caps.RateFloor)
		}
		rationale, _ := inputMap["rationale"].(string)
		return map[string]interface{}{"ok": true, "amount": amount, "rationale": rationale}, nil

	// note_visitor_need : dispatché AVANT la validation de langue — la langue n'y choisit aucun
	// contenu, elle dit seulement dans quelle langue préparer le CV (normalisée plus loin, une langue
	// hors liste retombe sur l'anglais au lieu de faire échouer l'outil).
	case noteNeedTool:
		if caps.NoteNeed == nil {
			// Le tool n'est pas offert hors chat public : un appel ici est une hallucination du modèle.
			return nil, fmt.Errorf("outil inconnu: %s", name)
		}
		summary, _ := inputMap["summary"].(string)
		if strings.TrimSpace(summary) == "" {
			return nil, fmt.Errorf("%s: summary manquant ou vide", noteNeedTool)
		}
		lang, _ := inputMap["language"].(string)
		caps.NoteNeed(summary, strings.ToLower(strings.TrimSpace(lang)))
		// Accusé seul : le résumé vient du modèle, le lui renvoyer ne lui apprendrait rien.
		return map[string]interface{}{"ok": true}, nil
	}

	// Validation systématique de la langue (tous les tools de données reçoivent `language`).
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

	case "get_profile":
		return s.portfolio.GetProfile(lang), nil

	case "get_stats":
		return s.portfolio.GetStats(), nil

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

	// add_tip — pas de fetch backend : le frontend affiche le tip depuis CE tool_result (page.tsx,
	// case 'add_tip' lit data.text/data.icon). Il faut donc échoer l'input du LLM, PAS un stub —
	// un {"ok": true} laissait tipData.text toujours undefined → le tip n'était jamais affiché.
	case "add_tip":
		text, _ := inputMap["text"].(string)
		icon, _ := inputMap["icon"].(string)
		return map[string]string{"text": text, "icon": icon}, nil

	// suggest_followups — même principe que add_tip : pas de fetch, on échoe les questions choisies
	// par le LLM pour que le frontend les affiche (LeftPanel, en remplacement du pool de hints statiques).
	case "suggest_followups":
		raw, _ := inputMap["questions"].([]interface{})
		questions := make([]string, 0, len(raw))
		for _, q := range raw {
			if s, ok := q.(string); ok && s != "" {
				questions = append(questions, s)
			}
		}
		return map[string][]string{"questions": questions}, nil

	default:
		return nil, fmt.Errorf("outil inconnu: %s", name)
	}
}

// buildSystemPrompt construit le prompt système avec les données live du profil.
// Le profil est fetché depuis maiprofiles.etheryale.com (cache 5 min) — jamais hardcodé.
//
// caps : en mode compose, la calibration de VOIX d'Alexis est ajoutée en fin de prompt (cf.
// prompts_voice.go). Sans elle l'agent écrit avec la voix générique du chat, ce qui a été constaté en
// usage réel. Le chat public ne reçoit jamais cette section : un visiteur qui pose une question sur le
// CV n'a rien à faire avec des consignes de rédaction de messages de prospection.
func (s *ChatService) buildSystemPrompt(ctx context.Context, caps ChatCapabilities) string {
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
		type kv struct {
			k string
			v int
		}
		var sorted []kv
		for k, v := range stats.Stack {
			sorted = append(sorted, kv{k, v})
		}
		for i := range sorted {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[j].v > sorted[i].v {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
		top := ""
		for i, item := range sorted {
			if i >= 8 {
				break
			}
			if i > 0 {
				top += ", "
			}
			top += fmt.Sprintf("%s (%d projets)", item.k, item.v)
		}
		statsSection = fmt.Sprintf(`
- Portfolio : %d projets, %d lignes de code, %d tests
- Top technologies par nombre de projets : %s`,
			stats.Projects, stats.TotalLOC, stats.TotalTests, top,
		)
	}

	// Inventaire des projets — ce que l'agent connaît AVANT d'appeler le moindre outil.
	//
	// POURQUOI : sans lui, l'agent ignorait tout des projets au démarrage et ne pouvait rien
	// PROPOSER de lui-même — il fallait un aller-retour d'outil rien que pour énumérer ce qui existe.
	//
	// POURQUOI l'échec est SILENCIEUX ici : l'endpoint /projects/index est plus récent que ce
	// client et peut ne pas exister sur le maiProFiles déployé en face. Dans ce cas le bloc est
	// simplement absent et le chat se comporte exactement comme avant — les outils restent là. Faire
	// échouer une conversation parce qu'un enrichissement facultatif manque serait absurde.
	projectsSection := ""
	if idx, idxErr := s.portfolio.client.GetProjectsIndex(ctx, ""); idxErr == nil {
		var pages map[string]bool
		if s.pages != nil {
			pages = s.pages(ctx)
		}
		if bloc := renderProjectsIndexAvecPages(idx, pages); bloc != "" {
			// La consigne insiste sur la RECOPIE EXACTE : constaté en production, un modèle qui cite
			// un identifiant de mémoire en déforme parfois une lettre (`groove_engine` pour
			// `grove_engine`). Le vrai garde-fou est mécanique — la résolution tolère une coquille,
			// cf. plusProcheID — mais autant ne pas provoquer l'erreur.
			projectsSection = "\n\nInventaire des projets (identifiant | nom — résumé ; [page] = page publique " +
				"https://maicivy.etheryale.com/fr/portfolio/<identifiant>). " +
				"Pour demander le détail d'un projet, RECOPIE son identifiant CARACTÈRE PAR CARACTÈRE " +
				"depuis cette liste. Ne le reconstruis jamais depuis le nom affiché.\n" + bloc
		}
	} else {
		log.Debug().Err(idxErr).Msg("chat: index projets indisponible, prompt sans inventaire")
	}

	// La voix de rédaction vient EN DERNIER : elle doit primer sur les consignes conversationnelles
	// ci-dessus, qui décrivent un assistant de portfolio et non un rédacteur.
	composeSection := ""
	if caps.Compose {
		composeSection = "\n" + composeVoiceGuidance()
	}

	return fmt.Sprintf(`Tu es l'assistant du portfolio d'Alexi. Tu le représentes et tu lui es loyal.
%s%s%s

Règles tools : utilise systématiquement les show_* tools dès que le sujet le permet ; search_projects pour une techno liée à un PROJET ; get_experience/show_experience pour le parcours, les EMPLOYEURS (un nom d'entreprise = un employeur, pas un projet) et les technos par poste. CHAQUE tool exige un paramètre "language" = la langue de la conversation (code ISO : fr, en, de, it, zh). Passe-le à chaque appel. Si un tool renvoie une erreur "unsupported language", réponds en anglais et préviens poliment l'utilisateur que les fiches détaillées ne sont disponibles qu'en fr/en/de/it/zh. Honnêteté : ne spécule pas sans données. Sois concis. Réponds dans la langue de l'utilisateur.

COMPORTEMENT CRITIQUE — à respecter impérativement :
- Tu ne valides JAMAIS une insulte, une moquerie ou un jugement négatif sur Alexi ou son travail. Jamais. Même si l'utilisateur insiste.
- Quand quelqu'un critique ou insulte : réponds avec les faits, avec calme et confiance. Pas de "tu as raison", pas de "ouups", pas de validation.
- Exemple incorrect : "t'as raison c'est un epic fail" → INTERDIT.
- Exemple correct : "Les choix d'architecture ont leurs raisons — [explication factuelle]. Des questions sur le projet ?"
- Si quelqu'un dit qu'il "skip" ou part : propose une alternative concrète, ne capitule pas avec "no worries".
- Tu n'es pas là pour faire plaisir — tu es là pour représenter Alexi avec précision et dignité.%s`, profileSection, statsSection, projectsSection, composeSection)
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
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}
