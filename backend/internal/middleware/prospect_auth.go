package middleware

import (
	"crypto/hmac"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Cookie PROSPECTEUR signé HMAC — format `prospect:<id>:<expiryUnix>.<sig>`.
//
// QUOI : jeton de session d'un apporteur d'affaires, portant son IDENTITÉ (id du compte) et, par son
// préfixe, son RÔLE.
//
// POURQUOI un cookie distinct de maicivy_admin plutôt qu'un rôle ajouté au cookie owner : le cookie
// owner et tout ce qui le lit (AIRateLimit, sus_ratelimit, GetStatsAuth) restent strictement
// inchangés. Et un jeton prospecteur ne peut JAMAIS valider comme owner : VerifyAdminCookie exige le
// préfixe "admin:", celui-ci porte "prospect:" — la signature couvre le préfixe, il est infalsifiable.
//
// COMMENT : même primitive que le cookie owner (sessionSig, HMAC-SHA256 base64url). La signature ne
// suffit PAS à autoriser : elle prouve l'identité, l'état actif du compte est relu en base à chaque
// requête par un ProspectResolver — c'est ce qui rend la révocation immédiate.

// ProspectCookieName : nom du cookie de session prospecteur.
const ProspectCookieName = "maicivy_prospect"

const prospectCookiePrefix = "prospect:"

// SignProspectCookie produit le jeton de session du prospecteur `id`, valable `ttl`.
func SignProspectCookie(secret, id string, ttl time.Duration) string {
	payload := prospectCookiePrefix + id + ":" + strconv.FormatInt(time.Now().Add(ttl).Unix(), 10)
	return payload + "." + sessionSig(payload, secret)
}

// VerifyProspectCookie renvoie l'id du prospecteur si `token` a été émis avec `secret` et n'est pas
// expiré. Ne dit RIEN de l'état du compte (cf. ProspectResolver).
func VerifyProspectCookie(token, secret string) (string, bool) {
	if secret == "" || token == "" {
		return "", false
	}
	i := strings.LastIndexByte(token, '.')
	if i <= 0 || i >= len(token)-1 {
		return "", false
	}
	payload, sig := token[:i], token[i+1:]
	if !hmac.Equal([]byte(sig), []byte(sessionSig(payload, secret))) {
		return "", false
	}
	if !strings.HasPrefix(payload, prospectCookiePrefix) {
		return "", false
	}
	rest := payload[len(prospectCookiePrefix):]
	j := strings.LastIndexByte(rest, ':')
	if j <= 0 {
		return "", false
	}
	expUnix, err := strconv.ParseInt(rest[j+1:], 10, 64)
	if err != nil || time.Now().Unix() >= expUnix {
		return "", false
	}
	return rest[:j], true
}

// ProspectIdentity : ce que le reste du backend sait d'un prospecteur authentifié ET actif.
type ProspectIdentity struct {
	ID   string
	Name string
	Ref  string
}

// ProspectResolver authentifie un prospecteur depuis la requête : cookie valide ET compte actif en
// base. Interface (et non accès direct à la base) pour que le middleware ignore le stockage.
type ProspectResolver interface {
	ResolveProspect(c *fiber.Ctx) (ProspectIdentity, bool)
}

// localsProspect : clé des locals où AdminRoleGuard / AIRateLimit déposent l'identité résolue.
const localsProspect = "prospect"

// ProspectFromCtx renvoie le prospecteur déposé dans les locals par un middleware de ce paquet.
func ProspectFromCtx(c *fiber.Ctx) (ProspectIdentity, bool) {
	p, ok := c.Locals(localsProspect).(ProspectIdentity)
	return p, ok
}

// AllowedRoute : une route du panneau ouverte au rôle prospecteur. Path est un MOTIF Fiber complet
// (ex. /api/v1/admin/compose/drafts/:id) : un segment `:x` accepte n'importe quel segment non vide.
type AllowedRoute struct {
	Method string
	Path   string
}

// matches dit si la requête (méthode, chemin concret) correspond au motif.
func (r AllowedRoute) matches(method, path string) bool {
	if r.Method != method {
		return false
	}
	want := strings.Split(strings.TrimSuffix(r.Path, "/"), "/")
	got := strings.Split(strings.TrimSuffix(path, "/"), "/")
	if len(want) != len(got) {
		return false
	}
	for i := range want {
		if strings.HasPrefix(want[i], ":") {
			if got[i] == "" {
				return false
			}
			continue
		}
		if want[i] != got[i] {
			return false
		}
	}
	return true
}

// AdminRoleGuard : autorisation PAR DÉFAUT-REFUS du rôle prospecteur sur tout /admin/*.
//
// POURQUOI un middleware monté sur le préfixe plutôt qu'un contrôle dans chaque handler : un handler
// owner ajouté demain (ou oublié aujourd'hui) n'a rien à faire pour rester fermé aux prospecteurs.
// La seule façon d'ouvrir une route est de l'inscrire dans `allowed` — et le test qui énumère les
// routes réellement enregistrées échoue pour toute autre.
//
// COMMENT :
//  1. cookie owner valide → rien ne change pour l'owner, on passe ;
//  2. prospecteur résolu (cookie signé + compte actif en base) → identité déposée dans les locals ;
//     route hors liste → 403 ;
//  3. personne d'identifié → on passe : chaque handler applique sa garde habituelle (401), et les
//     routes publiques (login, acceptation d'invitation) restent atteignables.
func AdminRoleGuard(sessionSecret string, prospects ProspectResolver, allowed []AllowedRoute) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if VerifyAdminCookie(c.Cookies("maicivy_admin"), sessionSecret) {
			return c.Next()
		}
		p, ok := prospects.ResolveProspect(c)
		if !ok {
			return c.Next()
		}
		c.Locals(localsProspect, p)
		for _, r := range allowed {
			if r.matches(c.Method(), c.Path()) {
				return c.Next()
			}
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "forbidden for prospect role",
			"code":  "PROSPECT_FORBIDDEN",
		})
	}
}
