package services

import (
	"context"
	"time"
)

// ImapMessage — un message IMAP réduit au strict nécessaire pour MailboxService (dédup, filtrage,
// stockage, transfert). From est l'adresse email BRUTE de l'expéditeur ("mailbox@host"), pas un nom
// affiché — c'est sur cette adresse que MailboxService extrait le domaine pour matcher l'allowlist
// (cf. EmailDomain).
type ImapMessage struct {
	UID        uint32
	MessageID  string
	From       string
	Subject    string
	BodyText   string
	ReceivedAt time.Time
}

// ImapFetcher abstrait la connexion à une boîte IMAP.
//
// POURQUOI une interface : MailboxService doit être testable sans réseau réel — cf.
// mailbox_service_test.go, qui injecte un fake en mémoire. La vraie implémentation (connexion Gmail via
// go-imap/v2) arrive dans imap_fetcher_gmail.go.
type ImapFetcher interface {
	// MailboxStatus retourne UIDValidity et UIDNext de la boîte suivie SANS fetcher aucun message —
	// c'est ce qui permet à MailboxService de seeder son curseur au premier démarrage (ou après un
	// changement d'UIDValidity) sans jamais ingérer l'historique existant de la boîte.
	MailboxStatus(ctx context.Context) (uidValidity uint32, uidNext uint32, err error)

	// FetchSince retourne tous les messages dont l'UID est STRICTEMENT supérieur à sinceUID, triés
	// par UID croissant. MailboxService s'appuie sur cet ordre pour avancer son curseur message par
	// message sans trou (reprise propre après crash) — une implémentation qui ne garantit pas l'ordre
	// doit trier avant de retourner.
	FetchSince(ctx context.Context, sinceUID uint32) ([]ImapMessage, error)

	// Close libère la connexion IMAP sous-jacente.
	Close() error
}
