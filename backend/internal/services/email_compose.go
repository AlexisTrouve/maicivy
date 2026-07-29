package services

import (
	"crypto/rand"
	"encoding/hex"
	"mime"
)

// MimeWord encode un en-tête en RFC 2047 si non-ASCII (accents dans le nom/sujet) — sinon tel quel.
// Extrait de internal/api/newsletter.go (partagé avec la composition des emails de transfert mailbox).
func MimeWord(s string) string {
	for _, r := range s {
		if r > 127 {
			return mime.QEncoding.Encode("UTF-8", s)
		}
	}
	return s
}

// RandHex16 génère 16 caractères hex aléatoires (boundary MIME, Message-ID...).
func RandHex16() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
