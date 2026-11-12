package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/config"
)

// QUOI : la forme RÉELLE du corps envoyé par Complete selon `Effort`.
//
// POURQUOI : deux pièges muets des modèles 5.x (CLAUDE.md) — `budget_tokens` rend un bloc de
// réflexion vide, `temperature` un 400. Un effort demandé doit donc partir en
// `thinking:{type:"adaptive"}` + `output_config:{effort}`, et RIEN d'autre ; sans effort, le corps
// reste nu (la passerelle pose la forme par défaut du modèle). Le test lit le JSON tel qu'il arrive
// sur le fil, pas la structure Go : c'est le fil que la passerelle juge.
func TestComplete_EffortPartEnFormeAdaptive(t *testing.T) {
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"m","type":"message","role":"assistant","model":"x","stop_reason":"end_turn",` +
			`"content":[{"type":"thinking","thinking":"…","signature":"s"},{"type":"text","text":"ok"}],` +
			`"usage":{"input_tokens":10,"output_tokens":20}}`))
	}))
	defer srv.Close()

	ai, err := NewAIService(&config.AIConfig{AnthropicAPIKey: "k", AnthropicBaseURL: srv.URL, ClaudeModel: "x",
		MaxRequestsPerMinute: 1, MaxTokensPerRequest: 100, RequestTimeout: 5 * time.Second}, nil)
	require.NoError(t, err)

	c, err := ai.Complete(context.Background(), CompletionRequest{Model: "claude-sonnet-5-5", Prompt: "p", MaxTokens: 900, Effort: "medium", Timeout: 10 * time.Second})
	require.NoError(t, err)
	assert.Equal(t, "ok", c.Text, "seuls les blocs `text` sont lus")
	// Sonnet 5.5 : $2/$10 (source corrigée le 2026-10-09, cf. ai_pricing.go).
	assert.InDelta(t, 10.0/1e6*2+20.0/1e6*10, c.Metrics.EstimatedCost, 1e-12, "tarif Sonnet 5.5")

	_, err = ai.Complete(context.Background(), CompletionRequest{Model: "claude-haiku-5-5", Prompt: "p", MaxTokens: 900})
	require.NoError(t, err)

	require.Len(t, bodies, 2)
	withEffort, without := bodies[0], bodies[1]
	assert.Equal(t, map[string]any{"type": "adaptive"}, withEffort["thinking"])
	assert.Equal(t, map[string]any{"effort": "medium"}, withEffort["output_config"])
	assert.EqualValues(t, 900, withEffort["max_tokens"])
	for _, b := range bodies {
		_, temp := b["temperature"]
		assert.False(t, temp, "jamais de temperature")
		if th, ok := b["thinking"].(map[string]any); ok {
			_, budget := th["budget_tokens"]
			assert.False(t, budget, "jamais de budget_tokens")
		}
	}
	_, th := without["thinking"]
	_, oc := without["output_config"]
	assert.False(t, th || oc, "sans effort, aucune forme de réflexion n'est imposée")
}
