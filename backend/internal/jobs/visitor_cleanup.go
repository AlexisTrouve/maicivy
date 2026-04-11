package jobs

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"maicivy/internal/services"
)

// VisitorCleanupJob nettoie périodiquement les visiteurs inactifs
type VisitorCleanupJob struct {
	service       *services.AnalyticsService
	interval      time.Duration
	stopChan      chan struct{}
	cleanupPeriod time.Duration
}

// NewVisitorCleanupJob crée un nouveau job de nettoyage des visiteurs
func NewVisitorCleanupJob(service *services.AnalyticsService, cleanupInterval time.Duration) *VisitorCleanupJob {
	if cleanupInterval == 0 {
		cleanupInterval = 1 * time.Minute // Par défaut: toutes les minutes
	}

	return &VisitorCleanupJob{
		service:       service,
		interval:      cleanupInterval,
		stopChan:      make(chan struct{}),
		cleanupPeriod: 5 * time.Minute, // Nettoie les visiteurs inactifs > 5 minutes
	}
}

// Start démarre le job en arrière-plan
func (j *VisitorCleanupJob) Start(ctx context.Context) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	log.Info().
		Dur("interval", j.interval).
		Dur("cleanup_period", j.cleanupPeriod).
		Msg("Visitor cleanup job started")

	// Exécuter immédiatement au démarrage
	j.cleanup(ctx)

	for {
		select {
		case <-ticker.C:
			j.cleanup(ctx)

		case <-j.stopChan:
			log.Info().Msg("Visitor cleanup job stopped")
			return

		case <-ctx.Done():
			log.Info().Msg("Visitor cleanup job stopped (context cancelled)")
			return
		}
	}
}

// cleanup effectue le nettoyage des visiteurs inactifs
func (j *VisitorCleanupJob) cleanup(ctx context.Context) {
	cleanupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	removed, err := j.service.CleanupInactiveVisitors(cleanupCtx)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to cleanup inactive visitors")
		return
	}

	if removed > 0 {
		log.Info().
			Int64("removed", removed).
			Msg("Cleaned up inactive visitors")
	}
}

// Stop arrête le job
func (j *VisitorCleanupJob) Stop() {
	close(j.stopChan)
}
