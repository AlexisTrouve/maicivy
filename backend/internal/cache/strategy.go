package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheService encapsulates caching logic with Redis
type CacheService struct {
	client *redis.Client
}

// NewCacheService creates a new cache service instance
func NewCacheService(redisClient *redis.Client) *CacheService {
	return &CacheService{
		client: redisClient,
	}
}

// TTL constants for different data types
const (
	// Quasi-static data (changes rarely)
	TTL_SKILLS   = 24 * time.Hour
	TTL_PROJECTS = 24 * time.Hour
	TTL_CV       = 1 * time.Hour // CV adapted by theme

	// Dynamic data
	TTL_LETTERS   = 30 * 24 * time.Hour // Generated letters cached 30 days
	TTL_COMPANY   = 7 * 24 * time.Hour  // Company info cached 7 days
	TTL_ANALYTICS = 5 * time.Minute     // Real-time stats, short TTL
)

// GetCV retrieves CV from cache or returns empty string if not found
func (cs *CacheService) GetCV(ctx context.Context, theme string) (string, error) {
	key := fmt.Sprintf("cv:%s", theme)
	val, err := cs.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Cache miss
	}
	return val, err
}

// SetCV stores CV in cache with TTL
func (cs *CacheService) SetCV(ctx context.Context, theme string, cvData interface{}) error {
	key := fmt.Sprintf("cv:%s", theme)
	cvJSON, err := json.Marshal(cvData)
	if err != nil {
		return fmt.Errorf("failed to marshal CV data: %w", err)
	}
	return cs.client.Set(ctx, key, string(cvJSON), TTL_CV).Err()
}

// GetLetter retrieves generated letter from cache
func (cs *CacheService) GetLetter(ctx context.Context, companyName, letterType string) (string, error) {
	// Hash company name to avoid excessively long keys
	hash := fmt.Sprintf("%x", companyName) // In production, use crypto/sha256
	key := fmt.Sprintf("letter:%s:%s", hash, letterType) // "motivation" or "anti-motivation"
	val, err := cs.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// SetLetter stores generated letter in cache
func (cs *CacheService) SetLetter(ctx context.Context, companyName, letterType, content string) error {
	hash := fmt.Sprintf("%x", companyName)
	key := fmt.Sprintf("letter:%s:%s", hash, letterType)
	return cs.client.Set(ctx, key, content, TTL_LETTERS).Err()
}

// GetCompanyInfo retrieves scraped company info from cache
func (cs *CacheService) GetCompanyInfo(ctx context.Context, companyName string) (string, error) {
	hash := fmt.Sprintf("%x", companyName)
	key := fmt.Sprintf("company_info:%s", hash)
	val, err := cs.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// SetCompanyInfo stores company info in cache
func (cs *CacheService) SetCompanyInfo(ctx context.Context, companyName, info string) error {
	hash := fmt.Sprintf("%x", companyName)
	key := fmt.Sprintf("company_info:%s", hash)
	return cs.client.Set(ctx, key, info, TTL_COMPANY).Err()
}

// InvalidateCV invalidates CV cache (after modification)
func (cs *CacheService) InvalidateCV(ctx context.Context, theme string) error {
	key := fmt.Sprintf("cv:%s", theme)
	return cs.client.Del(ctx, key).Err()
}

// InvalidateAllCVs invalidates all CVs in cache
func (cs *CacheService) InvalidateAllCVs(ctx context.Context) error {
	keys, err := cs.client.Keys(ctx, "cv:*").Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return cs.client.Del(ctx, keys...).Err()
	}
	return nil
}

// GetOrSet retrieves value from cache or fetches it using provided function
func (cs *CacheService) GetOrSet(ctx context.Context, key string, ttl time.Duration, fetchFn func() (interface{}, error)) (string, error) {
	// Try to get from cache
	val, err := cs.client.Get(ctx, key).Result()
	if err == nil {
		return val, nil // Cache hit
	}
	if err != redis.Nil {
		return "", err // Real error
	}

	// Cache miss, fetch data
	data, err := fetchFn()
	if err != nil {
		return "", err
	}

	// Store in cache
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := cs.client.Set(ctx, key, string(dataJSON), ttl).Err(); err != nil {
		// Log error but return data anyway
		return string(dataJSON), nil
	}

	return string(dataJSON), nil
}

// IncrementCounter increments a counter in cache
func (cs *CacheService) IncrementCounter(ctx context.Context, key string) (int64, error) {
	return cs.client.Incr(ctx, key).Result()
}

// GetCounter retrieves counter value
func (cs *CacheService) GetCounter(ctx context.Context, key string) (int64, error) {
	val, err := cs.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// ExpireKey sets expiration on a key
func (cs *CacheService) ExpireKey(ctx context.Context, key string, ttl time.Duration) error {
	return cs.client.Expire(ctx, key, ttl).Err()
}

// DeleteKey deletes a key from cache
func (cs *CacheService) DeleteKey(ctx context.Context, key string) error {
	return cs.client.Del(ctx, key).Err()
}

// DeletePattern deletes all keys matching pattern
func (cs *CacheService) DeletePattern(ctx context.Context, pattern string) error {
	keys, err := cs.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return cs.client.Del(ctx, keys...).Err()
	}
	return nil
}

// Exists checks if key exists in cache
func (cs *CacheService) Exists(ctx context.Context, key string) (bool, error) {
	val, err := cs.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}
