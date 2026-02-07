package flagkit

// This file exports internal types for testing and advanced usage.
// Users typically don't need to use these directly.

import (
	"time"

	"github.com/teracrafts/flagkit-go/internal/core"
	"github.com/teracrafts/flagkit-go/internal/http"
	"github.com/teracrafts/flagkit-go/internal/persistence"
	inttypes "github.com/teracrafts/flagkit-go/internal/types"
)

// Cache wraps core.Cache for public access
type Cache struct {
	*core.Cache
}

// CacheConfig is an alias for core.CacheConfig
type CacheConfig struct {
	TTL     time.Duration
	MaxSize int
	Logger  Logger
}

// NewCache creates a new cache.
func NewCache(config *CacheConfig) *Cache {
	return &Cache{
		Cache: core.NewCache(&core.CacheConfig{
			TTL:     config.TTL,
			MaxSize: config.MaxSize,
			Logger:  config.Logger,
		}),
	}
}

// Get retrieves a flag from the cache.
func (c *Cache) Get(key string) *FlagState {
	intFlag := c.Cache.Get(key)
	if intFlag == nil {
		return nil
	}
	return internalToPublicFlagState(intFlag)
}

// GetStale retrieves a flag from the cache even if expired.
func (c *Cache) GetStale(key string) *FlagState {
	intFlag := c.Cache.GetStale(key)
	if intFlag == nil {
		return nil
	}
	return internalToPublicFlagState(intFlag)
}

// Set stores a flag in the cache.
func (c *Cache) Set(key string, flag FlagState, ttl ...time.Duration) {
	c.Cache.Set(key, publicToInternalFlagState(flag), ttl...)
}

// SetMany stores multiple flags in the cache.
func (c *Cache) SetMany(flags []FlagState, ttl ...time.Duration) {
	internalFlags := make([]inttypes.FlagState, len(flags))
	for i, f := range flags {
		internalFlags[i] = publicToInternalFlagState(f)
	}
	c.Cache.SetMany(internalFlags, ttl...)
}

// DefaultCacheConfig returns the default cache configuration.
func DefaultCacheConfig() *CacheConfig {
	cfg := core.DefaultCacheConfig()
	return &CacheConfig{
		TTL:     cfg.TTL,
		MaxSize: cfg.MaxSize,
	}
}

// CircuitBreaker type aliases for testing
type CircuitBreaker = http.CircuitBreaker
type CircuitBreakerConfig = http.CircuitBreakerConfig
type CircuitState = http.CircuitState

const (
	CircuitClosed   = http.CircuitClosed
	CircuitOpen     = http.CircuitOpen
	CircuitHalfOpen = http.CircuitHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	return http.NewCircuitBreaker(config)
}

// DefaultCircuitBreakerConfig returns the default circuit breaker configuration.
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return http.DefaultCircuitBreakerConfig()
}

// Retry type aliases for testing
type RetryConfig = http.RetryConfig

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() *RetryConfig {
	return http.DefaultRetryConfig()
}

// CalculateBackoff calculates the backoff delay for a retry attempt.
func CalculateBackoff(attempt int, config *RetryConfig) time.Duration {
	return http.CalculateBackoff(attempt, config)
}

// WithRetry executes a function with retry logic.
func WithRetry[T any](fn func() (T, error), config *RetryConfig) (T, error) {
	return http.WithRetry(fn, config)
}

// Event type aliases for testing
type Event = core.Event

// EventQueue wraps core.EventQueue for public access
type EventQueue struct {
	*core.EventQueue
}

// EventQueueConfig is an alias for core.EventQueueConfig
type EventQueueConfig = core.EventQueueConfig

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
		EventQueue: core.NewEventQueue(&core.EventQueueOptions{
			SessionID:     opts.SessionID,
			EnvironmentID: opts.EnvironmentID,
			SDKVersion:    opts.SDKVersion,
			Logger:        opts.Logger,
			Config:        opts.Config,
		}),
	}
}

// TrackWithContext adds an event with context to the queue.
func (eq *EventQueue) TrackWithContext(eventType string, data map[string]any, ctx *EvaluationContext) {
	eq.EventQueue.TrackWithContext(eventType, data, publicToInternalContext(ctx))
}

// DefaultEventQueueConfig returns the default event queue configuration.
func DefaultEventQueueConfig() *EventQueueConfig {
	return core.DefaultEventQueueConfig()
}

// Polling type aliases for testing
type PollingManager = core.PollingManager
type PollingConfig = core.PollingConfig

// NewPollingManager creates a new polling manager.
func NewPollingManager(onPoll func(), config *PollingConfig, logger Logger) *PollingManager {
	return core.NewPollingManager(onPoll, config, logger)
}

// DefaultPollingConfig returns the default polling configuration.
func DefaultPollingConfig() *PollingConfig {
	return core.DefaultPollingConfig()
}

// HTTP client type aliases for advanced usage
type HTTPClient = http.HTTPClient
type HTTPClientConfig = http.HTTPClientConfig
type HTTPResponse = http.HTTPResponse

// NewHTTPClient creates a new HTTP client.
func NewHTTPClient(config *HTTPClientConfig) *HTTPClient {
	return http.NewHTTPClient(config)
}

// EventPersistence type aliases for testing
type EventPersistence = persistence.EventPersistence
type EventPersisterAdapter = persistence.EventPersisterAdapter
type PersistedEvent = persistence.PersistedEvent
type EventStatus = persistence.EventStatus
type EventPersistenceConfig = persistence.EventPersistenceConfig

// Event status constants
const (
	EventStatusPending = persistence.EventStatusPending
	EventStatusSending = persistence.EventStatusSending
	EventStatusSent    = persistence.EventStatusSent
	EventStatusFailed  = persistence.EventStatusFailed
)

// NewEventPersistence creates a new event persistence instance.
func NewEventPersistence(storagePath string, maxEvents int, flushInterval time.Duration, logger Logger) (*EventPersistence, error) {
	return persistence.NewEventPersistence(storagePath, maxEvents, flushInterval, logger)
}

// NewEventPersisterAdapter creates an adapter that implements the EventPersister interface.
func NewEventPersisterAdapter(ep *EventPersistence) *EventPersisterAdapter {
	return persistence.NewEventPersisterAdapter(ep)
}

// DefaultEventPersistenceConfig returns the default event persistence configuration.
func DefaultEventPersistenceConfig() *EventPersistenceConfig {
	return persistence.DefaultEventPersistenceConfig()
}

// GenerateEventID generates a unique event ID.
func GenerateEventID() string {
	return persistence.GenerateEventID()
}

// Helper functions for type conversion

func internalToPublicFlagState(f *inttypes.FlagState) *FlagState {
	return &FlagState{
		Key:          f.Key,
		Value:        f.Value,
		Enabled:      f.Enabled,
		Version:      f.Version,
		FlagType:     FlagType(f.FlagType),
		LastModified: f.LastModified,
	}
}

func publicToInternalFlagState(f FlagState) inttypes.FlagState {
	return inttypes.FlagState{
		Key:          f.Key,
		Value:        f.Value,
		Enabled:      f.Enabled,
		Version:      f.Version,
		FlagType:     inttypes.FlagType(f.FlagType),
		LastModified: f.LastModified,
	}
}

func publicToInternalContext(ctx *EvaluationContext) *inttypes.EvaluationContext {
	if ctx == nil {
		return nil
	}
	return &inttypes.EvaluationContext{
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
