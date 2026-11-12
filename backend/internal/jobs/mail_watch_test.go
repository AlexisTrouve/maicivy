package jobs_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"maicivy/internal/jobs"
	"maicivy/internal/models"
	"maicivy/internal/services"
)

func setupJobDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.MailboxEmail{}, &models.MailboxCursor{}))
	return db
}

// webhookEspion retient les messages postés, pour pouvoir affirmer aussi bien la présence d'une
// alerte que son ABSENCE (ici, l'absence est l'assertion la plus importante).
type webhookEspion struct {
	srv      *httptest.Server
	messages []string
}

func nouveauWebhookEspion(t *testing.T) *webhookEspion {
	t.Helper()
	e := &webhookEspion{}
	e.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corps, _ := io.ReadAll(r.Body)
		var charge map[string]any
		_ = json.Unmarshal(corps, &charge)
		contenu, _ := charge["content"].(string)
		e.messages = append(e.messages, contenu)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(e.srv.Close)
	return e
}

func nouveauJobTest(t *testing.T, db *gorm.DB, seuil time.Duration) (*jobs.MailWatchJob, *webhookEspion) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	e := nouveauWebhookEspion(t)
	watch := services.NewMailWatch(rdb, services.NewDiscordNotifier(e.srv.URL, "243388469169094656"))
	j := jobs.NewMailWatchJob(db, watch, seuil, 21)
	require.NotNil(t, j)
	return j, e
}

// insererMail pose un mail dans l'état voulu. forwardedAt nil = jamais transféré. Relevé dès sa
// réception (le cas normal) : cf. insererMailReleve pour une relève en retard.
func insererMail(t *testing.T, db *gorm.DB, recu time.Time, forwardedAt *time.Time, bloque bool) {
	t.Helper()
	insererMailReleve(t, db, recu, recu, forwardedAt, bloque)
}

// insererMailReleve distingue la date du mail (en-tête, `recu`) de sa relève IMAP (`releve`, qui
// devient created_at) — c'est l'écart entre les deux qui a produit la fausse alerte du 19/09.
func insererMailReleve(t *testing.T, db *gorm.DB, recu, releve time.Time, forwardedAt *time.Time, bloque bool) {
	t.Helper()
	// L'ID vient de BaseModel (hook GORM) — on ne le pose pas à la main.
	require.NoError(t, db.Create(&models.MailboxEmail{
		BaseModel:      models.BaseModel{CreatedAt: releve},
		MessageID:      uuid.NewString() + "@test",
		FromAddress:    "noreply@malt.com",
		FromDomain:     "malt.com",
		Platform:       "malt",
		Subject:        "Nouvelle opportunité",
		ReceivedAt:     recu,
		ForwardedAt:    forwardedAt,
		ForwardBlocked: bloque,
	}).Error)
}

var maintenantTest = time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)

// Le cas des 13 jours : du courrier jamais transféré, plus vieux que le seuil → alerte.
func TestMailWatchJob_BlocageAncien_Alerte(t *testing.T) {
	db := setupJobDB(t)
	j, e := nouveauJobTest(t, db, 6*time.Hour)

	insererMail(t, db, maintenantTest.Add(-13*24*time.Hour), nil, false)
	insererMail(t, db, maintenantTest.Add(-10*time.Hour), nil, false)

	j.VerifierBlocage(context.Background(), maintenantTest)

	require.Len(t, e.messages, 1)
	assert.Contains(t, e.messages[0], "2 mail(s) en attente")
	assert.Contains(t, e.messages[0], "17/08/2026", "l'alerte doit nommer la date du plus ancien")
}

// Un mail récent en attente est NORMAL : il part au prochain cycle de poll. Alerter dessus rendrait
// l'alerte permanente, donc inutile.
func TestMailWatchJob_EnAttenteRecent_PasDAlerte(t *testing.T) {
	db := setupJobDB(t)
	j, e := nouveauJobTest(t, db, 6*time.Hour)

	insererMail(t, db, maintenantTest.Add(-30*time.Minute), nil, false)

	j.VerifierBlocage(context.Background(), maintenantTest)

	assert.Empty(t, e.messages, "un mail reçu il y a 30 min n'est pas bloqué, il attend son tour")
}

// Un mail écarté par le filtre de pertinence est une DÉCISION, pas une panne : il resterait éternel
// en base et produirait une alerte permanente.
func TestMailWatchJob_BloqueParPertinence_PasDAlerte(t *testing.T) {
	db := setupJobDB(t)
	j, e := nouveauJobTest(t, db, 6*time.Hour)

	insererMail(t, db, maintenantTest.Add(-20*24*time.Hour), nil, true)

	j.VerifierBlocage(context.Background(), maintenantTest)

	assert.Empty(t, e.messages)
}

func TestMailWatchJob_ToutTransfere_PasDAlerte(t *testing.T) {
	db := setupJobDB(t)
	j, e := nouveauJobTest(t, db, 6*time.Hour)

	transfere := maintenantTest.Add(-9 * 24 * time.Hour)
	insererMail(t, db, maintenantTest.Add(-10*24*time.Hour), &transfere, false)

	j.VerifierBlocage(context.Background(), maintenantTest)

	assert.Empty(t, e.messages)
}

// Le job tourne toutes les heures : le blocage persiste, mais on n'alerte pas à chaque tour.
func TestMailWatchJob_BlocagePersistant_UneSeuleAlerteParFenetre(t *testing.T) {
	db := setupJobDB(t)
	j, e := nouveauJobTest(t, db, 6*time.Hour)

	insererMail(t, db, maintenantTest.Add(-13*24*time.Hour), nil, false)

	j.VerifierBlocage(context.Background(), maintenantTest)
	j.VerifierBlocage(context.Background(), maintenantTest.Add(time.Hour))

	assert.Len(t, e.messages, 1)
}

// Avant l'heure du récapitulatif, rien ne part — même s'il y a eu de l'activité.
func TestMailWatchJob_DigestAvantLHeure_NePublieRien(t *testing.T) {
	db := setupJobDB(t)
	j, e := nouveauJobTest(t, db, 6*time.Hour)

	j.PublierDigestSiHeure(context.Background(), maintenantTest) // 12h UTC, digest à 21h

	assert.Empty(t, e.messages)
}

func TestNewMailWatchJob_SansSurveillanceDesactive(t *testing.T) {
	db := setupJobDB(t)
	assert.Nil(t, jobs.NewMailWatchJob(db, nil, time.Hour, 21))
	assert.Nil(t, jobs.NewMailWatchJob(nil, nil, time.Hour, 21))
}

// Vécu le 19/09/2026 10:28 UTC : trois offres Malt datées du 17 et du 18/09, relevées en IMAP
// seulement ce matin-là, et l'alerte « relai SMTP ne répond pas » est partie AVANT même leur première
// tentative de transfert (qui a réussi 13 s plus tard). L'attente d'un TRANSFERT se mesure depuis la
// relève, pas depuis la date inscrite dans le mail.
func TestMailWatchJob_ReleveTardive_PasDeFausseAlerteTransfert(t *testing.T) {
	db := setupJobDB(t)
	j, e := nouveauJobTest(t, db, 6*time.Hour)

	insererMailReleve(t, db, maintenantTest.Add(-2*24*time.Hour), maintenantTest.Add(-time.Minute), nil, false)

	j.VerifierBlocage(context.Background(), maintenantTest)

	assert.Empty(t, e.messages, "relevé il y a une minute : il attend son premier essai, il n'est pas bloqué")
}

// nouveauJobAvecWatch rend aussi le MailWatch, pour pouvoir marquer des relèves.
func nouveauJobAvecWatch(t *testing.T, db *gorm.DB) (*jobs.MailWatchJob, *services.MailWatch, *webhookEspion) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	e := nouveauWebhookEspion(t)
	watch := services.NewMailWatch(rdb, services.NewDiscordNotifier(e.srv.URL, "243388469169094656"))
	j := jobs.NewMailWatchJob(db, watch, 6*time.Hour, 21)
	require.NotNil(t, j)
	return j, watch, e
}

// Le vrai blocage du 17 au 19/09 : la relève IMAP ne lisait plus rien, et RIEN ne le disait — un mail
// jamais relevé n'existe pas en base, aucune alerte de transfert ne peut le voir. La seule trace est
// l'âge de la dernière relève réussie.
func TestMailWatchJob_ReleveMuette_Alerte(t *testing.T) {
	db := setupJobDB(t)
	j, watch, e := nouveauJobAvecWatch(t, db)

	watch.MarquerReleve(context.Background(), maintenantTest.Add(-3*time.Hour))
	j.VerifierReleve(context.Background(), maintenantTest)

	require.Len(t, e.messages, 1)
	assert.Contains(t, e.messages[0], "Relève IMAP muette")
	assert.Contains(t, e.messages[0], "30/08/2026 09:00 UTC", "l'alerte doit dater la dernière relève réussie")
}

func TestMailWatchJob_ReleveRecente_PasDAlerte(t *testing.T) {
	db := setupJobDB(t)
	j, watch, e := nouveauJobAvecWatch(t, db)

	watch.MarquerReleve(context.Background(), maintenantTest.Add(-4*time.Minute))
	j.VerifierReleve(context.Background(), maintenantTest)

	assert.Empty(t, e.messages)
}

// Boîte non configurée (ou jamais relevée) : aucune marque, donc rien à dire — pas une panne.
func TestMailWatchJob_AucuneReleveEnregistree_PasDAlerte(t *testing.T) {
	db := setupJobDB(t)
	j, _, e := nouveauJobAvecWatch(t, db)

	j.VerifierReleve(context.Background(), maintenantTest)

	assert.Empty(t, e.messages)
}

// Le job tourne toutes les heures : une relève muette qui dure n'alerte qu'une fois par fenêtre.
func TestMailWatchJob_ReleveMuettePersistante_UneSeuleAlerteParFenetre(t *testing.T) {
	db := setupJobDB(t)
	j, watch, e := nouveauJobAvecWatch(t, db)

	watch.MarquerReleve(context.Background(), maintenantTest.Add(-3*time.Hour))
	j.VerifierReleve(context.Background(), maintenantTest)
	j.VerifierReleve(context.Background(), maintenantTest.Add(10*time.Minute))

	assert.Len(t, e.messages, 1)
}
