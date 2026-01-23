package flagkit

import (
	"time"

	"github.com/flagkit/flagkit-go/internal/core"
)

// CacheEntry represents a cached flag with metadata.
type CacheEntry = core.CacheEntry

// Cache is an in-memory cache for flag states.
type Cache = core.Cache

// CacheConfig contains cache configuration.
type CacheConfig struct {
	TTL     time.Duration
	MaxSize int
	Logger  Logger
}

// DefaultCacheConfig returns default cache configuration.
func DefaultCacheConfig() *CacheConfig {
	cfg := core.DefaultCacheConfig()
	return &CacheConfig{
		TTL:     cfg.TTL,
		MaxSize: cfg.MaxSize,
	}
}

// NewCache creates a new cache with the given configuration.
func NewCache(config *CacheConfig) *Cache {
	var coreConfig *core.CacheConfig
	if config != nil {
		coreConfig = &core.CacheConfig{
			TTL:     config.TTL,
			MaxSize: config.MaxSize,
			Logger:  config.Logger,
		}
	}
	return core.NewCache(coreConfig)
}
