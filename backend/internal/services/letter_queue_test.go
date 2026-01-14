package services

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestLetterQueueService_EnqueueJob(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	service := NewLetterQueueService(redisClient)

	jobID, err := service.EnqueueJob("visitor-123", "Google", "Software Engineer", "backend")

	assert.NoError(t, err)
	assert.NotEmpty(t, jobID)

	// Vérifier que le job est dans la queue
	queueLength, _ := service.GetQueueLength()
	assert.Equal(t, int64(1), queueLength)
}

func TestLetterQueueService_GetJobStatus(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	service := NewLetterQueueService(redisClient)

	jobID, _ := service.EnqueueJob("visitor-123", "Google", "", "")

	job, err := service.GetJobStatus(jobID)

	assert.NoError(t, err)
	assert.Equal(t, jobID, job.JobID)
	assert.Equal(t, "visitor-123", job.VisitorID)
	assert.Equal(t, "Google", job.CompanyName)
	assert.Equal(t, JobStatusQueued, job.Status)
	assert.Equal(t, 0, job.Progress)
}

func TestLetterQueueService_UpdateJobStatus(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	service := NewLetterQueueService(redisClient)

	jobID, _ := service.EnqueueJob("visitor-123", "Google", "", "")

	err := service.UpdateJobStatus(jobID, JobStatusProcessing, 50)
	assert.NoError(t, err)

	job, _ := service.GetJobStatus(jobID)
	assert.Equal(t, JobStatusProcessing, job.Status)
	assert.Equal(t, 50, job.Progress)
}

func TestLetterQueueService_CompleteJob(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	service := NewLetterQueueService(redisClient)

	jobID, _ := service.EnqueueJob("visitor-123", "Google", "", "")

	uuid1 := uuid.New()
	uuid2 := uuid.New()
	err := service.CompleteJob(jobID, uuid1, uuid2)
	assert.NoError(t, err)

	job, _ := service.GetJobStatus(jobID)
	assert.Equal(t, JobStatusCompleted, job.Status)
	assert.Equal(t, 100, job.Progress)
	assert.NotNil(t, job.LetterMotivationID)
	assert.NotNil(t, job.LetterAntiMotivationID)
	assert.Equal(t, uuid1, *job.LetterMotivationID)
	assert.Equal(t, uuid2, *job.LetterAntiMotivationID)
}

func TestLetterQueueService_FailJob(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	service := NewLetterQueueService(redisClient)

	jobID, _ := service.EnqueueJob("visitor-123", "Google", "", "")

	errMsg := "API timeout"
	err := service.FailJob(jobID, errMsg)
	assert.NoError(t, err)

	job, _ := service.GetJobStatus(jobID)
	assert.Equal(t, JobStatusFailed, job.Status)
	assert.NotNil(t, job.Error)
	assert.Equal(t, errMsg, *job.Error)
}

func TestLetterQueueService_PopJob(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	service := NewLetterQueueService(redisClient)

	jobID1, _ := service.EnqueueJob("visitor-123", "Google", "", "")
	jobID2, _ := service.EnqueueJob("visitor-456", "Meta", "", "")

	// Pop first job (FIFO)
	poppedID, err := service.PopJob()
	assert.NoError(t, err)
	assert.Equal(t, jobID1, poppedID)

	// Pop second job
	poppedID, err = service.PopJob()
	assert.NoError(t, err)
	assert.Equal(t, jobID2, poppedID)

	// Queue should be empty now
	queueLength, _ := service.GetQueueLength()
	assert.Equal(t, int64(0), queueLength)
}

func TestLetterQueueService_RetryJob(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	service := NewLetterQueueService(redisClient)

	jobID, _ := service.EnqueueJob("visitor-123", "Google", "", "")

	// Marquer comme failed
	service.FailJob(jobID, "Temporary error")

	// Retry
	err := service.RetryJob(jobID)
	assert.NoError(t, err)

	job, _ := service.GetJobStatus(jobID)
	assert.Equal(t, JobStatusQueued, job.Status)
	assert.Equal(t, 1, job.RetryCount)

	// Vérifier que le job est de nouveau dans la queue
	queueLength, _ := service.GetQueueLength()
	assert.Equal(t, int64(1), queueLength)
}

func TestLetterQueueService_MaxRetriesReached(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	service := NewLetterQueueService(redisClient)

	jobID, _ := service.EnqueueJob("visitor-123", "Google", "", "")

	job, _ := service.GetJobStatus(jobID)
	job.RetryCount = 3 // Max retries
	service.saveJob(job)

	err := service.RetryJob(jobID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max retries")
}

func TestLetterJob_EstimateRemainingTime(t *testing.T) {
	tests := []struct {
		progress int
		expected int
	}{
		{0, 30},
		{50, 15},
		{75, 7},
		{100, 0},
	}

	for _, test := range tests {
		job := LetterJob{
			Progress: test.progress,
		}
		result := job.EstimateRemainingTime()
		assert.Equal(t, test.expected, result)
	}
}

func TestLetterQueueService_GetQueueLength(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	service := NewLetterQueueService(redisClient)

	// Queue vide
	length, err := service.GetQueueLength()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), length)

	// Enqueue 3 jobs
	service.EnqueueJob("visitor-1", "Google", "", "")
	service.EnqueueJob("visitor-2", "Meta", "", "")
	service.EnqueueJob("visitor-3", "Amazon", "", "")

	length, err = service.GetQueueLength()
	assert.NoError(t, err)
	assert.Equal(t, int64(3), length)
}

func TestLetterQueueService_JobNotFound(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	service := NewLetterQueueService(redisClient)

	_, err := service.GetJobStatus("non-existent-job")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
