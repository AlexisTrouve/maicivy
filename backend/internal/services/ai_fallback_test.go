package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/config"
)

// QUOI : verrouille le comportement de repli d'AIService quand le modèle primaire échoue.
//
// POURQUOI ces tests existent : le repli passait autrefois par un SECOND client (OpenAI) qui tapait
// directement api.openai.com, hors passerelle. Trois conséquences, toutes silencieuses : un message
// signé Alexis pouvait sortir d'un autre fournisseur sans trace, le coût échappait à la comptabilité
// de la passerelle, et une seconde clé vivait dans le conteneur. Le repli passe désormais par LA MÊME
// passerelle, avec la MÊME clé — seul l'identifiant de modèle change. Ces tests vérifient exactement
// ça : une seule destination réseau, quoi qu'il arrive.
//
// COMMENT : un faux serveur joue le rôle de la passerelle et enregistre chaque requête reçue (chemin
// + modèle demandé). Le SDK Anthropic y est pointé via AnthropicBaseURL, exactement comme en prod où
// il pointe sur ai.etheryale.com. Aucun appel réseau réel.

// gatewayStub : faux serveur passerelle. Enregistre les modèles demandés dans l'ordre et répond selon
// un scénario indexé par modèle (statut HTTP à renvoyer).
type gatewayStub struct {
	srv *httptest.Server

	mu     sync.Mutex
	models []string // modèles demandés, dans l'ordre d'arrivée
	paths  []string // chemins appelés, dans l'ordre — sert à prouver l'unicité de la destination

	// statusFor : statut à renvoyer pour un modèle donné. Absent = 200.
	statusFor map[string]int
}

// newGatewayStub monte le faux serveur. statusFor liste les modèles qui doivent ÉCHOUER (et avec quel
// statut) ; tout modèle absent de la table réussit.
//
// COMMENT : la réponse de succès est le minimum qu'attend le SDK Anthropic pour construire un Message
// (id, type, role, model, content, stop_reason, usage). Le champ `model` est renvoyé tel que demandé,
// ce qui permet au test de vérifier quel modèle a réellement produit la réponse.
func newGatewayStub(t *testing.T, statusFor map[string]int) *gatewayStub {
	t.Helper()

	g := &gatewayStub{statusFor: statusFor}
	g.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		g.mu.Lock()
		g.models = append(g.models, body.Model)
		g.paths = append(g.paths, r.URL.Path)
		g.mu.Unlock()

		if code, ko := g.statusFor[body.Model]; ko {
			// 400 : erreur NON retryable (cf. isRetryableError) → une seule tentative, le test reste
			// rapide et le repli est le seul comportement observable.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(code)
			_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"modele indisponible"}}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"msg_stub","type":"message","role":"assistant","model":"` + body.Model + `",
			"content":[{"type":"text","text":"reponse du modele ` + body.Model + `"}],
			"stop_reason":"end_turn",
			"usage":{"input_tokens":10,"output_tokens":5}
		}`))
	}))
	t.Cleanup(g.srv.Close)
	return g
}

func (g *gatewayStub) requested() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]string(nil), g.models...)
}

func (g *gatewayStub) requestedPaths() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]string(nil), g.paths...)
}

// newAIServiceForStub : service pointé sur le faux serveur.
//
// COMMENT : MaxRetries=0 et RetryBaseDelay=0 pour qu'un échec soit immédiat (sinon chaque cas d'échec
// attendrait les backoffs réels). MaxRequestsPerMinute élevé pour que le rate limiter interne ne
// vienne pas fausser le compte de requêtes.
func newAIServiceForStub(t *testing.T, g *gatewayStub, primaire, repli string) *AIService {
	t.Helper()

	svc, err := NewAIService(&config.AIConfig{
		AnthropicAPIKey:      "eai_test",
		AnthropicBaseURL:     g.srv.URL,
		ClaudeModel:          primaire,
		FallbackModel:        repli,
		MaxRequestsPerMinute: 6000,
		MaxTokensPerRequest:  256,
		MaxRetries:           0,
		RetryBaseDelay:       0,
		RequestTimeout:       5 * time.Second,
	}, nil)
	require.NoError(t, err)
	return svc
}

// Le cas central : le modèle primaire échoue, le repli DOIT partir sur la même passerelle.
func TestGenerateText_ReplitSurLaMemePasserelle(t *testing.T) {
	const primaire, repli = "claude-haiku-latest", "gpt-5.6-terra"

	g := newGatewayStub(t, map[string]int{primaire: http.StatusBadRequest})
	svc := newAIServiceForStub(t, g, primaire, repli)

	text, metrics, err := svc.GenerateText(context.Background(), "bonjour")

	require.NoError(t, err, "le repli doit rattraper l'échec du modèle primaire")
	assert.Contains(t, text, repli, "la réponse doit venir du modèle de repli")

	// LE point du changement : deux appels, une seule destination.
	assert.Equal(t, []string{primaire, repli}, g.requested(),
		"le primaire doit être tenté d'abord, puis le repli")
	for _, p := range g.requestedPaths() {
		assert.Equal(t, "/v1/messages", p,
			"TOUT doit passer par la passerelle en protocole Anthropic — aucun appel direct chez un autre fournisseur")
	}

	require.NotNil(t, metrics)
	assert.Equal(t, repli, metrics.Model, "les métriques doivent porter le modèle qui a RÉELLEMENT répondu")
	assert.True(t, metrics.Success)
}

// Le repli ne doit pas se déclencher quand le primaire répond : un appel, pas deux.
func TestGenerateText_PasDeRepliSiLePrimaireRepond(t *testing.T) {
	const primaire, repli = "claude-haiku-latest", "gpt-5.6-terra"

	g := newGatewayStub(t, nil) // personne n'échoue
	svc := newAIServiceForStub(t, g, primaire, repli)

	_, metrics, err := svc.GenerateText(context.Background(), "bonjour")

	require.NoError(t, err)
	assert.Equal(t, []string{primaire}, g.requested(), "aucun appel de repli ne doit partir inutilement")
	require.NotNil(t, metrics)
	assert.Equal(t, primaire, metrics.Model)
}

// Repli non configuré = échec FRANC. POURQUOI ce test : la doctrine du projet proscrit les replis
// masquants ; l'absence de repli doit produire une erreur visible, pas un silence.
func TestGenerateText_SansRepliConfigureEchecFranc(t *testing.T) {
	const primaire = "claude-haiku-latest"

	g := newGatewayStub(t, map[string]int{primaire: http.StatusBadRequest})
	svc := newAIServiceForStub(t, g, primaire, "") // pas de repli

	_, _, err := svc.GenerateText(context.Background(), "bonjour")

	require.Error(t, err)
	assert.Equal(t, []string{primaire}, g.requested(), "sans repli configuré, un seul appel doit partir")
}

// Un repli identique au modèle primaire n'a aucun sens : le rejouer coûterait un appel pour reproduire
// le même échec. Il doit être ignoré.
func TestGenerateText_ReplIdentiqueAuPrimaireIgnore(t *testing.T) {
	const modele = "claude-haiku-latest"

	g := newGatewayStub(t, map[string]int{modele: http.StatusBadRequest})
	svc := newAIServiceForStub(t, g, modele, modele)

	_, _, err := svc.GenerateText(context.Background(), "bonjour")

	require.Error(t, err)
	assert.Equal(t, []string{modele}, g.requested(),
		"un repli égal au modèle primaire doit être ignoré, pas rejoué")
}

// Le repli s'applique aussi quand l'appelant impose un modèle (cas owner : Opus explicite).
// POURQUOI ce test : la comparaison « repli == primaire » doit se faire sur le modèle EFFECTIF, pas
// sur la valeur par défaut de la configuration.
func TestGenerateTextWithModel_ReplISurModeleImpose(t *testing.T) {
	const impose, repli = "claude-opus-5", "gpt-5.6-terra"

	g := newGatewayStub(t, map[string]int{impose: http.StatusBadRequest})
	svc := newAIServiceForStub(t, g, "claude-haiku-latest", repli)

	_, metrics, err := svc.GenerateTextWithModel(context.Background(), "bonjour", impose)

	require.NoError(t, err)
	assert.Equal(t, []string{impose, repli}, g.requested(),
		"le modèle imposé doit être tenté d'abord, pas celui de la configuration")
	require.NotNil(t, metrics)
	assert.Equal(t, repli, metrics.Model)
}
