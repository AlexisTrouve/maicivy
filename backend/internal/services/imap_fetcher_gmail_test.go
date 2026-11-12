package services

import (
	"bufio"
	"bytes"
	"context"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Vécu du 17 au 19/09/2026 (et du 10 au 14/09) : trois offres Malt reçues dans la boîte n'ont été
// relevées qu'au redéploiement suivant — deux jours plus tard, puis transférées en 13 s. La relève
// tenait UNE session IMAP ouverte pendant toute la vie du conteneur, et aucun appel n'avait de délai
// (le ctx de 60 s du poll n'atteignait jamais le réseau). Session gelée ou appel pendu : dans les
// deux cas, plus rien n'était lu jusqu'au redémarrage. Ces tests verrouillent les deux remèdes.

const (
	testImapUser = "offres@test"
	testImapPass = "mot-de-passe-app"
)

// serveurMemoire démarre un vrai serveur IMAP en mémoire (sans TLS) et compte les connexions
// acceptées — c'est ce compteur qui prouve qu'une relève ouvre bien sa PROPRE session.
func serveurMemoire(t *testing.T) (*imapmemserver.User, func(ctx context.Context) (net.Conn, error), *atomic.Int32) {
	t.Helper()
	mem := imapmemserver.New()
	user := imapmemserver.NewUser(testImapUser, testImapPass)
	require.NoError(t, user.Create("INBOX", nil))
	mem.AddUser(user)

	srv := imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return mem.NewSession(), nil, nil
		},
		InsecureAuth: true,
		Caps:         imap.CapSet{imap.CapIMAP4rev1: {}, imap.CapIMAP4rev2: {}},
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	var connexions atomic.Int32
	dial := func(ctx context.Context) (net.Conn, error) {
		connexions.Add(1)
		var d net.Dialer
		return d.DialContext(ctx, "tcp", ln.Addr().String())
	}
	return user, dial, &connexions
}

// deposer ajoute un message dans INBOX côté serveur, comme le ferait l'arrivée d'un mail.
func deposer(t *testing.T, user *imapmemserver.User, sujet string) {
	t.Helper()
	brut := "From: noreply@malt.com\r\nSubject: " + sujet + "\r\nMessage-ID: <" + sujet + "@malt>\r\n" +
		"Date: Thu, 17 Sep 2026 14:28:58 +0000\r\n\r\ncorps\r\n"
	_, err := user.Append("INBOX", bytes.NewReader([]byte(brut)), &imap.AppendOptions{})
	require.NoError(t, err)
}

// Une relève = une session neuve : un mail arrivé entre deux relèves est vu à la suivante, et chaque
// relève a ouvert sa propre connexion (plus de session tenue des jours).
func TestGmailImapFetcher_ChaqueReleveOuvreUneSessionNeuve(t *testing.T) {
	user, dial, connexions := serveurMemoire(t)
	f, err := newImapFetcher(dial, testImapUser, testImapPass, "INBOX")
	require.NoError(t, err)
	defer f.Close()

	ctx := context.Background()
	_, uidNext, err := f.MailboxStatus(ctx)
	require.NoError(t, err)
	msgs, err := f.FetchSince(ctx, uidNext-1)
	require.NoError(t, err)
	assert.Empty(t, msgs)

	deposer(t, user, "offre-karbonalpha")

	_, _, err = f.MailboxStatus(ctx)
	require.NoError(t, err)
	msgs, err = f.FetchSince(ctx, uidNext-1)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "offre-karbonalpha", msgs[0].Subject)
	assert.Equal(t, "noreply@malt.com", msgs[0].From)

	// Constructeur (échec tôt) + une session par relève : 3 connexions pour 2 relèves.
	assert.Equal(t, int32(3), connexions.Load())
}

// serveurMuet répond au LOGIN et au SELECT, puis se tait pour toujours — un pair qui ne répond plus,
// exactement ce qui gelait la relève.
func serveurMuet(t *testing.T) func(ctx context.Context) (net.Conn, error) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_, _ = c.Write([]byte("* OK IMAP4rev1 pret\r\n"))
				r := bufio.NewReader(c)
				for {
					ligne, err := r.ReadString('\n')
					if err != nil {
						return
					}
					champs := strings.Fields(ligne)
					if len(champs) < 2 {
						continue
					}
					tag, cmd := champs[0], strings.ToUpper(champs[1])
					switch cmd {
					case "LOGIN":
						_, _ = c.Write([]byte(tag + " OK connecte\r\n"))
					case "SELECT":
						_, _ = c.Write([]byte("* 0 EXISTS\r\n* OK [UIDVALIDITY 1] v\r\n* OK [UIDNEXT 1] n\r\n" +
							tag + " OK [READ-WRITE] selectionne\r\n"))
					default:
						// Silence : ni réponse, ni fermeture.
					}
				}
			}(c)
		}
	}()
	return func(ctx context.Context) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, "tcp", ln.Addr().String())
	}
}

// Un pair muet ne gèle plus la relève : l'appel échoue dans le délai du ctx, et la relève suivante
// peut repartir. Avant, STATUS attendait indéfiniment et bloquait la boucle de poll entière.
func TestGmailImapFetcher_PairMuet_EchoueDansLeDelai(t *testing.T) {
	f, err := newImapFetcher(serveurMuet(t), testImapUser, testImapPass, "INBOX")
	require.NoError(t, err)
	defer f.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	fini := make(chan error, 1)
	go func() {
		_, _, err := f.MailboxStatus(ctx)
		fini <- err
	}()

	select {
	case err := <-fini:
		require.Error(t, err, "un pair muet doit rendre une erreur, pas un succès")
	case <-time.After(5 * time.Second):
		t.Fatal("MailboxStatus pend encore 5 s après l'échéance de 500 ms — la relève resterait gelée")
	}
}
