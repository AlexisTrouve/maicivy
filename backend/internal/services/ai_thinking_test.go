package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"maicivy/internal/config"
)

// TestGenerateText_IgnoreLeBlocDeReflexion : la réponse d'un modèle à réflexion toujours active
// (claude-haiku-5-5, Opus 5) commence par un bloc `thinking`, PAS par le texte.
// POURQUOI : GenerateTextWithModel lisait `resp.Content[0].Text` — sur un bloc thinking ce champ est
// vide, et la fonction rendait "" SANS erreur : lettre, message ou traduction Composer vides, en
// silence. Découvert le 2026-10-16 en préparant la bascule Haiku 4.5 → 5.5 (demande de la passerelle).
// COMMENT : faux serveur passerelle qui répond [thinking, text, text] ; on attend la concaténation
// des seuls blocs texte.
func TestGenerateText_IgnoreLeBlocDeReflexion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"msg_t","type":"message","role":"assistant","model":"claude-haiku-5-5",
			"content":[
				{"type":"thinking","thinking":"je reflechis","signature":"sig"},
				{"type":"text","text":"Bonjour "},
				{"type":"text","text":"le monde"}
			],
			"stop_reason":"end_turn",
			"usage":{"input_tokens":10,"output_tokens":5}
		}`))
	}))
	t.Cleanup(srv.Close)

	svc, err := NewAIService(&config.AIConfig{
		AnthropicAPIKey: "eai_test", AnthropicBaseURL: srv.URL, ClaudeModel: "claude-haiku-5-5",
		MaxRequestsPerMinute: 6000, MaxTokensPerRequest: 256, RequestTimeout: 5 * time.Second,
	}, nil)
	require.NoError(t, err)

	out, _, err := svc.GenerateTextWithModel(context.Background(), "x", "claude-haiku-5-5")
	require.NoError(t, err)
	require.Equal(t, "Bonjour le monde", out)
}
