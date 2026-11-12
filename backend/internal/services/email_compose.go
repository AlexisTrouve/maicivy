package services

import (
	"crypto/rand"
	"encoding/hex"
	"mime"
	"mime/quotedprintable"
	"strings"
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

// QuotedPrintable encode un corps de message en quoted-printable (RFC 2045).
//
// POURQUOI cette fonction existe — panne de production du 2026-08-01 : un corps écrit tel quel sous
// un Content-Type SANS en-tête Content-Transfer-Encoding est interprété comme du `7bit`, qui
// plafonne une ligne à 998 octets (RFC 5321 §4.5.3.1.6). Passé ce seuil le serveur destinataire
// répond `550 maximum allowed line length is 998 octets` et le message n'arrive JAMAIS. Le piège est
// que les contenus courts passent : la panne ne se déclenche que sur les messages longs, donc en
// général sur ceux qui comptent.
//
// TOUT corps de mail du projet doit passer par ici, et l'appelant DOIT déclarer
// `Content-Transfer-Encoding: quoted-printable` dans les en-têtes de la partie concernée — sans quoi
// le client mail affiche l'encodage brut (« Transf=C3=A9r=C3=A9 ») au lieu du texte.
//
// COMMENT : l'encodeur standard coupe les lignes trop longues par des retours SOUPLES (`=` en fin de
// ligne) que le client recolle à l'affichage — le texte reste intact pour le lecteur. Il échappe au
// passage les octets non-ASCII, ce qui évite d'envoyer de l'UTF-8 brut sous une déclaration `7bit`.
//
// POURQUOI pas base64 : il rendrait le message illisible pour tout ce qui ne le décode pas (journaux
// serveur, inspection manuelle) alors que ces contenus sont majoritairement de l'ASCII.
//
// Les erreurs d'écriture sont ignorées sciemment : la destination est un strings.Builder, dont le
// Write ne peut pas échouer.
func QuotedPrintable(s string) string {
	var b strings.Builder
	w := quotedprintable.NewWriter(&b)
	_, _ = w.Write([]byte(s))
	_ = w.Close()
	return b.String()
}

// RandHex16 génère 16 caractères hex aléatoires (boundary MIME, Message-ID...).
func RandHex16() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
