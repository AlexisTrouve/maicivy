package api

import (
	"bufio"
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"maicivy/internal/dossiervivant"
	"maicivy/internal/middleware"
	"maicivy/internal/services"
)

// DossierVivantHandler — un run RÉEL de l'équipe d'agents du dossier vivant, diffusé en SSE
// (contrat : frontend/lib/dossier-vivant/events.ts).
//
// DEUX ROUTES, un seul handler :
//   - POST /api/v1/dossier-vivant/run (PUBLIQUE) — tout visiteur, sans connexion, lance la vraie
//     équipe sur SON besoin. Décision d'Alexis (« pas de limite, juste spawn le système complet et
//     on voit ») : aucun quota par visiteur, aucun plafond journalier, aucun rate limit IA, et pas
//     de verrou « un run à la fois » — un second visiteur ne doit pas recevoir de refus parce
//     qu'un premier regarde son run. Le service est sans état entre deux runs (chaque run a son
//     compteur, son tempo et son contexte), des runs concurrents ne se voient pas.
//   - POST /api/v1/admin/dossier-vivant/run (OWNER) — la même chose derrière RequireOwner (cookie
//     admin ou X-Owner-Key → sinon 401), sous /admin donc derrière AdminRoleGuard. Gardée pour les
//     appels serveur à serveur (X-Owner-Key).
//
// Ce qui BORNE un run, sans limiter l'usage : la validation de la saisie avant tout appel
// (dossiervivant.ParseVisitor — besoin de 20 à 2 000 caractères, présentation ≤ 120) et le plafond
// d'entrée de CHAQUE appel (dossiervivant/callsize.go). Ni la consommation ni la durée d'un run ne
// l'arrêtent (décision d'Alexis : le temps est rappelé aux agents, cf. dossiervivant/tempo.go). Seule
// reste la limite de la CONNEXION (connectionLimit), qui est celle du transport, pas du produit.
//
// COMMENT : le run tourne dans sa goroutine et émet dans un canal ; le writer SSE vide le canal
// vers le client. Écriture en échec (client parti) → le contexte du run est annulé : les agents
// s'arrêtent, plus aucun token n'est dépensé pour personne. Entre le run et le canal, un
// RecordingEmitter garde la copie serveur du dossier remis (`dossiers`) et en donne l'id dans
// run.completed : c'est cette copie, et elle seule, que la route publique du PDF imprime
// (dossier_vivant_pdf.go).
type DossierVivantHandler struct {
	service       *dossiervivant.Service
	dossiers      dossiervivant.DossierSaver
	sessionSecret string
	ownerKey      string
	// needs : le besoin saisi est aussi rangé pour préparer le CV adapté du visiteur (cf.
	// cv_context.go). nil = rien.
	needs NeedRecorder
}

// SetNeedRecorder branche la préparation du CV adapté au besoin du visiteur.
func (h *DossierVivantHandler) SetNeedRecorder(r NeedRecorder) { h.needs = r }

// connectionLimit — le WriteTimeout de Fiber (cmd/main.go) : au-delà, la connexion est coupée.
//
// CE QUI SE PASSE vraiment à 120 s (vérifié dans fasthttp v1.51 › serveConn) : l'échéance
// d'écriture est posée UNE fois, au retour du handler, et couvre tout le corps en flux — une trame
// écrite après 120 s échoue, le writer SSE ci-dessous s'arrête et annule le run. Le front-door
// (frontdoor.go, ReadTimeout du client de relais, 120 s) borne la même réponse de son côté. Le
// visiteur voit alors « connexion perdue » (`network`), pas un arrêt nommé du run. Ce n'est pas un
// arrêt voulu : c'est le plafond du transport, ×4 au-dessus de la cible de 20–30 s (tempo.go).
const connectionLimit = 120 * time.Second

func NewDossierVivantHandler(service *dossiervivant.Service, dossiers dossiervivant.DossierSaver, sessionSecret, ownerKey string) *DossierVivantHandler {
	return &DossierVivantHandler{service: service, dossiers: dossiers, sessionSecret: sessionSecret, ownerKey: ownerKey}
}

// RegisterPublicRoutes — la route du visiteur (hors /admin, cf. cmd/main.go).
func (h *DossierVivantHandler) RegisterPublicRoutes(api fiber.Router) {
	api.Post("/dossier-vivant/run", h.run)
}

// RegisterOwnerRoutes — la route owner (à enregistrer avec la surface /admin).
func (h *DossierVivantHandler) RegisterOwnerRoutes(api fiber.Router) {
	api.Post("/admin/dossier-vivant/run", middleware.RequireOwner(h.sessionSecret, h.ownerKey), h.run)
}

// dossierRunRequest — `who` : « Nom, rôle chez organisation » ; `lang` : langue de l'interface,
// dans laquelle les agents écrivent.
type dossierRunRequest struct {
	Need string `json:"need"`
	Who  string `json:"who"`
	Lang string `json:"lang"`
}

func (h *DossierVivantHandler) run(c *fiber.Ctx) error {
	var req dossierRunRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body", "code": "INVALID_REQUEST"})
	}
	// Entrée externe : validée AVANT le flux et avant tout appel de modèle (un besoin de 50 ko ne
	// part jamais à la passerelle).
	brief, err := dossiervivant.ParseVisitor(req.Who, req.Need)
	if err != nil {
		code := "NEED_LENGTH"
		if errors.Is(err, dossiervivant.ErrVisitorShape) {
			code = "WHO_FORMAT"
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error(), "code": code})
	}

	// Le besoin VALIDÉ (jamais le nom ni l'organisation du visiteur) prépare aussi son CV adapté.
	// Le besoin saisi est déjà l'énoncé du visiteur, borné par ParseVisitor : il n'y a rien à
	// résumer, le service le recoupe. Hors du chemin du run : un échec ne touche pas le dossier.
	if v, ok := visitorOf(c); ok && h.needs != nil {
		recordNeed(h.needs, v, services.NeedSourceDossier, brief.Need, req.Lang)
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no") // nginx ne doit pas retenir les trames

	// Contexte de la CONNEXION, détaché de la requête Fiber (recyclée après le retour du handler).
	// Borné au WriteTimeout du serveur : si le writer SSE n'était jamais appelé (client parti avant
	// la première trame), rien d'autre ne fermerait ce contexte et le run resterait bloqué sur un
	// canal plein, ses agents en vol.
	conn, hangUp := context.WithTimeout(context.Background(), connectionLimit)
	emitter := dossiervivant.NewStreamEmitter(conn, 64)
	recorded := dossiervivant.NewRecordingEmitter(conn, emitter, h.dossiers)
	lang := req.Lang
	go func() {
		defer emitter.Close()
		_ = h.service.Run(conn, brief, lang, recorded) // l'issue est déjà dans le flux (run.failed) et le journal
	}()

	c.Response().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer hangUp()
		for frame := range emitter.Frames() {
			if _, err := w.Write(frame); err != nil {
				log.Info().Msg("dossier vivant: client parti, run annulé")
				return
			}
			if err := w.Flush(); err != nil {
				log.Info().Msg("dossier vivant: client parti, run annulé")
				return
			}
		}
	})
	return nil
}
