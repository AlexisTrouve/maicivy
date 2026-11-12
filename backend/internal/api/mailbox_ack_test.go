package api

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"maicivy/internal/models"
	"maicivy/internal/services"
)

// Lien « ✓ 标记为已处理 » — tests du handler PUBLIC (cf. mailbox_ack.go).

const ackTestSecret = "secret-hmac"

// ackFixture : une opportunité en base + l'app montée, sans service IMAP (le lien doit marcher même
// quand le service n'est pas configuré : la donnée est en base, le lien a déjà été envoyé).
func ackFixture(t *testing.T) (*gorm.DB, *fiber.App, uuid.UUID) {
	t.Helper()
	db := setupMailboxHandlerTestDB(t)
	score := 87
	e := models.MailboxEmail{
		MessageID: "op@malt.fr", FromAddress: "n@malt.fr", FromDomain: "malt.fr", Platform: "malt",
		Subject: "Caraxo", ReceivedAt: time.Now(), IsOpportunity: true, RelevanceScore: &score,
	}
	e.ID = uuid.New()
	if err := db.Create(&e).Error; err != nil {
		t.Fatal(err)
	}
	return db, newMailboxApp(db, ackTestSecret, nil), e.ID
}

func ackChemin(id uuid.UUID, sig string) string {
	return "/api/v1/mailbox/ack/" + id.String() + "?sig=" + sig
}

func ackFaire(t *testing.T, app *fiber.App, method, path string) (int, string) {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest(method, path, nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func handledAt(t *testing.T, db *gorm.DB, id uuid.UUID) *time.Time {
	t.Helper()
	var e models.MailboxEmail
	if err := db.First(&e, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	return e.HandledAt
}

// Le GET NE MARQUE RIEN : les messageries ouvrent les liens pour les analyser. Un GET qui marquerait
// ferait passer chaque offre pour traitée à sa réception et couperait les rappels en silence.
func TestMailboxAck_GetAfficheLaConfirmationSansMarquer(t *testing.T) {
	db, app, id := ackFixture(t)
	sig := services.MailboxAckSig(id.String(), ackTestSecret)

	code, body := ackFaire(t, app, "GET", ackChemin(id, sig))

	if code != 200 {
		t.Fatalf("GET signé: attendu 200, reçu %d", code)
	}
	if !strings.Contains(body, `method="post"`) || !strings.Contains(body, `data-testid="ack-confirm"`) {
		t.Errorf("la page doit proposer un bouton qui POSTe, reçu:\n%s", body)
	}
	if h := handledAt(t, db, id); h != nil {
		t.Errorf("un GET ne doit JAMAIS marquer l'offre traitée (scanner de liens), handled_at=%v", h)
	}
}

// Le POST signé marque l'offre ; un second POST garde la date du PREMIER (elle mesure le délai).
func TestMailboxAck_PostMarqueUneSeuleFois(t *testing.T) {
	db, app, id := ackFixture(t)
	sig := services.MailboxAckSig(id.String(), ackTestSecret)

	code, body := ackFaire(t, app, "POST", ackChemin(id, sig))
	if code != 200 || !strings.Contains(body, "已处理") {
		t.Fatalf("POST signé: attendu 200 + confirmation, reçu %d:\n%s", code, body)
	}
	premier := handledAt(t, db, id)
	if premier == nil {
		t.Fatal("handled_at doit être posé après le POST")
	}

	time.Sleep(5 * time.Millisecond)
	code, body = ackFaire(t, app, "POST", ackChemin(id, sig))
	if code != 200 || !strings.Contains(body, "此前已标记为已处理") {
		t.Fatalf("second POST: attendu 200 « déjà marqué », reçu %d:\n%s", code, body)
	}
	if second := handledAt(t, db, id); second == nil || !second.Equal(*premier) {
		t.Errorf("la date du premier clic doit être conservée: %v puis %v", premier, second)
	}
}

// Signature absente, fausse, d'un autre mail, ou d'un autre secret → 403 et rien n'est marqué.
func TestMailboxAck_SignatureInvalideRefusee(t *testing.T) {
	db, app, id := ackFixture(t)
	autre := uuid.New().String()

	for nom, sig := range map[string]string{
		"absente":      "",
		"bidon":        "AAAA",
		"autre mail":   services.MailboxAckSig(autre, ackTestSecret),
		"autre secret": services.MailboxAckSig(id.String(), "un-autre-secret"),
	} {
		for _, m := range []string{"GET", "POST"} {
			if code, _ := ackFaire(t, app, m, ackChemin(id, sig)); code != 403 {
				t.Errorf("%s signature %s: attendu 403, reçu %d", m, nom, code)
			}
		}
	}
	if h := handledAt(t, db, id); h != nil {
		t.Errorf("aucune signature invalide ne doit marquer l'offre, handled_at=%v", h)
	}
}

// Id inconnu mais bien signé → 404, pas de 500 ni de faux « c'est fait ».
func TestMailboxAck_IdInconnu(t *testing.T) {
	_, app, _ := ackFixture(t)
	inconnu := uuid.New()
	code, _ := ackFaire(t, app, "POST", ackChemin(inconnu, services.MailboxAckSig(inconnu.String(), ackTestSecret)))
	if code != 404 {
		t.Errorf("id inconnu: attendu 404, reçu %d", code)
	}
}
