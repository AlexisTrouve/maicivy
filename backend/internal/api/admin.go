package api

import (
	"crypto/subtle"
	"time"

	"github.com/gofiber/fiber/v2"

	"maicivy/internal/middleware"
)

// AdminHandler — auth du panneau /admin : login (mot de passe → cookie owner signé), logout, me.
// POURQUOI : exposer les outils owner (génération CV depuis offre, etc.) à Alexi sans surface
// publique. Le cookie maicivy_admin posé ici est reconnu comme OWNER par le middleware AIRateLimit
// (via VerifyAdminCookie) → l'admin hérite des privilèges owner (Opus, bypass rate-limit) sans
// jamais manipuler la clé API côté browser.
type AdminHandler struct {
	adminPassword string
	sessionSecret string
}

// NewAdminHandler crée le handler. adminPassword/sessionSecret vides → login désactivé (cf. Login).
func NewAdminHandler(adminPassword, sessionSecret string) *AdminHandler {
	return &AdminHandler{adminPassword: adminPassword, sessionSecret: sessionSecret}
}

const (
	adminCookieName = "maicivy_admin"
	adminCookieTTL  = 12 * time.Hour // session admin courte — un cookie volé périme vite

	// Badge dev "je suis un copain" : cookie longue durée posé au login, exemptant le sus-rate-limit
	// (cf. middleware.SusConfig.FriendSecret). POURQUOI : le owner qui dev génère des 404 légitimes →
	// se faisait throttler. Ce cookie IP-indépendant le marque ami sur le checkpoint, sans allowlister
	// son IP (qui tourne). 90 j = pose-et-oublie ; révoqué en tournant SESSION_SECRET.
	friendCookieName = "maicivy_friend"
	friendCookieTTL  = 90 * 24 * time.Hour
)

func (h *AdminHandler) RegisterRoutes(api fiber.Router) {
	admin := api.Group("/admin")
	admin.Post("/login", h.Login)
	admin.Post("/logout", h.Logout)
	admin.Get("/me", h.Me) // sonde pour le guard frontend
}

type adminLoginRequest struct {
	Password string `json:"password"`
}

// setAdminCookie pose (ou efface si value=="") le cookie admin avec les mêmes attributs que la session.
func (h *AdminHandler) setAdminCookie(c *fiber.Ctx, value string, expires time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:     adminCookieName,
		Value:    value,
		Expires:  expires,
		HTTPOnly: true,  // inaccessible au JS → pas de vol via XSS
		Secure:   true,  // HTTPS only
		SameSite: "Lax", // pas envoyé en cross-site POST → anti-CSRF de base
	})
}

// setFriendCookie pose (ou efface si value=="") le badge dev maicivy_friend (mêmes attributs de
// sécurité que le cookie admin). Cookie séparé : durée longue (90 j) et finalité distincte
// (exemption sus côté front-door), indépendant de la session admin courte.
func (h *AdminHandler) setFriendCookie(c *fiber.Ctx, value string, expires time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:     friendCookieName,
		Value:    value,
		Expires:  expires,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
	})
}

// Login valide le mot de passe (constant-time) et pose le cookie admin signé. 401 sinon.
func (h *AdminHandler) Login(c *fiber.Ctx) error {
	// Login DÉSACTIVÉ si pas de mot de passe / secret configuré → 401 systématique (jamais de porte
	// ouverte par config absente).
	if h.adminPassword == "" || h.sessionSecret == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "admin login disabled"})
	}

	var req adminLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Comparaison constant-time → pas de fuite de timing sur le mot de passe.
	if subtle.ConstantTimeCompare([]byte(req.Password), []byte(h.adminPassword)) != 1 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid password"})
	}

	h.setAdminCookie(c, middleware.SignAdminCookie(h.sessionSecret, adminCookieTTL), time.Now().Add(adminCookieTTL))
	// Pose aussi le badge dev (90 j) → ton trafic navigateur est exempté du sus, IP-indépendamment.
	// On réutilise la MÊME signature (jeton admin: signé) ; seul le nom de cookie + le TTL diffèrent.
	h.setFriendCookie(c, middleware.SignAdminCookie(h.sessionSecret, friendCookieTTL), time.Now().Add(friendCookieTTL))
	return c.JSON(fiber.Map{"authenticated": true})
}

// Logout efface les cookies admin ET badge dev (valeur vide + déjà expiré) → logout propre.
func (h *AdminHandler) Logout(c *fiber.Ctx) error {
	h.setAdminCookie(c, "", time.Now().Add(-time.Hour))
	h.setFriendCookie(c, "", time.Now().Add(-time.Hour))
	return c.JSON(fiber.Map{"authenticated": false})
}

// Me renvoie 200 si le cookie admin est valide (sonde pour le guard frontend), 401 sinon.
func (h *AdminHandler) Me(c *fiber.Ctx) error {
	if middleware.VerifyAdminCookie(c.Cookies(adminCookieName), h.sessionSecret) {
		return c.JSON(fiber.Map{"authenticated": true})
	}
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"authenticated": false})
}
