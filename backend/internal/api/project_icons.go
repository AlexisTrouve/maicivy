package api

import (
	"regexp"

	"github.com/gofiber/fiber/v2"

	"maicivy/internal/services"
)

// ProjectIconsHandler — les icônes des projets, prises dans leurs dépôts Gitea (cf.
// services/project_icons.go pour la recherche, le cache et les garde-fous).
//
//   - GET /api/v1/project-icons     → { "icons": { "<id projet>": "/api/v1/project-icons/<id>?v=<version>" } }
//     les seuls projets qui ont une icône ; vide tant que l'index se construit (le front affiche
//     alors les monogrammes).
//   - GET /api/v1/project-icons/:id → l'icône, PNG réduit et ré-encodé par le backend ; 404 si le
//     projet n'en a pas — ou n'est pas affiché, ou vit dans un dépôt exclu : la réponse ne
//     distingue pas ces cas, pour ne rien apprendre sur les dépôts à qui sonde la route.
//
// Publiques et sans session : ce sont des images de la vitrine, sans coût IA.
type ProjectIconsHandler struct {
	service *services.ProjectIconService
}

// NewProjectIconsHandler — service nil accepté (Gitea non configuré) : aucune icône, routes vivantes.
func NewProjectIconsHandler(service *services.ProjectIconService) *ProjectIconsHandler {
	return &ProjectIconsHandler{service: service}
}

func (h *ProjectIconsHandler) RegisterRoutes(api fiber.Router) {
	api.Get("/project-icons", h.list)
	api.Get("/project-icons/:id", h.icon)
}

// idProjet — forme d'un id maiProFiles (« grove_engine », « video_to_mp3_transcriptor »). Tout autre
// paramètre est refusé AVANT de consulter l'index : ni chemin, ni nom de dépôt n'y entre.
var idProjet = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)

const prefixeIcone = "/api/v1/project-icons/"

func (h *ProjectIconsHandler) list(c *fiber.Ctx) error {
	icons := map[string]string{}
	for id, v := range h.service.Versions(c.Context()) {
		icons[id] = prefixeIcone + id + "?v=" + v
	}
	// Court : la liste change quand une icône apparaît dans un dépôt, et l'index vide du premier
	// passage ne doit pas rester collé dans un cache intermédiaire.
	c.Set(fiber.HeaderCacheControl, "public, max-age=300")
	return c.JSON(fiber.Map{"icons": icons})
}

func (h *ProjectIconsHandler) icon(c *fiber.Ctx) error {
	id := c.Params("id")
	if !idProjet.MatchString(id) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "icône introuvable"})
	}
	data, version, ok := h.service.Icon(c.Context(), id)
	if !ok {
		c.Set(fiber.HeaderCacheControl, "public, max-age=300")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "icône introuvable"})
	}
	// L'URL de la liste porte la version : à version égale, l'image ne change jamais.
	if c.Query("v") == version {
		c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
	} else {
		c.Set(fiber.HeaderCacheControl, "public, max-age=3600")
	}
	c.Set(fiber.HeaderContentType, "image/png")
	c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
	// Servie au navigateur d'une autre origine (front ≠ API en local) et lue par <img> : CORP explicite.
	c.Set("Cross-Origin-Resource-Policy", "cross-origin")
	return c.Send(data)
}
