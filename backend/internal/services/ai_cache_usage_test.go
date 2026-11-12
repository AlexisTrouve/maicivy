package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/config"
)

// QUOI : verrouille la lecture des tokens de CACHE dans la réponse de la passerelle, et leur prix.
//
// POURQUOI : la passerelle pose un cache de prompt d'une heure sur les requêtes, et son `usage`
// sépare alors l'entrée en trois postes — `input_tokens` (hors cache), `cache_creation_input_tokens`
// (écriture), `cache_read_input_tokens` (lecture). Mesuré : 1 454 tokens d'écriture sur 1 876 d'entrée,
// absents de `input_tokens`. Lire ce seul champ sous-comptait donc l'entrée de 3/4 sur un premier
// appel, et le coût et le budget du dossier vivant avec elle — sans aucune erreur.
//
// COMMENT : faux serveur passerelle (motif ai_fallback_test.go) qui renvoie un `usage` à la forme de
// la passerelle réelle ; le vrai AIService.Complete le lit. Les coûts attendus sont écrits en
// arithmétique explicite sur les tarifs cités dans ai_pricing.go — pas recalculés par le code testé.

// cacheGateway : passerelle factice qui renvoie le `usage` brut fourni, tel quel.
func cacheGateway(t *testing.T, usage string) *AIService {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_cache","type":"message","role":"assistant","model":"claude-haiku-5-5",
			"content":[{"type":"text","text":"{\"ok\":true}"}],"stop_reason":"end_turn","usage":` + usage + `}`))
	}))
	t.Cleanup(srv.Close)
	svc, err := NewAIService(&config.AIConfig{
		AnthropicAPIKey: "eai_test", AnthropicBaseURL: srv.URL, MaxRequestsPerMinute: 6000,
		RequestTimeout: 5 * time.Second,
	}, nil)
	require.NoError(t, err)
	return svc
}

func TestComplete_CompteEtFactureLeCache(t *testing.T) {
	for _, tc := range []struct {
		nom, model, usage      string
		entree, cacheW, cacheR int
		cout                   float64 // $ attendus, arithmétique sur les tarifs cités
	}{
		{
			// Premier appel : la passerelle ÉCRIT le cache (TTL 1 h, la sienne). Haiku 5.5 :
			// 422 × 0,10 + 1 454 × 0,20 (écriture 1 h = 2 × l'entrée) + 40 × 0,50.
			nom: "ecriture 1h", model: "claude-haiku-5-5",
			usage:  `{"input_tokens":422,"cache_creation_input_tokens":1454,"cache_read_input_tokens":0,"cache_creation":{"ephemeral_1h_input_tokens":1454,"ephemeral_5m_input_tokens":0},"output_tokens":40}`,
			entree: 422 + 1454, cacheW: 1454,
			cout: (422*0.10 + 1454*0.20 + 40*0.50) / 1e6,
		},
		{
			// Appel suivant dans l'heure : le même préfixe est LU. 1 454 × 0,01 (lecture Haiku 5.5).
			nom: "lecture", model: "claude-haiku-5-5",
			usage:  `{"input_tokens":422,"cache_creation_input_tokens":0,"cache_read_input_tokens":1454,"output_tokens":40}`,
			entree: 422 + 1454, cacheR: 1454,
			cout: (422*0.10 + 1454*0.01 + 40*0.50) / 1e6,
		},
		{
			// Écriture 5 min déclarée : tarif `cacheWrite` de la table (Sonnet 5.5 : 2,50).
			nom: "ecriture 5min", model: "claude-sonnet-5-5",
			usage:  `{"input_tokens":100,"cache_creation_input_tokens":2000,"cache_read_input_tokens":0,"cache_creation":{"ephemeral_1h_input_tokens":0,"ephemeral_5m_input_tokens":2000},"output_tokens":10}`,
			entree: 100 + 2000, cacheW: 2000,
			cout: (100*2.00 + 2000*2.50 + 10*10.00) / 1e6,
		},
		{
			// Écriture SANS ventilation par TTL : imputée au TTL de la passerelle (1 h, le plus cher —
			// ne jamais sous-estimer). Sonnet 5.5 : 2 × 2,00.
			nom: "ecriture non ventilee", model: "claude-sonnet-5-5",
			usage:  `{"input_tokens":100,"cache_creation_input_tokens":2000,"cache_read_input_tokens":0,"output_tokens":10}`,
			entree: 100 + 2000, cacheW: 2000,
			cout: (100*2.00 + 2000*4.00 + 10*10.00) / 1e6,
		},
		{
			// Le palier ×5 de Haiku 5.5 porte sur le prompt TOTAL : 1 000 hors cache + 99 500 lus
			// = 100 500 > 100 000 → toute la requête au palier haut (0,50 / 0,05 / 2,50).
			nom: "palier haiku sur prompt total", model: "claude-haiku-5-5",
			usage:  `{"input_tokens":1000,"cache_creation_input_tokens":0,"cache_read_input_tokens":99500,"output_tokens":100}`,
			entree: 1000 + 99500, cacheR: 99500,
			cout: (1000*0.50 + 99500*0.05 + 100*2.50) / 1e6,
		},
	} {
		t.Run(tc.nom, func(t *testing.T) {
			svc := cacheGateway(t, tc.usage)
			c, err := svc.Complete(context.Background(), CompletionRequest{Model: tc.model, Prompt: "p", MaxTokens: 100})
			require.NoError(t, err)

			assert.Equal(t, tc.entree, c.Metrics.TokensInput, "l'entrée compte les trois postes : hors cache + écriture + lecture")
			assert.Equal(t, tc.cacheW, c.Metrics.TokensCacheWrite, "détail : écriture de cache")
			assert.Equal(t, tc.cacheR, c.Metrics.TokensCacheRead, "détail : lecture de cache")
			assert.Equal(t, tc.entree+c.Metrics.TokensOutput, c.Metrics.TotalTokens)
			assert.InDelta(t, tc.cout, c.Metrics.EstimatedCost, 1e-12, "chaque poste à son tarif")
		})
	}
}
