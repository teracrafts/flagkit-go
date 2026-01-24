package flagkit

// This file exports internal types for testing and advanced usage.
// Users typically don't need to use these directly.

import (
	"time"

	"github.com/flagkit/flagkit-go/internal"
)

// Cache wraps internal.Cache for public access
type Cache struct {
	*internal.Cache
}

// CacheConfig is an alias for internal.CacheConfig
type CacheConfig struct {
	TTL     time.Duration
	MaxSize int
	Logger  Logger
}

// NewCache creates a new cache.
func NewCache(config *CacheConfig) *Cache {
	return &Cache{
		Cache: internal.NewCache(&internal.CacheConfig{
			TTL:     config.TTL,
			MaxSize: config.MaxSize,
			Logger:  config.Logger,
		}),
	}
}

// Get retrieves a flag from the cache.
func (c *Cache) Get(key string) *FlagState {
	internal := c.Cache.Get(key)
	if internal == nil {
		return nil
	}
	return internalToPublicFlagState(internal)
}

// GetStale retrieves a flag from the cache even if expired.
func (c *Cache) GetStale(key string) *FlagState {
	internal := c.Cache.GetStale(key)
	if internal == nil {
		return nil
	}
	return internalToPublicFlagState(internal)
}

// Set stores a flag in the cache.
func (c *Cache) Set(key string, flag FlagState, ttl ...time.Duration) {
	c.Cache.Set(key, publicToInternalFlagState(flag), ttl...)
}

// SetMany stores multiple flags in the cache.
func (c *Cache) SetMany(flags []FlagState, ttl ...time.Duration) {
	internalFlags := make([]internal.FlagState, len(flags))
	for i, f := range flags {
		internalFlags[i] = publicToInternalFlagState(f)
	}
	c.Cache.SetMany(internalFlags, ttl...)
}

// DefaultCacheConfig returns the default cache configuration.
func DefaultCacheConfig() *CacheConfig {
	cfg := internal.DefaultCacheConfig()
	return &CacheConfig{
		TTL:     cfg.TTL,
		MaxSize: cfg.MaxSize,
	}
}

// CircuitBreaker type aliases for testing
type CircuitBreaker = internal.CircuitBreaker
type CircuitBreakerConfig = internal.CircuitBreakerConfig
type CircuitState = internal.CircuitState

const (
	CircuitClosed   = internal.CircuitClosed
	CircuitOpen     = internal.CircuitOpen
	CircuitHalfOpen = internal.CircuitHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	return internal.NewCircuitBreaker(config)
}

// DefaultCircuitBreakerConfig returns the default circuit breaker configuration.
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return internal.DefaultCircuitBreakerConfig()
}

// Retry type aliases for testing
type RetryConfig = internal.RetryConfig

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() *RetryConfig {
	return internal.DefaultRetryConfig()
}

// calculateBackoff calculates the backoff delay for a retry attempt.
func calculateBackoff(attempt int, config *RetryConfig) time.Duration {
	return internal.CalculateBackoff(attempt, config)
}

// WithRetry executes a function with retry logic.
func WithRetry[T any](fn func() (T, error), config *RetryConfig) (T, error) {
	return internal.WithRetry(fn, config)
}

// Event type aliases for testing
type Event = internal.Event

// EventQueue wraps internal.EventQueue for public access
type EventQueue struct {
	*internal.EventQueue
}

// EventQueueConfig is an alias for internal.EventQueueConfig
type EventQueueConfig = internal.EventQueueConfig

// EventQueueOptions contains options for creating an event queue
type EventQueueOptions struct {
	SessionID     string
	EnvironmentID string
	SDKVersion    string
	Logger        Logger
	Config        *EventQueueConfig
}

// NewEventQueue creates a new event queue.
func NewEventQueue(opts *EventQueueOptions) *EventQueue {
	return &EventQueue{
		EventQueue: internal.NewEventQueue(&internal.EventQueueOptions{
			SessionID:     opts.SessionID,
			EnvironmentID: opts.EnvironmentID,
			SDKVersion:    opts.SDKVersion,
			Logger:        opts.Logger,
			Config:        opts.Config,
		}),
	}
}

// TrackWithContext adds an event with context to the queue.
func (eq *EventQueue) TrackWithContext(eventType string, data map[string]interface{}, ctx *EvaluationContext) {
	eq.EventQueue.TrackWithContext(eventType, data, publicToInternalContext(ctx))
}

// DefaultEventQueueConfig returns the default event queue configuration.
func DefaultEventQueueConfig() *EventQueueConfig {
	return internal.DefaultEventQueueConfig()
}

// Polling type aliases for testing
type PollingManager = internal.PollingManager
type PollingConfig = internal.PollingConfig

// NewPollingManager creates a new polling manager.
func NewPollingManager(onPoll func(), config *PollingConfig, logger Logger) *PollingManager {
	return internal.NewPollingManager(onPoll, config, logger)
}

// DefaultPollingConfig returns the default polling configuration.
func DefaultPollingConfig() *PollingConfig {
	return internal.DefaultPollingConfig()
}

// HTTP client type aliases for advanced usage
type HTTPClient = internal.HTTPClient
type HTTPClientConfig = internal.HTTPClientConfig
type HTTPResponse = internal.HTTPResponse

// NewHTTPClient creates a new HTTP client.
func NewHTTPClient(config *HTTPClientConfig) *HTTPClient {
	return internal.NewHTTPClient(config)
}

// Helper functions for type conversion

func internalToPublicFlagState(f *internal.FlagState) *FlagState {
	return &FlagState{
		Key:          f.Key,
		Value:        f.Value,
		Enabled:      f.Enabled,
		Version:      f.Version,
		FlagType:     FlagType(f.FlagType),
		LastModified: f.LastModified,
	}
}

func publicToInternalFlagState(f FlagState) internal.FlagState {
	return internal.FlagState{
		Key:          f.Key,
		Value:        f.Value,
		Enabled:      f.Enabled,
		Version:      f.Version,
		FlagType:     internal.FlagType(f.FlagType),
		LastModified: f.LastModified,
	}
}

func publicToInternalContext(ctx *EvaluationContext) *internal.EvaluationContext {
	if ctx == nil {
		return nil
	}
	return &internal.EvaluationContext{
		UserID:            ctx.UserID,
		Email:             ctx.Email,
		Name:              ctx.Name,
		Anonymous:         ctx.Anonymous,
		Country:           ctx.Country,
		DeviceType:        ctx.DeviceType,
		OS:                ctx.OS,
		Browser:           ctx.Browser,
		Custom:            ctx.Custom,
		PrivateAttributes: ctx.PrivateAttributes,
	}
}
