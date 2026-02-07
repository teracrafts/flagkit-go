package core

import (
	"sync"
	"time"

	"github.com/teracrafts/flagkit-go/internal/types"
)

// CacheEntry represents a cached flag with metadata.
type CacheEntry struct {
	Flag      types.FlagState
	FetchedAt time.Time
	ExpiresAt time.Time
}

// Cache is an in-memory cache for flag states.
type Cache struct {
	entries map[string]*CacheEntry
	mu      sync.RWMutex
	ttl     time.Duration
	maxSize int
	logger  types.Logger
}

// CacheConfig contains cache configuration.
type CacheConfig struct {
	TTL     time.Duration
	MaxSize int
	Logger  types.Logger
}

// DefaultCacheConfig returns default cache configuration.
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		TTL:     5 * time.Minute,
		MaxSize: 1000,
	}
}

// NewCache creates a new cache with the given configuration.
func NewCache(config *CacheConfig) *Cache {
	if config == nil {
		config = DefaultCacheConfig()
	}
	return &Cache{
		entries: make(map[string]*CacheEntry),
		ttl:     config.TTL,
		maxSize: config.MaxSize,
		logger:  config.Logger,
	}
}

// Get retrieves a flag from the cache.
// Returns nil if not found or expired.
func (c *Cache) Get(key string) *types.FlagState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil
	}

	if time.Now().After(entry.ExpiresAt) {
		if c.logger != nil {
			c.logger.Debug("Cache miss (expired)", "key", key)
		}
		return nil
	}

	if c.logger != nil {
		c.logger.Debug("Cache hit", "key", key)
	}
	return &entry.Flag
}

// GetStale retrieves a flag from the cache even if expired.
func (c *Cache) GetStale(key string) *types.FlagState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil
	}
	return &entry.Flag
}

// IsStale checks if a cached entry is expired.
func (c *Cache) IsStale(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return false
	}
	return time.Now().After(entry.ExpiresAt)
}

// Set stores a flag in the cache.
func (c *Cache) Set(key string, flag types.FlagState, ttl ...time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Enforce max size
	if len(c.entries) >= c.maxSize {
		if _, exists := c.entries[key]; !exists {
			c.evictOldest()
		}
	}

	cacheTTL := c.ttl
	if len(ttl) > 0 {
		cacheTTL = ttl[0]
	}

	now := time.Now()
	c.entries[key] = &CacheEntry{
		Flag:      flag,
		FetchedAt: now,
		ExpiresAt: now.Add(cacheTTL),
	}

	if c.logger != nil {
		c.logger.Debug("Cache set", "key", key, "ttl", cacheTTL)
	}
}

// SetMany stores multiple flags in the cache.
func (c *Cache) SetMany(flags []types.FlagState, ttl ...time.Duration) {
	for _, flag := range flags {
		c.Set(flag.Key, flag, ttl...)
	}
}

// Delete removes a flag from the cache.
func (c *Cache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.entries[key]; ok {
		delete(c.entries, key)
		if c.logger != nil {
			c.logger.Debug("Cache delete", "key", key)
		}
		return true
	}
	return false
}

// Clear removes all entries from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	size := len(c.entries)
	c.entries = make(map[string]*CacheEntry)
	if c.logger != nil {
		c.logger.Debug("Cache cleared", "entries", size)
	}
}

// Has checks if a flag exists in the cache (including stale).
func (c *Cache) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.entries[key]
	return ok
}

// GetAllKeys returns all cached flag keys.
func (c *Cache) GetAllKeys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.entries))
	for key := range c.entries {
		keys = append(keys, key)
	}
	return keys
}

// GetAll returns all cached flags (including stale).
func (c *Cache) GetAll() []types.FlagState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	flags := make([]types.FlagState, 0, len(c.entries))
	for _, entry := range c.entries {
		flags = append(flags, entry.Flag)
	}
	return flags
}

// GetAllValid returns all non-expired cached flags.
func (c *Cache) GetAllValid() []types.FlagState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	flags := make([]types.FlagState, 0)
	for _, entry := range c.entries {
		if now.Before(entry.ExpiresAt) {
			flags = append(flags, entry.Flag)
		}
	}
	return flags
}

// Size returns the number of cached entries.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Stats returns cache statistics.
func (c *Cache) Stats() map[string]int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	validCount := 0
	staleCount := 0

	for _, entry := range c.entries {
		if now.Before(entry.ExpiresAt) {
			validCount++
		} else {
			staleCount++
		}
	}

	return map[string]int{
		"size":        len(c.entries),
		"valid_count": validCount,
		"stale_count": staleCount,
		"max_size":    c.maxSize,
	}
}

// evictOldest removes the oldest entry from the cache.
func (c *Cache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	first := true
	for key, entry := range c.entries {
		if first || entry.FetchedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.FetchedAt
			first = false
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
		if c.logger != nil {
			c.logger.Debug("Cache evicted oldest", "key", oldestKey)
		}
	}
}
