package dossiervivant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/config"
	"maicivy/internal/services"
)

// QUOI : l'événement `usage` compte TOUTE l'entrée facturée, cache compris.
//
// POURQUOI : la passerelle pose un cache de prompt d'une heure ; son `input_tokens` EXCLUT alors les
// tokens écrits ou lus en cache (mesuré : 1 454 d'écriture sur 1 876 d'entrée). Un compteur réglé sur
// `input_tokens` seul afficherait environ quatre fois moins que ce que le run consomme réellement.
//
// COMMENT : faux serveur passerelle (motif run_test.go) renvoyant un `usage` à la forme réelle, le vrai
// AIService.Complete, et UN appel par le chemin unique `run.complete`.

func TestComplete_UsageCompteLeCache(t *testing.T) {
	const in, cacheW, cacheR, out = 422, 1454, 300, 40
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_cache","type":"message","role":"assistant","model":"claude-haiku-5-5",
			"content":[{"type":"text","text":"{\"ok\":true}"}],"stop_reason":"end_turn",
			"usage":{"input_tokens":422,"cache_creation_input_tokens":1454,"cache_read_input_tokens":300,
				"cache_creation":{"ephemeral_1h_input_tokens":1454,"ephemeral_5m_input_tokens":0},"output_tokens":40}}`))
	}))
	t.Cleanup(srv.Close)
	ai, err := services.NewAIService(&config.AIConfig{
		AnthropicAPIKey: "eai_test", AnthropicBaseURL: srv.URL, MaxRequestsPerMinute: 1, RequestTimeout: 5 * time.Second,
	}, nil)
	require.NoError(t, err)

	rec := &recorder{}
	r := &run{ai: ai, emit: rec, meter: newMeter(), tempo: tempo{start: time.Now(), now: time.Now}}
	var answer struct{ OK bool }
	require.NoError(t, r.complete(context.Background(), Reader,
		call{task: "cache", model: testModel, system: "s", prompt: "p", maxTokens: 100}, &answer))

	events := rec.all()
	require.Len(t, events, 1)
	u := events[0].(Usage)
	assert.Equal(t, in+cacheW+cacheR, u.InputTokens, "`usage.inputTokens` : toute l'entrée, cache compris")
	assert.Equal(t, out, u.OutputTokens)
	// Haiku 5.5 : 0,10 hors cache, 0,20 écriture 1 h (2 × l'entrée), 0,01 lecture, 0,50 sortie.
	assert.InDelta(t, (in*0.10+cacheW*0.20+cacheR*0.01+out*0.50)/1e6, u.CostUsd, 1e-12)
}
