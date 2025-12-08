package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"maicivy/internal/models"
	"maicivy/internal/services"
)

// LetterWorker worker pour traiter les jobs de génération de lettres
type LetterWorker struct {
	db           *gorm.DB
	queueService *services.LetterQueueService
	// aiService, scraperService, pdfService seront implémentés dans Doc 08
	// Pour l'instant on les mock dans les commentaires

	stopChan chan bool
	running  bool
}

// NewLetterWorker crée une nouvelle instance du worker
func NewLetterWorker(
	db *gorm.DB,
	queueService *services.LetterQueueService,
) *LetterWorker {
	return &LetterWorker{
		db:           db,
		queueService: queueService,
		stopChan:     make(chan bool),
		running:      false,
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

	log.Printf("[LetterWorker] Job %s completed. Letters: %d, %d", jobID, motivationID, antiMotivationID)
}

// generateLetters génère les deux lettres (motivation + anti-motivation)
func (w *LetterWorker) generateLetters(job *services.LetterJob) (uint, uint, error) {
	ctx := context.Background()
	startTime := time.Now()

	// NOTE: Les services AI, Scraper et PDF seront implémentés dans Doc 08
	// Pour l'instant, on crée un mock/placeholder

	// 1. Scraper infos entreprise (20% progress)
	w.queueService.UpdateJobStatus(job.JobID, services.JobStatusProcessing, 20)

	companyInfo := mockCompanyInfo(job.CompanyName)

	// 2. Génération lettre motivation (40% progress)
	w.queueService.UpdateJobStatus(job.JobID, services.JobStatusProcessing, 40)

	motivationContent := mockGenerateMotivationLetter(job.CompanyName, job.JobTitle)
	motivationTokens := 500

	// 3. Génération lettre anti-motivation (60% progress)
	w.queueService.UpdateJobStatus(job.JobID, services.JobStatusProcessing, 60)

	antiMotivationContent := mockGenerateAntiMotivationLetter(job.CompanyName, job.JobTitle)
	antiMotivationTokens := 500

	// 4. Sauvegarder en DB (70% progress)
	w.queueService.UpdateJobStatus(job.JobID, services.JobStatusProcessing, 70)

	// Convertir VisitorID string en UUID
	var visitorUUID models.BaseModel
	var visitor models.Visitor
	result := w.db.Where("session_id = ?", job.VisitorID).First(&visitor)
	if result.Error != nil {
		return 0, 0, fmt.Errorf("visitor not found: %w", result.Error)
	}
	visitorUUID.ID = visitor.ID

	// Lettre de motivation
	motivationLetter := models.GeneratedLetter{
		VisitorID:    visitor.ID,
		CompanyName:  job.CompanyName,
		LetterType:   models.LetterTypeMotivation,
		Content:      motivationContent,
		AIModel:      "claude-3-sonnet",
		TokensUsed:   motivationTokens,
		GenerationMS: int(time.Since(startTime).Milliseconds()),
		CompanyInfo:  companyInfo,
	}

	result = w.db.Create(&motivationLetter)
	if result.Error != nil {
		return 0, 0, fmt.Errorf("failed to save motivation letter: %w", result.Error)
	}

	// 5. Lettre anti-motivation (85% progress)
	w.queueService.UpdateJobStatus(job.JobID, services.JobStatusProcessing, 85)

	antiMotivationLetter := models.GeneratedLetter{
		VisitorID:    visitor.ID,
		CompanyName:  job.CompanyName,
		LetterType:   models.LetterTypeAntiMotivation,
		Content:      antiMotivationContent,
		AIModel:      "claude-3-sonnet",
		TokensUsed:   antiMotivationTokens,
		GenerationMS: int(time.Since(startTime).Milliseconds()),
		CompanyInfo:  companyInfo,
	}

	result = w.db.Create(&antiMotivationLetter)
	if result.Error != nil {
		return 0, 0, fmt.Errorf("failed to save anti-motivation letter: %w", result.Error)
	}

	// 6. Générer PDFs (90-100% progress) - Optionnel pour MVP
	w.queueService.UpdateJobStatus(job.JobID, services.JobStatusProcessing, 90)

	// TODO: Implémenter génération PDF (Doc 08)
	// Pour l'instant on skip cette étape

	w.queueService.UpdateJobStatus(job.JobID, services.JobStatusProcessing, 100)

	log.Printf("[LetterWorker] Letters generated in %dms (total tokens: %d)",
		time.Since(startTime).Milliseconds(),
		motivationTokens+antiMotivationTokens,
	)

	return uint(motivationLetter.ID), uint(antiMotivationLetter.ID), nil
}

// --- MOCK FUNCTIONS (à remplacer par les vrais services du Doc 08) ---

// mockCompanyInfo mock du scraper d'infos entreprise
func mockCompanyInfo(companyName string) string {
	info := map[string]interface{}{
		"name":        companyName,
		"industry":    "Technology",
		"size":        "1000-5000",
		"description": fmt.Sprintf("%s est une entreprise innovante dans le secteur technologique.", companyName),
		"source":      "mock",
	}

	jsonData, _ := json.Marshal(info)
	return string(jsonData)
}

// mockGenerateMotivationLetter mock de génération lettre motivation
func mockGenerateMotivationLetter(companyName, jobTitle string) string {
	if jobTitle == "" {
		jobTitle = "Développeur"
	}

	return fmt.Sprintf(`Madame, Monsieur,

C'est avec un grand intérêt que je vous soumets ma candidature pour un poste de %s au sein de %s.

Fort d'une expérience significative en développement logiciel et d'une passion pour l'innovation technologique, je suis convaincu que je pourrais apporter une contribution précieuse à votre équipe.

Mes compétences en backend (Go, Node.js), frontend (React, Next.js) et DevOps (Docker, Kubernetes) me permettent d'aborder des projets complexes avec une vision complète du cycle de développement.

Je serais honoré de pouvoir discuter de cette opportunité avec vous et de vous démontrer comment mon expertise pourrait bénéficier à %s.

Cordialement,
Alexi`, jobTitle, companyName, companyName)
}

// mockGenerateAntiMotivationLetter mock de génération lettre anti-motivation
func mockGenerateAntiMotivationLetter(companyName, jobTitle string) string {
	if jobTitle == "" {
		jobTitle = "Développeur"
	}

	return fmt.Sprintf(`Cher %s,

Après mûre réflexion, je dois avouer que je ne suis probablement PAS le candidat idéal pour votre poste de %s.

Voici pourquoi vous ne devriez pas m'embaucher :

1. Je préfère coder avec les pieds qu'avec les mains (c'est plus original)
2. Mon café préféré est le décaféiné (une hérésie dans le milieu tech)
3. Je considère que "ça marche sur ma machine" est une réponse acceptable
4. Je pense que les tests unitaires sont optionnels (comme les légumes dans un burger)
5. Mon style de code préféré ? Le "spaghetti code" bien sûr !

Plus sérieusement, je serais ravi de démontrer que derrière l'humour se cache un développeur passionné et compétent.

Avec humour (et sérieux en même temps),
Alexi

P.S. : Cette lettre d'anti-motivation est générée par IA. Si elle vous a fait sourire, imaginez ce que nous pourrions créer ensemble !`, companyName, jobTitle)
}
