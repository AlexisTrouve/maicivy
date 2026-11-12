package services_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/services"
)

// captureWebhook monte un faux webhook Discord et rend le corps JSON du dernier POST reçu.
func captureWebhook(t *testing.T) (*httptest.Server, func() map[string]any, func() int) {
	t.Helper()
	var dernier map[string]any
	appels := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appels++
		corps, _ := io.ReadAll(r.Body)
		dernier = map[string]any{}
		_ = json.Unmarshal(corps, &dernier)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	return srv, func() map[string]any { return dernier }, func() int { return appels }
}

func TestNewDiscordNotifier_WebhookVideDesactive(t *testing.T) {
	assert.Nil(t, services.NewDiscordNotifier("", "243388469169094656"),
		"webhook vide = fonctionnalité désactivée, même convention que les autres services optionnels")
}

// Le cas « ne me ping pas s'il n'y a rien » ne doit pas reposer sur la discipline de l'appelant :
// sans ping, la charge utile INTERDIT toute mention côté Discord (allowed_mentions.parse vide).
// Même si un texte contenait « @everyone » ou un <@id>, Discord ne notifierait personne.
func TestDiscordNotifier_SansPing_AucuneMentionAutorisee(t *testing.T) {
	srv, dernier, appels := captureWebhook(t)
	n := services.NewDiscordNotifier(srv.URL, "243388469169094656")
	require.NotNil(t, n)

	require.NoError(t, n.Post(context.Background(), "16 mails envoyés aujourd'hui", false))

	assert.Equal(t, 1, appels())
	corps := dernier()
	assert.Equal(t, "16 mails envoyés aujourd'hui", corps["content"])
	assert.NotContains(t, corps["content"], "<@", "sans ping, aucune mention dans le texte")

	mentions, ok := corps["allowed_mentions"].(map[string]any)
	require.True(t, ok, "allowed_mentions doit toujours être présent")
	parse, ok := mentions["parse"].([]any)
	require.True(t, ok, "parse doit être une liste explicitement vide")
	assert.Empty(t, parse, "parse vide = Discord ne résout AUCUNE mention")
	assert.NotContains(t, mentions, "users")
}

func TestDiscordNotifier_AvecPing_MentionneLUtilisateur(t *testing.T) {
	srv, dernier, _ := captureWebhook(t)
	n := services.NewDiscordNotifier(srv.URL, "243388469169094656")
	require.NotNil(t, n)

	require.NoError(t, n.Post(context.Background(), "le transfert casse", true))

	corps := dernier()
	contenu, _ := corps["content"].(string)
	assert.True(t, strings.HasPrefix(contenu, "<@243388469169094656>"),
		"le ping doit être en tête du message, pas noyé dedans : %q", contenu)
	assert.Contains(t, contenu, "le transfert casse")

	mentions, _ := corps["allowed_mentions"].(map[string]any)
	users, ok := mentions["users"].([]any)
	require.True(t, ok)
	assert.Equal(t, []any{"243388469169094656"}, users)
}

// Sans user ID configuré, un ping demandé ne doit pas fabriquer une mention vide « <@> » : on poste
// le message sans mention plutôt que d'émettre un texte cassé.
func TestDiscordNotifier_PingSansUserID_PosteSansMention(t *testing.T) {
	srv, dernier, _ := captureWebhook(t)
	n := services.NewDiscordNotifier(srv.URL, "")
	require.NotNil(t, n)

	require.NoError(t, n.Post(context.Background(), "alerte", true))

	corps := dernier()
	assert.Equal(t, "alerte", corps["content"])
	mentions, _ := corps["allowed_mentions"].(map[string]any)
	assert.NotContains(t, mentions, "users")
}

// Un webhook en panne ne doit jamais faire tomber l'appelant (on notifie À PROPOS d'un envoi de
// mail : perdre la notification est ennuyeux, perdre le mail serait grave).
func TestDiscordNotifier_ErreurHTTP_RemonteMaisNePanique(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	n := services.NewDiscordNotifier(srv.URL, "1")
	require.NotNil(t, n)
	assert.Error(t, n.Post(context.Background(), "peu importe", false))
}
