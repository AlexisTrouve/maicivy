package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache is a wrapper around redis.Client with helper methods
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new Redis cache wrapper
func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{
		client: client,
	}
}

// Get retrieves a value from Redis
func (rc *RedisCache) Get(ctx context.Context, key string) (string, error) {
	val, err := rc.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Cache miss, not an error
	}
	return val, err
}

// Set stores a value in Redis with TTL
func (rc *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return rc.client.Set(ctx, key, value, ttl).Err()
}

// SetJSON stores a JSON-encoded value in Redis
func (rc *RedisCache) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return rc.client.Set(ctx, key, jsonData, ttl).Err()
}

// GetJSON retrieves and decodes a JSON value from Redis
func (rc *RedisCache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	val, err := rc.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return fmt.Errorf("key not found: %s", key)
	}
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return nil
}

// Delete removes a key from Redis
func (rc *RedisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return rc.client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists in Redis
func (rc *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := rc.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Expire sets an expiration on a key
func (rc *RedisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return rc.client.Expire(ctx, key, ttl).Err()
}

// TTL returns the remaining time to live of a key
func (rc *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return rc.client.TTL(ctx, key).Result()
}

// Incr increments a counter
func (rc *RedisCache) Incr(ctx context.Context, key string) (int64, error) {
	return rc.client.Incr(ctx, key).Result()
}

// Decr decrements a counter
func (rc *RedisCache) Decr(ctx context.Context, key string) (int64, error) {
	return rc.client.Decr(ctx, key).Result()
}

// IncrBy increments a counter by a specific amount
func (rc *RedisCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return rc.client.IncrBy(ctx, key, value).Result()
}

// GetOrSet retrieves value from cache or sets it using the fetch function
func (rc *RedisCache) GetOrSet(ctx context.Context, key string, ttl time.Duration, fetchFn func() (interface{}, error)) (string, error) {
	// Try to get from cache
	val, err := rc.Get(ctx, key)
	if err == nil && val != "" {
		return val, nil // Cache hit
	}

	// Cache miss, fetch data
	data, err := fetchFn()
	if err != nil {
		return "", err
	}

	// Store in cache
	if err := rc.SetJSON(ctx, key, data, ttl); err != nil {
		// Log error but return data anyway
		fmt.Printf("Warning: failed to cache data for key %s: %v\n", key, err)
	}

	// Return as JSON string
	jsonData, _ := json.Marshal(data)
	return string(jsonData), nil
}

// Keys retrieves all keys matching a pattern
func (rc *RedisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	return rc.client.Keys(ctx, pattern).Result()
}

// DeletePattern deletes all keys matching a pattern
func (rc *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	keys, err := rc.Keys(ctx, pattern)
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return rc.Delete(ctx, keys...)
	}
	return nil
}

// FlushDB flushes the entire database (use with caution!)
func (rc *RedisCache) FlushDB(ctx context.Context) error {
	return rc.client.FlushDB(ctx).Err()
}

// Ping checks if Redis connection is alive
func (rc *RedisCache) Ping(ctx context.Context) error {
	return rc.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (rc *RedisCache) Close() error {
	return rc.client.Close()
}

// HashSet sets a field in a hash
func (rc *RedisCache) HashSet(ctx context.Context, key string, field string, value interface{}) error {
	return rc.client.HSet(ctx, key, field, value).Err()
}

// HashGet retrieves a field from a hash
func (rc *RedisCache) HashGet(ctx context.Context, key string, field string) (string, error) {
	val, err := rc.client.HGet(ctx, key, field).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// HashGetAll retrieves all fields from a hash
func (rc *RedisCache) HashGetAll(ctx context.Context, key string) (map[string]string, error) {
	return rc.client.HGetAll(ctx, key).Result()
}

// HashDelete deletes fields from a hash
func (rc *RedisCache) HashDelete(ctx context.Context, key string, fields ...string) error {
	return rc.client.HDel(ctx, key, fields...).Err()
}

// ListPush adds elements to the beginning of a list
func (rc *RedisCache) ListPush(ctx context.Context, key string, values ...interface{}) error {
	return rc.client.LPush(ctx, key, values...).Err()
}

// ListRange retrieves a range of elements from a list
func (rc *RedisCache) ListRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return rc.client.LRange(ctx, key, start, stop).Result()
}

// SetAdd adds members to a set
func (rc *RedisCache) SetAdd(ctx context.Context, key string, members ...interface{}) error {
	return rc.client.SAdd(ctx, key, members...).Err()
}

// SetMembers retrieves all members of a set
func (rc *RedisCache) SetMembers(ctx context.Context, key string) ([]string, error) {
	return rc.client.SMembers(ctx, key).Result()
}

// SetRemove removes members from a set
func (rc *RedisCache) SetRemove(ctx context.Context, key string, members ...interface{}) error {
	return rc.client.SRem(ctx, key, members...).Err()
}

// SetIsMember checks if a value is a member of a set
func (rc *RedisCache) SetIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return rc.client.SIsMember(ctx, key, member).Result()
}
