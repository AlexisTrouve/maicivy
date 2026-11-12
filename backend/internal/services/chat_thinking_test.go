package services

import (
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseStreamEvents : variante plurielle de parseStreamEvent (cf. chat_service_test.go), pour décrire
// un flux complet sans répéter l'appel ligne à ligne.
func parseStreamEvents(t *testing.T, raws []string) []anthropic.MessageStreamEventUnion {
	t.Helper()
	out := make([]anthropic.MessageStreamEventUnion, 0, len(raws))
	for _, r := range raws {
		out = append(out, parseStreamEvent(t, r))
	}
	return out
}

// QUOI : vérifie l'exposition du raisonnement de l'agent (thinking / CoT) et le budget de tokens qui
// va avec.
//
// POURQUOI : sur l'écran compose, l'opérateur ne lit pas le français et ne peut pas juger le message.
// Voir le RAISONNEMENT de l'agent — pourquoi il a retenu tel projet, pourquoi il propose tel tarif —
// est la seule fenêtre qu'il a sur la qualité du travail avant de l'envoyer à un client. Sans ça il ne
// voit qu'un résultat opaque.
//
// POURQUOI un budget dédié : max_tokens plafonne réflexion ET réponse ensemble. Le plafond du chat
// conversationnel (1500) laisserait la réflexion dévorer le budget et tronquer le message en plein
// milieu. Le mode compose a donc ses propres valeurs.
//
// POURQUOI le chat public n'y touche pas : la réflexion coûte des tokens à chaque tour, et un visiteur
// qui demande "c'est quoi tes projets Rust ?" n'en a aucun besoin.

// Mode compose : réflexion ADAPTATIVE, avec le résumé rendu visible.
//
// POURQUOI adaptatif et non plus un budget fixe : `budget_tokens` est REJETÉ (400) à partir d'Opus 4.7.
// Le laisser en place aurait cassé toute génération dès la montée en Opus 5.
// POURQUOI `summarized` explicitement : sans lui, les blocs de réflexion arrivent avec un texte VIDE
// sur les modèles récents — l'opérateur verrait une section « Raisonnement » sans contenu.
func TestThinkingConfig_AdaptatifEtVisibleEnCompose(t *testing.T) {
	cfg := thinkingConfigFor(ChatCapabilities{Compose: true})

	require.NotNil(t, cfg.OfAdaptive, "l'agent compose doit raisonner en mode adaptatif")
	assert.Equal(t, anthropic.ThinkingConfigAdaptiveDisplaySummarized, cfg.OfAdaptive.Display,
		"sans display=summarized, le bloc de raisonnement s'affiche vide")
	assert.Nil(t, cfg.OfEnabled, "budget_tokens est rejeté par l'API dès Opus 4.7")
}

// Hors compose, la réflexion est coupée EXPLICITEMENT.
//
// POURQUOI c'est indispensable : sur Opus 5 la réflexion est ACTIVE PAR DÉFAUT quand le champ est
// absent. Or max_tokens plafonne réflexion ET réponse ensemble ; le chat owner tourne à 1500. Sans
// coupure explicite, la réflexion mangerait le budget et tronquerait la réponse en plein milieu.
func TestThinkingConfig_CoupeeHorsCompose(t *testing.T) {
	cfg := thinkingConfigFor(ChatCapabilities{})

	require.NotNil(t, cfg.OfDisabled,
		"champ absent = réflexion ACTIVE par défaut sur Opus 5 : il faut la couper explicitement")
	assert.Nil(t, cfg.OfAdaptive)
}

func TestMaxTokens_CompoSuperieurAuChatPublic(t *testing.T) {
	assert.Equal(t, int64(maxResponseTokens), maxTokensFor(ChatCapabilities{}),
		"le chat public garde son plafond d'origine")
	assert.Greater(t, maxTokensFor(ChatCapabilities{Compose: true}), int64(maxResponseTokens))
}

// Le raisonnement doit être STREAMÉ au fil de l'eau, comme le texte : l'opérateur voit l'agent
// travailler au lieu d'attendre devant un écran figé.
func TestAccumulateStream_TransmetLeRaisonnement(t *testing.T) {
	raw := []string{
		`{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","content":[],"model":"claude-opus-4-6","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":0}}}`,
		`{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":"","signature":""}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"Leur vrai problème "}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"c'est la mise en production."}}`,
		`{"type":"content_block_stop","index":0}`,
		`{"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"Voici le message."}}`,
		`{"type":"content_block_stop","index":1}`,
		`{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":20}}`,
		`{"type":"message_stop"}`,
	}

	streamEvents := parseStreamEvents(t, raw)

	eventCh := make(chan ChatEvent, 40)
	_, err := accumulateStream(&fakeStream{events: streamEvents}, eventCh)
	require.NoError(t, err)
	close(eventCh)

	var raisonnement, texte string
	ordre := []ChatEventType{}
	for ev := range eventCh {
		switch ev.Type {
		case ChatEventThinking:
			raisonnement += ev.Delta
		case ChatEventText:
			texte += ev.Delta
		}
		ordre = append(ordre, ev.Type)
	}

	assert.Equal(t, "Leur vrai problème c'est la mise en production.", raisonnement)
	assert.Equal(t, "Voici le message.", texte)

	// Le raisonnement précède le texte : c'est ce qui permet à l'UI de l'afficher AVANT la réponse.
	require.NotEmpty(t, ordre)
	assert.Equal(t, ChatEventThinking, ordre[0],
		"le raisonnement doit arriver avant le texte, sinon l'UI ne peut pas le placer au-dessus")
}
