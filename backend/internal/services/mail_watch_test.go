package services_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/services"
)

// journalWebhook retient TOUS les messages postés, pour pouvoir vérifier à la fois ce qui est dit et
// ce qui ne l'est pas (l'absence de post est ici une assertion de premier plan).
type journalWebhook struct {
	srv      *httptest.Server
	messages []string
	pings    []bool
}

func nouveauJournalWebhook(t *testing.T) *journalWebhook {
	t.Helper()
	j := &journalWebhook{}
	j.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corps, _ := io.ReadAll(r.Body)
		var charge map[string]any
		_ = json.Unmarshal(corps, &charge)
		contenu, _ := charge["content"].(string)
		mentions, _ := charge["allowed_mentions"].(map[string]any)
		_, pingue := mentions["users"]
		j.messages = append(j.messages, contenu)
		j.pings = append(j.pings, pingue)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(j.srv.Close)
	return j
}

func nouveauMailWatch(t *testing.T) (*services.MailWatch, *journalWebhook, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	j := nouveauJournalWebhook(t)
	w := services.NewMailWatch(rdb, services.NewDiscordNotifier(j.srv.URL, "243388469169094656"))
	require.NotNil(t, w)
	return w, j, mr
}

var jourTest = time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)

func TestMailWatch_Succes_CompteParOrigine(t *testing.T) {
	w, j, _ := nouveauMailWatch(t)
	ctx := context.Background()

	w.Succes(ctx, services.OrigineMalt, jourTest)
	w.Succes(ctx, services.OrigineMalt, jourTest)
	w.Succes(ctx, services.OrigineWanMira, jourTest)

	compte, err := w.Compteurs(ctx, jourTest)
	require.NoError(t, err)
	assert.Equal(t, 2, compte.Envoyes[services.OrigineMalt])
	assert.Equal(t, 1, compte.Envoyes[services.OrigineWanMira])
	assert.Equal(t, 0, compte.Envoyes[services.OrigineRappelMalt])

	assert.Empty(t, j.messages, "un envoi réussi ne poste rien sur le moment — il n'apparaît que dans le digest")
}

// Un échec ping tout de suite : c'est exactement le signal qui manquait pendant les 13 jours.
func TestMailWatch_Echec_PingImmediat(t *testing.T) {
	w, j, _ := nouveauMailWatch(t)
	ctx := context.Background()

	w.Echec(ctx, services.OrigineMalt, "dial tcp: connection timed out", jourTest)

	require.Len(t, j.messages, 1)
	assert.True(t, j.pings[0], "un envoi cassé DOIT pinguer")
	assert.Contains(t, j.messages[0], "Malt", "l'alerte nomme l'origine en clair, pas la clé Redis")
	assert.Contains(t, j.messages[0], "connection timed out", "le motif doit être lisible sans ouvrir les logs")

	compte, err := w.Compteurs(ctx, jourTest)
	require.NoError(t, err)
	assert.Equal(t, 1, compte.Echecs[services.OrigineMalt])
}

// 11 mails en attente qui échouent tous = 1 alerte, pas 11. Sinon la première vraie panne noie le
// salon et l'alerte suivante ne sera plus lue.
func TestMailWatch_Echec_CooldownEvitLeSpam(t *testing.T) {
	w, j, _ := nouveauMailWatch(t)
	ctx := context.Background()

	for i := 0; i < 11; i++ {
		w.Echec(ctx, services.OrigineMalt, "connection timed out", jourTest)
	}

	assert.Len(t, j.messages, 1, "un seul ping malgré 11 échecs de la même origine")

	compte, err := w.Compteurs(ctx, jourTest)
	require.NoError(t, err)
	assert.Equal(t, 11, compte.Echecs[services.OrigineMalt], "mais les 11 échecs restent comptés")
}

// Le cooldown est PAR ORIGINE : une newsletter cassée ne doit pas masquer un transfert Malt cassé.
func TestMailWatch_Echec_CooldownParOrigine(t *testing.T) {
	w, j, _ := nouveauMailWatch(t)
	ctx := context.Background()

	w.Echec(ctx, services.OrigineMalt, "timeout", jourTest)
	w.Echec(ctx, services.OrigineWanMira, "relai refusé", jourTest)

	assert.Len(t, j.messages, 2)
}

// « Faut pas me ping si y'a juste rien » : une journée sans le moindre envoi ni échec ne produit
// AUCUN message. Pas un digest à zéro, rien du tout.
func TestMailWatch_Digest_SilencieuxQuandRienNeSestPasse(t *testing.T) {
	w, j, _ := nouveauMailWatch(t)
	ctx := context.Background()

	poste, err := w.PosterDigest(ctx, jourTest)
	require.NoError(t, err)
	assert.False(t, poste)
	assert.Empty(t, j.messages)
}

func TestMailWatch_Digest_ListeLesOriginesSansPinguer(t *testing.T) {
	w, j, _ := nouveauMailWatch(t)
	ctx := context.Background()

	for i := 0; i < 16; i++ {
		w.Succes(ctx, services.OrigineMalt, jourTest)
	}
	w.Succes(ctx, services.OrigineWanMira, jourTest)

	poste, err := w.PosterDigest(ctx, jourTest)
	require.NoError(t, err)
	assert.True(t, poste)

	require.Len(t, j.messages, 1)
	assert.False(t, j.pings[0], "le récapitulatif quotidien ne ping JAMAIS")
	assert.Contains(t, j.messages[0], "16")
	assert.Contains(t, j.messages[0], "Malt")
	assert.Contains(t, j.messages[0], "WanMira")
}

// Le job tourne toutes les heures : le digest ne doit partir qu'une fois par jour.
func TestMailWatch_Digest_UneSeuleFoisParJour(t *testing.T) {
	w, j, _ := nouveauMailWatch(t)
	ctx := context.Background()
	w.Succes(ctx, services.OrigineMalt, jourTest)

	poste, err := w.PosterDigest(ctx, jourTest)
	require.NoError(t, err)
	assert.True(t, poste)

	poste, err = w.PosterDigest(ctx, jourTest)
	require.NoError(t, err)
	assert.False(t, poste)
	assert.Len(t, j.messages, 1)
}

// L'alerte qui aurait coupé les 13 jours : des mails en attente, et RIEN qui échoue bruyamment
// (le relai était injoignable, donc plus personne ne tentait après le premier cycle).
func TestMailWatch_AlerteBlocage_Ping(t *testing.T) {
	w, j, _ := nouveauMailWatch(t)
	ctx := context.Background()

	w.AlerterBlocage(ctx, 11, jourTest.Add(-13*24*time.Hour))

	require.Len(t, j.messages, 1)
	assert.True(t, j.pings[0])
	assert.Contains(t, j.messages[0], "11")
}

func TestMailWatch_AlerteBlocage_CooldownEvitLeRappelHoraire(t *testing.T) {
	w, j, _ := nouveauMailWatch(t)
	ctx := context.Background()

	w.AlerterBlocage(ctx, 11, jourTest.Add(-13*24*time.Hour))
	w.AlerterBlocage(ctx, 11, jourTest.Add(-13*24*time.Hour))

	assert.Len(t, j.messages, 1, "le job tourne toutes les heures : une seule alerte par fenêtre")
}

// Sans webhook configuré, la comptabilité doit continuer de fonctionner : on veut les chiffres même
// si personne n'est notifié, et surtout aucun panic sur un notifier nil.
func TestMailWatch_SansNotifier_CompteQuandMeme(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	w := services.NewMailWatch(rdb, nil)
	require.NotNil(t, w)
	ctx := context.Background()

	w.Succes(ctx, services.OrigineMalt, jourTest)
	w.Echec(ctx, services.OrigineWanMira, "peu importe", jourTest)

	compte, err := w.Compteurs(ctx, jourTest)
	require.NoError(t, err)
	assert.Equal(t, 1, compte.Envoyes[services.OrigineMalt])
	assert.Equal(t, 1, compte.Echecs[services.OrigineWanMira])
}

func TestNewMailWatch_SansRedisDesactive(t *testing.T) {
	assert.Nil(t, services.NewMailWatch(nil, nil), "sans Redis on ne peut rien compter")
}
