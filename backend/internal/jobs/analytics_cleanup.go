package jobs

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"maicivy/internal/services"
)

// AnalyticsCleanupJob gère le nettoyage périodique des anciennes données analytics
type AnalyticsCleanupJob struct {
	service       *services.AnalyticsService
	retentionDays int
}

// NewAnalyticsCleanupJob crée un nouveau job de cleanup
func NewAnalyticsCleanupJob(service *services.AnalyticsService, retentionDays int) *AnalyticsCleanupJob {
	if retentionDays <= 0 {
		retentionDays = 90 // Par défaut: 90 jours
	}

	return &AnalyticsCleanupJob{
		service:       service,
		retentionDays: retentionDays,
	}
}

// Run exécute le job de nettoyage
func (j *AnalyticsCleanupJob) Run(ctx context.Context) error {
	log.Info().
		Int("retention_days", j.retentionDays).
		Msg("Starting analytics cleanup job")

	start := time.Now()

	deleted, err := j.service.CleanupOldEvents(ctx)
	if err != nil {
		log.Error().
			Err(err).
			Dur("duration", time.Since(start)).
			Msg("Analytics cleanup failed")
		return err
	}

	log.Info().
		Int64("deleted_events", deleted).
		Dur("duration", time.Since(start)).
		Msg("Analytics cleanup completed successfully")

	return nil
}

// Start démarre le job en mode cron (exécution quotidienne)
func (j *AnalyticsCleanupJob) Start(ctx context.Context) {
	log.Info().
		Int("retention_days", j.retentionDays).
		Msg("Starting analytics cleanup scheduler")

	// Calculer délai jusqu'à 2h du matin
	now := time.Now()
	next2AM := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())

	// Si on est déjà passé 2h aujourd'hui, planifier pour demain
	if now.After(next2AM) {
		next2AM = next2AM.Add(24 * time.Hour)
	}

	timeUntil2AM := time.Until(next2AM)

	log.Info().
		Time("next_run", next2AM).
		Dur("time_until", timeUntil2AM).
		Msg("Scheduled first cleanup run")

	// Première exécution à 2h
	timer := time.NewTimer(timeUntil2AM)

	// Ticker pour exécutions suivantes (toutes les 24h)
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			// Première exécution
			if err := j.Run(ctx); err != nil {
				log.Error().Err(err).Msg("Cleanup job failed")
			}

			// Planifier prochaine exécution
			next2AM = next2AM.Add(24 * time.Hour)
			log.Info().
				Time("next_run", next2AM).
				Msg("Scheduled next cleanup run")

		case <-ticker.C:
			// Exécutions quotidiennes suivantes
			if err := j.Run(ctx); err != nil {
				log.Error().Err(err).Msg("Cleanup job failed")
			}

			// Planifier prochaine
			next2AM = time.Date(
				time.Now().Year(),
				time.Now().Month(),
				time.Now().Day()+1,
				2, 0, 0, 0,
				time.Now().Location(),
			)
			log.Info().
				Time("next_run", next2AM).
				Msg("Scheduled next cleanup run")

		case <-ctx.Done():
			log.Info().Msg("Analytics cleanup job stopped (context cancelled)")
			return
		}
	}
}

// RunOnce exécute le job une seule fois (utile pour tests ou exécution manuelle)
func (j *AnalyticsCleanupJob) RunOnce(ctx context.Context) error {
	return j.Run(ctx)
}
