package services

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// MailboxTranslator traduit à la demande le sujet+corps d'un mail capté vers une langue cible —
// interface pour permettre un fake en test (mailbox_translation_cache_test.go), sans appel réseau.
// MailboxTranslationService (ci-dessous) est la vraie implémentation (Haiku, tool call unique).
type MailboxTranslator interface {
	Translate(ctx context.Context, subject, body, lang string) (translatedSubject, translatedBody string, err error)
}

// translationMaxBodyChars plafonne le corps NETTOYÉ (cf. StripHTMLNoise) envoyé au modèle — coût, un
// mail Malt fait rarement plus de quelques milliers de caractères utiles, même limite d'ordre de
// grandeur que le filtre de pertinence (cf. maxRelevanceBodyChars).
const translationMaxBodyChars = 8000

// translationModel — Haiku : traduction pure, pas de raisonnement complexe requis, volume faible
// (uniquement à la demande — jamais en arrière-plan, cf. GetOrTranslateMailboxEmail).
const translationModel = "claude-haiku-4-5-20251001"

const translationMaxTokens = 4096

// translationSystemPrompt insiste explicitement sur les noms propres — c'est la classe de bug déjà
// rencontrée une fois (un nom de client anonymisé "REMO" traduit à tort en "REMOTE" par un traducteur
// aveugle, cf. maiProFiles/translations.py). Un mail Malt contient des noms de sociétés clientes, des
// noms de plateformes (Malt, Lilycare...), des URLs — aucun ne doit être touché.
const translationSystemPrompt = `You are translating a French email into %s for a non-French-speaking reader.

Rules:
- Translate naturally — convey meaning, not word-for-word literalism.
- NEVER translate proper nouns: company names, client names, platform names (Malt, Lilycare, etc.), person names, product names. Keep them EXACTLY as written in the source.
- NEVER alter URLs, email addresses, or numbers/amounts/dates.
- Preserve paragraph breaks and overall structure.

Call submit_translation EXACTLY ONCE with the translated subject and body.`

// MailboxTranslationService — implémentation réelle de MailboxTranslator.
type MailboxTranslationService struct {
	client *anthropic.Client
}

var _ MailboxTranslator = (*MailboxTranslationService)(nil)

// NewMailboxTranslationService — nil si baseURL/apiKey absents (même convention que le reste de
// mailbox : feature optionnelle, absence de credentials = désactivée proprement).
func NewMailboxTranslationService(baseURL, apiKey string) *MailboxTranslationService {
	if baseURL == "" || apiKey == "" {
		return nil
	}
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	client := anthropic.NewClient(opts...)
	return &MailboxTranslationService{client: &client}
}

// Translate traduit subject+body vers lang (nom de langue humain, ex "English", "German") en UN seul
// appel (pas de boucle agentic — la traduction n'a besoin d'aucun tool/recherche, contrairement au
// jugement de pertinence). ToolChoice forcé dès le premier tour : garantit une sortie structurée.
func (s *MailboxTranslationService) Translate(ctx context.Context, subject, body, lang string) (string, string, error) {
	body = StripHTMLNoise(body)
	if len(body) > translationMaxBodyChars {
		body = body[:translationMaxBodyChars] + "..."
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(translationModel),
		MaxTokens: translationMaxTokens,
		System:    []anthropic.TextBlockParam{{Text: fmt.Sprintf(translationSystemPrompt, langDisplayName(lang))}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(fmt.Sprintf("Subject: %s\n\nBody:\n%s", subject, body))),
		},
		Tools:      buildTranslationTools(),
		ToolChoice: anthropic.ToolChoiceParamOfTool("submit_translation"),
	}

	resp, err := s.client.Messages.New(ctx, params)
	if err != nil {
		return "", "", fmt.Errorf("mailbox translation call: %w", err)
	}
	for _, block := range resp.Content {
		if tb, ok := block.AsAny().(anthropic.ToolUseBlock); ok && tb.Name == "submit_translation" {
			return parseTranslationToolInput(tb.Input)
		}
	}
	return "", "", fmt.Errorf("mailbox translation: no submit_translation call in response")
}

func buildTranslationTools() []anthropic.ToolUnionParam {
	p := anthropic.ToolParam{
		Name:        "submit_translation",
		Description: anthropic.String("Submit the translated subject and body. MUST be called exactly once."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]interface{}{
				"subject": map[string]interface{}{"type": "string", "description": "Translated subject line."},
				"body":    map[string]interface{}{"type": "string", "description": "Translated body text."},
			},
			Required: []string{"subject", "body"},
		},
	}
	return []anthropic.ToolUnionParam{{OfTool: &p}}
}

func parseTranslationToolInput(input interface{}) (string, string, error) {
	m, ok := parseInput(input)
	if !ok {
		return "", "", fmt.Errorf("invalid submit_translation input")
	}
	subject, _ := m["subject"].(string)
	body, _ := m["body"].(string)
	return subject, body, nil
}

// langDisplayName convertit un code de langue (locale next-intl : en/de/it/zh) en nom humain pour le
// prompt — un modèle traduit plus fiablement vers "German" que vers le code brut "de".
func langDisplayName(lang string) string {
	switch lang {
	case "en":
		return "English"
	case "de":
		return "German"
	case "it":
		return "Italian"
	case "zh":
		return "Simplified Chinese"
	default:
		return lang
	}
}
