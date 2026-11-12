package api

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"maicivy/internal/middleware"
	"maicivy/internal/services"
)

// CV adapté au besoin du visiteur — côté HTTP (le métier est dans services/cv_context.go).
//
//   GET /api/v1/cv/context?lang=xx      → { status, need?, source?, lang?, cv? }
//   GET /api/v1/cv/context/pdf?lang=xx  → PDF du CV adapté PRÊT, rendu à la demande (404 sinon)
//
// Un visiteur sans session SIGNÉE (pas de cookie, cookie forgé) reçoit toujours `none` : aucune
// lecture Redis, aucune génération. Les réponses portent `Cache-Control: private, no-store` — elles
// dépendent du cookie, un cache partagé servirait le CV d'un visiteur à un autre.

// NeedRecorder : range le besoin exprimé par un visiteur (CVContextService en prod). Interface pour
// que le chat et le dossier vivant ne dépendent que de ce geste.
type NeedRecorder interface {
	RecordNeed(ctx context.Context, v services.CVVisitor, source, summary, lang string) error
}

// cvContextSource : ce que les routes de lecture demandent au service.
type cvContextSource interface {
	Ensure(ctx context.Context, v services.CVVisitor, lang string) (services.CVContextView, error)
	AdaptedCV(ctx context.Context, sessionID, lang string) (*services.AdaptiveCVResponse, string, error)
}

// adaptedPDFRenderer : rend un CV adapté déjà généré (CVGenerationService.RenderDynamicPDF en prod).
type adaptedPDFRenderer func(cv *services.AdaptiveCVResponse, lang string) ([]byte, error)

// visitorOf : l'identité budgétaire du visiteur, seulement si sa session est signée par nous.
func visitorOf(c *fiber.Ctx) (services.CVVisitor, bool) {
	sid, ok := middleware.VerifiedSessionID(c)
	if !ok {
		return services.CVVisitor{}, false
	}
	return services.CVVisitor{SessionID: sid, IP: c.IP()}, true
}

// recordNeed : range un besoin hors du chemin critique de la requête appelante. Une erreur est
// journalisée et c'est tout : le chat ou le dossier du visiteur ne doit jamais échouer parce que la
// préparation (facultative) de son CV n'a pas pu être lancée.
func recordNeed(rec NeedRecorder, v services.CVVisitor, source, summary, lang string) {
	if err := rec.RecordNeed(context.Background(), v, source, summary, lang); err != nil {
		log.Warn().Err(err).Str("source", source).Msg("cv context: besoin non enregistré")
	}
}

// SetContext branche le CV adapté au besoin. nil → les routes répondent `none` / 404.
func (h *CVHandler) SetContext(src cvContextSource, render adaptedPDFRenderer) {
	h.cvCtx, h.renderAdapted = src, render
}

// GetContext : état du CV adapté pour la session, et lancement de sa génération s'il en manque une
// pour cette langue (single-flight et budget dans le service).
func (h *CVHandler) GetContext(c *fiber.Ctx) error {
	c.Set("Cache-Control", "private, no-store")
	v, ok := visitorOf(c)
	if !ok || h.cvCtx == nil {
		return c.JSON(services.CVContextView{Status: services.CVContextNone})
	}
	view, err := h.cvCtx.Ensure(c.Context(), v, c.Query("lang", "fr"))
	if err != nil {
		log.Error().Err(err).Msg("cv context: lecture impossible")
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "context unavailable"})
	}
	return c.JSON(view)
}

// GetContextPDF : le PDF du CV adapté, rendu depuis le JSON stocké — aucun appel IA ici.
func (h *CVHandler) GetContextPDF(c *fiber.Ctx) error {
	c.Set("Cache-Control", "private, no-store")
	v, ok := visitorOf(c)
	if !ok || h.cvCtx == nil || h.renderAdapted == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no adapted CV"})
	}
	cv, lang, err := h.cvCtx.AdaptedCV(c.Context(), v.SessionID, c.Query("lang", "fr"))
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "context unavailable"})
	}
	if cv == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no adapted CV"})
	}
	pdf, err := h.renderAdapted(cv, lang)
	if err != nil {
		log.Error().Err(err).Msg("cv context: rendu PDF échoué")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate PDF"})
	}
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", `attachment; filename="cv_adapte_`+lang+`.pdf"`)
	return c.Send(pdf)
}
