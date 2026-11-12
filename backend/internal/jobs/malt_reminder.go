package jobs

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"maicivy/internal/services"
)

// MaltReminderJob vérifie toutes les heures s'il faut envoyer le rappel de disponibilité Malt (cf.
// services.MaltReminderService.CheckAndSend) — granularité horaire suffisante pour un rappel 2x/semaine,
// pas besoin d'un vrai scheduler cron.
type MaltReminderJob struct {
	service  *services.MaltReminderService
	stopChan chan struct{}
}

func NewMaltReminderJob(service *services.MaltReminderService) *MaltReminderJob {
	return &MaltReminderJob{service: service, stopChan: make(chan struct{})}
}

// Start démarre le job en arrière-plan (bloquant — à lancer via `go job.Start(ctx)`).
func (j *MaltReminderJob) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	log.Info().Msg("Malt reminder job started")

	j.service.CheckAndSend(ctx, time.Now())

	for {
		select {
		case <-ticker.C:
			j.service.CheckAndSend(ctx, time.Now())

		case <-j.stopChan:
			log.Info().Msg("Malt reminder job stopped")
			return

		case <-ctx.Done():
			log.Info().Msg("Malt reminder job stopped (context cancelled)")
			return
		}
	}
}

// Stop arrête le job.
func (j *MaltReminderJob) Stop() {
	close(j.stopChan)
}
