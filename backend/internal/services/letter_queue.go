package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// JobStatus représente le status d'un job de génération
type JobStatus string

const (
	JobStatusQueued     JobStatus = "queued"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

// LetterJob représente un job de génération de lettre
type LetterJob struct {
	JobID       string    `json:"job_id"`
	VisitorID   string    `json:"visitor_id"`    // Session ID du visiteur
	CompanyName string    `json:"company_name"`
	JobTitle    string    `json:"job_title,omitempty"`
	Theme       string    `json:"theme,omitempty"`
	Status      JobStatus `json:"status"`
	Progress    int       `json:"progress"` // 0-100

	// Résultats (si completed)
	LetterMotivationID     *uuid.UUID `json:"letter_motivation_id,omitempty"`
	LetterAntiMotivationID *uuid.UUID `json:"letter_anti_motivation_id,omitempty"`

	// Erreur (si failed)
	Error *string `json:"error,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Retry tracking
	RetryCount int `json:"retry_count"`
	MaxRetries int `json:"max_retries"`
}

// LetterQueueService service de gestion de la queue de génération de lettres
type LetterQueueService struct {
	redis *redis.Client
	ctx   context.Context
}

// NewLetterQueueService crée une nouvelle instance du service
func NewLetterQueueService(redis *redis.Client) *LetterQueueService {
	return &LetterQueueService{
		redis: redis,
		ctx:   context.Background(),
	}
}

// EnqueueJob ajoute un job dans la queue
func (s *LetterQueueService) EnqueueJob(visitorID, companyName string, jobTitle, theme string) (string, error) {
	jobID := uuid.New().String()

	job := LetterJob{
		JobID:       jobID,
		VisitorID:   visitorID,
		CompanyName: companyName,
		JobTitle:    jobTitle,
		Theme:       theme,
		Status:      JobStatusQueued,
		Progress:    0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		RetryCount:  0,
		MaxRetries:  3,
	}

	// Sérialiser le job
	jobJSON, err := json.Marshal(job)
	if err != nil {
		return "", fmt.Errorf("failed to marshal job: %w", err)
	}

	// Stocker le job dans Redis (Hash)
	jobKey := fmt.Sprintf("job:letter:%s", jobID)
	err = s.redis.Set(s.ctx, jobKey, jobJSON, 24*time.Hour).Err() // TTL 24h
	if err != nil {
		return "", fmt.Errorf("failed to store job: %w", err)
	}

	// Ajouter à la queue (List FIFO)
	queueKey := "queue:letters"
	err = s.redis.RPush(s.ctx, queueKey, jobID).Err()
	if err != nil {
		return "", fmt.Errorf("failed to enqueue job: %w", err)
	}

	return jobID, nil
}

// GetJobStatus récupère le status d'un job
func (s *LetterQueueService) GetJobStatus(jobID string) (*LetterJob, error) {
	jobKey := fmt.Sprintf("job:letter:%s", jobID)

	jobJSON, err := s.redis.Get(s.ctx, jobKey).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("job not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	var job LetterJob
	err = json.Unmarshal([]byte(jobJSON), &job)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// UpdateJobStatus met à jour le status d'un job
func (s *LetterQueueService) UpdateJobStatus(jobID string, status JobStatus, progress int) error {
	job, err := s.GetJobStatus(jobID)
	if err != nil {
		return err
	}

	job.Status = status
	job.Progress = progress
	job.UpdatedAt = time.Now()

	return s.saveJob(job)
}

// CompleteJob marque un job comme complété
func (s *LetterQueueService) CompleteJob(jobID string, motivationID, antiMotivationID uuid.UUID) error {
	job, err := s.GetJobStatus(jobID)
	if err != nil {
		return err
	}

	job.Status = JobStatusCompleted
	job.Progress = 100
	job.LetterMotivationID = &motivationID
	job.LetterAntiMotivationID = &antiMotivationID
	job.UpdatedAt = time.Now()

	return s.saveJob(job)
}

// FailJob marque un job comme échoué
func (s *LetterQueueService) FailJob(jobID string, errorMsg string) error {
	job, err := s.GetJobStatus(jobID)
	if err != nil {
		return err
	}

	job.Status = JobStatusFailed
	job.Error = &errorMsg
	job.UpdatedAt = time.Now()

	return s.saveJob(job)
}

// RetryJob incrémente le compteur de retry et re-enqueue si max pas atteint
func (s *LetterQueueService) RetryJob(jobID string) error {
	job, err := s.GetJobStatus(jobID)
	if err != nil {
		return err
	}

	if job.RetryCount >= job.MaxRetries {
		return fmt.Errorf("max retries reached")
	}

	job.RetryCount++
	job.Status = JobStatusQueued
	job.Progress = 0
	job.UpdatedAt = time.Now()

	err = s.saveJob(job)
	if err != nil {
		return err
	}

	// Re-enqueue
	queueKey := "queue:letters"
	return s.redis.RPush(s.ctx, queueKey, jobID).Err()
}

// PopJob récupère le prochain job de la queue (pour worker)
func (s *LetterQueueService) PopJob() (string, error) {
	queueKey := "queue:letters"

	// BLPOP avec timeout de 1 seconde (blocking)
	result, err := s.redis.BLPop(s.ctx, 1*time.Second, queueKey).Result()
	if err == redis.Nil {
		return "", nil // Queue vide (timeout)
	}
	if err != nil {
		return "", fmt.Errorf("failed to pop job: %w", err)
	}

	if len(result) < 2 {
		return "", nil
	}

	return result[1], nil // result[0] est la clé, result[1] est la valeur
}

// GetQueueLength retourne le nombre de jobs en attente
func (s *LetterQueueService) GetQueueLength() (int64, error) {
	queueKey := "queue:letters"
	length, err := s.redis.LLen(s.ctx, queueKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get queue length: %w", err)
	}
	return length, nil
}

// CleanupOldJobs nettoie les jobs expirés (>24h)
func (s *LetterQueueService) CleanupOldJobs() error {
	// Cette fonction peut être appelée par un cronjob
	// Les jobs ont déjà un TTL de 24h dans Redis, donc cleanup automatique
	// Cette fonction est optionnelle pour cleanup manuel si nécessaire
	return nil
}

// saveJob sauvegarde un job dans Redis
func (s *LetterQueueService) saveJob(job *LetterJob) error {
	jobJSON, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	jobKey := fmt.Sprintf("job:letter:%s", job.JobID)
	err = s.redis.Set(s.ctx, jobKey, jobJSON, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to save job: %w", err)
	}

	return nil
}

// EstimateRemainingTime estime le temps restant basé sur le progress
func (job *LetterJob) EstimateRemainingTime() int {
	// Temps total estimé: 30 secondes
	totalSeconds := 30
	if job.Progress >= 100 {
		return 0
	}
	remaining := (100 - job.Progress) * totalSeconds / 100
	return remaining
}
