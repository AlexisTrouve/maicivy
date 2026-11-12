package api

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"maicivy/internal/middleware"
	"maicivy/internal/services"
)

// DossierPDFHandler — « Télécharger le dossier » du dossier vivant : le dossier remis, en PDF A4.
//
// RÈGLE : un PDF servi par une route PUBLIQUE ne contient que du contenu produit par le serveur. Le
// document porte le nom et les coordonnées d'Alexis Trouvé ; avant, ces routes imprimaient le dossier
// que le NAVIGATEUR envoyait, et n'importe qui pouvait fabriquer avec curl un PDF signé Alexis
// contenant n'importe quel texte. Désormais le corps ne porte plus que des identifiants et des choix
// fermés ; tout autre champ est ignoré.
//
// TROIS ROUTES, un seul rendu (services.DossierPDFService) :
//   - POST /api/v1/dossier-vivant/pdf (PUBLIQUE) `{lang}` — le dossier du REJEU Tessera, tel que le
//     serveur le connaît (services/dossier_tessera.go), avec la mention du rejeu. Ouverte parce
//     qu'elle ne coûte ni IA ni écriture. Charge bornée par le moteur (un Chromium à la fois) et par
//     le rate-limit global par IP.
//   - POST /api/v1/dossier-vivant/run/pdf (PUBLIQUE) `{dossierId, lang}` — le dossier d'un run RÉEL
//     exécuté par le backend : la copie gardée à la remise (RecordingEmitter → services.DossierStore,
//     24 h), désignée par l'id reçu dans run.completed. Id inconnu, mal formé ou expiré : 404
//     DOSSIER_NOT_FOUND, que le front nomme à l'écran.
//   - POST /api/v1/admin/dossier-vivant/pdf (OWNER) — le corps EST le dossier (contrat
//     DossierPDFRequest, validé et borné), derrière RequireOwner (sous /admin, donc derrière
//     AdminRoleGuard : un prospecteur y reçoit 403). Outil du propriétaire, inchangé.
//
// Et la section « Compétences mobilisées » du dossier À L'ÉCRAN :
//   - POST /api/v1/dossier-vivant/skills (PUBLIQUE) — laissée telle quelle, sections en entrée : elle
//     ne rend que du JSON, au seul demandeur, et ce JSON ne contient que des données du serveur
//     (compétences et niveaux du profil, technologies des fiches) indexées par les ids de preuve
//     reçus. Un corps forgé ne change que ce que voit celui qui l'a forgé : rien n'est imprimé, rien
//     n'est signé, rien n'est montré à un tiers. Le PDF, lui, recompose ses compétences depuis la
//     copie serveur.
type DossierPDFHandler struct {
	service       *services.DossierPDFService
	dossiers      dossierLoader
	sessionSecret string
	ownerKey      string
}

// dossierLoader — d'où vient le dossier d'un run réel (services.DossierStore).
type dossierLoader interface {
	Load(ctx context.Context, id string) (*services.DossierRecord, error)
}

func NewDossierPDFHandler(service *services.DossierPDFService, dossiers dossierLoader, sessionSecret, ownerKey string) *DossierPDFHandler {
	return &DossierPDFHandler{service: service, dossiers: dossiers, sessionSecret: sessionSecret, ownerKey: ownerKey}
}

// RegisterPublicRoutes — le PDF du rejeu, celui d'un run réel de visiteur, et les compétences.
func (h *DossierPDFHandler) RegisterPublicRoutes(api fiber.Router) {
	api.Post("/dossier-vivant/pdf", h.replay)
	api.Post("/dossier-vivant/run/pdf", h.run)
	api.Post("/dossier-vivant/skills", h.skills)
}

// RegisterOwnerRoutes — la route du dossier fourni par le propriétaire (à enregistrer avec /admin).
func (h *DossierPDFHandler) RegisterOwnerRoutes(api fiber.Router) {
	api.Post("/admin/dossier-vivant/pdf", middleware.RequireOwner(h.sessionSecret, h.ownerKey), h.owner)
}

// dossierPDFQuery — le corps des routes publiques : un identifiant et une langue, rien d'imprimable.
type dossierPDFQuery struct {
	DossierID string `json:"dossierId"`
	Lang      string `json:"lang"`
}

func (h *DossierPDFHandler) replay(c *fiber.Ctx) error {
	var q dossierPDFQuery
	if err := c.BodyParser(&q); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body", "code": "INVALID_REQUEST"})
	}
	pdf, err := h.service.RenderReplay(c.UserContext(), q.Lang)
	return h.send(c, pdf, err, "replay")
}

func (h *DossierPDFHandler) run(c *fiber.Ctx) error {
	var q dossierPDFQuery
	if err := c.BodyParser(&q); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body", "code": "INVALID_REQUEST"})
	}
	rec, err := h.dossiers.Load(c.UserContext(), q.DossierID)
	if err != nil {
		return h.send(c, nil, err, "run")
	}
	pdf, err := h.service.RenderRecord(c.UserContext(), rec, q.Lang)
	return h.send(c, pdf, err, "run")
}

func (h *DossierPDFHandler) owner(c *fiber.Ctx) error {
	var req services.DossierPDFRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body", "code": "INVALID_REQUEST"})
	}
	pdf, err := h.service.Render(c.UserContext(), &req, false)
	return h.send(c, pdf, err, "owner")
}

// send — le PDF, ou l'erreur NOMMÉE (le front affiche chaque code dans la langue du visiteur).
func (h *DossierPDFHandler) send(c *fiber.Ctx, pdf []byte, err error, route string) error {
	switch {
	case errors.Is(err, services.ErrDossierNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error(), "code": "DOSSIER_NOT_FOUND"})
	case errors.Is(err, services.ErrDossierPDFInvalid):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error(), "code": "INVALID_DOSSIER"})
	case errors.Is(err, services.ErrDossierPDFBusy):
		c.Set("Retry-After", "10")
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "pdf engine busy", "code": "PDF_BUSY"})
	case err != nil:
		log.Error().Err(err).Str("route", route).Msg("dossier vivant: PDF du dossier en échec")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "pdf generation failed", "code": "PDF_FAILED"})
	}
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", `attachment; filename="dossier-de-conception.pdf"`)
	c.Set("Cache-Control", "no-store")
	return c.Send(pdf)
}

// dossierSkillsRequest — la langue du dossier et ses sections (forme de la requête du PDF).
type dossierSkillsRequest struct {
	Lang     string                       `json:"lang"`
	Sections []services.DossierPDFSection `json:"sections"`
}

func (h *DossierPDFHandler) skills(c *fiber.Ctx) error {
	var req dossierSkillsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body", "code": "INVALID_REQUEST"})
	}
	skills, err := h.service.Skills(c.UserContext(), req.Lang, req.Sections)
	switch {
	case errors.Is(err, services.ErrDossierPDFInvalid):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error(), "code": "INVALID_DOSSIER"})
	case err != nil:
		log.Error().Err(err).Msg("dossier vivant: compétences du dossier en échec")
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "skills unavailable", "code": "SKILLS_FAILED"})
	}
	return c.JSON(skills)
}
