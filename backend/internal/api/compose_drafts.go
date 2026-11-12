package api

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"maicivy/internal/middleware"
	"maicivy/internal/models"
)

// ComposeDraftsHandler — persistance de l'atelier de composition (owner-only).
//
// POURQUOI : sans elle, un rafraîchissement de page perd le message en cours. L'opérateur ne relit pas
// le français et vient potentiellement de passer plusieurs tours à faire ajuster le texte : il ne peut
// pas reconstituer le travail perdu.
//
// Le front crée un atelier au lancement, puis sauvegarde l'état complet après chaque tour d'agent. Au
// chargement, il relit le plus récent — d'où le tri décroissant sur updated_at.
type ComposeDraftsHandler struct {
	db            *gorm.DB
	sessionSecret string
}

func NewComposeDraftsHandler(db *gorm.DB, sessionSecret string) *ComposeDraftsHandler {
	return &ComposeDraftsHandler{db: db, sessionSecret: sessionSecret}
}

func (h *ComposeDraftsHandler) RegisterRoutes(api fiber.Router) {
	g := api.Group("/admin/compose/drafts")
	g.Get("/", h.list)
	g.Get("/:id", h.get)
	g.Post("/", h.create)
	g.Put("/:id", h.save)
	g.Patch("/:id/meta", h.meta)
}

// draftsListLimit : profondeur d'historique proposée dans la liste. Au-delà, l'opératrice cherchera
// plutôt dans les archives — et une liste déroulante interminable ne sert personne.
const draftsListLimit = 30

// composeDraftPayload : état complet échangé avec le front. Messages et Checks restent du JSON opaque —
// le backend n'a pas à connaître leur forme, et la faire évoluer côté front ne doit pas demander une
// migration ici.
type composeDraftPayload struct {
	Title         string          `json:"title"`
	Announce      string          `json:"announce"`
	Draft         string          `json:"draft"`
	Translation   string          `json:"translation"`
	Rate          int             `json:"rate"`
	RateRationale string          `json:"rate_rationale"`
	Messages      json.RawMessage `json:"messages"`
	Checks        json.RawMessage `json:"checks"`
}

// composeDraftResponse : l'atelier COMPLET, servi à l'unité quand on l'ouvre.
type composeDraftResponse struct {
	ID string `json:"id"`
	composeDraftPayload
	UpdatedAt string `json:"updated_at"`
	// SentAt : date d'envoi, absente si l'atelier n'est pas envoyé. Nécessaire à l'écran pour relire
	// l'état du bouton « Envoyé » à l'ouverture d'un atelier (régression du 2026-10-12 : seule la
	// liste le portait, le bouton retombait à zéro au rechargement).
	SentAt string `json:"sent_at,omitempty"`
}

// composeDraftSummary : ce que voit la LISTE. Volontairement sans announce, sans draft, sans
// traduction et surtout SANS le fil de conversation.
//
// POURQUOI c'est structurel et pas une micro-optimisation : le fil contient les captures d'écran en
// base64. Mesuré en production avant correction — 3 ateliers pesaient 468 kB, dont un seul à 455 kB.
// L'écran téléchargeait l'intégralité des derniers ateliers à chaque ouverture pour n'en restaurer
// qu'un. Garder l'historique, ce qui est précisément la demande, aggrave ce défaut à chaque atelier
// supplémentaire.
//
// Ce qui reste est exactement ce qu'il faut pour CHOISIR : de quoi ça parle, quand, à quel tarif, et
// si c'était prêt et envoyé.
type composeDraftSummary struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Rate  int    `json:"rate"`
	// HasDraft : un atelier peut n'être qu'une annonce collée sans message produit. La liste doit
	// distinguer les deux, sinon on rouvre un atelier vide en croyant retrouver un message.
	HasDraft    bool   `json:"has_draft"`
	ChecksPass  int    `json:"checks_pass"`
	ChecksTotal int    `json:"checks_total"`
	SentAt      string `json:"sent_at,omitempty"`
	ArchivedAt  string `json:"archived_at,omitempty"`
	UpdatedAt   string `json:"updated_at"`
}

// draftScope : à qui appartiennent les ateliers visibles par cette requête.
//
// POURQUOI : un prospecteur ne voit et ne modifie QUE ses ateliers ; l'owner, les siens (prospect_id
// NULL, ce qui inclut tous les ateliers antérieurs). Chaque requête passe par ce filtre — un atelier
// d'un autre compte est donc INTROUVABLE (404), on ne révèle même pas qu'il existe.
type draftScope struct {
	prospectID *string // nil = owner
}

func (s draftScope) apply(q *gorm.DB) *gorm.DB {
	if s.prospectID == nil {
		return q.Where("prospect_id IS NULL")
	}
	return q.Where("prospect_id = ?", *s.prospectID)
}

// scope authentifie (owner ou prospecteur actif, cf. AdminRoleGuard) et renvoie le périmètre. 401
// sinon.
func (h *ComposeDraftsHandler) scope(c *fiber.Ctx) (draftScope, bool) {
	if GetStatsAuth(c, h.sessionSecret) { // réutilise VerifyAdminCookie (cf. admin_stats.go)
		return draftScope{}, true
	}
	if p, ok := middleware.ProspectFromCtx(c); ok {
		id := p.ID
		return draftScope{prospectID: &id}, true
	}
	_ = c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "owner only"})
	return draftScope{}, false
}

// applyPayload reporte le payload sur la ligne. COMMENT : les champs JSON opaques ont un défaut non
// vide — une colonne jsonb à chaîne vide casse la relecture postgres, alors qu'un tableau/objet vide
// se relit sans cas particulier côté front.
func applyPayload(row *models.ComposeDraft, p composeDraftPayload) {
	row.Title = p.Title
	row.Announce = p.Announce
	row.Draft = p.Draft
	row.Translation = p.Translation
	row.Rate = p.Rate
	row.RateRationale = p.RateRationale

	row.Messages = "[]"
	if len(p.Messages) > 0 {
		row.Messages = string(p.Messages)
	}
	row.Checks = "{}"
	if len(p.Checks) > 0 {
		row.Checks = string(p.Checks)
	}
}

func toResponse(row models.ComposeDraft) composeDraftResponse {
	return composeDraftResponse{
		ID: row.ID,
		composeDraftPayload: composeDraftPayload{
			Title:         row.Title,
			Announce:      row.Announce,
			Draft:         row.Draft,
			Translation:   row.Translation,
			Rate:          row.Rate,
			RateRationale: row.RateRationale,
			Messages:      json.RawMessage(row.Messages),
			Checks:        json.RawMessage(row.Checks),
		},
		UpdatedAt: row.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		SentAt:    iso(row.SentAt),
	}
}

// checksSummary compte les contrôles au vert sur le total, à partir du JSON opaque de la checklist.
//
// POURQUOI le backend regarde ici un JSON qu'il traite partout ailleurs comme opaque : la liste doit
// dire « prêt / pas prêt » d'un coup d'œil, et faire ce comptage côté front obligerait à lui envoyer
// toutes les checklists — donc à réintroduire exactement le poids qu'on cherche à éviter. On lit la
// forme minimale (deux tableaux, un champ status) et toute déviation donne simplement 0/0.
func checksSummary(raw string) (pass, total int) {
	var parsed struct {
		Mechanical []struct {
			Status string `json:"status"`
		} `json:"mechanical"`
		Semantic []struct {
			Status string `json:"status"`
		} `json:"semantic"`
	}
	if json.Unmarshal([]byte(raw), &parsed) != nil {
		return 0, 0
	}
	for _, c := range parsed.Mechanical {
		// Les items « non applicables » ne comptent NI en réussite NI au total : les faire peser
		// afficherait 5/6 sur un message pourtant irréprochable, ce qui inquiéterait pour rien.
		if c.Status == "na" {
			continue
		}
		total++
		if c.Status == "pass" {
			pass++
		}
	}
	for _, c := range parsed.Semantic {
		if c.Status == "na" {
			continue
		}
		total++
		if c.Status == "pass" {
			pass++
		}
	}
	return pass, total
}

func iso(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02T15:04:05Z07:00")
}

// list — ateliers, le plus récent d'abord. `?archived=true` bascule sur les archives.
//
// Ne renvoie que des RÉSUMÉS : cf. composeDraftSummary pour la raison. Le contenu se lit à l'unité.
func (h *ComposeDraftsHandler) list(c *fiber.Ctx) error {
	sc, ok := h.scope(c)
	if !ok {
		return nil
	}

	q := sc.apply(h.db.Order("updated_at DESC").Limit(draftsListLimit))
	if c.Query("archived") == "true" {
		q = q.Where("archived_at IS NOT NULL")
	} else {
		q = q.Where("archived_at IS NULL")
	}

	var rows []models.ComposeDraft
	if err := q.Find(&rows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "list failed"})
	}

	out := make([]composeDraftSummary, 0, len(rows))
	for _, row := range rows {
		pass, total := checksSummary(row.Checks)
		out = append(out, composeDraftSummary{
			ID:          row.ID,
			Title:       row.Title,
			Rate:        row.Rate,
			HasDraft:    strings.TrimSpace(row.Draft) != "",
			ChecksPass:  pass,
			ChecksTotal: total,
			SentAt:      iso(row.SentAt),
			ArchivedAt:  iso(row.ArchivedAt),
			UpdatedAt:   row.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return c.JSON(fiber.Map{"drafts": out})
}

// get — un atelier COMPLET. Pendant de la liste allégée : on ne paie le poids du fil (captures
// comprises) que pour celui qu'on ouvre réellement.
//
// Sert aussi bien à reprendre un atelier actif qu'à relire une archive : rien n'est perdu à
// l'archivage, c'est précisément ce qui le distingue d'une suppression.
func (h *ComposeDraftsHandler) get(c *fiber.Ctx) error {
	sc, ok := h.scope(c)
	if !ok {
		return nil
	}

	var row models.ComposeDraft
	if err := sc.apply(h.db).First(&row, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "draft not found"})
	}
	return c.JSON(toResponse(row))
}

// composeDraftMeta : métadonnées modifiables sans toucher à l'état de l'atelier.
//
// POURQUOI des POINTEURS : il faut distinguer « ce champ n'est pas dans la requête » de « ce champ
// vaut faux/vide ». Sans ça, marquer un atelier comme envoyé effacerait son titre au passage — un
// effet de bord silencieux et pénible à diagnostiquer.
type composeDraftMeta struct {
	Title    *string `json:"title"`
	Archived *bool   `json:"archived"`
	Sent     *bool   `json:"sent"`
}

// meta — renommer, archiver/réactiver, marquer envoyé.
//
// POURQUOI un endpoint distinct du PUT d'état : renommer ne doit pas obliger le front à réémettre
// l'annonce, le draft, la traduction et tout le fil. C'est absurde en volume, et surtout dangereux —
// un renommage écraserait alors un état plus récent que celui que le front a en mémoire.
//
// On n'expose AUCUNE suppression, à dessein : un message envoyé à un client est une trace qu'on peut
// vouloir relire des mois plus tard, et une suppression est irréversible là où un archivage ne
// l'est pas.
func (h *ComposeDraftsHandler) meta(c *fiber.Ctx) error {
	sc, ok := h.scope(c)
	if !ok {
		return nil
	}

	var m composeDraftMeta
	if err := c.BodyParser(&m); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var row models.ComposeDraft
	if err := sc.apply(h.db).First(&row, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "draft not found"})
	}

	if m.Title != nil {
		// Même borne que le nommage automatique (composeTitleMaxRunes) : la colonne est un
		// varchar(200) et postgres TRONQUE en silence sur un cast trop court. En runes, pour ne pas
		// couper un caractère accentué ou chinois en deux.
		titre := strings.TrimSpace(*m.Title)
		if r := []rune(titre); len(r) > draftTitleMaxRunes {
			titre = strings.TrimSpace(string(r[:draftTitleMaxRunes]))
		}
		row.Title = titre
	}
	if m.Archived != nil {
		row.ArchivedAt = nowIf(*m.Archived)
	}
	if m.Sent != nil {
		row.SentAt = nowIf(*m.Sent)
	}

	if err := h.db.Save(&row).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "save failed"})
	}
	return c.JSON(fiber.Map{"id": row.ID})
}

// draftTitleMaxRunes : même borne que le nommage automatique côté service. Dupliquée plutôt
// qu'importée — la constante du service n'est pas exportée, et un titre saisi à la main n'a pas de
// raison de dépendre du module qui en génère.
const draftTitleMaxRunes = 60

// nowIf : horodate si l'état est demandé, remet à nul sinon. Un seul chemin pour les deux bascules
// (archivé, envoyé), qui se comportent exactement pareil.
func nowIf(on bool) *time.Time {
	if !on {
		return nil
	}
	t := time.Now()
	return &t
}

func (h *ComposeDraftsHandler) create(c *fiber.Ctx) error {
	sc, ok := h.scope(c)
	if !ok {
		return nil
	}

	var p composeDraftPayload
	if err := c.BodyParser(&p); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Le propriétaire vient de la SESSION, jamais du corps de la requête.
	row := models.ComposeDraft{ID: uuid.NewString(), ProspectID: sc.prospectID}
	applyPayload(&row, p)
	if err := h.db.Create(&row).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "create failed"})
	}
	return c.JSON(fiber.Map{"id": row.ID})
}

// save — écrase l'état de l'atelier. POURQUOI un écrasement complet et non un patch : l'état est un
// tout cohérent (le draft, sa traduction et sa checklist décrivent la MÊME version). Sauvegarder un
// sous-ensemble laisserait afficher une traduction qui ne correspond plus au message.
func (h *ComposeDraftsHandler) save(c *fiber.Ctx) error {
	sc, ok := h.scope(c)
	if !ok {
		return nil
	}

	var p composeDraftPayload
	if err := c.BodyParser(&p); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var row models.ComposeDraft
	if err := sc.apply(h.db).First(&row, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "draft not found"})
	}

	applyPayload(&row, p)
	if err := h.db.Save(&row).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "save failed"})
	}
	return c.JSON(fiber.Map{"id": row.ID})
}
