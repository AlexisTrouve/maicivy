package jobs

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"maicivy/internal/models"
	"maicivy/internal/services"
)

// MailWatchJob veille sur le courrier sortant, une fois par heure.
//
// Il fait deux choses que les émetteurs eux-mêmes ne peuvent pas faire :
//
//  1. Repérer un BLOCAGE. Un émetteur ne sait signaler que ses propres échecs, au moment où il
//     essaie. Or la panne du 2026-08-30 a duré 13 jours précisément parce que personne ne regardait
//     l'état global : des mails s'accumulaient, non transférés, et rien ne le disait. Ici on
//     interroge l'état — « du courrier attend-il depuis trop longtemps ? » — ce qui reste vrai tant
//     que le problème dure.
//  2. Publier le récapitulatif quotidien, sans ping.
//
// Granularité horaire : suffisante pour les deux usages, et elle évite un vrai scheduler cron —
// même choix que MaltReminderJob.
type MailWatchJob struct {
	db            *gorm.DB
	watch         *services.MailWatch
	seuilBlocage  time.Duration // au-delà, un mail non transféré est considéré bloqué
	seuilReleve   time.Duration // au-delà, une relève IMAP qui ne réussit plus est signalée
	heureDigetUTC int           // heure UTC à laquelle publier le récapitulatif du jour
	stopChan      chan struct{}
}

// seuilReleveMuette : la relève tourne toutes les 2 min ; 30 min sans une seule réussite, ce n'est
// plus un aléa réseau, c'est une relève arrêtée (vécu : 2 jours, du 17 au 19/09/2026).
const seuilReleveMuette = 30 * time.Minute

// NewMailWatchJob rend nil si la surveillance n'est pas configurée (pas de Redis, ou pas de webhook
// et donc rien à publier) — même convention que les autres jobs optionnels.
func NewMailWatchJob(db *gorm.DB, watch *services.MailWatch, seuilBlocage time.Duration, heureDigestUTC int) *MailWatchJob {
	if db == nil || watch == nil {
		return nil
	}
	return &MailWatchJob{
		db:            db,
		watch:         watch,
		seuilBlocage:  seuilBlocage,
		seuilReleve:   seuilReleveMuette,
		heureDigetUTC: heureDigestUTC,
		stopChan:      make(chan struct{}),
	}
}

// VerifierBlocage compte les mails jamais transférés, RELEVÉS depuis plus longtemps que le seuil,
// puis alerte.
//
// Depuis la RELÈVE (created_at), pas depuis la date du mail (received_at, lue dans l'en-tête) : un
// transfert ne peut pas être en retard sur un mail que la relève n'avait pas encore lu. Vécu le
// 19/09/2026 : trois offres datées du 17 et du 18, relevées ce matin-là, et « le relai SMTP ne
// répond pas » est parti avant même leur premier essai, réussi 13 s plus tard. Une relève en retard
// est l'affaire de VerifierReleve.
//
// EXCLUT les mails bloqués par le filtre de pertinence (forward_blocked) : ce sont des décisions,
// pas des pannes — les compter ici produirait une alerte permanente et donc ignorée, exactement ce
// qu'on cherche à éviter.
func (j *MailWatchJob) VerifierBlocage(ctx context.Context, maintenant time.Time) {
	seuil := maintenant.Add(-j.seuilBlocage)

	var enAttente []models.MailboxEmail
	if err := j.db.Where("forwarded_at IS NULL AND forward_blocked = ? AND created_at < ?", false, seuil).
		Order("created_at ASC").Find(&enAttente).Error; err != nil {
		log.Error().Err(err).Msg("mailwatch: lecture des transferts en attente impossible")
		return
	}
	if len(enAttente) == 0 {
		return
	}

	log.Warn().Int("en_attente", len(enAttente)).Time("plus_vieux", enAttente[0].CreatedAt).
		Msg("mailwatch: des transferts sont bloqués")
	j.watch.AlerterBlocage(ctx, len(enAttente), enAttente[0].CreatedAt)
}

// VerifierReleve alerte si la dernière relève IMAP réussie est plus vieille que le seuil.
//
// POURQUOI en plus de VerifierBlocage : un mail que la relève n'a jamais lu n'existe pas en base —
// VerifierBlocage ne peut pas le voir. Du 17 au 19/09/2026, trois offres Malt ont ainsi attendu deux
// jours sans qu'aucune alerte ne parte. Aucune relève jamais marquée (boîte non configurée) = rien à dire.
func (j *MailWatchJob) VerifierReleve(ctx context.Context, maintenant time.Time) {
	derniere, ok, err := j.watch.DerniereReleve(ctx)
	if err != nil {
		log.Error().Err(err).Msg("mailwatch: âge de la relève illisible")
		return
	}
	if !ok || maintenant.Sub(derniere) <= j.seuilReleve {
		return
	}
	log.Warn().Time("derniere_releve", derniere).Msg("mailwatch: la relève IMAP ne réussit plus")
	j.watch.AlerterReleveMuette(ctx, derniere)
}

// PublierDigestSiHeure publie le récapitulatif quand l'heure configurée est atteinte. La
// déduplication par date vit dans MailWatch.PosterDigest — le job peut donc appeler à chaque heure
// sans risque de doublon.
func (j *MailWatchJob) PublierDigestSiHeure(ctx context.Context, maintenant time.Time) {
	if maintenant.UTC().Hour() < j.heureDigetUTC {
		return
	}
	poste, err := j.watch.PosterDigest(ctx, maintenant)
	if err != nil {
		log.Warn().Err(err).Msg("mailwatch: récapitulatif quotidien non publié")
		return
	}
	if poste {
		log.Info().Msg("mailwatch: récapitulatif quotidien publié")
	}
}

// Start démarre le job en arrière-plan (bloquant — à lancer via `go job.Start(ctx)`).
func (j *MailWatchJob) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	log.Info().Dur("seuil_blocage", j.seuilBlocage).Int("heure_digest_utc", j.heureDigetUTC).
		Msg("Mail watch job started")

	j.cycle(ctx)

	for {
		select {
		case <-ticker.C:
			j.cycle(ctx)

		case <-j.stopChan:
			log.Info().Msg("Mail watch job stopped")
			return

		case <-ctx.Done():
			log.Info().Msg("Mail watch job stopped (context cancelled)")
			return
		}
	}
}

func (j *MailWatchJob) cycle(ctx context.Context) {
	maintenant := time.Now()
	j.VerifierBlocage(ctx, maintenant)
	j.VerifierReleve(ctx, maintenant)
	j.PublierDigestSiHeure(ctx, maintenant)
}

// Stop arrête le job.
func (j *MailWatchJob) Stop() {
	close(j.stopChan)
}
