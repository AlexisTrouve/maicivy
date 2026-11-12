package jobs

import (
	"context"
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"maicivy/internal/models"
	"maicivy/internal/services"
)

// GitHubAutoSyncJob gère la synchronisation automatique quotidienne
type GitHubAutoSyncJob struct {
	db          *gorm.DB
	syncService *services.GitHubSyncService
	scheduler   *cron.Cron
}

// NewGitHubAutoSyncJob crée une nouvelle instance
func NewGitHubAutoSyncJob(db *gorm.DB, syncService *services.GitHubSyncService) *GitHubAutoSyncJob {
	return &GitHubAutoSyncJob{
		db:          db,
		syncService: syncService,
		scheduler:   cron.New(),
	}
}

// Start démarre le cron job
func (j *GitHubAutoSyncJob) Start() error {
	// Sync quotidien à 2h du matin
	_, err := j.scheduler.AddFunc("0 2 * * *", j.syncAllProfiles)
	if err != nil {
		return err
	}

	log.Println("[GitHubAutoSync] Cron job scheduled: daily at 2:00 AM")
	j.scheduler.Start()
	return nil
}

// Stop arrête le cron job
func (j *GitHubAutoSyncJob) Stop() {
	if j.scheduler != nil {
		j.scheduler.Stop()
		log.Println("[GitHubAutoSync] Cron job stopped")
	}
}

// syncAllProfiles synchronise tous les profils GitHub connectés
func (j *GitHubAutoSyncJob) syncAllProfiles() {
	log.Println("[GitHubAutoSync] Starting daily sync for all GitHub profiles")

	ctx := context.Background()

	// Récupérer tous les profils connectés
	var profiles []models.GitHubProfile
	if err := j.db.Find(&profiles).Error; err != nil {
		log.Printf("[GitHubAutoSync] Error fetching profiles: %v", err)
		return
	}

	if len(profiles) == 0 {
		log.Println("[GitHubAutoSync] No profiles to sync")
		return
	}

	log.Printf("[GitHubAutoSync] Found %d profiles to sync", len(profiles))

	successCount := 0
	errorCount := 0

	// Synchroniser chaque profil
	for _, profile := range profiles {
		log.Printf("[GitHubAutoSync] Syncing profile: %s", profile.Username)

		// Vérifier si token valide
		if profile.Token.AccessToken == "" {
			log.Printf("[GitHubAutoSync] Profile %s has no valid token, skipping", profile.Username)
			errorCount++
			continue
		}

		// Appeler service de sync
		err := j.syncService.SyncRepositories(ctx, profile.Username, profile.Token.AccessToken)
		if err != nil {
			log.Printf("[GitHubAutoSync] Error syncing %s: %v", profile.Username, err)
			errorCount++

			// Graceful degradation: continuer avec les autres profils
			continue
		}

		log.Printf("[GitHubAutoSync] Successfully synced %s", profile.Username)
		successCount++

		// Petit délai pour éviter rate limiting GitHub
		time.Sleep(2 * time.Second)
	}

	log.Printf("[GitHubAutoSync] Daily sync completed: %d success, %d errors", successCount, errorCount)
}

// SyncNow déclenche une synchronisation immédiate (utile pour tests)
func (j *GitHubAutoSyncJob) SyncNow() {
	go j.syncAllProfiles()
}

// GetSchedulerInfo retourne les infos sur les jobs programmés
func (j *GitHubAutoSyncJob) GetSchedulerInfo() []cron.Entry {
	return j.scheduler.Entries()
}
