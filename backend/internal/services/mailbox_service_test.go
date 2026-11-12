package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"maicivy/internal/models"
	"maicivy/internal/services"
)

func setupMailboxTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.MailboxEmail{}, &models.MailboxCursor{}))
	return db
}

// --- fake ImapFetcher : boîte IMAP en mémoire, aucun réseau ---

type fakeFetcher struct {
	uidValidity uint32
	uidNext     uint32
	messages    []services.ImapMessage
	statusErr   error
	fetchErr    error
	fetchCalls  int
}

func (f *fakeFetcher) MailboxStatus(ctx context.Context) (uint32, uint32, error) {
	if f.statusErr != nil {
		return 0, 0, f.statusErr
	}
	return f.uidValidity, f.uidNext, nil
}

func (f *fakeFetcher) FetchSince(ctx context.Context, sinceUID uint32) ([]services.ImapMessage, error) {
	f.fetchCalls++
	if f.fetchErr != nil {
		return nil, f.fetchErr
	}
	var out []services.ImapMessage
	for _, m := range f.messages {
		if m.UID > sinceUID {
			out = append(out, m)
		}
	}
	return out, nil
}

func (f *fakeFetcher) Close() error { return nil }

// --- fake relay SMTP : enregistre les envois, aucun réseau ---

type relayCall struct {
	addr, from, to, msg string
}

type fakeRelay struct {
	calls   []relayCall
	failing bool
}

func (r *fakeRelay) send(addr, from, to string, msg []byte) error {
	r.calls = append(r.calls, relayCall{addr, from, to, string(msg)})
	if r.failing {
		return errors.New("relay down")
	}
	return nil
}

func testAllowlist() map[string]string {
	return services.ParseMailboxAllowlist("malt.fr:malt")
}

func newMailboxService(db *gorm.DB, fetcher services.ImapFetcher, relay *fakeRelay) *services.MailboxService {
	return newMailboxServiceWithRelevance(db, fetcher, relay, nil)
}

func newMailboxServiceWithRelevance(db *gorm.DB, fetcher services.ImapFetcher, relay *fakeRelay, relevance services.MailboxRelevanceEvaluator) *services.MailboxService {
	return services.NewMailboxService(db, fetcher, testAllowlist(), "smtp.local:25", "dest@example.com", "mailbox@etheryale.com", "Mailbox Etheryale", relay.send, relevance)
}

// Premier démarrage (pas de curseur) : on seed à UIDNext-1 SANS fetcher — l'historique existant de la
// boîte ne doit jamais être ingéré ni transféré.
func TestMailboxService_ColdStart_NoBackfill(t *testing.T) {
	db := setupMailboxTestDB(t)
	fetcher := &fakeFetcher{uidValidity: 100, uidNext: 50, messages: []services.ImapMessage{
		{UID: 10, MessageID: "old@malt.fr", From: "n@malt.fr", Subject: "old", ReceivedAt: time.Now()},
	}}
	relay := &fakeRelay{}
	svc := newMailboxService(db, fetcher, relay)

	require.NoError(t, svc.PollOnce(context.Background()))

	assert.Equal(t, 0, fetcher.fetchCalls, "cold start ne doit jamais fetcher")

	var cursor models.MailboxCursor
	require.NoError(t, db.Where("mailbox_key = ?", "primary").First(&cursor).Error)
	assert.Equal(t, uint32(100), cursor.UIDValidity)
	assert.Equal(t, uint32(49), cursor.LastUID)

	var count int64
	db.Model(&models.MailboxEmail{}).Count(&count)
	assert.Equal(t, int64(0), count, "aucun mail ne doit être ingéré au cold start")
	assert.Empty(t, relay.calls)
}

// Changement d'UIDValidity (boîte recréée) : même comportement que le cold start — reset sans fetch.
func TestMailboxService_UIDValidityChanged_ResetsWithoutFetch(t *testing.T) {
	db := setupMailboxTestDB(t)
	require.NoError(t, db.Create(&models.MailboxCursor{MailboxKey: "primary", UIDValidity: 1, LastUID: 999}).Error)

	fetcher := &fakeFetcher{uidValidity: 2, uidNext: 5}
	relay := &fakeRelay{}
	svc := newMailboxService(db, fetcher, relay)

	require.NoError(t, svc.PollOnce(context.Background()))
	assert.Equal(t, 0, fetcher.fetchCalls)

	var cursor models.MailboxCursor
	require.NoError(t, db.Where("mailbox_key = ?", "primary").First(&cursor).Error)
	assert.Equal(t, uint32(2), cursor.UIDValidity)
	assert.Equal(t, uint32(4), cursor.LastUID)
}

// Fetch incrémental : seuls les domaines allowlistés (exact + sous-domaine, PAS le bypass de
// frontière) sont stockés — mais le curseur avance sur TOUS les messages inspectés.
func TestMailboxService_IncrementalFetch_AllowlistFilter(t *testing.T) {
	db := setupMailboxTestDB(t)
	require.NoError(t, db.Create(&models.MailboxCursor{MailboxKey: "primary", UIDValidity: 1, LastUID: 5}).Error)

	now := time.Now()
	fetcher := &fakeFetcher{
		uidValidity: 1, uidNext: 12,
		messages: []services.ImapMessage{
			{UID: 6, MessageID: "m6@malt.fr", From: "notif@malt.fr", Subject: "Mission 1", BodyText: "corps 1", ReceivedAt: now},
			{UID: 7, MessageID: "m7@gmail.com", From: "friend@gmail.com", Subject: "perso", BodyText: "corps perso", ReceivedAt: now},
			{UID: 8, MessageID: "m8@evilmalt.fr", From: "attacker@evilmalt.fr", Subject: "phishing", BodyText: "corps phishing", ReceivedAt: now},
			{UID: 9, MessageID: "m9@sub.malt.fr", From: "notif@sub.malt.fr", Subject: "Mission 2", BodyText: "corps 2", ReceivedAt: now},
		},
	}
	relay := &fakeRelay{}
	svc := newMailboxService(db, fetcher, relay)

	require.NoError(t, svc.PollOnce(context.Background()))

	var emails []models.MailboxEmail
	db.Order("imap_uid asc").Find(&emails)
	require.Len(t, emails, 2, "seuls malt.fr et sub.malt.fr doivent être retenus")
	assert.Equal(t, "notif@malt.fr", emails[0].FromAddress)
	assert.Equal(t, "malt", emails[0].Platform)
	assert.Equal(t, "notif@sub.malt.fr", emails[1].FromAddress)

	var cursor models.MailboxCursor
	require.NoError(t, db.Where("mailbox_key = ?", "primary").First(&cursor).Error)
	assert.Equal(t, uint32(9), cursor.LastUID, "le curseur doit avancer sur TOUS les messages inspectés, pas seulement les retenus")

	assert.Len(t, relay.calls, 2, "les 2 mails retenus doivent être transférés immédiatement à l'ingestion")
}

// Échec de transfert à l'ingestion → l'erreur est enregistrée, ForwardedAt reste nil (candidat au
// retry du prochain cycle).
func TestMailboxService_ForwardFailure_RecordsErrorForRetry(t *testing.T) {
	db := setupMailboxTestDB(t)
	require.NoError(t, db.Create(&models.MailboxCursor{MailboxKey: "primary", UIDValidity: 1, LastUID: 0}).Error)

	fetcher := &fakeFetcher{
		uidValidity: 1, uidNext: 2,
		messages: []services.ImapMessage{
			{UID: 1, MessageID: "fail@malt.fr", From: "notif@malt.fr", Subject: "Mission", ReceivedAt: time.Now()},
		},
	}
	relay := &fakeRelay{failing: true}
	svc := newMailboxService(db, fetcher, relay)
	require.NoError(t, svc.PollOnce(context.Background()))

	var email models.MailboxEmail
	require.NoError(t, db.Where("message_id = ?", "fail@malt.fr").First(&email).Error)
	assert.Nil(t, email.ForwardedAt)
	assert.Contains(t, email.ForwardError, "relay down")
}

// Le mécanisme de retry : un mail stocké mais jamais transféré est retenté au cycle suivant, avant
// même le fetch incrémental.
func TestMailboxService_RetryFailedForwards(t *testing.T) {
	db := setupMailboxTestDB(t)
	require.NoError(t, db.Create(&models.MailboxCursor{MailboxKey: "primary", UIDValidity: 1, LastUID: 5}).Error)

	stuck := models.MailboxEmail{
		MessageID: "stuck@malt.fr", ImapUID: 3, FromAddress: "notif@malt.fr", FromDomain: "malt.fr",
		Platform: "malt", Subject: "old fail", ReceivedAt: time.Now().Add(-time.Hour),
	}
	require.NoError(t, db.Create(&stuck).Error)

	fetcher := &fakeFetcher{uidValidity: 1, uidNext: 6} // rien de nouveau à fetcher
	relay := &fakeRelay{}
	svc := newMailboxService(db, fetcher, relay)
	require.NoError(t, svc.PollOnce(context.Background()))

	var reloaded models.MailboxEmail
	require.NoError(t, db.First(&reloaded, "id = ?", stuck.ID).Error)
	assert.NotNil(t, reloaded.ForwardedAt, "le retry doit avoir marqué le mail comme transféré")
	require.Len(t, relay.calls, 1)
	assert.Equal(t, "dest@example.com", relay.calls[0].to)
}

// Dédup : un Message-ID déjà en base (rejeu après crash — le poll précédent a stocké le mail mais
// n'a pas pu persister l'avance du curseur) ne doit jamais créer une seconde ligne. Le curseur avance
// quand même.
func TestMailboxService_Dedup_SameMessageIDNotStoredTwice(t *testing.T) {
	db := setupMailboxTestDB(t)
	require.NoError(t, db.Create(&models.MailboxCursor{MailboxKey: "primary", UIDValidity: 1, LastUID: 0}).Error)

	require.NoError(t, db.Create(&models.MailboxEmail{
		MessageID: "dup@malt.fr", ImapUID: 1, FromAddress: "notif@malt.fr", FromDomain: "malt.fr",
		Platform: "malt", Subject: "déjà stocké", ReceivedAt: time.Now(),
	}).Error)

	fetcher := &fakeFetcher{
		uidValidity: 1, uidNext: 2,
		messages: []services.ImapMessage{
			{UID: 1, MessageID: "dup@malt.fr", From: "notif@malt.fr", Subject: "rejeu", ReceivedAt: time.Now()},
		},
	}
	relay := &fakeRelay{}
	svc := newMailboxService(db, fetcher, relay)
	require.NoError(t, svc.PollOnce(context.Background()))

	var count int64
	db.Model(&models.MailboxEmail{}).Where("message_id = ?", "dup@malt.fr").Count(&count)
	assert.Equal(t, int64(1), count, "le même Message-ID ne doit jamais être stocké deux fois")

	var cursor models.MailboxCursor
	require.NoError(t, db.Where("mailbox_key = ?", "primary").First(&cursor).Error)
	assert.Equal(t, uint32(1), cursor.LastUID, "le curseur doit quand même avancer")
}

// RetryForward (retry manuel depuis l'admin panel) : succès → ForwardedAt posé, erreur retournée nil.
func TestMailboxService_RetryForward_Success(t *testing.T) {
	db := setupMailboxTestDB(t)
	email := models.MailboxEmail{
		MessageID: "manual@malt.fr", ImapUID: 1, FromAddress: "notif@malt.fr", FromDomain: "malt.fr",
		Platform: "malt", Subject: "à retenter", ReceivedAt: time.Now(),
	}
	require.NoError(t, db.Create(&email).Error)

	relay := &fakeRelay{}
	svc := newMailboxService(db, &fakeFetcher{}, relay)

	require.NoError(t, svc.RetryForward(context.Background(), email.ID))

	var reloaded models.MailboxEmail
	require.NoError(t, db.First(&reloaded, "id = ?", email.ID).Error)
	assert.NotNil(t, reloaded.ForwardedAt)
	require.Len(t, relay.calls, 1)
}

// RetryForward : échec de transfert → l'erreur est retournée ET persistée pour le prochain retry auto.
func TestMailboxService_RetryForward_Failure(t *testing.T) {
	db := setupMailboxTestDB(t)
	email := models.MailboxEmail{
		MessageID: "manual-fail@malt.fr", ImapUID: 1, FromAddress: "notif@malt.fr", FromDomain: "malt.fr",
		Platform: "malt", Subject: "échoue", ReceivedAt: time.Now(),
	}
	require.NoError(t, db.Create(&email).Error)

	relay := &fakeRelay{failing: true}
	svc := newMailboxService(db, &fakeFetcher{}, relay)

	err := svc.RetryForward(context.Background(), email.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "relay down")

	var reloaded models.MailboxEmail
	require.NoError(t, db.First(&reloaded, "id = ?", email.ID).Error)
	assert.Nil(t, reloaded.ForwardedAt)
	assert.Contains(t, reloaded.ForwardError, "relay down")
}

// RetryForward sur un ID inconnu → erreur explicite (pas de panic, pas de silence).
func TestMailboxService_RetryForward_NotFound(t *testing.T) {
	db := setupMailboxTestDB(t)
	relay := &fakeRelay{}
	svc := newMailboxService(db, &fakeFetcher{}, relay)

	err := svc.RetryForward(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Empty(t, relay.calls)
}

// --- fake MailboxRelevanceEvaluator : jugement en mémoire, aucun appel LLM réel ---

type fakeRelevance struct {
	verdict   services.MailboxRelevanceVerdict
	err       error
	threshold int
	calls     int
}

func (f *fakeRelevance) Evaluate(ctx context.Context, subject, body string) (services.MailboxRelevanceVerdict, error) {
	f.calls++
	if f.err != nil {
		return services.MailboxRelevanceVerdict{}, f.err
	}
	return f.verdict, nil
}

func (f *fakeRelevance) Threshold() int { return f.threshold }

// Opportunité SOUS le seuil → transfert quand même (l'opératrice doit tout voir), mais le message porte la
// consigne « ✕ 建议拒绝 » dans le sujet et « REFUSE » dans le corps — jamais un blocage. La consigne est en anglais (la
// destinataire ne lit pas le français) et explique POURQUOI refuser (une offre en attente bloque les
// autres sur Malt).
func TestMailboxService_Relevance_LowScore_ForwardsWithRefuseAction(t *testing.T) {
	db := setupMailboxTestDB(t)
	require.NoError(t, db.Create(&models.MailboxCursor{MailboxKey: "primary", UIDValidity: 1, LastUID: 0}).Error)

	fetcher := &fakeFetcher{
		uidValidity: 1, uidNext: 2,
		messages: []services.ImapMessage{
			{UID: 1, MessageID: "low@malt.fr", From: "notif@malt.fr", Subject: "Mission hors profil", ReceivedAt: time.Now()},
		},
	}
	relay := &fakeRelay{}
	relevance := &fakeRelevance{
		verdict:   services.MailboxRelevanceVerdict{IsOpportunity: true, Score: 20, Reason: "out of profile"},
		threshold: 50,
	}
	svc := newMailboxServiceWithRelevance(db, fetcher, relay, relevance)

	require.NoError(t, svc.PollOnce(context.Background()))

	var email models.MailboxEmail
	require.NoError(t, db.Where("message_id = ?", "low@malt.fr").First(&email).Error)
	assert.True(t, email.IsOpportunity)
	require.NotNil(t, email.RelevanceScore)
	assert.Equal(t, 20, *email.RelevanceScore)
	assert.False(t, email.ForwardBlocked, "le score bas ne doit plus jamais bloquer le transfert")
	assert.NotNil(t, email.ForwardedAt)
	require.Len(t, relay.calls, 1)
	msg := relay.calls[0].msg
	assert.Equal(t, "[malt][20 ✕ 建议拒绝] Mission hors profil", sujetDecode(t, msg), "sujet : score + refus")
	assert.Contains(t, msg, "REFUSE this offer on Malt")
	assert.Contains(t, msg, "20/100")
	assert.Contains(t, msg, "out of profile")
	assert.Contains(t, msg, "blocks the other offers", "doit expliquer pourquoi refuser")
	assert.NotContains(t, msg, "Pertinence faible", "plus d'ancien texte français")
}

// Opportunité AU-DESSUS du seuil → consigne « postuler sinon refuser » (postuler, sinon refuser) : même au-dessus du
// seuil, une offre laissée en attente bloque la file, donc elle doit finir postulée OU refusée.
func TestMailboxService_Relevance_HighScore_ForwardsWithReviewAction(t *testing.T) {
	db := setupMailboxTestDB(t)
	require.NoError(t, db.Create(&models.MailboxCursor{MailboxKey: "primary", UIDValidity: 1, LastUID: 0}).Error)

	fetcher := &fakeFetcher{
		uidValidity: 1, uidNext: 2,
		messages: []services.ImapMessage{
			{UID: 1, MessageID: "high@malt.fr", From: "notif@malt.fr", Subject: "Good mission", ReceivedAt: time.Now()},
		},
	}
	relay := &fakeRelay{}
	relevance := &fakeRelevance{
		verdict:   services.MailboxRelevanceVerdict{IsOpportunity: true, Score: 80, Reason: "great fit"},
		threshold: 50,
	}
	svc := newMailboxServiceWithRelevance(db, fetcher, relay, relevance)

	require.NoError(t, svc.PollOnce(context.Background()))

	require.Len(t, relay.calls, 1)
	msg := relay.calls[0].msg
	// Le TAG est dans le SUJET (en-tête, lisible tel quel) ; la consigne est dans le CORPS, qui est
	// encodé en quoted-printable — d'où le décodage avant assertion. Sans lui, une phrase coupée par
	// un retour souple (« otherwise =\r\nREFUSE ») ferait échouer un test sur un contenu pourtant
	// intact à la lecture.
	assert.Equal(t, "[malt][80 ✓ 优先处理] Good mission", sujetDecode(t, msg), "une bonne opportunité porte aussi un tag")
	assert.NotContains(t, sujetDecode(t, msg), "拒绝", "au-dessus du seuil, pas de refus imposé")

	corps := corpsDecode(t, msg)
	assert.Contains(t, corps, "Apply if it fits")
	assert.Contains(t, corps, "otherwise REFUSE")
	assert.Contains(t, corps, "80/100")
}

// Non-opportunité (newsletter/notif) → AUCUNE consigne : rien à refuser sur Malt. Le mail est transféré
// tel quel, tag plateforme seul.
func TestMailboxService_Relevance_NonOpportunity_NoAction(t *testing.T) {
	db := setupMailboxTestDB(t)
	require.NoError(t, db.Create(&models.MailboxCursor{MailboxKey: "primary", UIDValidity: 1, LastUID: 0}).Error)

	fetcher := &fakeFetcher{
		uidValidity: 1, uidNext: 2,
		messages: []services.ImapMessage{
			{UID: 1, MessageID: "news@malt.fr", From: "notif@malt.fr", Subject: "Newsletter", ReceivedAt: time.Now()},
		},
	}
	relay := &fakeRelay{}
	relevance := &fakeRelevance{
		verdict:   services.MailboxRelevanceVerdict{IsOpportunity: false, Reason: "digest"},
		threshold: 50,
	}
	svc := newMailboxServiceWithRelevance(db, fetcher, relay, relevance)

	require.NoError(t, svc.PollOnce(context.Background()))

	require.Len(t, relay.calls, 1)
	msg := relay.calls[0].msg
	assert.Equal(t, "[malt] Newsletter", sujetDecode(t, msg), "non-opportunité : tag plateforme seul")
	assert.NotNil(t, mustFindEmail(t, db, "news@malt.fr").ForwardedAt)
}

func mustFindEmail(t *testing.T, db *gorm.DB, messageID string) models.MailboxEmail {
	t.Helper()
	var email models.MailboxEmail
	require.NoError(t, db.Where("message_id = ?", messageID).First(&email).Error)
	return email
}

// Opportunité au-dessus du seuil → transfert auto normal, verdict persisté, AUCUN avertissement dans
// le message transféré (on ne pollue pas les vraies bonnes opportunités).
func TestMailboxService_Relevance_ForwardsHighScoreOpportunity(t *testing.T) {
	db := setupMailboxTestDB(t)
	require.NoError(t, db.Create(&models.MailboxCursor{MailboxKey: "primary", UIDValidity: 1, LastUID: 0}).Error)

	fetcher := &fakeFetcher{
		uidValidity: 1, uidNext: 2,
		messages: []services.ImapMessage{
			{UID: 1, MessageID: "high@malt.fr", From: "notif@malt.fr", Subject: "Mission Go parfaite", ReceivedAt: time.Now()},
		},
	}
	relay := &fakeRelay{}
	relevance := &fakeRelevance{
		verdict: services.MailboxRelevanceVerdict{
			IsOpportunity: true, Score: 80, Reason: "bon match",
			CoT: "Checked get_experience: 3 Go backend roles found.", Link: "https://malt.fr/mission/123",
		},
		threshold: 50,
	}
	svc := newMailboxServiceWithRelevance(db, fetcher, relay, relevance)

	require.NoError(t, svc.PollOnce(context.Background()))

	var email models.MailboxEmail
	require.NoError(t, db.Where("message_id = ?", "high@malt.fr").First(&email).Error)
	assert.True(t, email.IsOpportunity)
	require.NotNil(t, email.RelevanceScore)
	assert.Equal(t, 80, *email.RelevanceScore)
	assert.Equal(t, "Checked get_experience: 3 Go backend roles found.", email.RelevanceCoT)
	assert.Equal(t, "https://malt.fr/mission/123", email.RelevanceLink)
	assert.False(t, email.ForwardBlocked)
	assert.NotNil(t, email.ForwardedAt)
	require.Len(t, relay.calls, 1)
	// Bonne opportunité : tag de priorité, jamais de refus.
	assert.Equal(t, "[malt][80 ✓ 优先处理] Mission Go parfaite", sujetDecode(t, relay.calls[0].msg))
}

// Newsletter/digest (pas une opportunité) → jamais scoré, jamais bloqué, transfert inchangé.
func TestMailboxService_Relevance_BypassesNonOpportunity(t *testing.T) {
	db := setupMailboxTestDB(t)
	require.NoError(t, db.Create(&models.MailboxCursor{MailboxKey: "primary", UIDValidity: 1, LastUID: 0}).Error)

	fetcher := &fakeFetcher{
		uidValidity: 1, uidNext: 2,
		messages: []services.ImapMessage{
			{UID: 1, MessageID: "newsletter@malt.fr", From: "community-fr@malt.fr", Subject: "Vos prochains rendez-vous Malt", ReceivedAt: time.Now()},
		},
	}
	relay := &fakeRelay{}
	relevance := &fakeRelevance{
		verdict:   services.MailboxRelevanceVerdict{IsOpportunity: false, Score: 0, Reason: "newsletter, pas une mission"},
		threshold: 50,
	}
	svc := newMailboxServiceWithRelevance(db, fetcher, relay, relevance)

	require.NoError(t, svc.PollOnce(context.Background()))

	var email models.MailboxEmail
	require.NoError(t, db.Where("message_id = ?", "newsletter@malt.fr").First(&email).Error)
	assert.False(t, email.IsOpportunity)
	assert.Nil(t, email.RelevanceScore, "une non-opportunité ne doit jamais recevoir de score (pas 0, qui se lirait comme jugé et mauvais)")
	assert.False(t, email.ForwardBlocked)
	assert.NotNil(t, email.ForwardedAt)
}

// Panne LLM (réseau/proxy) → fail-open : transfert quand même, jamais de perte d'opportunité pour
// une raison technique.
func TestMailboxService_Relevance_FailOpenOnLLMError(t *testing.T) {
	db := setupMailboxTestDB(t)
	require.NoError(t, db.Create(&models.MailboxCursor{MailboxKey: "primary", UIDValidity: 1, LastUID: 0}).Error)

	fetcher := &fakeFetcher{
		uidValidity: 1, uidNext: 2,
		messages: []services.ImapMessage{
			{UID: 1, MessageID: "panne@malt.fr", From: "notif@malt.fr", Subject: "Mission", ReceivedAt: time.Now()},
		},
	}
	relay := &fakeRelay{}
	relevance := &fakeRelevance{err: errors.New("proxy down"), threshold: 50}
	svc := newMailboxServiceWithRelevance(db, fetcher, relay, relevance)

	require.NoError(t, svc.PollOnce(context.Background()))

	var email models.MailboxEmail
	require.NoError(t, db.Where("message_id = ?", "panne@malt.fr").First(&email).Error)
	assert.False(t, email.IsOpportunity)
	assert.Nil(t, email.RelevanceScore)
	assert.False(t, email.ForwardBlocked)
	assert.NotNil(t, email.ForwardedAt, "une panne LLM ne doit jamais bloquer le transfert (fail-open)")
	assert.Len(t, relay.calls, 1)
}

// ForwardBlocked n'est plus jamais posé par le filtre de pertinence (cf. judgeRelevance) mais reste
// un état possible (backfill historique, blocage manuel) — le retry automatique (par lot, à chaque
// cycle) ne doit JAMAIS le ressusciter, seul un forçage manuel (RetryForward) peut passer outre.
func TestMailboxService_Relevance_AutoRetryNeverResurrectsBlocked(t *testing.T) {
	db := setupMailboxTestDB(t)
	blocked := models.MailboxEmail{
		MessageID: "blocked@malt.fr", ImapUID: 1, FromAddress: "notif@malt.fr", FromDomain: "malt.fr",
		Platform: "malt", Subject: "s", ReceivedAt: time.Now(),
		IsOpportunity: true, ForwardBlocked: true,
	}
	require.NoError(t, db.Create(&blocked).Error)
	require.NoError(t, db.Create(&models.MailboxCursor{MailboxKey: "primary", UIDValidity: 1, LastUID: 5}).Error)

	fetcher := &fakeFetcher{uidValidity: 1, uidNext: 6}
	relay := &fakeRelay{}
	svc := newMailboxService(db, fetcher, relay)
	require.NoError(t, svc.PollOnce(context.Background()))

	assert.Empty(t, relay.calls, "un mail bloqué par le filtre ne doit jamais être retenté automatiquement")

	var reloaded models.MailboxEmail
	require.NoError(t, db.First(&reloaded, "id = ?", blocked.ID).Error)
	assert.Nil(t, reloaded.ForwardedAt)
	assert.True(t, reloaded.ForwardBlocked)
}

// Le forçage manuel (bouton "retenter"/"forcer" du panel admin) passe outre un blocage de pertinence.
func TestMailboxService_Relevance_ManualRetryForwardOverridesBlock(t *testing.T) {
	db := setupMailboxTestDB(t)
	blocked := models.MailboxEmail{
		MessageID: "override@malt.fr", ImapUID: 1, FromAddress: "notif@malt.fr", FromDomain: "malt.fr",
		Platform: "malt", Subject: "s", ReceivedAt: time.Now(),
		IsOpportunity: true, ForwardBlocked: true,
	}
	require.NoError(t, db.Create(&blocked).Error)

	relay := &fakeRelay{}
	svc := newMailboxService(db, &fakeFetcher{}, relay)

	require.NoError(t, svc.RetryForward(context.Background(), blocked.ID))

	require.Len(t, relay.calls, 1)
	var reloaded models.MailboxEmail
	require.NoError(t, db.First(&reloaded, "id = ?", blocked.ID).Error)
	assert.False(t, reloaded.ForwardBlocked)
	assert.NotNil(t, reloaded.ForwardedAt)
}

// --- marque de relève : ce que VerifierReleve lit pour voir une relève arrêtée ---

// Une relève réussie laisse sa marque ; une relève en échec n'en laisse pas — c'est l'âge de cette
// marque qui a manqué du 17 au 19/09/2026, quand plus rien n'était lu et que rien ne le disait.
func TestMailboxService_PollOnce_MarqueLaReleveSeulementSiElleReussit(t *testing.T) {
	mr := miniredis.RunT(t)
	watch := services.NewMailWatch(redis.NewClient(&redis.Options{Addr: mr.Addr()}), nil)
	ctx := context.Background()

	// Relève en échec : aucune marque.
	enPanne := &fakeFetcher{statusErr: errors.New("imap status: connexion fermée")}
	s := newMailboxService(setupMailboxTestDB(t), enPanne, &fakeRelay{})
	s.SetMailWatch(watch)
	require.Error(t, s.PollOnce(ctx))
	_, marquee, err := watch.DerniereReleve(ctx)
	require.NoError(t, err)
	assert.False(t, marquee, "une relève en échec ne doit pas passer pour une relève réussie")

	// Relève réussie (premier passage : curseur seedé) puis relève incrémentale : marque posée.
	sain := &fakeFetcher{uidValidity: 1, uidNext: 10}
	s = newMailboxService(setupMailboxTestDB(t), sain, &fakeRelay{})
	s.SetMailWatch(watch)
	avant := time.Now().Add(-time.Second)
	require.NoError(t, s.PollOnce(ctx))
	require.NoError(t, s.PollOnce(ctx))
	derniere, marquee, err := watch.DerniereReleve(ctx)
	require.NoError(t, err)
	require.True(t, marquee)
	assert.False(t, derniere.Before(avant.Truncate(time.Second)))
}
