package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"maicivy/internal/models"
	"maicivy/internal/services"
)

// LetterWorker worker pour traiter les jobs de génération de lettres
type LetterWorker struct {
	db              *gorm.DB
	queueService    *services.LetterQueueService
	aiService       *services.AIService
	scraper         *services.CompanyScraper
	letterGenerator *services.LetterGenerator
	profileBuilder  *services.ProfileBuilder

	stopChan chan bool
	running  bool
}

// NewLetterWorker crée une nouvelle instance du worker
func NewLetterWorker(
	db *gorm.DB,
	queueService *services.LetterQueueService,
	aiService *services.AIService,
	scraper *services.CompanyScraper,
	letterGenerator *services.LetterGenerator,
	profileBuilder *services.ProfileBuilder,
) *LetterWorker {
	return &LetterWorker{
		db:              db,
		queueService:    queueService,
		aiService:       aiService,
		scraper:         scraper,
		letterGenerator: letterGenerator,
		profileBuilder:  profileBuilder,
		stopChan:        make(chan bool),
		running:         false,
	}
}

// Start démarre le worker
func (w *LetterWorker) Start() {
	if w.running {
		log.Println("[LetterWorker] Already running")
		return
	}

	w.running = true
	log.Println("[LetterWorker] Starting...")

	for {
		select {
		case <-w.stopChan:
			log.Println("[LetterWorker] Stopped")
			w.running = false
			return
		default:
			w.processNextJob()
		}
	}
}

// Stop arrête le worker
func (w *LetterWorker) Stop() {
	if !w.running {
		return
	}

	log.Println("[LetterWorker] Stopping...")
	w.stopChan <- true
}

// IsRunning retourne true si le worker est en cours d'exécution
func (w *LetterWorker) IsRunning() bool {
	return w.running
}

// processNextJob traite le prochain job dans la queue
func (w *LetterWorker) processNextJob() {
	jobID, err := w.queueService.PopJob()
	if err != nil {
		log.Printf("[LetterWorker] Error popping job: %v", err)
		time.Sleep(2 * time.Second) // Attendre avant retry
		return
	}

	if jobID == "" {
		// Queue vide, attendre un peu
		time.Sleep(1 * time.Second)
		return
	}

	log.Printf("[LetterWorker] Processing job: %s", jobID)

	// Récupérer les détails du job
	job, err := w.queueService.GetJobStatus(jobID)
	if err != nil {
		log.Printf("[LetterWorker] Error getting job status: %v", err)
		return
	}

	// Marquer comme en cours
	err = w.queueService.UpdateJobStatus(jobID, services.JobStatusProcessing, 10)
	if err != nil {
		log.Printf("[LetterWorker] Error updating job status: %v", err)
	}

	// Exécuter la génération
	motivationID, antiMotivationID, err := w.generateLetters(job)
	if err != nil {
		log.Printf("[LetterWorker] Error generating letters: %v", err)

		// Retry logic
		if job.RetryCount < job.MaxRetries {
			log.Printf("[LetterWorker] Retrying job %s (attempt %d/%d)", jobID, job.RetryCount+1, job.MaxRetries)
			w.queueService.RetryJob(jobID)
		} else {
			log.Printf("[LetterWorker] Max retries reached for job %s", jobID)
			w.queueService.FailJob(jobID, err.Error())
		}
		return
	}

	// Marquer comme complété
	err = w.queueService.CompleteJob(jobID, motivationID, antiMotivationID)
	if err != nil {
		log.Printf("[LetterWorker] Error completing job: %v", err)
		return
	}

	log.Printf("[LetterWorker] Job %s completed. Letters: %s, %s", jobID, motivationID, antiMotivationID)
}

// generateLetters génère les deux lettres (motivation + anti-motivation)
func (w *LetterWorker) generateLetters(job *services.LetterJob) (uuid.UUID, uuid.UUID, error) {
	startTime := time.Now()
	ctx := context.Background()

	// 1. Générer les deux lettres en parallèle (20-80% progress)
	w.queueService.UpdateJobStatus(job.JobID, services.JobStatusProcessing, 20)

	motivationLetter, antiMotivationLetter, err := w.letterGenerator.GenerateDualLetters(ctx, job.CompanyName)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("failed to generate letters: %w", err)
	}

	// 2. Sauvegarder en DB (80% progress)
	w.queueService.UpdateJobStatus(job.JobID, services.JobStatusProcessing, 80)

	// Récupérer ou créer le visitor
	var visitor models.Visitor
	result := w.db.Where("session_id = ?", job.VisitorID).First(&visitor)
	if result.Error != nil {
		// Créer le visiteur s'il n'existe pas (pour les appels API directs)
		now := time.Now()
		visitor = models.Visitor{
			SessionID:  job.VisitorID,
			VisitCount: 1,
			FirstVisit: now,
			LastVisit:  now,
		}
		if err := w.db.Create(&visitor).Error; err != nil {
			return uuid.Nil, uuid.Nil, fmt.Errorf("failed to create visitor: %w", err)
		}
	}

	// Marshaller CompanyInfo en JSON
	companyInfoJSON, err := json.Marshal(motivationLetter.CompanyInfo)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("failed to marshal company info: %w", err)
	}

	// Sauvegarder lettre de motivation
	motivationDB := models.GeneratedLetter{
		VisitorID:    visitor.ID,
		CompanyName:  job.CompanyName,
		LetterType:   models.LetterTypeMotivation,
		Content:      motivationLetter.Content,
		AIModel:      motivationLetter.Provider,
		TokensUsed:   motivationLetter.TokensUsed,
		GenerationMS: int(time.Since(startTime).Milliseconds()),
		CompanyInfo:  string(companyInfoJSON),
	}

	result = w.db.Create(&motivationDB)
	if result.Error != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("failed to save motivation letter: %w", result.Error)
	}

	// 3. Sauvegarder lettre anti-motivation (90% progress)
	w.queueService.UpdateJobStatus(job.JobID, services.JobStatusProcessing, 90)

	antiMotivationDB := models.GeneratedLetter{
		VisitorID:    visitor.ID,
		CompanyName:  job.CompanyName,
		LetterType:   models.LetterTypeAntiMotivation,
		Content:      antiMotivationLetter.Content,
		AIModel:      antiMotivationLetter.Provider,
		TokensUsed:   antiMotivationLetter.TokensUsed,
		GenerationMS: int(time.Since(startTime).Milliseconds()),
		CompanyInfo:  string(companyInfoJSON),
	}

	result = w.db.Create(&antiMotivationDB)
	if result.Error != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("failed to save anti-motivation letter: %w", result.Error)
	}

	// 4. Terminé (100% progress)
	w.queueService.UpdateJobStatus(job.JobID, services.JobStatusProcessing, 100)

	log.Printf("[LetterWorker] Letters generated in %dms (total tokens: %d, total cost: $%.4f)",
		time.Since(startTime).Milliseconds(),
		motivationLetter.TokensUsed+antiMotivationLetter.TokensUsed,
		motivationLetter.EstimatedCost+antiMotivationLetter.EstimatedCost,
	)

	return motivationDB.ID, antiMotivationDB.ID, nil
}
