package middleware

import "github.com/gofiber/fiber/v2"

// RequireOwner — refuse (401) toute requête qui n'est pas celle de l'owner.
//
// QUOI : owner = cookie admin signé et non expiré (maicivy_admin, posé par le login du panneau)
// OU en-tête X-Owner-Key égal à la clé owner. Exactement la règle d'AIRateLimit, sans son repli :
// là où AIRateLimit laisse passer un visiteur sous quota, RequireOwner l'arrête.
//
// POURQUOI un middleware et pas un test dans le handler : une route réservée doit l'être AVANT
// d'entrer dans le handler — rien de ce qu'il ouvre (flux SSE, goroutines, appels de modèle) ne
// doit exister pour un visiteur. Clé owner vide = aucune clé ne passe (une chaîne vide n'ouvre rien).
func RequireOwner(sessionSecret, ownerKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if VerifyAdminCookie(c.Cookies("maicivy_admin"), sessionSecret) ||
			(ownerKey != "" && c.Get("X-Owner-Key") == ownerKey) {
			return c.Next()
		}
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "owner only", "code": "OWNER_ONLY"})
	}
}
