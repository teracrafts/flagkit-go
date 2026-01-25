package flagkit

import (
	"sync"
	"time"

	"github.com/flagkit/flagkit-go/internal"
)

func init() {
	// Set SDK version in internal http package
	internal.SDKVersion = SDKVersion
}

// Client is the FlagKit SDK client.
type Client struct {
	options        *Options
	cache          *internal.Cache
	httpClient     *internal.HTTPClient
	eventQueue     *internal.EventQueue
	pollingManager *internal.PollingManager
	context        *EvaluationContext
	sessionID      string
	lastUpdateTime string
	ready          bool
	closed         bool
	logger         Logger
	mu             sync.RWMutex
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
	cache := internal.NewCache(&internal.CacheConfig{
		TTL:     options.CacheTTL,
		MaxSize: 1000,
		Logger:  logger,
	})

	// Create HTTP client
	httpClient := internal.NewHTTPClient(&internal.HTTPClientConfig{
		APIKey:                 options.APIKey,
		SecondaryAPIKey:        options.SecondaryAPIKey,
		KeyRotationGracePeriod: options.KeyRotationGracePeriod,
		EnableRequestSigning:   options.EnableRequestSigning,
		Timeout:                options.Timeout,
		Retry: &internal.RetryConfig{
			MaxAttempts:       options.Retries,
			BaseDelay:         time.Second,
			MaxDelay:          30 * time.Second,
			BackoffMultiplier: 2.0,
			Jitter:            100 * time.Millisecond,
		},
		Logger:    logger,
		LocalPort: options.LocalPort,
	})

	// Create event queue
	eventQueue := internal.NewEventQueue(&internal.EventQueueOptions{
		HTTPClient: httpClient,
		SessionID:  sessionID,
		SDKVersion: SDKVersion,
		Logger:     logger,
	})

	client := &Client{
		options:    options,
		cache:      cache,
		httpClient: httpClient,
		eventQueue: eventQueue,
		sessionID:  sessionID,
		logger:     logger,
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

	// Convert to internal FlagState and store in cache
	internalFlags := make([]internal.FlagState, len(data.Flags))
	for i, f := range data.Flags {
		internalFlags[i] = internal.FlagState{
			Key:          f.Key,
			Value:        f.Value,
			Enabled:      f.Enabled,
			Version:      f.Version,
			FlagType:     internal.FlagType(f.FlagType),
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
func (c *Client) GetJSONValue(key string, defaultValue map[string]interface{}, ctx ...*EvaluationContext) map[string]interface{} {
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
func (c *Client) SetContext(ctx *EvaluationContext) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.context = ctx
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
func (c *Client) Identify(userID string, attributes ...map[string]interface{}) {
	ctx := NewContext(userID)
	if len(attributes) > 0 {
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

	c.eventQueue.Track("context.identified", map[string]interface{}{"userId": userID})
}

// Reset resets to anonymous user.
func (c *Client) Reset() {
	c.mu.Lock()
	c.context = NewAnonymousContext()
	c.mu.Unlock()

	c.eventQueue.Track("context.reset", nil)
}

// Track tracks a custom event.
func (c *Client) Track(eventType string, data ...map[string]interface{}) {
	var eventData map[string]interface{}
	if len(data) > 0 {
		eventData = data[0]
	}
	c.eventQueue.Track(eventType, eventData)
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

	// Close HTTP client
	c.httpClient.Close()

	c.logger.Info("SDK closed")
	return nil
}

// evaluate performs flag evaluation.
func (c *Client) evaluate(key string, defaultValue interface{}, ctx *EvaluationContext, expectedType FlagType) *EvaluationResult {
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
	for key, value := range c.options.Bootstrap {
		flag := internal.FlagState{
			Key:          key,
			Value:        value,
			Enabled:      true,
			Version:      0,
			FlagType:     internal.FlagType(InferFlagType(value)),
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

	c.pollingManager = internal.NewPollingManager(c.refresh, &internal.PollingConfig{
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
		internalFlags := make([]internal.FlagState, len(data.Flags))
		for i, f := range data.Flags {
			internalFlags[i] = internal.FlagState{
				Key:          f.Key,
				Value:        f.Value,
				Enabled:      f.Enabled,
				Version:      f.Version,
				FlagType:     internal.FlagType(f.FlagType),
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

// generateSessionID generates a random session ID.
func generateSessionID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
