package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"maicivy/internal/dossiervivant"
	"maicivy/internal/middleware"
	"maicivy/internal/services"
)

// AdminSurfaceDeps : dépendances de toutes les routes /admin.
type AdminSurfaceDeps struct {
	DB                *gorm.DB
	Redis             *redis.Client
	SessionSecret     string
	AdminPassword     string
	ComposeService    *services.ComposeService // nil = pas de /admin/compose/verify (IA absente)
	MailboxService    *services.MailboxService
	MailboxTranslator services.MailboxTranslator
	// ProspectVerifyDailyMax : plafond quotidien des vérifications Composer d'un prospecteur
	// (compteur distinct du budget IA principal, cf. middleware.ProspectVerifyQuota). < 0 = levé, 0 = aucune.
	ProspectVerifyDailyMax int
	// DossierVivant : la route OWNER des runs réels du dossier vivant (la route publique, elle, est
	// enregistrée hors de /admin — cmd/main.go). nil = pas de route (IA absente).
	DossierVivant *dossiervivant.Service
	// DossierPDF : la route OWNER du PDF (corps = dossier). Les routes publiques (rejeu, run réel d'un
	// visiteur, sur la copie serveur) sont enregistrées hors de /admin (cmd/main.go). nil = pas de route.
	DossierPDF *services.DossierPDFService
	// OwnerAPIKey : la clé X-Owner-Key, second moyen d'être owner (appels serveur à serveur).
	OwnerAPIKey string
}

// RegisterAdminSurface enregistre TOUTES les routes /admin derrière le garde de rôle.
//
// POURQUOI un point unique (et non sept RegisterRoutes éparpillés dans main) : le test de
// défaut-refus (admin_routes_test.go) appelle cette MÊME fonction et énumère les routes réellement
// enregistrées. Une route owner ajoutée ici demain est donc testée d'office : si elle n'est pas dans
// ProspectAllowedAdminRoutes, un prospecteur doit y recevoir 403, et le test le vérifie.
//
// COMMENT : le garde est monté sur le préfixe /admin AVANT les routes — Fiber exécute la pile dans
// l'ordre d'enregistrement, il passe donc devant chaque handler.
func RegisterAdminSurface(apiV1 fiber.Router, d AdminSurfaceDeps) *ProspectsHandler {
	prospectAuth := NewProspectAuth(d.DB, d.SessionSecret)
	apiV1.Use("/admin", middleware.AdminRoleGuard(d.SessionSecret, prospectAuth, ProspectAllowedAdminRoutes))

	NewAdminHandler(d.AdminPassword, d.SessionSecret).RegisterRoutes(apiV1)
	NewAdminStatsHandler(d.DB, d.Redis, d.SessionSecret).RegisterRoutes(apiV1)
	NewAdminChatHandler(d.DB, d.SessionSecret).RegisterRoutes(apiV1)
	if d.ComposeService != nil {
		compose := NewComposeHandler(d.ComposeService, d.SessionSecret)
		compose.SetProspectVerifyQuota(middleware.NewProspectVerifyQuota(d.Redis, d.ProspectVerifyDailyMax))
		compose.RegisterRoutes(apiV1)
	}
	NewComposeDraftsHandler(d.DB, d.SessionSecret).RegisterRoutes(apiV1)
	NewMailboxHandler(d.DB, d.SessionSecret, d.MailboxService, d.MailboxTranslator).RegisterRoutes(apiV1)
	// Les runs owner gardent leur dossier comme les runs publics (même Redis, même durée de vie).
	dossiers := services.NewDossierStore(d.Redis)
	if d.DossierVivant != nil {
		NewDossierVivantHandler(d.DossierVivant, dossiers, d.SessionSecret, d.OwnerAPIKey).RegisterOwnerRoutes(apiV1)
	}
	if d.DossierPDF != nil {
		NewDossierPDFHandler(d.DossierPDF, dossiers, d.SessionSecret, d.OwnerAPIKey).RegisterOwnerRoutes(apiV1)
	}

	prospects := NewProspectsHandler(d.DB, d.SessionSecret)
	prospects.RegisterRoutes(apiV1)
	return prospects
}
