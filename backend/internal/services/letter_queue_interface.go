package services

import "github.com/google/uuid"

// LetterQueueServiceInterface defines the interface for letter queue operations
type LetterQueueServiceInterface interface {
	EnqueueJob(visitorID, companyName string, jobTitle, theme string) (string, error)
	GetJobStatus(jobID string) (*LetterJob, error)
	UpdateJobStatus(jobID string, status JobStatus, progress int) error
	CompleteJob(jobID string, motivationID, antiMotivationID uuid.UUID) error
	FailJob(jobID string, errorMsg string) error
	RetryJob(jobID string) error
	PopJob() (string, error)
	GetQueueLength() (int64, error)
	CleanupOldJobs() error
}
