package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"
)

// GmailImapFetcher — implémentation réelle d'ImapFetcher via IMAPS (Gmail par défaut ; tout serveur
// IMAP standard avec App Password fonctionne).
//
// UNE SESSION PAR RELÈVE, jamais une session tenue des jours. Vécu du 17 au 19/09/2026 (et du 10 au
// 14/09) : la relève gardait une seule session ouverte toute la vie du conteneur, et plus aucun mail
// n'était lu jusqu'au redéploiement suivant — deux jours de retard sur des offres Malt. Session gelée
// ou appel pendu, les deux mènent là ; une session neuve à chaque relève ferme les deux portes, pour
// une connexion toutes les 2 min.
//
// DÉLAI RÉEL sur chaque appel : go-imap ne prend pas de ctx, donc l'échéance du ctx ferme la
// connexion TCP elle-même (cf. echeance). Sans ça, le ctx de 60 s du poll n'atteignait jamais le
// réseau, et un pair muet bloquait la boucle de poll pour toujours.
type GmailImapFetcher struct {
	dial       func(ctx context.Context) (net.Conn, error)
	user, pass string
	folder     string
	conn       net.Conn           // session de la relève en cours ; nil entre deux relèves
	client     *imapclient.Client // idem
	desarme    func()             // annule la fermeture programmée par echeance
}

// delaiParDefautImap borne un appel dont le ctx n'a pas d'échéance (le constructeur, un outil CLI) —
// pour qu'aucun chemin ne puisse attendre indéfiniment.
const delaiParDefautImap = 60 * time.Second

// NewGmailImapFetcher prépare la relève IMAPS et ouvre une première session tout de suite : des
// credentials faux échouent au démarrage du job, pas au premier poll. App Password — PAS le mot de
// passe du compte, Gmail refuse l'auth basique depuis 2022.
func NewGmailImapFetcher(host string, port int, user, appPassword, folder string) (*GmailImapFetcher, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	d := &tls.Dialer{NetDialer: &net.Dialer{Timeout: 15 * time.Second}, Config: &tls.Config{ServerName: host}}
	return newImapFetcher(func(ctx context.Context) (net.Conn, error) {
		return d.DialContext(ctx, "tcp", addr)
	}, user, appPassword, folder)
}

// newImapFetcher — même chose avec un dial injecté (tests : serveur IMAP en mémoire, pair muet).
func newImapFetcher(dial func(ctx context.Context) (net.Conn, error), user, pass, folder string) (*GmailImapFetcher, error) {
	f := &GmailImapFetcher{dial: dial, user: user, pass: pass, folder: folder}
	if err := f.ouvrirSession(context.Background()); err != nil {
		return nil, err
	}
	return f, nil
}

// ouvrirSession ferme la session précédente puis en ouvre une neuve : dial, échéance, LOGIN, SELECT.
func (f *GmailImapFetcher) ouvrirSession(ctx context.Context) error {
	f.fermerSession()
	conn, err := f.dial(ctx)
	if err != nil {
		return fmt.Errorf("imap dial: %w", err)
	}
	f.conn, f.client = conn, imapclient.New(conn, nil)
	f.echeance(ctx)
	if err := f.client.Login(f.user, f.pass).Wait(); err != nil {
		f.fermerSession()
		return fmt.Errorf("imap login: %w", err)
	}
	if _, err := f.client.Select(f.folder, nil).Wait(); err != nil {
		f.fermerSession()
		return fmt.Errorf("imap select %s: %w", f.folder, err)
	}
	return nil
}

// echeance arme la fermeture de la connexion à l'échéance du ctx (ou du délai par défaut s'il n'en a
// pas) — c'est ce qui transforme un pair muet en erreur.
//
// POURQUOI fermer plutôt que SetDeadline : go-imap gère lui-même les délais de lecture de sa
// connexion, et remet à 0 (= infini) celui de l'attente d'une réponse (`idleReadTimeout`, imapclient
// v2.0.0-beta.8). Une échéance posée par nous est écrasée à la première commande — mesuré par
// TestGmailImapFetcher_PairMuet_EchoueDansLeDelai. Fermer le TCP, en revanche, fait sortir sa boucle
// de lecture, qui termine alors toutes les commandes en attente avec une erreur.
//
// COMMENT : context.AfterFunc sur la connexion capturée (conn.Close est sûr depuis une autre
// goroutine) ; l'armement précédent est désarmé à chaque appel. Conséquence voulue : à la fin d'un
// poll, l'annulation de son ctx ferme la session — aucune session ne reste ouverte entre deux relèves.
func (f *GmailImapFetcher) echeance(ctx context.Context) {
	f.desarmer()
	var cancel context.CancelFunc = func() {}
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, delaiParDefautImap)
	}
	conn := f.conn
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	f.desarme = func() { stop(); cancel() }
}

// desarmer annule la fermeture programmée par echeance, s'il y en a une.
func (f *GmailImapFetcher) desarmer() {
	if f.desarme != nil {
		f.desarme()
		f.desarme = nil
	}
}

// fermerSession libère la session courante s'il y en a une. Pas de LOGOUT : il attendrait une réponse
// que le pair ne donnera peut-être jamais — fermer le TCP suffit, le serveur libère de son côté.
func (f *GmailImapFetcher) fermerSession() {
	f.desarmer()
	if f.client != nil {
		_ = f.client.Close()
	}
	f.conn, f.client = nil, nil
}

// session rend le client de la relève en cours, en ouvrant une session si aucune n'est utilisable
// (après une erreur, après la fin du ctx qui l'avait ouverte, ou pour un appel isolé comme
// FetchByUIDs), avec l'échéance de ce ctx.
func (f *GmailImapFetcher) session(ctx context.Context) (*imapclient.Client, error) {
	if f.client == nil || f.client.State() == imap.ConnStateLogout {
		if err := f.ouvrirSession(ctx); err != nil {
			return nil, err
		}
	}
	f.echeance(ctx)
	return f.client, nil
}

// MailboxStatus ouvre la session de CETTE relève (c'est le premier appel de chaque PollOnce) puis
// interroge UIDValidity/UIDNext via STATUS — qui n'affecte pas l'état \Recent, contrairement à un
// second SELECT.
func (f *GmailImapFetcher) MailboxStatus(ctx context.Context) (uint32, uint32, error) {
	if err := f.ouvrirSession(ctx); err != nil {
		return 0, 0, err
	}
	data, err := f.client.Status(f.folder, &imap.StatusOptions{UIDNext: true, UIDValidity: true}).Wait()
	if err != nil {
		f.fermerSession()
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
	client, err := f.session(ctx)
	if err != nil {
		return nil, err
	}
	buffers, err := client.Fetch(uidSet, fetchOptions).Collect()
	if err != nil {
		f.fermerSession()
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
	client, err := f.session(ctx)
	if err != nil {
		return nil, err
	}
	buffers, err := client.Fetch(uidSet, fetchOptions).Collect()
	if err != nil {
		f.fermerSession()
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

// Close libère la session en cours (au shutdown, cf. jobs/mailbox_poll.go).
func (f *GmailImapFetcher) Close() error {
	f.fermerSession()
	return nil
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
