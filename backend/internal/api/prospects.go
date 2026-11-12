package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"maicivy/internal/middleware"
	"maicivy/internal/models"
)

// Comptes prospecteurs (apporteurs d'affaires) : invitation, login, gestion owner, attribution.
//
// POURQUOI dans un handler à part de AdminHandler : le login owner (mot de passe partagé →
// maicivy_admin) ne doit PAS changer de comportement. Tout ce qui concerne les prospecteurs vit ici,
// sur un cookie distinct (maicivy_prospect), et n'accorde jamais les privilèges owner.

const (
	prospectCookieTTL = 12 * time.Hour                    // même durée que la session owner
	prospectInviteTTL = 72 * time.Hour                    // une invitation oubliée ne reste pas ouverte indéfiniment
	prospectPwMin     = 10                                // longueur minimale du mot de passe choisi
	prospectPwMax     = 72                                // limite de bcrypt (au-delà, les octets sont ignorés)
	prospectRefLen    = 6                                 // court : il apparaît dans chaque lien envoyé
	prospectRefChars  = "abcdefghjkmnpqrstuvwxyz23456789" // sans 0/o/1/l/i : lisible à la dictée
)

// ProspectAllowedAdminRoutes : LA liste des routes /admin ouvertes au rôle prospecteur. Tout le reste
// est refusé (403) par middleware.AdminRoleGuard. Strictement ce qu'il faut au Composer (atelier +
// vérification), à l'Agent (ses conversations) et à la session du prospecteur ; le CV depuis offre passe par /cv/generate, hors
// /admin, sous le plafond prospecteur d'AIRateLimit.
var ProspectAllowedAdminRoutes = []middleware.AllowedRoute{
	{Method: "GET", Path: "/api/v1/admin/me"},
	{Method: "POST", Path: "/api/v1/admin/prospect/login"},
	{Method: "POST", Path: "/api/v1/admin/prospect/logout"},
	{Method: "GET", Path: "/api/v1/admin/invite/:token"},
	{Method: "POST", Path: "/api/v1/admin/invite/:token"},
	{Method: "POST", Path: "/api/v1/admin/compose/verify"},
	{Method: "GET", Path: "/api/v1/admin/compose/drafts"},
	{Method: "POST", Path: "/api/v1/admin/compose/drafts"},
	{Method: "GET", Path: "/api/v1/admin/compose/drafts/:id"},
	{Method: "PUT", Path: "/api/v1/admin/compose/drafts/:id"},
	{Method: "PATCH", Path: "/api/v1/admin/compose/drafts/:id/meta"},
	// Onglet Agent : CRUD de SES conversations (cloisonnées par AdminChatHandler.scope). Le streaming
	// passe par /chat/stream, hors /admin, sous AIRateLimit (modèle visiteur, toolset public).
	{Method: "GET", Path: "/api/v1/admin/chat/conversations"},
	{Method: "POST", Path: "/api/v1/admin/chat/conversations"},
	{Method: "GET", Path: "/api/v1/admin/chat/conversations/:id"},
	{Method: "PUT", Path: "/api/v1/admin/chat/conversations/:id"},
	{Method: "DELETE", Path: "/api/v1/admin/chat/conversations/:id"},
}

// ProspectAuth : implémentation de middleware.ProspectResolver. Relit le compte EN BASE à chaque
// requête : c'est ce qui rend la révocation effective dès la requête suivante, sans attendre
// l'expiration du cookie.
type ProspectAuth struct {
	db     *gorm.DB
	secret string
}

func NewProspectAuth(db *gorm.DB, secret string) *ProspectAuth {
	return &ProspectAuth{db: db, secret: secret}
}

func (a *ProspectAuth) ResolveProspect(c *fiber.Ctx) (middleware.ProspectIdentity, bool) {
	id, ok := middleware.VerifyProspectCookie(c.Cookies(middleware.ProspectCookieName), a.secret)
	if !ok {
		return middleware.ProspectIdentity{}, false
	}
	var p models.Prospect
	if err := a.db.Where("id = ? AND active = ?", id, true).First(&p).Error; err != nil {
		return middleware.ProspectIdentity{}, false
	}
	return middleware.ProspectIdentity{ID: p.ID, Name: p.Name, Ref: p.Ref}, true
}

// ProspectsHandler : routes des comptes prospecteurs.
type ProspectsHandler struct {
	db     *gorm.DB
	secret string
}

func NewProspectsHandler(db *gorm.DB, secret string) *ProspectsHandler {
	return &ProspectsHandler{db: db, secret: secret}
}

func (h *ProspectsHandler) RegisterRoutes(api fiber.Router) {
	// Owner : gestion des accès + attribution.
	api.Get("/admin/prospects", h.list)
	api.Post("/admin/prospects/invitations", h.invite)
	api.Post("/admin/prospects/:id/revoke", h.revoke)
	api.Post("/admin/prospects/:id/activate", h.activate)
	// Publiques : acceptation d'invitation et session prospecteur.
	api.Get("/admin/invite/:token", h.inviteInfo)
	api.Post("/admin/invite/:token", h.acceptInvite)
	api.Post("/admin/prospect/login", h.login)
	api.Post("/admin/prospect/logout", h.logout)
	// Publique, hors /admin : attribution d'une visite arrivée par ?ref=.
	api.Post("/ref", h.recordVisit)
}

func randomString(alphabet string, n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}

func hashInviteToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (h *ProspectsHandler) ownerOnly(c *fiber.Ctx) bool {
	if GetStatsAuth(c, h.secret) {
		return true
	}
	_ = c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "owner only"})
	return false
}

// prospectSummary : une ligne de l'écran owner, attribution comprise.
type prospectSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Ref       string `json:"ref"`
	Active    bool   `json:"active"`
	Pending   bool   `json:"pending"` // invitation pas encore acceptée
	Visits    int64  `json:"visits"`
	CVs       int64  `json:"cvs"`
	CreatedAt string `json:"created_at"`
}

func (h *ProspectsHandler) list(c *fiber.Ctx) error {
	if !h.ownerOnly(c) {
		return nil
	}
	var rows []models.Prospect
	if err := h.db.Order("created_at DESC").Find(&rows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "list failed"})
	}
	// Une seule requête d'agrégat pour tous les comptes, plutôt que deux COUNT par ligne.
	type agg struct {
		ProspectID string
		Kind       string
		N          int64
	}
	var counts []agg
	h.db.Model(&models.ProspectEvent{}).Select("prospect_id, kind, count(*) as n").
		Group("prospect_id, kind").Scan(&counts)
	byKey := map[string]int64{}
	for _, a := range counts {
		byKey[a.ProspectID+"|"+a.Kind] = a.N
	}
	out := make([]prospectSummary, 0, len(rows))
	for _, p := range rows {
		out = append(out, prospectSummary{
			ID: p.ID, Name: p.Name, Ref: p.Ref, Active: p.Active,
			Pending:   p.PasswordHash == "",
			Visits:    byKey[p.ID+"|"+models.ProspectEventVisit],
			CVs:       byKey[p.ID+"|"+models.ProspectEventCV],
			CreatedAt: p.CreatedAt.Format(time.RFC3339),
		})
	}
	return c.JSON(fiber.Map{"prospects": out})
}

type inviteRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// invite crée le compte en attente et renvoie le jeton EN CLAIR — une seule fois : seule son
// empreinte est stockée. Le front en fait le lien à copier.
func (h *ProspectsHandler) invite(c *fiber.Ctx) error {
	if !h.ownerOnly(c) {
		return nil
	}
	var req inviteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name required", "code": "NAME_REQUIRED"})
	}
	tokBytes := make([]byte, 32)
	_, _ = rand.Read(tokBytes)
	token := base64.RawURLEncoding.EncodeToString(tokBytes)
	exp := time.Now().Add(prospectInviteTTL)
	p := models.Prospect{
		ID: uuid.NewString(), Name: name, Ref: randomString(prospectRefChars, prospectRefLen),
		InviteTokenHash: hashInviteToken(token), InviteExpiresAt: &exp,
	}
	if email := strings.ToLower(strings.TrimSpace(req.Email)); email != "" {
		p.Email = &email
	}
	if err := h.db.Create(&p).Error; err != nil {
		// Nom/email déjà pris (index unique) — ou, très improbable, collision du code ref.
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "already exists", "code": "PROSPECT_EXISTS"})
	}
	return c.JSON(fiber.Map{"id": p.ID, "token": token, "ref": p.Ref, "expires_at": exp.Format(time.RFC3339)})
}

func (h *ProspectsHandler) setActive(c *fiber.Ctx, active bool) error {
	if !h.ownerOnly(c) {
		return nil
	}
	var p models.Prospect
	if err := h.db.First(&p, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	// Réactiver un compte dont l'invitation n'a jamais été acceptée lui donnerait un accès sans mot
	// de passe défini — impossible à utiliser, mais incohérent : refusé.
	if active && p.PasswordHash == "" {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "invitation pending", "code": "INVITE_PENDING"})
	}
	now := time.Now()
	updates := map[string]interface{}{"active": active}
	if active {
		updates["revoked_at"] = nil
	} else {
		updates["revoked_at"] = &now
		// Une révocation ferme aussi une invitation en attente.
		updates["invite_token_hash"] = ""
	}
	if err := h.db.Model(&p).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "update failed"})
	}
	return c.JSON(fiber.Map{"id": p.ID, "active": active})
}

func (h *ProspectsHandler) revoke(c *fiber.Ctx) error   { return h.setActive(c, false) }
func (h *ProspectsHandler) activate(c *fiber.Ctx) error { return h.setActive(c, true) }

// pendingInvite : invitation valide = empreinte connue, non consommée, non expirée.
func (h *ProspectsHandler) pendingInvite(token string) (models.Prospect, bool) {
	var p models.Prospect
	if token == "" {
		return p, false
	}
	if err := h.db.Where("invite_token_hash = ?", hashInviteToken(token)).First(&p).Error; err != nil {
		return p, false
	}
	if p.InviteExpiresAt == nil || time.Now().After(*p.InviteExpiresAt) || p.PasswordHash != "" {
		return p, false
	}
	return p, true
}

func (h *ProspectsHandler) inviteInfo(c *fiber.Ctx) error {
	p, ok := h.pendingInvite(c.Params("token"))
	if !ok {
		return c.Status(fiber.StatusGone).JSON(fiber.Map{"error": "invalid invitation", "code": "INVITE_INVALID"})
	}
	return c.JSON(fiber.Map{"name": p.Name})
}

type passwordRequest struct {
	Password string `json:"password"`
}

// acceptInvite : le prospecteur choisit son mot de passe. Le jeton est effacé dans la MÊME mise à
// jour, conditionnée à son empreinte : deux acceptations concurrentes ne peuvent pas réussir toutes
// les deux (usage unique, y compris en course).
func (h *ProspectsHandler) acceptInvite(c *fiber.Ctx) error {
	var req passwordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if len(req.Password) < prospectPwMin || len(req.Password) > prospectPwMax {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "password length", "code": "PASSWORD_LENGTH"})
	}
	p, ok := h.pendingInvite(c.Params("token"))
	if !ok {
		return c.Status(fiber.StatusGone).JSON(fiber.Map{"error": "invalid invitation", "code": "INVITE_INVALID"})
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "hash failed"})
	}
	now := time.Now()
	res := h.db.Model(&models.Prospect{}).
		Where("id = ? AND invite_token_hash = ?", p.ID, p.InviteTokenHash).
		Updates(map[string]interface{}{
			"password_hash": string(hash), "invite_token_hash": "", "active": true, "activated_at": &now,
		})
	if res.Error != nil || res.RowsAffected != 1 {
		return c.Status(fiber.StatusGone).JSON(fiber.Map{"error": "invalid invitation", "code": "INVITE_INVALID"})
	}
	return c.JSON(fiber.Map{"ok": true})
}

type prospectLoginRequest struct {
	Login    string `json:"login"` // nom ou email
	Password string `json:"password"`
}

// dummyHash : comparé quand le compte n'existe pas, pour que la durée de réponse ne révèle pas quels
// noms existent.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("maicivy-dummy-password"), bcrypt.DefaultCost)

func (h *ProspectsHandler) login(c *fiber.Ctx) error {
	if h.secret == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "login disabled"})
	}
	var req prospectLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	login := strings.TrimSpace(req.Login)
	var p models.Prospect
	err := h.db.Where("name = ? OR email = ?", login, strings.ToLower(login)).First(&p).Error
	hash := dummyHash
	if err == nil && p.PasswordHash != "" {
		hash = []byte(p.PasswordHash)
	}
	pwErr := bcrypt.CompareHashAndPassword(hash, []byte(req.Password))
	if err != nil || p.PasswordHash == "" || pwErr != nil || !p.Active {
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Error().Err(err).Msg("prospect login: lookup failed")
		}
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}
	h.setCookie(c, middleware.SignProspectCookie(h.secret, p.ID, prospectCookieTTL), time.Now().Add(prospectCookieTTL))
	return c.JSON(fiber.Map{"authenticated": true, "role": "prospect"})
}

func (h *ProspectsHandler) logout(c *fiber.Ctx) error {
	h.setCookie(c, "", time.Now().Add(-time.Hour))
	return c.JSON(fiber.Map{"authenticated": false})
}

// setCookie : mêmes attributs de sécurité que le cookie owner.
func (h *ProspectsHandler) setCookie(c *fiber.Ctx, value string, expires time.Time) {
	c.Cookie(&fiber.Cookie{
		Name: middleware.ProspectCookieName, Value: value, Expires: expires,
		HTTPOnly: true, Secure: true, SameSite: "Lax",
	})
}

type refRequest struct {
	Ref string `json:"ref"`
}

// recordVisit : attribution d'une visite arrivée par un lien `?ref=<code>`.
//
// CHOIX (le plus simple qui tienne) : le front poste le code UNE fois à l'arrivée ; on enregistre
// un événement, dédupliqué par (prospecteur, IP). Pas de cookie de suivi : l'IP suffit à ne pas
// compter les rechargements, et ne se contourne pas en vidant ses cookies. Toujours 204 : la réponse
// ne révèle pas si un code existe.
func (h *ProspectsHandler) recordVisit(c *fiber.Ctx) error {
	var req refRequest
	if err := c.BodyParser(&req); err != nil || len(req.Ref) == 0 || len(req.Ref) > 16 {
		return c.SendStatus(fiber.StatusNoContent)
	}
	var p models.Prospect
	if err := h.db.Where("ref = ?", strings.ToLower(req.Ref)).First(&p).Error; err != nil {
		return c.SendStatus(fiber.StatusNoContent)
	}
	key := c.IP()
	var n int64
	h.db.Model(&models.ProspectEvent{}).
		Where("prospect_id = ? AND kind = ? AND visitor_key = ?", p.ID, models.ProspectEventVisit, key).Count(&n)
	if n == 0 {
		h.db.Create(&models.ProspectEvent{ProspectID: p.ID, Kind: models.ProspectEventVisit, VisitorKey: key})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// RecordCV : attribution d'un CV généré par un prospecteur (branché sur CVHandler).
func (h *ProspectsHandler) RecordCV(prospectID string) {
	if err := h.db.Create(&models.ProspectEvent{ProspectID: prospectID, Kind: models.ProspectEventCV}).Error; err != nil {
		log.Error().Err(err).Msg("prospect: attribution CV échouée")
	}
}
