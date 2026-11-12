package jobs

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"maicivy/internal/services"
)

// MailboxPollJob interroge périodiquement la boîte IMAP suivie (cf. services.MailboxService.PollOnce) :
// ingestion des nouveaux mails allowlistés (Malt, puis autres plateformes) + retry des transferts en échec.
type MailboxPollJob struct {
	service  *services.MailboxService
	interval time.Duration
	stopChan chan struct{}
}

// NewMailboxPollJob crée le job. interval<=0 → défaut 2 minutes.
func NewMailboxPollJob(service *services.MailboxService, interval time.Duration) *MailboxPollJob {
	if interval <= 0 {
		interval = 2 * time.Minute
	}
	return &MailboxPollJob{
		service:  service,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start démarre le job en arrière-plan (bloquant — à lancer via `go job.Start(ctx)`).
func (j *MailboxPollJob) Start(ctx context.Context) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	log.Info().Dur("interval", j.interval).Msg("Mailbox poll job started")

	j.poll(ctx)

	for {
		select {
		case <-ticker.C:
			j.poll(ctx)

		case <-j.stopChan:
			log.Info().Msg("Mailbox poll job stopped")
			return

		case <-ctx.Done():
			log.Info().Msg("Mailbox poll job stopped (context cancelled)")
			return
		}
	}
}

// poll exécute un cycle, borné à 60s (IMAP fetch + relais SMTP peuvent bloquer sur un réseau lent).
func (j *MailboxPollJob) poll(ctx context.Context) {
	pollCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	if err := j.service.PollOnce(pollCtx); err != nil {
		log.Error().Err(err).Msg("mailbox: échec du poll")
	}
}

// Stop arrête le job.
func (j *MailboxPollJob) Stop() {
	close(j.stopChan)
}
