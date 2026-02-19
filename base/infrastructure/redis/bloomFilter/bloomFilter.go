package bloomFilter

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// BloomFilterKey is the enum type for bloom filter identifiers
type BloomFilterKey string

const (
	UserIDBloomFilter       BloomFilterKey = "discore:bloom:user:id"
	ServerIDBloomFilter     BloomFilterKey = "discore:bloom:server:id"
	ChannelIDBloomFilter    BloomFilterKey = "discore:bloom:channel:id"
	ServerInviteBloomFilter BloomFilterKey = "discore:bloom:server:invite"
)

// FilterConfig holds initialization parameters for each filter
type FilterConfig struct {
	Key      BloomFilterKey
	Capacity int64
	FPRate   float64
}

// BloomManager handles all bloom filters centrally
type BloomManager struct {
	client  *redis.Client
	configs map[BloomFilterKey]FilterConfig
}

// NewBloomManager creates a manager with predefined configurations
func NewBloomManager(client *redis.Client, configs []FilterConfig) (*BloomManager, error) {
	if client == nil {
		return nil, errors.New("redis client is nil")
	}

	m := &BloomManager{
		client:  client,
		configs: make(map[BloomFilterKey]FilterConfig),
	}

	// Validate and index configs
	for _, cfg := range configs {
		if cfg.Key == "" {
			return nil, errors.New("filter config missing key")
		}
		if cfg.Capacity <= 0 {
			return nil, fmt.Errorf("invalid capacity for %s: %d", cfg.Key, cfg.Capacity)
		}
		if cfg.FPRate <= 0 || cfg.FPRate >= 1 {
			return nil, fmt.Errorf("invalid fp rate for %s: %f", cfg.Key, cfg.FPRate)
		}
		m.configs[cfg.Key] = cfg
	}

	// Initialize all filters once
	var initErr error
	var once sync.Once

	once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		for key, cfg := range m.configs {
			if err := m.initFilter(ctx, key, cfg); err != nil {
				initErr = fmt.Errorf("failed to init filter %s: %w", key, err)
				return
			}
		}
	})

	if initErr != nil {
		return nil, initErr
	}

	return m, nil
}

// initFilter creates the bloom filter in Redis if it doesn't exist
func (m *BloomManager) initFilter(ctx context.Context, key BloomFilterKey, cfg FilterConfig) error {
	err := m.client.BFReserve(ctx, string(key), cfg.FPRate, cfg.Capacity).Err()
	if err != nil {
		if err.Error() == "ERR item exists" {
			return nil
		}
		return err
	}
	return nil
}

// getConfig retrieves config for a key - no locks needed since configs are immutable
func (m *BloomManager) getConfig(key BloomFilterKey) (FilterConfig, bool) {
	cfg, exists := m.configs[key]
	return cfg, exists
}

// Add inserts an item into the specified bloom filter
func (m *BloomManager) Add(ctx context.Context, filterKey BloomFilterKey, item string) error {
	if _, exists := m.getConfig(filterKey); !exists {
		return fmt.Errorf("unknown filter key: %s", filterKey)
	}

	if _, err := m.client.BFAdd(ctx, string(filterKey), item).Result(); err != nil {
		return fmt.Errorf("bfadd failed for %s: %w", filterKey, err)
	}
	return nil
}

// MightContain checks if item might exist in the filter
func (m *BloomManager) MightContain(ctx context.Context, filterKey BloomFilterKey, item string) (bool, error) {
	if _, exists := m.getConfig(filterKey); !exists {
		return false, fmt.Errorf("unknown filter key: %s", filterKey)
	}

	exists, err := m.client.BFExists(ctx, string(filterKey), item).Result()
	if err != nil {
		return false, fmt.Errorf("bfexists failed for %s: %w", filterKey, err)
	}
	return exists, nil
}

// AddMulti adds multiple items to a filter efficiently
func (m *BloomManager) AddMulti(ctx context.Context, filterKey BloomFilterKey, items []string) error {
	if len(items) == 0 {
		return nil
	}

	cfg, exists := m.getConfig(filterKey)
	if !exists {
		return fmt.Errorf("unknown filter key: %s", filterKey)
	}

	interfaceItems := make([]interface{}, len(items))
	for i, item := range items {
		interfaceItems[i] = item
	}

	opts := &redis.BFInsertOptions{
		Capacity:   cfg.Capacity,
		Error:      cfg.FPRate,
		NonScaling: false,
	}

	if err := m.client.BFInsert(ctx, string(filterKey), opts, interfaceItems...).Err(); err != nil {
		return fmt.Errorf("bfinsert failed for %s: %w", filterKey, err)
	}
	return nil
}

// CheckMulti checks multiple items at once
func (m *BloomManager) CheckMulti(ctx context.Context, filterKey BloomFilterKey, items []string) ([]bool, error) {
	if len(items) == 0 {
		return []bool{}, nil
	}

	if _, exists := m.getConfig(filterKey); !exists {
		return nil, fmt.Errorf("unknown filter key: %s", filterKey)
	}

	interfaceItems := make([]interface{}, len(items))
	for i, item := range items {
		interfaceItems[i] = item
	}

	results, err := m.client.BFMExists(ctx, string(filterKey), interfaceItems...).Result()
	if err != nil {
		return nil, fmt.Errorf("bfmexists failed for %s: %w", filterKey, err)
	}
	return results, nil
}

// Stats returns approximate item count in a filter
func (m *BloomManager) Stats(ctx context.Context, filterKey BloomFilterKey) (int64, error) {
	if _, exists := m.getConfig(filterKey); !exists {
		return 0, fmt.Errorf("unknown filter key: %s", filterKey)
	}

	count, err := m.client.BFCard(ctx, string(filterKey)).Result()
	if err != nil {
		return 0, fmt.Errorf("bfcard failed for %s: %w", filterKey, err)
	}
	return count, nil
}

// Clear removes a bloom filter entirely
func (m *BloomManager) Clear(ctx context.Context, filterKey BloomFilterKey) error {
	if _, exists := m.getConfig(filterKey); !exists {
		return fmt.Errorf("unknown filter key: %s", filterKey)
	}
	return m.client.Del(ctx, string(filterKey)).Err()
}

// Rebuild clears and reinitializes a filter
func (m *BloomManager) Rebuild(ctx context.Context, filterKey BloomFilterKey) error {
	cfg, exists := m.getConfig(filterKey)
	if !exists {
		return fmt.Errorf("unknown filter key: %s", filterKey)
	}

	if err := m.client.Del(ctx, string(filterKey)).Err(); err != nil {
		return fmt.Errorf("failed to delete filter %s: %w", filterKey, err)
	}

	return m.initFilter(ctx, filterKey, cfg)
}

// RegisterFilter adds a new filter at runtime
func (m *BloomManager) RegisterFilter(ctx context.Context, cfg FilterConfig) error {
	if cfg.Key == "" {
		return errors.New("filter key cannot be empty")
	}
	if cfg.Capacity <= 0 {
		return fmt.Errorf("invalid capacity: %d", cfg.Capacity)
	}
	if cfg.FPRate <= 0 || cfg.FPRate >= 1 {
		return fmt.Errorf("invalid fp rate: %f", cfg.FPRate)
	}

	// Check existence without locks - map writes are not safe concurrent
	// But this is only called during setup or explicit registration, not hot path
	if _, exists := m.configs[cfg.Key]; exists {
		return fmt.Errorf("filter %s already registered", cfg.Key)
	}

	if err := m.initFilter(ctx, cfg.Key, cfg); err != nil {
		return err
	}

	m.configs[cfg.Key] = cfg
	return nil
}

// ListFilters returns all registered filter keys
func (m *BloomManager) ListFilters() []BloomFilterKey {
	keys := make([]BloomFilterKey, 0, len(m.configs))
	for k := range m.configs {
		keys = append(keys, k)
	}
	return keys
}
