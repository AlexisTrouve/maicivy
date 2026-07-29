package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"maicivy/internal/services"
)

type reminderRelayCall struct {
	addr, from, to, msg string
}

func newTestReminderService(t *testing.T, relay services.MailboxRelaySender) (*services.MaltReminderService, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	svc := services.NewMaltReminderService(
		rdb, "smtp.local:25", "dest@qq.com", "mailbox@etheryale.com", "Mailbox Etheryale",
		[]time.Weekday{time.Monday, time.Thursday}, 1, relay,
	)
	require.NotNil(t, svc)
	return svc, rdb
}

func TestNewMaltReminderService_NilWhenNoRecipient(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	svc := services.NewMaltReminderService(rdb, "smtp.local:25", "", "a@b.c", "Name", nil, 1, nil)
	assert.Nil(t, svc)
}

// Jour non configuré (mardi, ni lundi ni jeudi) → jamais d'envoi.
func TestMaltReminderService_WrongDay_NoSend(t *testing.T) {
	var calls []reminderRelayCall
	relay := func(addr, from, to string, msg []byte) error {
		calls = append(calls, reminderRelayCall{addr, from, to, string(msg)})
		return nil
	}
	svc, _ := newTestReminderService(t, relay)

	tuesday := time.Date(2026, 7, 14, 2, 0, 0, 0, time.UTC) // mardi
	svc.CheckAndSend(context.Background(), tuesday)

	assert.Empty(t, calls, "mardi n'est pas un jour configuré, aucun envoi")
}

// Bon jour mais avant l'heure configurée → pas encore d'envoi.
func TestMaltReminderService_RightDayBeforeHour_NoSend(t *testing.T) {
	var calls []reminderRelayCall
	relay := func(addr, from, to string, msg []byte) error {
		calls = append(calls, reminderRelayCall{addr, from, to, string(msg)})
		return nil
	}
	svc, _ := newTestReminderService(t, relay)

	monday0h := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC) // lundi, avant 1h UTC configuré
	svc.CheckAndSend(context.Background(), monday0h)

	assert.Empty(t, calls, "avant l'heure configurée, aucun envoi")
}

// Bon jour + heure atteinte → envoi réel, contenu correct.
func TestMaltReminderService_RightDayAtHour_Sends(t *testing.T) {
	var calls []reminderRelayCall
	relay := func(addr, from, to string, msg []byte) error {
		calls = append(calls, reminderRelayCall{addr, from, to, string(msg)})
		return nil
	}
	svc, _ := newTestReminderService(t, relay)

	monday := time.Date(2026, 7, 13, 1, 0, 0, 0, time.UTC) // lundi, 1h UTC pile
	svc.CheckAndSend(context.Background(), monday)

	require.Len(t, calls, 1)
	assert.Equal(t, "dest@qq.com", calls[0].to)
	// Anglais : le destinataire (Tingting) ne lit pas le français, cf. renderMaltReminder.
	assert.Contains(t, calls[0].msg, "availability")
	assert.NotContains(t, calls[0].msg, "disponibilit", "le contenu ne doit plus être en français")
	assert.Contains(t, calls[0].msg, "malt.fr")
}

// Déjà envoyé aujourd'hui → un 2e check (même jour, heure plus tardive) ne renvoie pas.
func TestMaltReminderService_AlreadySentToday_NoDuplicate(t *testing.T) {
	var calls []reminderRelayCall
	relay := func(addr, from, to string, msg []byte) error {
		calls = append(calls, reminderRelayCall{addr, from, to, string(msg)})
		return nil
	}
	svc, _ := newTestReminderService(t, relay)

	monday1h := time.Date(2026, 7, 13, 1, 0, 0, 0, time.UTC)
	monday5h := time.Date(2026, 7, 13, 5, 0, 0, 0, time.UTC)
	svc.CheckAndSend(context.Background(), monday1h)
	svc.CheckAndSend(context.Background(), monday5h)

	assert.Len(t, calls, 1, "un seul envoi par jour, peu importe le nombre de checks")
}

// Échec d'envoi → PAS marqué comme envoyé, retenté au prochain check le même jour.
func TestMaltReminderService_SendFails_RetriesNextCheckSameDay(t *testing.T) {
	callCount := 0
	relay := func(addr, from, to string, msg []byte) error {
		callCount++
		if callCount == 1 {
			return errors.New("smtp down")
		}
		return nil
	}
	svc, _ := newTestReminderService(t, relay)

	monday1h := time.Date(2026, 7, 13, 1, 0, 0, 0, time.UTC)
	monday2h := time.Date(2026, 7, 13, 2, 0, 0, 0, time.UTC)
	svc.CheckAndSend(context.Background(), monday1h) // échoue
	svc.CheckAndSend(context.Background(), monday2h) // réessaie, réussit

	assert.Equal(t, 2, callCount, "l'échec ne doit pas bloquer un retry le même jour")
}

// Jour suivant configuré (jeudi) après un envoi le lundi → nouvel envoi (clé par date, pas globale).
func TestMaltReminderService_NextConfiguredDay_SendsAgain(t *testing.T) {
	var calls []reminderRelayCall
	relay := func(addr, from, to string, msg []byte) error {
		calls = append(calls, reminderRelayCall{addr, from, to, string(msg)})
		return nil
	}
	svc, _ := newTestReminderService(t, relay)

	monday := time.Date(2026, 7, 13, 1, 0, 0, 0, time.UTC)
	thursday := time.Date(2026, 7, 16, 1, 0, 0, 0, time.UTC)
	svc.CheckAndSend(context.Background(), monday)
	svc.CheckAndSend(context.Background(), thursday)

	assert.Len(t, calls, 2, "lundi et jeudi sont deux jours configurés distincts, deux envois")
}
