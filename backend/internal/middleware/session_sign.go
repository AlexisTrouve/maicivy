package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"

	"github.com/google/uuid"
)

// Signature HMAC du cookie de session — format `<uuid>.<sig>`.
//
// POURQUOI : distinguer une session que LE SERVEUR a émise d'une chaîne inventée par le client.
// Le tracking ne persiste un visiteur (ni ne le considère "réel") QUE pour un cookie valide → un
// scanner qui envoie des cookies bidons ne déclenche plus aucun INSERT en base (amplification fermée).
// COMMENT : sig = base64url(HMAC-SHA256(uuid, secret)). base64url (sans padding) = caractères
// cookie-safe (A-Za-z0-9-_). Vérification constant-time via hmac.Equal.

// sessionSig calcule la signature base64url d'un id avec le secret.
func sessionSig(id, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(id))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// signSession produit le token de cookie signé : `<uuid>.<sig>`.
func signSession(id, secret string) string {
	return id + "." + sessionSig(id, secret)
}

// newSignedSession génère un nouvel identifiant de session signé (uuid v4 + signature).
func newSignedSession(secret string) string {
	return signSession(uuid.NewString(), secret)
}

// verifySession dit si `token` est un cookie de session que NOUS avons émis avec `secret`.
// Retourne false pour un cookie absent, sans signature, à signature forgée, ou signé d'un autre
// secret. Comparaison constant-time (hmac.Equal) — pas de fuite de timing sur la signature.
func verifySession(token, secret string) bool {
	i := strings.LastIndexByte(token, '.')
	if i <= 0 || i >= len(token)-1 { // pas de '.', '.' en tête, ou signature vide
		return false
	}
	id, sig := token[:i], token[i+1:]
	return hmac.Equal([]byte(sig), []byte(sessionSig(id, secret)))
}
