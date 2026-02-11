package redisDatabase

import (
	"bytes"
	"context"
	"discore/internal/base/infrastructure/redis/bloomFilter"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const NULL_Value = "__NULL__"

// CacheManager handles all cache operations with optional Bloom filter integration
type CacheManager struct {
	client   *redis.Client
	bloomMgr *bloomFilter.BloomManager
	group    singleflight.Group
}

// NewCacheManager creates a new cache manager
// bloomMgr is optional - pass nil to disable bloom filter
func NewCacheManager(client *redis.Client, bloomMgr *bloomFilter.BloomManager) *CacheManager {
	return &CacheManager{
		client:   client,
		bloomMgr: bloomMgr,
	}
}

// Set stores data in cache and optionally adds to bloom filter
// data can be any type - will be JSON marshaled
func (cm *CacheManager) Set(ctx context.Context, cacheKey string, bloomKey *bloomFilter.BloomFilterKey, data interface{}, ttl time.Duration) error {
	// Marshal data

	isNilData := data == nil
	// Check for "Typed" Nil (nil pointer passed inside interface)
	if !isNilData {
		val := reflect.ValueOf(data)
		// We must check the Kind first to avoid panics on non-pointer types
		if val.Kind() == reflect.Ptr && val.IsNil() {
			isNilData = true
		}
	}
	if isNilData {
		// Write NULL to cache only
		if err := cm.client.Set(ctx, cacheKey, NULL_Value, ttl).Err(); err != nil {
			return fmt.Errorf("cache set failed: %w", err)
		}
		return nil
	}

	byteData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("cache marshal failed: %w", err)
	}

	// Write to cache
	if err := cm.client.Set(ctx, cacheKey, byteData, ttl).Err(); err != nil {
		return fmt.Errorf("cache set failed: %w", err)
	}

	// Add to bloom filter if configured
	if cm.bloomMgr != nil && bloomKey != nil && *bloomKey != "" {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = cm.bloomMgr.Add(ctx, *bloomKey, cacheKey)
		}()
	}

	return nil
}

// Get retrieves data from cache with optional bloom filter check
// Returns raw bytes or nil (if not exists) - caller handles unmarshaling
func (cm *CacheManager) Get(ctx context.Context, cacheKey string, bloomKey *bloomFilter.BloomFilterKey) ([]byte, error) {
	result, err, _ := cm.group.Do(cacheKey, func() (interface{}, error) {
		// Check bloom filter first if configured
		if cm.bloomMgr != nil && *bloomKey != "" {
			// logrus.Info("[Bloom check]")
			mightExist, err := cm.bloomMgr.MightContain(ctx, *bloomKey, cacheKey)
			if err != nil {
				// Log error but continue (fail open)
			} else if !mightExist {
				return nil, nil
			}

		}
		// Get from cache
		// logrus.Info("[Cache check]")
		val, err := cm.client.Get(ctx, cacheKey).Bytes()
		if err != nil {
			if err == redis.Nil {
				return nil, redis.Nil
			}
			return nil, fmt.Errorf("cache get failed: %w", err)
		}

		return val, nil
	})

	if err != nil {
		return nil, err
	}

	resultBytes, ok := result.([]byte) // go value to byte slice
	if !ok {
		return nil, fmt.Errorf("unexpected type from cache: %T", result)
	}

	if bytes.Equal(resultBytes, []byte(NULL_Value)) {
		return nil, nil
	}

	return resultBytes, nil
}

// Delete removes from cache
func (cm *CacheManager) Delete(ctx context.Context, cacheKey string) error {
	return cm.client.Del(ctx, cacheKey).Err()
}

// MGet retrieves multiple keys from Redis cache
// Returns a map of cacheKey -> raw value (string or []byte)
func (cm *CacheManager) MGet(ctx context.Context, cacheKeys []string) (map[string][]byte, error) {
	if len(cacheKeys) == 0 {
		return map[string][]byte{}, nil
	}

	vals, err := cm.client.MGet(ctx, cacheKeys...).Result()
	if err != nil {
		return nil, fmt.Errorf("cache MGet failed: %w", err)
	}

	result := make(map[string][]byte, len(cacheKeys))
	for i, v := range vals {
		if v != nil {
			var bytes []byte
			switch t := v.(type) {
			case string:
				bytes = []byte(t)
			case []byte:
				bytes = t
			default:
				// unexpected type, skip
				continue
			}
			result[cacheKeys[i]] = bytes
		}

	}
	return result, nil
}
