package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"
)

// GmailImapFetcher — implémentation réelle d'ImapFetcher via IMAPS (Gmail par défaut ; tout serveur
// IMAP standard avec App Password fonctionne). Une connexion, une boîte sélectionnée (folder), tenue
// ouverte pour tout le cycle de vie du service (fermée au shutdown, cf. jobs/mailbox_poll.go).
type GmailImapFetcher struct {
	client *imapclient.Client
	folder string
}

// NewGmailImapFetcher se connecte en IMAPS, s'authentifie (App Password — PAS le mot de passe du
// compte, Gmail refuse l'auth basique depuis 2022) et sélectionne folder. La connexion échoue tôt
// (au démarrage du job, pas au premier poll) si les credentials sont mauvais.
func NewGmailImapFetcher(host string, port int, user, appPassword, folder string) (*GmailImapFetcher, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := imapclient.DialTLS(addr, nil)
	if err != nil {
		return nil, fmt.Errorf("imap dial %s: %w", addr, err)
	}
	if err := client.Login(user, appPassword).Wait(); err != nil {
		client.Close()
		return nil, fmt.Errorf("imap login: %w", err)
	}
	if _, err := client.Select(folder, nil).Wait(); err != nil {
		client.Close()
		return nil, fmt.Errorf("imap select %s: %w", folder, err)
	}
	return &GmailImapFetcher{client: client, folder: folder}, nil
}

// MailboxStatus interroge UIDValidity/UIDNext via STATUS — pas besoin de re-SELECT (déjà fait au
// Dial), et STATUS n'affecte pas l'état \Recent des messages contrairement à un second SELECT.
func (f *GmailImapFetcher) MailboxStatus(ctx context.Context) (uint32, uint32, error) {
	data, err := f.client.Status(f.folder, &imap.StatusOptions{UIDNext: true, UIDValidity: true}).Wait()
	if err != nil {
		return 0, 0, fmt.Errorf("imap status %s: %w", f.folder, err)
	}
	return data.UIDValidity, uint32(data.UIDNext), nil
}

// FetchSince récupère l'UID range ouvert (sinceUID+1 : *) — UID FETCH garantit que le serveur ne
// renvoie que des messages existants dans cette plage, triés par UID croissant (comportement standard
// IMAP), ce qui satisfait le contrat de ImapFetcher.
func (f *GmailImapFetcher) FetchSince(ctx context.Context, sinceUID uint32) ([]ImapMessage, error) {
	var uidSet imap.UIDSet
	uidSet.AddRange(imap.UID(sinceUID+1), 0) // 0 = "*" (borne ouverte)

	bodySection := &imap.FetchItemBodySection{}
	fetchOptions := &imap.FetchOptions{
		UID:         true,
		Envelope:    true,
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}
	buffers, err := f.client.Fetch(uidSet, fetchOptions).Collect()
	if err != nil {
		return nil, fmt.Errorf("imap fetch: %w", err)
	}

	messages := make([]ImapMessage, 0, len(buffers))
	for _, buf := range buffers {
		if buf.UID == 0 {
			continue // pas d'UID retourné (ne devrait pas arriver en UID FETCH) — ignoré plutôt que planté
		}
		msg := ImapMessage{UID: uint32(buf.UID), ReceivedAt: time.Now()}
		if buf.Envelope != nil {
			msg.MessageID = buf.Envelope.MessageID
			msg.Subject = buf.Envelope.Subject
			if !buf.Envelope.Date.IsZero() {
				msg.ReceivedAt = buf.Envelope.Date
			}
			if len(buf.Envelope.From) > 0 {
				msg.From = buf.Envelope.From[0].Addr()
			}
		}
		msg.BodyText = extractPlainText(buf.FindBodySection(bodySection))
		messages = append(messages, msg)
	}
	return messages, nil
}

// FetchByUIDs récupère un ensemble PRÉCIS d'UID (pas une plage) — utilisé pour un backfill ciblé
// (ex: réimporter d'anciens mails identifiés par avance), par opposition à FetchSince qui couvre tout
// l'ouvert depuis un point. Même options de fetch que FetchSince, donc même contrat de sortie.
func (f *GmailImapFetcher) FetchByUIDs(ctx context.Context, uids []uint32) ([]ImapMessage, error) {
	var uidSet imap.UIDSet
	for _, u := range uids {
		uidSet.AddNum(imap.UID(u))
	}

	bodySection := &imap.FetchItemBodySection{}
	fetchOptions := &imap.FetchOptions{
		UID:         true,
		Envelope:    true,
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}
	buffers, err := f.client.Fetch(uidSet, fetchOptions).Collect()
	if err != nil {
		return nil, fmt.Errorf("imap fetch by uids: %w", err)
	}

	messages := make([]ImapMessage, 0, len(buffers))
	for _, buf := range buffers {
		if buf.UID == 0 {
			continue
		}
		msg := ImapMessage{UID: uint32(buf.UID), ReceivedAt: time.Now()}
		if buf.Envelope != nil {
			msg.MessageID = buf.Envelope.MessageID
			msg.Subject = buf.Envelope.Subject
			if !buf.Envelope.Date.IsZero() {
				msg.ReceivedAt = buf.Envelope.Date
			}
			if len(buf.Envelope.From) > 0 {
				msg.From = buf.Envelope.From[0].Addr()
			}
		}
		msg.BodyText = extractPlainText(buf.FindBodySection(bodySection))
		messages = append(messages, msg)
	}
	return messages, nil
}

// Close termine proprement la session IMAP (LOGOUT) puis ferme la connexion TCP.
func (f *GmailImapFetcher) Close() error {
	_ = f.client.Logout().Wait() // best-effort : on ferme la connexion de toute façon ensuite
	return f.client.Close()
}

// extractPlainText parse le RFC822 complet et renvoie la première partie text/plain rencontrée —
// fallback sur text/html brut (non nettoyé des balises) si aucune partie texte n'est présente, plutôt
// que de perdre le contenu du mail.
func extractPlainText(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	mr, err := mail.CreateReader(bytes.NewReader(raw))
	if err != nil {
		return string(raw)
	}
	var plain, html string
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		h, ok := p.Header.(*mail.InlineHeader)
		if !ok {
			continue // pièce jointe — hors scope de l'ingestion mailbox
		}
		ct, _, _ := h.ContentType()
		body, _ := io.ReadAll(p.Body)
		switch ct {
		case "text/plain":
			if plain == "" {
				plain = string(body)
			}
		case "text/html":
			if html == "" {
				html = string(body)
			}
		}
	}
	if plain != "" {
		return plain
	}
	return html
}
