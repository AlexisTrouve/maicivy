package jobs

import (
	"context"
	"log"

	"github.com/robfig/cron/v3"

	"maicivy/internal/services"
)

// ActivitySyncJob gère la synchronisation automatique du feed d'activité
type ActivitySyncJob struct {
	feedService *services.ActivityFeedService
	scheduler   *cron.Cron
}

// NewActivitySyncJob crée une nouvelle instance
func NewActivitySyncJob(feedService *services.ActivityFeedService) *ActivitySyncJob {
	return &ActivitySyncJob{
		feedService: feedService,
		scheduler:   cron.New(),
	}
}

// Start démarre le cron job
func (j *ActivitySyncJob) Start() error {
	// Sync toutes les 15 minutes
	_, err := j.scheduler.AddFunc("*/15 * * * *", j.syncFeed)
	if err != nil {
		return err
	}

	log.Println("[ActivitySync] Cron job scheduled: every 15 minutes")
	j.scheduler.Start()

	// Sync initial au démarrage
	go j.syncFeed()

	return nil
}

// Stop arrête le cron job
func (j *ActivitySyncJob) Stop() {
	if j.scheduler != nil {
		j.scheduler.Stop()
		log.Println("[ActivitySync] Cron job stopped")
	}
}

// syncFeed récupère et stocke le feed d'activité
func (j *ActivitySyncJob) syncFeed() {
	log.Println("[ActivitySync] Starting activity feed sync")

	ctx := context.Background()

	if err := j.feedService.FetchAndStore(ctx); err != nil {
		log.Printf("[ActivitySync] Error syncing feed: %v", err)
		return
	}

	log.Println("[ActivitySync] Activity feed sync completed")
}

// SyncNow déclenche une synchronisation immédiate
func (j *ActivitySyncJob) SyncNow() {
	go j.syncFeed()
}

// GetSchedulerInfo retourne les infos sur les jobs programmés
func (j *ActivitySyncJob) GetSchedulerInfo() []cron.Entry {
	return j.scheduler.Entries()
}
