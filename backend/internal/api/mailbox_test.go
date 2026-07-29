package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"maicivy/internal/middleware"
	"maicivy/internal/models"
	"maicivy/internal/services"
)

func setupMailboxHandlerTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.MailboxEmail{}, &models.MailboxCursor{}, &models.MailboxEmailTranslation{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	return db
}

func newMailboxApp(db *gorm.DB, secret string, service *services.MailboxService) *fiber.App {
	return newMailboxAppWithTranslator(db, secret, service, nil)
}

func newMailboxAppWithTranslator(db *gorm.DB, secret string, service *services.MailboxService, translator services.MailboxTranslator) *fiber.App {
	app := fiber.New()
	NewMailboxHandler(db, secret, service, translator).RegisterRoutes(app.Group("/api/v1"))
	return app
}

// fakeMailboxTranslator (tests API — traduction en mémoire, aucun appel réseau).
type fakeMailboxTranslator struct {
	subject, body string
	err           error
	calls         int
}

func (f *fakeMailboxTranslator) Translate(ctx context.Context, subject, body, lang string) (string, string, error) {
	f.calls++
	if f.err != nil {
		return "", "", f.err
	}
	return f.subject, f.body, nil
}

func mailboxAdminCookie(secret string) string {
	return adminCookieName + "=" + middleware.SignAdminCookie(secret, time.Hour)
}

func TestMailboxHandler_RequiresOwnerCookie(t *testing.T) {
	db := setupMailboxHandlerTestDB(t)
	app := newMailboxApp(db, "secret-hmac", nil)

	cases := []struct {
		method, path string
	}{
		{"GET", "/api/v1/admin/mailbox"},
		{"GET", "/api/v1/admin/mailbox/abc"},
		{"POST", "/api/v1/admin/mailbox/abc/read"},
		{"POST", "/api/v1/admin/mailbox/abc/forward"},
	}
	for _, tc := range cases {
		resp, _ := app.Test(httptest.NewRequest(tc.method, tc.path, nil))
		if resp.StatusCode != 401 {
			t.Fatalf("%s %s sans cookie : attendu 401, got %d", tc.method, tc.path, resp.StatusCode)
		}
	}
}

func TestMailboxHandler_List_PaginationFilterHidesBody(t *testing.T) {
	db := setupMailboxHandlerTestDB(t)
	now := time.Now()
	seed := []models.MailboxEmail{
		{MessageID: "m1@malt.fr", ImapUID: 1, FromAddress: "n@malt.fr", FromDomain: "malt.fr", Platform: "malt", Subject: "s1", BodyText: "corps confidentiel 1", ReceivedAt: now, Read: false},
		{MessageID: "m2@malt.fr", ImapUID: 2, FromAddress: "n@malt.fr", FromDomain: "malt.fr", Platform: "malt", Subject: "s2", BodyText: "corps confidentiel 2", ReceivedAt: now.Add(time.Minute), Read: true},
		{MessageID: "m3@comet.co", ImapUID: 3, FromAddress: "n@comet.co", FromDomain: "comet.co", Platform: "comet", Subject: "s3", BodyText: "corps confidentiel 3", ReceivedAt: now.Add(2 * time.Minute), Read: false},
	}
	for i := range seed {
		if err := db.Create(&seed[i]).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	app := newMailboxApp(db, "secret-hmac", nil)
	cookie := mailboxAdminCookie("secret-hmac")

	// Filtre platform=malt
	req := httptest.NewRequest("GET", "/api/v1/admin/mailbox?platform=malt", nil)
	req.Header.Set("Cookie", cookie)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("attendu 200, got %d", resp.StatusCode)
	}
	var body struct {
		Emails []map[string]interface{} `json:"emails"`
		Total  int                      `json:"total"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body.Total != 2 {
		t.Fatalf("filtre platform=malt : attendu 2 résultats, got %d", body.Total)
	}
	for _, e := range body.Emails {
		if _, ok := e["body_text"]; ok {
			t.Fatalf("la liste ne doit JAMAIS exposer body_text : %v", e)
		}
	}

	// Filtre unread=true
	req2 := httptest.NewRequest("GET", "/api/v1/admin/mailbox?unread=true", nil)
	req2.Header.Set("Cookie", cookie)
	resp2, _ := app.Test(req2)
	var body2 struct {
		Total int `json:"total"`
	}
	_ = json.NewDecoder(resp2.Body).Decode(&body2)
	if body2.Total != 2 {
		t.Fatalf("filtre unread=true : attendu 2 résultats, got %d", body2.Total)
	}
}

// Le verdict du filtre de pertinence (score/raison/blocage) doit apparaître dans la liste — c'est ce
// qui permet à Alexi de repêcher un faux négatif sans ouvrir chaque mail.
func TestMailboxHandler_List_ExposesRelevanceVerdict(t *testing.T) {
	db := setupMailboxHandlerTestDB(t)
	score := 25
	blocked := models.MailboxEmail{
		MessageID: "blocked@malt.fr", ImapUID: 1, FromAddress: "n@malt.fr", FromDomain: "malt.fr",
		Platform: "malt", Subject: "Mission hors profil", ReceivedAt: time.Now(),
		IsOpportunity: true, RelevanceScore: &score, RelevanceReason: "hors profil", ForwardBlocked: true,
		RelevanceCoT: "Checked get_experience: no matching domain found.", RelevanceLink: "https://malt.fr/mission/123",
	}
	if err := db.Create(&blocked).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	app := newMailboxApp(db, "secret-hmac", nil)
	cookie := mailboxAdminCookie("secret-hmac")

	req := httptest.NewRequest("GET", "/api/v1/admin/mailbox", nil)
	req.Header.Set("Cookie", cookie)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("attendu 200, got %d", resp.StatusCode)
	}
	var body struct {
		Emails []map[string]interface{} `json:"emails"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if len(body.Emails) != 1 {
		t.Fatalf("attendu 1 mail, got %d", len(body.Emails))
	}
	e := body.Emails[0]
	if e["is_opportunity"] != true {
		t.Errorf("is_opportunity attendu true, got %v", e["is_opportunity"])
	}
	if e["relevance_score"] != float64(25) {
		t.Errorf("relevance_score attendu 25, got %v", e["relevance_score"])
	}
	if e["relevance_reason"] != "hors profil" {
		t.Errorf("relevance_reason attendu 'hors profil', got %v", e["relevance_reason"])
	}
	if e["forward_blocked"] != true {
		t.Errorf("forward_blocked attendu true, got %v", e["forward_blocked"])
	}
	if e["relevance_link"] != "https://malt.fr/mission/123" {
		t.Errorf("relevance_link attendu, got %v", e["relevance_link"])
	}
	if _, ok := e["relevance_cot"]; ok {
		t.Fatalf("la liste ne doit JAMAIS exposer relevance_cot (potentiellement volumineux, réservé au détail): %v", e)
	}
}

func TestMailboxHandler_Detail_MarksRead(t *testing.T) {
	db := setupMailboxHandlerTestDB(t)
	email := models.MailboxEmail{
		MessageID: "detail@malt.fr", ImapUID: 1, FromAddress: "n@malt.fr", FromDomain: "malt.fr",
		Platform: "malt", Subject: "s", BodyText: "corps complet", ReceivedAt: time.Now(), Read: false,
	}
	if err := db.Create(&email).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	app := newMailboxApp(db, "secret-hmac", nil)
	cookie := mailboxAdminCookie("secret-hmac")

	req := httptest.NewRequest("GET", "/api/v1/admin/mailbox/"+email.ID.String(), nil)
	req.Header.Set("Cookie", cookie)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("attendu 200, got %d", resp.StatusCode)
	}
	var got models.MailboxEmail
	_ = json.NewDecoder(resp.Body).Decode(&got)
	if got.BodyText != "corps complet" {
		t.Fatalf("le détail doit inclure body_text, got %q", got.BodyText)
	}

	var reloaded models.MailboxEmail
	db.First(&reloaded, "id = ?", email.ID)
	if !reloaded.Read {
		t.Fatal("consulter le détail doit marquer le mail lu")
	}
}

// Régression réelle : un mail sans partie text/plain (fallback HTML brut, cf.
// ImapFetcher.extractPlainText) stocke des balises/CSS bruts en base — le détail doit les nettoyer
// (cf. services.StripHTMLNoise) SANS modifier la valeur persistée en DB.
func TestMailboxHandler_Detail_StripsHTMLFromBodyWithoutPersisting(t *testing.T) {
	db := setupMailboxHandlerTestDB(t)
	rawHTML := `<style type="text/css">.heading{font-size:32px;}</style>
<p class="heading">Bonjour Alexis,<br/>cette opportunité correspond à votre profil !</p>
<p class="section-details">Mission Go/Kubernetes, 3 mois, Paris.</p>`
	email := models.MailboxEmail{
		MessageID: "html@malt.fr", ImapUID: 1, FromAddress: "n@malt.fr", FromDomain: "malt.fr",
		Platform: "malt", Subject: "s", BodyText: rawHTML, ReceivedAt: time.Now(),
	}
	if err := db.Create(&email).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	app := newMailboxApp(db, "secret-hmac", nil)
	cookie := mailboxAdminCookie("secret-hmac")

	req := httptest.NewRequest("GET", "/api/v1/admin/mailbox/"+email.ID.String(), nil)
	req.Header.Set("Cookie", cookie)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("attendu 200, got %d", resp.StatusCode)
	}
	var got models.MailboxEmail
	_ = json.NewDecoder(resp.Body).Decode(&got)
	if strings.Contains(got.BodyText, "<style") || strings.Contains(got.BodyText, "<p") || strings.Contains(got.BodyText, "font-size") {
		t.Fatalf("le body_text renvoyé ne doit plus contenir de balises/CSS bruts, got: %q", got.BodyText)
	}
	if !strings.Contains(got.BodyText, "cette opportunité correspond à votre profil") {
		t.Fatalf("le texte utile doit être préservé, got: %q", got.BodyText)
	}

	// La valeur PERSISTÉE en DB doit rester brute (source de vérité) — seule la réponse est nettoyée.
	var reloaded models.MailboxEmail
	db.First(&reloaded, "id = ?", email.ID)
	if reloaded.BodyText != rawHTML {
		t.Fatalf("le body_text stocké en DB ne doit JAMAIS être modifié par la consultation, got: %q", reloaded.BodyText)
	}
}

func TestMailboxHandler_SetRead_Toggle(t *testing.T) {
	db := setupMailboxHandlerTestDB(t)
	email := models.MailboxEmail{
		MessageID: "toggle@malt.fr", ImapUID: 1, FromAddress: "n@malt.fr", FromDomain: "malt.fr",
		Platform: "malt", Subject: "s", ReceivedAt: time.Now(), Read: true,
	}
	if err := db.Create(&email).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	app := newMailboxApp(db, "secret-hmac", nil)
	cookie := mailboxAdminCookie("secret-hmac")

	req := httptest.NewRequest("POST", "/api/v1/admin/mailbox/"+email.ID.String()+"/read", strings.NewReader(`{"read":false}`))
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("attendu 200, got %d", resp.StatusCode)
	}

	var reloaded models.MailboxEmail
	db.First(&reloaded, "id = ?", email.ID)
	if reloaded.Read {
		t.Fatal("le toggle doit permettre de re-marquer non lu")
	}
}

func TestMailboxHandler_RetryForward_503WhenUnconfigured(t *testing.T) {
	db := setupMailboxHandlerTestDB(t)
	app := newMailboxApp(db, "secret-hmac", nil) // service nil = non configuré
	cookie := mailboxAdminCookie("secret-hmac")

	req := httptest.NewRequest("POST", "/api/v1/admin/mailbox/"+uuid.New().String()+"/forward", nil)
	req.Header.Set("Cookie", cookie)
	resp, _ := app.Test(req)
	if resp.StatusCode != 503 {
		t.Fatalf("service non configuré : attendu 503, got %d", resp.StatusCode)
	}
}

// noopFetcher — implémentation triviale d'ImapFetcher : RetryForward ne touche jamais à l'IMAP, mais
// un *services.MailboxService réel en a besoin au constructeur.
type noopFetcher struct{}

func (noopFetcher) MailboxStatus(ctx context.Context) (uint32, uint32, error) { return 0, 0, nil }
func (noopFetcher) FetchSince(ctx context.Context, sinceUID uint32) ([]services.ImapMessage, error) {
	return nil, nil
}
func (noopFetcher) Close() error { return nil }

func TestMailboxHandler_RetryForward_Success(t *testing.T) {
	db := setupMailboxHandlerTestDB(t)
	email := models.MailboxEmail{
		MessageID: "retry@malt.fr", ImapUID: 1, FromAddress: "n@malt.fr", FromDomain: "malt.fr",
		Platform: "malt", Subject: "s", ReceivedAt: time.Now(),
	}
	if err := db.Create(&email).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	var relayCalled bool
	relay := func(addr, from, to string, msg []byte) error {
		relayCalled = true
		return nil
	}
	service := services.NewMailboxService(db, noopFetcher{}, map[string]string{"malt.fr": "malt"},
		"smtp.local:25", "dest@example.com", "mailbox@etheryale.com", "Mailbox Etheryale", relay, nil)

	app := newMailboxApp(db, "secret-hmac", service)
	cookie := mailboxAdminCookie("secret-hmac")

	req := httptest.NewRequest("POST", "/api/v1/admin/mailbox/"+email.ID.String()+"/forward", nil)
	req.Header.Set("Cookie", cookie)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("attendu 200, got %d", resp.StatusCode)
	}
	if !relayCalled {
		t.Fatal("le relai SMTP doit avoir été appelé")
	}

	var reloaded models.MailboxEmail
	db.First(&reloaded, "id = ?", email.ID)
	if reloaded.ForwardedAt == nil {
		t.Fatal("ForwardedAt doit être posé après un retry réussi")
	}
}

func seedMailboxEmailForTranslation(t *testing.T, db *gorm.DB) models.MailboxEmail {
	t.Helper()
	e := models.MailboxEmail{
		MessageID: "trad@malt.fr", ImapUID: 1, FromAddress: "n@malt.fr", FromDomain: "malt.fr",
		Platform: "malt", Subject: "Sujet FR", BodyText: "Corps FR", ReceivedAt: time.Now(),
	}
	if err := db.Create(&e).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	return e
}

// GET translation : rien en cache → 404, ZÉRO appel au traducteur (jamais de coût juste en ouvrant un mail).
func TestMailboxHandler_GetTranslation_NoCache_404_NoTranslatorCall(t *testing.T) {
	db := setupMailboxHandlerTestDB(t)
	email := seedMailboxEmailForTranslation(t, db)
	translator := &fakeMailboxTranslator{subject: "EN subject", body: "EN body"}
	app := newMailboxAppWithTranslator(db, "secret-hmac", nil, translator)
	cookie := mailboxAdminCookie("secret-hmac")

	req := httptest.NewRequest("GET", "/api/v1/admin/mailbox/"+email.ID.String()+"/translation?lang=en", nil)
	req.Header.Set("Cookie", cookie)
	resp, _ := app.Test(req)

	if resp.StatusCode != 404 {
		t.Fatalf("attendu 404, got %d", resp.StatusCode)
	}
	if translator.calls != 0 {
		t.Fatal("un GET (check cache) ne doit jamais appeler le traducteur")
	}
}

// POST translation : traduit, met en cache, renvoie le résultat.
func TestMailboxHandler_TranslateNow_TranslatesAndCaches(t *testing.T) {
	db := setupMailboxHandlerTestDB(t)
	email := seedMailboxEmailForTranslation(t, db)
	translator := &fakeMailboxTranslator{subject: "EN subject", body: "EN body"}
	app := newMailboxAppWithTranslator(db, "secret-hmac", nil, translator)
	cookie := mailboxAdminCookie("secret-hmac")

	req := httptest.NewRequest("POST", "/api/v1/admin/mailbox/"+email.ID.String()+"/translation?lang=en", nil)
	req.Header.Set("Cookie", cookie)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("attendu 200, got %d", resp.StatusCode)
	}
	var body struct {
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body.Subject != "EN subject" || body.Body != "EN body" {
		t.Fatalf("réponse inattendue: %+v", body)
	}

	// Un GET (check cache) juste après doit maintenant trouver l'entrée, sans re-traduire.
	req2 := httptest.NewRequest("GET", "/api/v1/admin/mailbox/"+email.ID.String()+"/translation?lang=en", nil)
	req2.Header.Set("Cookie", cookie)
	resp2, _ := app.Test(req2)
	if resp2.StatusCode != 200 {
		t.Fatalf("attendu 200 (servi depuis le cache), got %d", resp2.StatusCode)
	}
	if translator.calls != 1 {
		t.Fatalf("attendu exactement 1 appel au traducteur (le 2e GET doit servir le cache), got %d", translator.calls)
	}
}

// Traducteur non configuré (credentials absentes) → 503 explicite, pas une 500 opaque.
func TestMailboxHandler_TranslateNow_503WhenUnconfigured(t *testing.T) {
	db := setupMailboxHandlerTestDB(t)
	email := seedMailboxEmailForTranslation(t, db)
	app := newMailboxAppWithTranslator(db, "secret-hmac", nil, nil) // translator nil
	cookie := mailboxAdminCookie("secret-hmac")

	req := httptest.NewRequest("POST", "/api/v1/admin/mailbox/"+email.ID.String()+"/translation?lang=en", nil)
	req.Header.Set("Cookie", cookie)
	resp, _ := app.Test(req)

	if resp.StatusCode != 503 {
		t.Fatalf("attendu 503, got %d", resp.StatusCode)
	}
}

// lang invalide/absente → 400, jamais transmis tel quel au prompt LLM.
func TestMailboxHandler_TranslateNow_InvalidLang_400(t *testing.T) {
	db := setupMailboxHandlerTestDB(t)
	email := seedMailboxEmailForTranslation(t, db)
	translator := &fakeMailboxTranslator{subject: "x", body: "y"}
	app := newMailboxAppWithTranslator(db, "secret-hmac", nil, translator)
	cookie := mailboxAdminCookie("secret-hmac")

	req := httptest.NewRequest("POST", "/api/v1/admin/mailbox/"+email.ID.String()+"/translation?lang=fr", nil)
	req.Header.Set("Cookie", cookie)
	resp, _ := app.Test(req)

	if resp.StatusCode != 400 {
		t.Fatalf("attendu 400 (fr n'est pas une cible valide, c'est la langue source), got %d", resp.StatusCode)
	}
	if translator.calls != 0 {
		t.Fatal("une lang invalide ne doit jamais atteindre le traducteur")
	}
}

// Routes translation : 401 sans cookie owner, comme le reste du handler.
func TestMailboxHandler_Translation_RequiresOwnerCookie(t *testing.T) {
	db := setupMailboxHandlerTestDB(t)
	email := seedMailboxEmailForTranslation(t, db)
	app := newMailboxAppWithTranslator(db, "secret-hmac", nil, &fakeMailboxTranslator{})

	cases := []struct{ method, path string }{
		{"GET", "/api/v1/admin/mailbox/" + email.ID.String() + "/translation?lang=en"},
		{"POST", "/api/v1/admin/mailbox/" + email.ID.String() + "/translation?lang=en"},
	}
	for _, tc := range cases {
		resp, _ := app.Test(httptest.NewRequest(tc.method, tc.path, nil))
		if resp.StatusCode != 401 {
			t.Fatalf("%s %s sans cookie : attendu 401, got %d", tc.method, tc.path, resp.StatusCode)
		}
	}
}
