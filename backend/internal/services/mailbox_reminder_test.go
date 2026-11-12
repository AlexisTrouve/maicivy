package services_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"mime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"maicivy/internal/models"
	"maicivy/internal/services"
)

// Lien « ✓ 标记为已处理 » + rappels des opportunités ouvertes (cf. mailbox_reminder.go).

const testAckSecret = "secret-hmac"

// sujetDecode lit l'en-tête Subject et le décode (RFC 2047) : le sujet porte désormais du chinois,
// il est donc encodé sur le fil. On asserte ce que la destinataire LIT, pas les octets transmis.
func sujetDecode(t *testing.T, msg string) string {
	t.Helper()
	for _, l := range strings.Split(msg, "\r\n") {
		if v, ok := strings.CutPrefix(l, "Subject: "); ok {
			dec, err := new(mime.WordDecoder).DecodeHeader(v)
			require.NoError(t, err)
			return dec
		}
	}
	t.Fatalf("pas d'en-tête Subject:\n%s", msg)
	return ""
}

// rappelSvc : service avec filtre de pertinence (seuil 50) et lien « traité » actif.
func rappelSvc(db *gorm.DB, relay *fakeRelay) *services.MailboxService {
	svc := newMailboxServiceWithRelevance(db, &fakeFetcher{}, relay, &fakeRelevance{threshold: 50})
	svc.SetAckLink("https://maicivy.example", testAckSecret)
	return svc
}

// opportuniteTransferee insère une opportunité déjà transférée il y a `age`.
func opportuniteTransferee(t *testing.T, db *gorm.DB, score int, age time.Duration, now time.Time) uuid.UUID {
	t.Helper()
	fwd := now.Add(-age)
	e := models.MailboxEmail{
		MessageID: uuid.NewString() + "@malt.fr", FromAddress: "n@malt.fr", FromDomain: "malt.fr",
		Platform: "malt", Subject: "Mission Go", BodyText: "Corps de l'offre.", ReceivedAt: fwd,
		IsOpportunity: true, RelevanceScore: &score, ForwardedAt: &fwd,
	}
	e.ID = uuid.New()
	require.NoError(t, db.Create(&e).Error)
	return e.ID
}

// Le sujet porte le score, un glyphe et la priorité EN CHINOIS, à trois paliers.
func TestMailboxForward_SujetScorePrioriteChinois(t *testing.T) {
	for _, c := range []struct {
		score int
		tag   string
	}{
		{87, "[malt][87 ✓ 优先处理] Mission"},
		{75, "[malt][75 ✓ 优先处理] Mission"},
		{62, "[malt][62 ? 待评估] Mission"},
		{20, "[malt][20 ✕ 建议拒绝] Mission"},
	} {
		db := setupMailboxTestDB(t)
		require.NoError(t, db.Create(&models.MailboxCursor{MailboxKey: "primary", UIDValidity: 1}).Error)
		relay := &fakeRelay{}
		fetcher := &fakeFetcher{uidValidity: 1, uidNext: 2, messages: []services.ImapMessage{
			{UID: 1, MessageID: "m@malt.fr", From: "n@malt.fr", Subject: "Mission", ReceivedAt: time.Now()},
		}}
		rel := &fakeRelevance{verdict: services.MailboxRelevanceVerdict{IsOpportunity: true, Score: c.score}, threshold: 50}
		require.NoError(t, newMailboxServiceWithRelevance(db, fetcher, relay, rel).PollOnce(context.Background()))
		require.Len(t, relay.calls, 1)
		assert.Equal(t, c.tag, sujetDecode(t, relay.calls[0].msg), "score %d", c.score)
	}
}

// Une opportunité transférée porte le lien signé ; une newsletter n'en porte pas (rien à traiter).
func TestMailboxForward_LienTraiteSurLesSeulesOpportunites(t *testing.T) {
	db := setupMailboxTestDB(t)
	require.NoError(t, db.Create(&models.MailboxCursor{MailboxKey: "primary", UIDValidity: 1}).Error)
	relay := &fakeRelay{}
	fetcher := &fakeFetcher{uidValidity: 1, uidNext: 2, messages: []services.ImapMessage{
		{UID: 1, MessageID: "op@malt.fr", From: "n@malt.fr", Subject: "Mission", ReceivedAt: time.Now()},
	}}
	rel := &fakeRelevance{verdict: services.MailboxRelevanceVerdict{IsOpportunity: true, Score: 80}, threshold: 50}
	svc := newMailboxServiceWithRelevance(db, fetcher, relay, rel)
	svc.SetAckLink("https://maicivy.example/", testAckSecret)
	require.NoError(t, svc.PollOnce(context.Background()))

	var e models.MailboxEmail
	require.NoError(t, db.First(&e, "message_id = ?", "op@malt.fr").Error)
	attendu := "https://maicivy.example/api/v1/mailbox/ack/" + e.ID.String() + "?sig=" + services.MailboxAckSig(e.ID.String(), testAckSecret)
	require.Len(t, relay.calls, 1)
	corps := corpsDecode(t, relay.calls[0].msg)
	assert.Contains(t, corps, attendu, "le lien doit sortir intact (pas coupé) une fois décodé")
	assert.Contains(t, corps, "已处理")

	// Newsletter : aucun lien.
	db2 := setupMailboxTestDB(t)
	require.NoError(t, db2.Create(&models.MailboxCursor{MailboxKey: "primary", UIDValidity: 1}).Error)
	relay2 := &fakeRelay{}
	fetcher2 := &fakeFetcher{uidValidity: 1, uidNext: 2, messages: []services.ImapMessage{
		{UID: 1, MessageID: "news@malt.fr", From: "n@malt.fr", Subject: "Newsletter", ReceivedAt: time.Now()},
	}}
	svc2 := newMailboxServiceWithRelevance(db2, fetcher2, relay2, &fakeRelevance{threshold: 50})
	svc2.SetAckLink("https://maicivy.example", testAckSecret)
	require.NoError(t, svc2.PollOnce(context.Background()))
	require.Len(t, relay2.calls, 1)
	assert.NotContains(t, corpsDecode(t, relay2.calls[0].msg), "/mailbox/ack/")
}

// La signature du lien n'est PAS un cookie de session valide pour le même uuid : le secret est
// partagé avec les sessions (HMAC(uuid)), seul le préfixe de domaine les sépare.
func TestMailboxAckSig_DomaineSepareDeLaSession(t *testing.T) {
	id := uuid.NewString()
	mac := hmac.New(sha256.New, []byte(testAckSecret))
	mac.Write([]byte(id))
	sigSession := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	assert.NotEqual(t, sigSession, services.MailboxAckSig(id, testAckSecret))
	assert.True(t, services.VerifyMailboxAck(id, services.MailboxAckSig(id, testAckSecret), testAckSecret))
	assert.False(t, services.VerifyMailboxAck(id, services.MailboxAckSig(id, ""), ""), "secret vide = tout refuser")
}

// Parcours complet : rien avant 4 h, rappel 1/2 à 4 h, rien juste après, rappel 2/2 à 12 h, puis plus rien.
func TestMailboxRappels_CalendrierBorne(t *testing.T) {
	db := setupMailboxTestDB(t)
	relay := &fakeRelay{}
	svc := rappelSvc(db, relay)
	t0 := time.Now()
	id := opportuniteTransferee(t, db, 87, 0, t0)
	ctx := context.Background()

	svc.SendReminders(ctx, t0.Add(3*time.Hour))
	assert.Empty(t, relay.calls, "aucun rappel avant 4 h")

	svc.SendReminders(ctx, t0.Add(4*time.Hour+time.Minute))
	require.Len(t, relay.calls, 1)
	msg := relay.calls[0].msg
	assert.Equal(t, "[malt][提醒 1/2][87 ✓ 优先处理] Mission Go", sujetDecode(t, msg))
	corps := corpsDecode(t, msg)
	assert.Contains(t, corps, "⏰ 提醒 1/2")
	assert.Contains(t, corps, "/api/v1/mailbox/ack/"+id.String())
	assert.Contains(t, corps, "Corps de l'offre.", "le rappel porte l'offre entière")

	svc.SendReminders(ctx, t0.Add(4*time.Hour+3*time.Minute))
	assert.Len(t, relay.calls, 1, "pas de second rappel au cycle suivant")

	svc.SendReminders(ctx, t0.Add(12*time.Hour+time.Minute))
	require.Len(t, relay.calls, 2)
	assert.Equal(t, "[malt][提醒 2/2][87 ✓ 优先处理] Mission Go", sujetDecode(t, relay.calls[1].msg))

	svc.SendReminders(ctx, t0.Add(30*time.Hour))
	assert.Len(t, relay.calls, 2, "jamais plus de deux rappels")
}

// Un clic « traité » arrête tout.
func TestMailboxRappels_ArretesParLeClic(t *testing.T) {
	db := setupMailboxTestDB(t)
	relay := &fakeRelay{}
	svc := rappelSvc(db, relay)
	t0 := time.Now()
	id := opportuniteTransferee(t, db, 87, 0, t0)

	deja, err := services.MarkHandled(db, id, t0.Add(time.Hour))
	require.NoError(t, err)
	assert.False(t, deja)

	svc.SendReminders(context.Background(), t0.Add(5*time.Hour))
	assert.Empty(t, relay.calls, "une offre traitée n'est plus relancée")
}

// Sous le seuil, au-delà de 48 h, ou sans secret : pas de rappel.
func TestMailboxRappels_HorsPerimetre(t *testing.T) {
	t0 := time.Now()
	ctx := context.Background()

	db := setupMailboxTestDB(t)
	relay := &fakeRelay{}
	svc := rappelSvc(db, relay)
	opportuniteTransferee(t, db, 20, 5*time.Hour, t0)  // « ✕ 建议拒绝 »
	opportuniteTransferee(t, db, 90, 49*time.Hour, t0) // trop vieille
	svc.SendReminders(ctx, t0)
	assert.Empty(t, relay.calls)

	db2 := setupMailboxTestDB(t)
	relay2 := &fakeRelay{}
	sansLien := newMailboxServiceWithRelevance(db2, &fakeFetcher{}, relay2, &fakeRelevance{threshold: 50})
	opportuniteTransferee(t, db2, 90, 5*time.Hour, t0)
	sansLien.SendReminders(ctx, t0)
	assert.Empty(t, relay2.calls, "sans lien pour l'arrêter, on ne relance pas")
}

// Au déploiement, une offre de 20 h a dépassé les DEUX délais : un seul rappel part, le second
// attend l'écart minimal au lieu de suivre deux minutes plus tard.
func TestMailboxRappels_RattrapageEspace(t *testing.T) {
	db := setupMailboxTestDB(t)
	relay := &fakeRelay{}
	svc := rappelSvc(db, relay)
	t0 := time.Now()
	opportuniteTransferee(t, db, 87, 20*time.Hour, t0)

	svc.SendReminders(context.Background(), t0)
	svc.SendReminders(context.Background(), t0.Add(2*time.Minute))
	assert.Len(t, relay.calls, 1)

	svc.SendReminders(context.Background(), t0.Add(4*time.Hour+time.Minute))
	assert.Len(t, relay.calls, 2)
}

// Un rappel en échec SMTP n'avance pas le compteur : il repart au cycle suivant.
func TestMailboxRappels_EchecRetente(t *testing.T) {
	db := setupMailboxTestDB(t)
	relay := &fakeRelay{failing: true}
	svc := rappelSvc(db, relay)
	t0 := time.Now()
	id := opportuniteTransferee(t, db, 87, 5*time.Hour, t0)

	svc.SendReminders(context.Background(), t0)
	var e models.MailboxEmail
	require.NoError(t, db.First(&e, "id = ?", id).Error)
	assert.Equal(t, 0, e.RemindersSent)

	relay.failing = false
	svc.SendReminders(context.Background(), t0.Add(2*time.Minute))
	require.NoError(t, db.First(&e, "id = ?", id).Error)
	assert.Equal(t, 1, e.RemindersSent)
}
