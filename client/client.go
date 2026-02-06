package client

import (
	"math/rand"
	"sync"
	"time"

	"github.com/flagkit/flagkit-go/config"
	"github.com/flagkit/flagkit-go/errors"
	"github.com/flagkit/flagkit-go/internal/core"
	"github.com/flagkit/flagkit-go/internal/http"
	"github.com/flagkit/flagkit-go/internal/persistence"
	inttypes "github.com/flagkit/flagkit-go/internal/types"
	"github.com/flagkit/flagkit-go/internal/version"
	"github.com/flagkit/flagkit-go/security"
	"github.com/flagkit/flagkit-go/types"
)

// Type aliases for convenience
type (
	Options              = config.Options
	OptionFunc           = config.OptionFunc
	EvaluationContext    = types.EvaluationContext
	EvaluationResult     = types.EvaluationResult
	FlagState            = types.FlagState
	FlagType             = types.FlagType
	Logger               = types.Logger
	EventPersistence     = persistence.EventPersistence
	EventPersisterAdapter = persistence.EventPersisterAdapter
)

// Function aliases
var (
	DefaultOptions              = config.DefaultOptions
	ParseInitResponse           = types.ParseInitResponse
	ParseUpdatesResponse        = types.ParseUpdatesResponse
	InferFlagType               = types.InferFlagType
	NewContext                  = types.NewContext
	NewAnonymousContext         = types.NewAnonymousContext
	NewDefaultLogger            = types.NewDefaultLogger
	NewEventPersistence         = persistence.NewEventPersistence
	NewEventPersisterAdapter    = persistence.NewEventPersisterAdapter
	ValidateLocalPort           = security.ValidateLocalPort
	CheckPIIWithStrictMode      = security.CheckPIIWithStrictMode
	VerifyBootstrapSignature    = security.VerifyBootstrapSignature
)

// Error function aliases
var (
	NewError          = errors.NewError
	NewErrorWithCause = errors.NewErrorWithCause
)

// Error code aliases
const (
	ErrInitFailed = errors.ErrInitFailed
)

// Config constant aliases
const (
	SDKVersion                      = config.SDKVersion
	DefaultMaxPersistedEvents       = config.DefaultMaxPersistedEvents
	DefaultPersistenceFlushInterval = config.DefaultPersistenceFlushInterval
)

// FlagType constant aliases
const (
	FlagTypeBoolean = types.FlagTypeBoolean
	FlagTypeString  = types.FlagTypeString
	FlagTypeNumber  = types.FlagTypeNumber
	FlagTypeJSON    = types.FlagTypeJSON
)

// EvaluationReason constant aliases
const (
	ReasonCached       = types.ReasonCached
	ReasonFallthrough  = types.ReasonFallthrough
	ReasonTargeted     = types.ReasonTargeted
	ReasonDefault      = types.ReasonDefault
	ReasonDisabled     = types.ReasonDisabled
	ReasonFlagNotFound = types.ReasonFlagNotFound
	ReasonError        = types.ReasonError
	ReasonStaleCache   = types.ReasonStaleCache
	ReasonBootstrap    = types.ReasonBootstrap
)

// NullLogger type alias
type NullLogger = types.NullLogger

// createDefaultResult is a helper function
func createDefaultResult(key string, defaultValue any, reason types.EvaluationReason) *types.EvaluationResult {
	return &types.EvaluationResult{
		FlagKey:   key,
		Value:     defaultValue,
		Enabled:   false,
		Reason:    reason,
		Version:   0,
		Timestamp: time.Now(),
	}
}

func init() {
	// Set SDK version in http package
	http.SDKVersion = SDKVersion
}

// Client is the FlagKit SDK client.
type Client struct {
	options          *Options
	cache            *core.Cache
	httpClient       *http.HTTPClient
	eventQueue       *core.EventQueue
	pollingManager   *core.PollingManager
	eventPersistence *EventPersistence
	context          *EvaluationContext
	sessionID        string
	lastUpdateTime   string
	ready            bool
	closed           bool
	logger           Logger
	mu               sync.RWMutex
}

// NewClient creates a new FlagKit client.
func NewClient(apiKey string, opts ...OptionFunc) (*Client, error) {
	options := DefaultOptions(apiKey)
	for _, opt := range opts {
		opt(options)
	}

	if err := options.Validate(); err != nil {
		return nil, err
	}

	// Validate LocalPort is not used in production
	if err := ValidateLocalPort(options.LocalPort); err != nil {
		return nil, err
	}

	// Set up logger
	var logger Logger
	if options.Logger != nil {
		logger = options.Logger
	} else if options.Debug {
		logger = NewDefaultLogger(true)
	} else {
		logger = &NullLogger{}
	}

	// Generate session ID
	sessionID := generateSessionID()

	// Create cache
	cache := core.NewCache(&core.CacheConfig{
		TTL:     options.CacheTTL,
		MaxSize: 1000,
		Logger:  logger,
	})

	// Create HTTP client
	httpClient := http.NewHTTPClient(&http.HTTPClientConfig{
		APIKey:                 options.APIKey,
		SecondaryAPIKey:        options.SecondaryAPIKey,
		KeyRotationGracePeriod: options.KeyRotationGracePeriod,
		EnableRequestSigning:   options.EnableRequestSigning,
		Timeout:                options.Timeout,
		Retry: &http.RetryConfig{
			MaxAttempts:       options.Retries,
			BaseDelay:         time.Second,
			MaxDelay:          30 * time.Second,
			BackoffMultiplier: 2.0,
			Jitter:            100 * time.Millisecond,
		},
		Logger:    logger,
		LocalPort: options.LocalPort,
	})

	// Create event persistence if enabled
	var eventPersistence *EventPersistence
	var persisterAdapter *EventPersisterAdapter
	if options.PersistEvents {
		storagePath := options.EventStoragePath
		maxEvents := options.MaxPersistedEvents
		if maxEvents <= 0 {
			maxEvents = DefaultMaxPersistedEvents
		}
		flushInterval := options.PersistenceFlushInterval
		if flushInterval <= 0 {
			flushInterval = DefaultPersistenceFlushInterval
		}

		var err error
		eventPersistence, err = NewEventPersistence(storagePath, maxEvents, flushInterval, logger)
		if err != nil {
			logger.Warn("Failed to create event persistence", "error", err.Error())
		} else {
			persisterAdapter = NewEventPersisterAdapter(eventPersistence)
			eventPersistence.Start()
		}
	}

	// Create event queue with persistence support
	eventQueueOpts := &core.EventQueueOptions{
		HTTPClient:     httpClient,
		SessionID:      sessionID,
		SDKVersion:     SDKVersion,
		Logger:         logger,
		PersistEnabled: options.PersistEvents && persisterAdapter != nil,
	}
	if persisterAdapter != nil {
		eventQueueOpts.Persister = persisterAdapter
	}
	eventQueue := core.NewEventQueue(eventQueueOpts)

	// Recover events if persistence is enabled
	if eventPersistence != nil {
		if err := eventQueue.RecoverEvents(); err != nil {
			logger.Warn("Failed to recover persisted events", "error", err.Error())
		}
	}

	client := &Client{
		options:          options,
		cache:            cache,
		httpClient:       httpClient,
		eventQueue:       eventQueue,
		eventPersistence: eventPersistence,
		sessionID:        sessionID,
		logger:           logger,
	}

	// Apply bootstrap values
	client.applyBootstrap()

	logger.Info("FlagKit client created",
		"offline", options.Offline,
	)

	return client, nil
}

// Initialize initializes the SDK by fetching flag configurations.
func (c *Client) Initialize() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return NewError(ErrInitFailed, "client is closed")
	}
	c.mu.Unlock()

	if c.options.Offline {
		c.logger.Info("Offline mode enabled, skipping initialization")
		c.setReady()
		return nil
	}

	c.logger.Debug("Initializing SDK")

	resp, err := c.httpClient.Get("/sdk/init")
	if err != nil {
		c.logger.Error("SDK initialization failed", "error", err.Error())
		if c.options.OnError != nil {
			c.options.OnError(err)
		}
		// Mark as ready anyway (will use cache/bootstrap/defaults)
		c.setReady()
		return err
	}

	data, err := ParseInitResponse(resp.Body)
	if err != nil {
		c.logger.Error("Failed to parse init response", "error", err.Error())
		c.setReady()
		return NewErrorWithCause(ErrInitFailed, "failed to parse init response", err)
	}

	// Set environment ID for event tracking
	c.eventQueue.SetEnvironmentID(data.EnvironmentID)

	// Check SDK version metadata and emit warnings
	c.checkVersionMetadata(data)

	// Convert to internal FlagState and store in cache
	internalFlags := make([]inttypes.FlagState, len(data.Flags))
	for i, f := range data.Flags {
		internalFlags[i] = inttypes.FlagState{
			Key:          f.Key,
			Value:        f.Value,
			Enabled:      f.Enabled,
			Version:      f.Version,
			FlagType:     inttypes.FlagType(f.FlagType),
			LastModified: f.LastModified,
		}
	}
	c.cache.SetMany(internalFlags, c.options.CacheTTL)
	c.lastUpdateTime = data.ServerTime

	// Start polling if enabled
	if c.options.EnablePolling {
		c.startPolling(time.Duration(data.PollingIntervalSeconds) * time.Second)
	}

	// Start event queue
	c.eventQueue.Start()

	c.setReady()

	c.logger.Info("SDK initialized",
		"flag_count", len(data.Flags),
		"environment", data.Environment,
	)

	return nil
}

// IsReady returns whether the SDK is ready.
func (c *Client) IsReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ready
}

// WaitForReady waits for the SDK to be ready.
func (c *Client) WaitForReady() {
	if c.IsReady() {
		return
	}
	_ = c.Initialize()
}

// GetBooleanValue evaluates a boolean flag.
func (c *Client) GetBooleanValue(key string, defaultValue bool, ctx ...*EvaluationContext) bool {
	result := c.evaluate(key, defaultValue, getContext(ctx), FlagTypeBoolean)
	return result.BoolValue()
}

// GetStringValue evaluates a string flag.
func (c *Client) GetStringValue(key string, defaultValue string, ctx ...*EvaluationContext) string {
	result := c.evaluate(key, defaultValue, getContext(ctx), FlagTypeString)
	return result.StringValue()
}

// GetNumberValue evaluates a number flag.
func (c *Client) GetNumberValue(key string, defaultValue float64, ctx ...*EvaluationContext) float64 {
	result := c.evaluate(key, defaultValue, getContext(ctx), FlagTypeNumber)
	return result.Float64Value()
}

// GetIntValue evaluates an integer flag.
func (c *Client) GetIntValue(key string, defaultValue int, ctx ...*EvaluationContext) int {
	result := c.evaluate(key, float64(defaultValue), getContext(ctx), FlagTypeNumber)
	return result.IntValue()
}

// GetJSONValue evaluates a JSON flag.
func (c *Client) GetJSONValue(key string, defaultValue map[string]any, ctx ...*EvaluationContext) map[string]any {
	result := c.evaluate(key, defaultValue, getContext(ctx), FlagTypeJSON)
	if v := result.JSONValue(); v != nil {
		return v
	}
	return defaultValue
}

// Evaluate evaluates a flag and returns the full result.
func (c *Client) Evaluate(key string, ctx ...*EvaluationContext) *EvaluationResult {
	return c.evaluate(key, nil, getContext(ctx), "")
}

// EvaluateAll evaluates all flags.
func (c *Client) EvaluateAll(ctx ...*EvaluationContext) map[string]*EvaluationResult {
	results := make(map[string]*EvaluationResult)
	for _, key := range c.GetAllFlagKeys() {
		results[key] = c.Evaluate(key, ctx...)
	}
	return results
}

// HasFlag checks if a flag exists.
func (c *Client) HasFlag(key string) bool {
	if c.cache.Has(key) {
		return true
	}
	_, ok := c.options.Bootstrap[key]
	return ok
}

// GetAllFlagKeys returns all flag keys.
func (c *Client) GetAllFlagKeys() []string {
	keys := make(map[string]bool)
	for _, k := range c.cache.GetAllKeys() {
		keys[k] = true
	}
	for k := range c.options.Bootstrap {
		keys[k] = true
	}

	result := make([]string, 0, len(keys))
	for k := range keys {
		result = append(result, k)
	}
	return result
}

// SetContext sets the global evaluation context.
// Returns an error if StrictPIIMode is enabled and PII is detected without privateAttributes.
func (c *Client) SetContext(ctx *EvaluationContext) error {
	if ctx != nil && len(ctx.PrivateAttributes) == 0 {
		// Check custom attributes for PII
		if ctx.Custom != nil {
			if err := CheckPIIWithStrictMode(ctx.Custom, "context", c.options.StrictPIIMode, c.logger); err != nil {
				return err
			}
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.context = ctx
	return nil
}

// GetContext returns the current global context.
func (c *Client) GetContext() *EvaluationContext {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.context
}

// ClearContext clears the global context.
func (c *Client) ClearContext() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.context = nil
}

// Identify identifies a user.
// Returns an error if StrictPIIMode is enabled and PII is detected in attributes.
func (c *Client) Identify(userID string, attributes ...map[string]any) error {
	ctx := NewContext(userID)
	if len(attributes) > 0 {
		// Security: check for potential PII in attributes
		if err := CheckPIIWithStrictMode(attributes[0], "context", c.options.StrictPIIMode, c.logger); err != nil {
			return err
		}

		for k, v := range attributes[0] {
			ctx.WithCustom(k, v)
		}
	}

	c.mu.Lock()
	if c.context != nil {
		c.context = c.context.Merge(ctx)
	} else {
		c.context = ctx
	}
	c.mu.Unlock()

	c.eventQueue.Track("context.identified", map[string]any{"userId": userID})
	return nil
}

// Reset resets to anonymous user.
func (c *Client) Reset() {
	c.mu.Lock()
	c.context = NewAnonymousContext()
	c.mu.Unlock()

	c.eventQueue.Track("context.reset", nil)
}

// Track tracks a custom event.
// Returns an error if StrictPIIMode is enabled and PII is detected in event data.
func (c *Client) Track(eventType string, data ...map[string]any) error {
	var eventData map[string]any
	if len(data) > 0 {
		eventData = data[0]

		// Security: check for potential PII in event data
		if err := CheckPIIWithStrictMode(eventData, "event", c.options.StrictPIIMode, c.logger); err != nil {
			return err
		}
	}
	c.eventQueue.Track(eventType, eventData)
	return nil
}

// Flush flushes pending events.
func (c *Client) Flush() {
	c.eventQueue.Flush()
}

// Refresh forces a refresh of flags from the server.
func (c *Client) Refresh() {
	if c.options.Offline || c.closed {
		return
	}

	c.refresh()
}

// Close closes the client and cleans up resources.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	c.logger.Debug("Closing SDK")

	// Stop polling
	if c.pollingManager != nil {
		c.pollingManager.Stop()
	}

	// Flush and stop events
	c.eventQueue.Stop()

	// Close event persistence
	if c.eventPersistence != nil {
		if err := c.eventPersistence.Close(); err != nil {
			c.logger.Warn("Failed to close event persistence", "error", err.Error())
		}
	}

	// Close HTTP client
	if err := c.httpClient.Close(); err != nil {
		c.logger.Warn("Failed to close HTTP client", "error", err.Error())
	}

	c.logger.Info("SDK closed")
	return nil
}

// evaluate performs flag evaluation.
func (c *Client) evaluate(key string, defaultValue any, ctx *EvaluationContext, expectedType FlagType) *EvaluationResult {
	// Apply evaluation jitter if enabled (cache timing attack protection)
	if c.options.EvaluationJitter.Enabled {
		c.applyEvaluationJitter()
	}

	// Validate key
	if key == "" {
		c.logger.Warn("Invalid flag key", "key", key)
		return createDefaultResult(key, defaultValue, ReasonDefault)
	}

	// Try cache first
	if cached := c.cache.Get(key); cached != nil {
		// Type check if expected type provided
		if expectedType != "" && FlagType(cached.FlagType) != expectedType {
			c.logger.Warn("Flag type mismatch",
				"key", key,
				"expected", expectedType,
				"got", cached.FlagType,
			)
			return createDefaultResult(key, defaultValue, ReasonError)
		}

		return &EvaluationResult{
			FlagKey:   key,
			Value:     cached.Value,
			Enabled:   cached.Enabled,
			Reason:    ReasonCached,
			Version:   cached.Version,
			Timestamp: time.Now(),
		}
	}

	// Try stale cache
	if stale := c.cache.GetStale(key); stale != nil {
		c.logger.Debug("Using stale cached value", "key", key)
		return &EvaluationResult{
			FlagKey:   key,
			Value:     stale.Value,
			Enabled:   stale.Enabled,
			Reason:    ReasonStaleCache,
			Version:   stale.Version,
			Timestamp: time.Now(),
		}
	}

	// Try bootstrap
	if value, ok := c.options.Bootstrap[key]; ok {
		c.logger.Debug("Using bootstrap value", "key", key)
		return createDefaultResult(key, value, ReasonBootstrap)
	}

	// Return default
	c.logger.Debug("Flag not found, using default", "key", key)
	return createDefaultResult(key, defaultValue, ReasonFlagNotFound)
}

// applyBootstrap applies bootstrap values to cache.
func (c *Client) applyBootstrap() {
	var flags map[string]any

	// Check for signed bootstrap first (takes precedence)
	if c.options.BootstrapWithSignature != nil {
		bootstrap := c.options.BootstrapWithSignature

		// Verify signature if present
		if bootstrap.Signature != "" {
			valid, err := VerifyBootstrapSignature(*bootstrap, c.options.APIKey, c.options.BootstrapVerification)

			if !valid {
				// Handle verification failure based on OnFailure setting
				switch c.options.BootstrapVerification.OnFailure {
				case "error":
					c.logger.Error("Bootstrap signature verification failed", "error", err.Error())
					if c.options.OnError != nil {
						c.options.OnError(err)
					}
					// Don't apply bootstrap values
					return
				case "ignore":
					// Silently continue with bootstrap values
				default: // "warn" is the default
					c.logger.Warn("Bootstrap signature verification failed, using values anyway", "error", err.Error())
				}
			} else {
				c.logger.Debug("Bootstrap signature verified successfully")
			}
		}

		flags = bootstrap.Flags
	} else {
		// Fall back to legacy bootstrap format
		flags = c.options.Bootstrap
	}

	// Apply the flags to cache
	for key, value := range flags {
		flag := inttypes.FlagState{
			Key:          key,
			Value:        value,
			Enabled:      true,
			Version:      0,
			FlagType:     inttypes.FlagType(InferFlagType(value)),
			LastModified: time.Now().UTC().Format(time.RFC3339),
		}
		// Bootstrap values don't expire (use very long TTL)
		c.cache.Set(key, flag, 365*24*time.Hour)
	}
}

// startPolling starts background polling.
func (c *Client) startPolling(interval time.Duration) {
	if c.pollingManager != nil {
		return
	}

	if interval < c.options.PollingInterval {
		interval = c.options.PollingInterval
	}

	c.pollingManager = core.NewPollingManager(c.refresh, &core.PollingConfig{
		Interval:          interval,
		Jitter:            time.Second,
		BackoffMultiplier: 2.0,
		MaxInterval:       5 * time.Minute,
	}, c.logger)

	c.pollingManager.Start()
}

// refresh refreshes flags from the server.
func (c *Client) refresh() {
	since := c.lastUpdateTime
	if since == "" {
		since = time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	}

	resp, err := c.httpClient.Get("/sdk/updates?since=" + since)
	if err != nil {
		c.logger.Warn("Failed to refresh flags", "error", err.Error())
		if c.pollingManager != nil {
			c.pollingManager.OnError()
		}
		return
	}

	data, err := ParseUpdatesResponse(resp.Body)
	if err != nil {
		c.logger.Warn("Failed to parse updates response", "error", err.Error())
		return
	}

	if len(data.Flags) > 0 {
		// Convert to internal FlagState
		internalFlags := make([]inttypes.FlagState, len(data.Flags))
		for i, f := range data.Flags {
			internalFlags[i] = inttypes.FlagState{
				Key:          f.Key,
				Value:        f.Value,
				Enabled:      f.Enabled,
				Version:      f.Version,
				FlagType:     inttypes.FlagType(f.FlagType),
				LastModified: f.LastModified,
			}
		}
		c.cache.SetMany(internalFlags)
		c.lastUpdateTime = data.CheckedAt

		c.logger.Debug("Flags refreshed", "count", len(data.Flags))

		if c.options.OnUpdate != nil {
			c.options.OnUpdate(data.Flags)
		}
	}

	if c.pollingManager != nil {
		c.pollingManager.OnSuccess()
	}
}

// setReady marks the client as ready.
func (c *Client) setReady() {
	c.mu.Lock()
	c.ready = true
	c.mu.Unlock()

	if c.options.OnReady != nil {
		c.options.OnReady()
	}
}

// getContext extracts context from variadic parameter.
func getContext(ctx []*EvaluationContext) *EvaluationContext {
	if len(ctx) > 0 {
		return ctx[0]
	}
	return nil
}

// applyEvaluationJitter applies a random delay for cache timing attack protection.
func (c *Client) applyEvaluationJitter() {
	minMs := c.options.EvaluationJitter.MinMs
	maxMs := c.options.EvaluationJitter.MaxMs

	// Ensure valid range
	if minMs < 0 {
		minMs = 0
	}
	if maxMs < minMs {
		maxMs = minMs
	}

	// Calculate jitter: min + rand.Intn(max-min+1)
	jitterMs := minMs
	if maxMs > minMs {
		jitterMs = minMs + rand.Intn(maxMs-minMs+1)
	}

	if jitterMs > 0 {
		time.Sleep(time.Duration(jitterMs) * time.Millisecond)
	}
}

// checkVersionMetadata checks SDK version metadata from init response and emits appropriate warnings.
//
// Per spec, the SDK should parse and surface:
//   - sdkVersionMin: Minimum required version (older may not work)
//   - sdkVersionRecommended: Recommended version for optimal experience
//   - sdkVersionLatest: Latest available version
//   - deprecationWarning: Server-provided deprecation message
func (c *Client) checkVersionMetadata(data *types.InitResponse) {
	if data.Metadata == nil {
		return
	}

	metadata := data.Metadata

	// Check for server-provided deprecation warning first
	if metadata.DeprecationWarning != "" {
		c.logger.Warn("[FlagKit] Deprecation Warning: " + metadata.DeprecationWarning)
	}

	// Check minimum version requirement
	if metadata.SDKVersionMin != "" && version.IsLessThan(SDKVersion, metadata.SDKVersionMin) {
		c.logger.Error("[FlagKit] SDK version " + SDKVersion + " is below minimum required version " + metadata.SDKVersionMin + ". " +
			"Some features may not work correctly. Please upgrade the SDK.")
	}

	// Check recommended version
	warnedAboutRecommended := false
	if metadata.SDKVersionRecommended != "" && version.IsLessThan(SDKVersion, metadata.SDKVersionRecommended) {
		c.logger.Warn("[FlagKit] SDK version " + SDKVersion + " is below recommended version " + metadata.SDKVersionRecommended + ". " +
			"Consider upgrading for the best experience.")
		warnedAboutRecommended = true
	}

	// Log if a newer version is available (info level, not a warning)
	// Only log if we haven't already warned about recommended
	if metadata.SDKVersionLatest != "" && version.IsLessThan(SDKVersion, metadata.SDKVersionLatest) && !warnedAboutRecommended {
		c.logger.Info("[FlagKit] SDK version " + SDKVersion + " - a newer version " + metadata.SDKVersionLatest + " is available.")
	}
}

// generateSessionID generates a random session ID.
func generateSessionID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
