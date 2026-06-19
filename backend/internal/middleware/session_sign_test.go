package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSessionSign_RoundTripAndForgery verrouille la primitive de signature : un token qu'on a émis
// est valide, toute falsification (pas de sig, sig forgée, uuid altéré, mauvais secret, vide) est
// rejetée.
func TestSessionSign_RoundTripAndForgery(t *testing.T) {
	const secret = "s3cr3t-test"

	tok := newSignedSession(secret)
	assert.True(t, verifySession(tok, secret), "un token qu'on a émis doit être valide")

	assert.False(t, verifySession("not-a-token", secret), "aucune signature")
	assert.False(t, verifySession("forged-uuid.deadbeef", secret), "signature forgée")
	assert.False(t, verifySession("", secret), "vide")
	assert.False(t, verifySession(".sig", secret), "uuid vide")
	assert.False(t, verifySession("id.", secret), "signature vide")
	assert.False(t, verifySession(tok, "autre-secret"), "secret différent")

	// Altérer le 1er caractère de l'uuid invalide la signature.
	assert.False(t, verifySession("x"+tok[1:], secret), "uuid altéré")
}
