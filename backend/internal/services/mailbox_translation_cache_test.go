package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"maicivy/internal/models"
	"maicivy/internal/services"
)

func setupTranslationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.MailboxEmail{}, &models.MailboxEmailTranslation{}))
	return db
}

type fakeTranslator struct {
	subject, body string
	err           error
	calls         int
}

func (f *fakeTranslator) Translate(ctx context.Context, subject, body, lang string) (string, string, error) {
	f.calls++
	if f.err != nil {
		return "", "", f.err
	}
	return f.subject, f.body, nil
}

func seedTranslationEmail(t *testing.T, db *gorm.DB) models.MailboxEmail {
	t.Helper()
	e := models.MailboxEmail{
		MessageID: "x@malt.fr", ImapUID: 1, FromAddress: "n@malt.fr", FromDomain: "malt.fr",
		Platform: "malt", Subject: "Sujet FR", BodyText: "Corps FR",
	}
	require.NoError(t, db.Create(&e).Error)
	return e
}

// Pas en cache, allowTranslate=false (check auto au chargement) → rien, ZÉRO appel LLM.
func TestGetOrTranslateMailboxEmail_NoCacheNoTranslate_ReturnsNilNoCall(t *testing.T) {
	db := setupTranslationTestDB(t)
	email := seedTranslationEmail(t, db)
	translator := &fakeTranslator{subject: "EN subject", body: "EN body"}

	got, err := services.GetOrTranslateMailboxEmail(context.Background(), db, translator, &email, "en", false)

	require.NoError(t, err)
	assert.Nil(t, got)
	assert.Zero(t, translator.calls, "un check cache-only ne doit jamais appeler le traducteur")
}

// Pas en cache, allowTranslate=true (clic "Traduire") → traduit et met en cache.
func TestGetOrTranslateMailboxEmail_NoCacheAllowTranslate_TranslatesAndCaches(t *testing.T) {
	db := setupTranslationTestDB(t)
	email := seedTranslationEmail(t, db)
	translator := &fakeTranslator{subject: "EN subject", body: "EN body"}

	got, err := services.GetOrTranslateMailboxEmail(context.Background(), db, translator, &email, "en", true)

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "EN subject", got.Subject)
	assert.Equal(t, "EN body", got.Body)
	assert.Equal(t, "en", got.Lang)
	assert.Equal(t, 1, translator.calls)

	var stored models.MailboxEmailTranslation
	require.NoError(t, db.Where("mailbox_email_id = ? AND lang = ?", email.ID, "en").First(&stored).Error)
	assert.Equal(t, "EN subject", stored.Subject)
}

// Déjà en cache → sert directement, ZÉRO nouvel appel LLM, même avec allowTranslate=true.
func TestGetOrTranslateMailboxEmail_CacheHit_NeverCallsTranslator(t *testing.T) {
	db := setupTranslationTestDB(t)
	email := seedTranslationEmail(t, db)
	require.NoError(t, db.Create(&models.MailboxEmailTranslation{
		MailboxEmailID: email.ID, Lang: "en", Subject: "Cached subject", Body: "Cached body",
	}).Error)
	translator := &fakeTranslator{subject: "SHOULD NOT BE USED", body: "SHOULD NOT BE USED"}

	got, err := services.GetOrTranslateMailboxEmail(context.Background(), db, translator, &email, "en", true)

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Cached subject", got.Subject)
	assert.Zero(t, translator.calls, "un cache hit ne doit jamais réappeler le traducteur")
}

// Cache miss, allowTranslate=true, mais translator nil (credentials absentes) → erreur explicite.
func TestGetOrTranslateMailboxEmail_TranslatorNilAndRequested_ReturnsConfigError(t *testing.T) {
	db := setupTranslationTestDB(t)
	email := seedTranslationEmail(t, db)

	_, err := services.GetOrTranslateMailboxEmail(context.Background(), db, nil, &email, "en", true)

	require.ErrorIs(t, err, services.ErrMailboxTranslationNotConfigured)
}

// Panne LLM → erreur remontée, RIEN n'est mis en cache (pas de traduction vide/partielle stockée).
func TestGetOrTranslateMailboxEmail_TranslatorError_NothingCached(t *testing.T) {
	db := setupTranslationTestDB(t)
	email := seedTranslationEmail(t, db)
	translator := &fakeTranslator{err: errors.New("proxy down")}

	_, err := services.GetOrTranslateMailboxEmail(context.Background(), db, translator, &email, "en", true)

	require.Error(t, err)
	var count int64
	db.Model(&models.MailboxEmailTranslation{}).Where("mailbox_email_id = ?", email.ID).Count(&count)
	assert.Zero(t, count, "une traduction ratée ne doit jamais laisser d'entrée en cache")
}

// Deux langues différentes pour le même mail → deux entrées de cache distinctes, indépendantes.
func TestGetOrTranslateMailboxEmail_DifferentLangs_SeparateCacheEntries(t *testing.T) {
	db := setupTranslationTestDB(t)
	email := seedTranslationEmail(t, db)
	translatorEn := &fakeTranslator{subject: "EN subject", body: "EN body"}
	translatorDe := &fakeTranslator{subject: "DE subject", body: "DE body"}

	_, err := services.GetOrTranslateMailboxEmail(context.Background(), db, translatorEn, &email, "en", true)
	require.NoError(t, err)
	_, err = services.GetOrTranslateMailboxEmail(context.Background(), db, translatorDe, &email, "de", true)
	require.NoError(t, err)

	gotEn, err := services.GetOrTranslateMailboxEmail(context.Background(), db, translatorEn, &email, "en", false)
	require.NoError(t, err)
	gotDe, err := services.GetOrTranslateMailboxEmail(context.Background(), db, translatorDe, &email, "de", false)
	require.NoError(t, err)

	assert.Equal(t, "EN subject", gotEn.Subject)
	assert.Equal(t, "DE subject", gotDe.Subject)
	assert.Equal(t, 1, translatorEn.calls)
	assert.Equal(t, 1, translatorDe.calls)
}
