package middleware

import (
	"testing"
	"time"
)

// Verrouille le cookie admin signé HMAC (login owner). Sécurité-critique → on teste surtout les REJETS.

func TestAdminCookie_RoundTrip(t *testing.T) {
	secret := "s3cr3t-hmac-key"
	tok := SignAdminCookie(secret, time.Hour)
	if tok == "" {
		t.Fatal("token vide")
	}
	if !VerifyAdminCookie(tok, secret) {
		t.Fatal("un cookie qu'on vient de signer doit être valide")
	}
}

func TestAdminCookie_RejectsForged(t *testing.T) {
	secret := "s3cr3t-hmac-key"
	if VerifyAdminCookie("admin:9999999999.signaturebidon", secret) {
		t.Fatal("signature forgée acceptée")
	}
	if VerifyAdminCookie("", secret) {
		t.Fatal("token vide accepté")
	}
	if VerifyAdminCookie("pasdepoint", secret) {
		t.Fatal("token sans signature accepté")
	}
}

func TestAdminCookie_RejectsWrongSecret(t *testing.T) {
	tok := SignAdminCookie("secretA", time.Hour)
	if VerifyAdminCookie(tok, "secretB") {
		t.Fatal("cookie validé avec un autre secret (forge possible)")
	}
}

func TestAdminCookie_RejectsExpired(t *testing.T) {
	secret := "s3cr3t-hmac-key"
	expired := SignAdminCookie(secret, -time.Minute) // déjà expiré
	if VerifyAdminCookie(expired, secret) {
		t.Fatal("cookie expiré accepté")
	}
}

func TestAdminCookie_EmptySecretRejectsAll(t *testing.T) {
	// Secret non configuré → AUCUN cookie ne doit valider (sinon login désactivé = porte ouverte).
	if VerifyAdminCookie("admin:9999999999.whatever", "") {
		t.Fatal("secret vide ne doit rien valider")
	}
}
