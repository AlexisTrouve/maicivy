package middleware

import (
	"crypto/hmac"
	"strconv"
	"strings"
	"time"
)

// Cookie admin signé HMAC avec EXPIRATION — format `admin:<expiryUnix>.<sig>`.
//
// POURQUOI : le login owner (mot de passe → cookie) doit produire un jeton (a) infalsifiable sans le
// secret serveur, (b) périssable (un cookie volé ne vaut pas éternellement). On réutilise le même
// secret HMAC + sessionSig que le cookie de session (cf. session_sign.go) — pas de nouvelle crypto.
// COMMENT : payload = "admin:<unix expiry>", sig = base64url(HMAC-SHA256(payload, secret)). Vérif =
// signature constant-time (hmac.Equal) PUIS expiry > maintenant. Secret vide → rien ne valide.

const adminCookiePrefix = "admin:"

// SignAdminCookie produit le jeton de cookie admin valable `ttl`.
func SignAdminCookie(secret string, ttl time.Duration) string {
	payload := adminCookiePrefix + strconv.FormatInt(time.Now().Add(ttl).Unix(), 10)
	return payload + "." + sessionSig(payload, secret)
}

// VerifyAdminCookie dit si `token` est un cookie admin qu'on a émis avec `secret` ET non expiré.
// Rejette : token vide, secret vide, sans signature, signature forgée, signé d'un autre secret,
// préfixe absent, expiry illisible, ou expiré.
func VerifyAdminCookie(token, secret string) bool {
	if secret == "" || token == "" {
		return false
	}
	i := strings.LastIndexByte(token, '.')
	if i <= 0 || i >= len(token)-1 {
		return false
	}
	payload, sig := token[:i], token[i+1:]
	// 1. signature authentique (constant-time, pas de fuite de timing)
	if !hmac.Equal([]byte(sig), []byte(sessionSig(payload, secret))) {
		return false
	}
	// 2. préfixe attendu + expiry parsable
	if !strings.HasPrefix(payload, adminCookiePrefix) {
		return false
	}
	expUnix, err := strconv.ParseInt(payload[len(adminCookiePrefix):], 10, 64)
	if err != nil {
		return false
	}
	// 3. non expiré
	return time.Now().Unix() < expUnix
}
