package config

import (
	"time"

	"github.com/flagkit/flagkit-go/errors"
	"github.com/flagkit/flagkit-go/types"
)

// Type aliases for convenience
type Logger = types.Logger
type FlagState = types.FlagState
type ErrorSanitizationConfig = errors.ErrorSanitizationConfig
type NullLogger = types.NullLogger

// UsageMetrics contains usage metrics extracted from API response headers.
type UsageMetrics struct {
	// ApiUsagePercent is the percentage of API call limit used this period (0-150+).
	ApiUsagePercent float64
	// EvaluationUsagePercent is the percentage of evaluation limit used (0-150+).
	EvaluationUsagePercent float64
	// RateLimitWarning indicates whether approaching rate limit threshold.
	RateLimitWarning bool
	// SubscriptionStatus is the current subscription status.
	// Valid values: "active", "trial", "past_due", "suspended", "cancelled"
	SubscriptionStatus string
}

// Error function aliases
var (
	NewError = errors.NewError
)

// Error code aliases
const (
	ErrConfigMissingRequired = errors.ErrConfigMissingRequired
	ErrConfigInvalidInterval = errors.ErrConfigInvalidInterval
	ErrAuthInvalidKey        = errors.ErrAuthInvalidKey
)

const (
	// DefaultBaseURL is the default FlagKit API base URL.
	DefaultBaseURL = "https://api.flagkit.dev/api/v1"

	// DefaultPollingInterval is the default polling interval.
	DefaultPollingInterval = 30 * time.Second

	// DefaultCacheTTL is the default cache TTL.
	DefaultCacheTTL = 5 * time.Minute

	// DefaultTimeout is the default HTTP timeout.
	DefaultTimeout = 5 * time.Second

	// DefaultRetries is the default number of retries.
	DefaultRetries = 3

	// SDKVersion is the current SDK version.
	SDKVersion = "1.0.0"
)

// Options configures the FlagKit client.
type Options struct {
	// APIKey is the API key for authentication (required).
	APIKey string

	// SecondaryAPIKey is a secondary API key for key rotation.
	// When the primary key receives a 401 error, the SDK will automatically
	// fail over to use this secondary key.
	SecondaryAPIKey string

	// KeyRotationGracePeriod is the duration to track key rotation state.
	// Default: 5 minutes.
	KeyRotationGracePeriod time.Duration

	// BaseURL is the FlagKit API base URL.
	BaseURL string

	// PollingInterval is the interval between flag updates.
	PollingInterval time.Duration

	// EnablePolling enables background polling for flag updates.
	EnablePolling bool

	// CacheEnabled enables local caching of flag values.
	CacheEnabled bool

	// CacheTTL is the time-to-live for cached values.
	CacheTTL time.Duration

	// EnableCacheEncryption enables AES-256-GCM encryption for cached data.
	// The encryption key is derived from the API key using PBKDF2.
	EnableCacheEncryption bool

	// Offline mode disables network requests.
	Offline bool

	// Timeout is the HTTP request timeout.
	Timeout time.Duration

	// Retries is the number of retry attempts for failed requests.
	Retries int

	// Bootstrap provides initial flag values (legacy format).
	Bootstrap map[string]any

	// BootstrapWithSignature provides signed bootstrap flag values.
	// If set, this takes precedence over Bootstrap.
	BootstrapWithSignature *BootstrapConfig

	// BootstrapVerification configures bootstrap signature verification.
	BootstrapVerification BootstrapVerificationConfig

	// Debug enables debug logging.
	Debug bool

	// LocalPort specifies a local port for development mode (0 means not set/production).
	LocalPort int

	// StrictPIIMode when enabled returns a SecurityError instead of warning
	// when PII is detected in context/events without proper PrivateAttributes.
	StrictPIIMode bool

	// EnableRequestSigning enables HMAC-SHA256 signing for POST requests.
	// Default: true.
	EnableRequestSigning bool

	// Logger is a custom logger implementation.
	Logger Logger

	// OnReady is called when the SDK is ready.
	OnReady func()

	// OnError is called when an error occurs.
	OnError func(error)

	// OnUpdate is called when flags are updated.
	OnUpdate func([]FlagState)

	// OnUsageUpdate is called when usage metrics are received from API responses.
	// Provides visibility into API usage, rate limits, and subscription status.
	OnUsageUpdate func(*UsageMetrics)

	// OnSubscriptionError is called when a subscription error occurs (e.g., suspended).
	// This allows applications to notify users of subscription issues.
	OnSubscriptionError func(message string)

	// OnConnectionLimitError is called when the streaming connection limit is reached.
	// Applications can use this to close other connections or implement backoff.
	OnConnectionLimitError func()

	// PersistEvents enables crash-resilient event persistence.
	// When enabled, events are written to disk before being queued for sending.
	PersistEvents bool

	// EventStoragePath is the directory for event storage files.
	// Defaults to OS temp directory if not specified.
	EventStoragePath string

	// MaxPersistedEvents is the maximum number of events to persist.
	// Default: 10000.
	MaxPersistedEvents int

	// PersistenceFlushInterval is the interval between disk writes.
	// Default: 1 second.
	PersistenceFlushInterval time.Duration

	// EvaluationJitter configures timing jitter for flag evaluations.
	// This provides protection against cache timing attacks.
	EvaluationJitter EvaluationJitterConfig

	// ErrorSanitization configures error message sanitization to prevent information leakage.
	ErrorSanitization ErrorSanitizationConfig
}

// EvaluationJitterConfig configures timing jitter for cache timing attack protection.
type EvaluationJitterConfig struct {
	// Enabled enables evaluation jitter. Default: false.
	Enabled bool

	// MinMs is the minimum jitter delay in milliseconds. Default: 5.
	MinMs int

	// MaxMs is the maximum jitter delay in milliseconds. Default: 15.
	MaxMs int
}

// BootstrapConfig represents bootstrap flag values with optional HMAC signature verification.
type BootstrapConfig struct {
	// Flags is the map of flag keys to their values.
	Flags map[string]any `json:"flags"`

	// Signature is the HMAC-SHA256 signature of the canonicalized flags JSON.
	// Optional: if empty, no verification is performed.
	Signature string `json:"signature,omitempty"`

	// Timestamp is the Unix timestamp (milliseconds) when the bootstrap was generated.
	// Used for staleness checking when signature verification is enabled.
	Timestamp int64 `json:"timestamp,omitempty"`
}

// BootstrapVerificationConfig configures bootstrap signature verification behavior.
type BootstrapVerificationConfig struct {
	// Enabled enables signature verification for bootstrap values. Default: true.
	Enabled bool

	// MaxAge is the maximum age of bootstrap data. Default: 24 hours.
	// Bootstrap data older than this will be rejected if verification is enabled.
	MaxAge time.Duration

	// OnFailure specifies the behavior when verification fails.
	// Valid values: "warn", "error", "ignore". Default: "warn".
	// - "warn": Log a warning but continue using bootstrap values
	// - "error": Return an error and don't use bootstrap values
	// - "ignore": Silently ignore verification failures
	OnFailure string
}

// DefaultKeyRotationGracePeriod is the default grace period for key rotation.
const DefaultKeyRotationGracePeriod = 5 * time.Minute

// DefaultMaxPersistedEvents is the default maximum number of persisted events.
const DefaultMaxPersistedEvents = 10000

// DefaultPersistenceFlushInterval is the default interval between persistence disk writes.
const DefaultPersistenceFlushInterval = time.Second

// Default evaluation jitter values for cache timing attack protection.
const (
	DefaultEvaluationJitterMinMs = 5
	DefaultEvaluationJitterMaxMs = 15
)

// Default bootstrap verification values.
const (
	DefaultBootstrapMaxAge    = 24 * time.Hour
	DefaultBootstrapOnFailure = "warn"
)

// DefaultOptions returns options with default values.
func DefaultOptions(apiKey string) *Options {
	return &Options{
		APIKey:                 apiKey,
		BaseURL:                DefaultBaseURL,
		PollingInterval:        DefaultPollingInterval,
		EnablePolling:          true,
		CacheEnabled:           true,
		CacheTTL:               DefaultCacheTTL,
		Offline:                false,
		Timeout:                DefaultTimeout,
		Retries:                DefaultRetries,
		Bootstrap:              make(map[string]any),
		Debug:                  false,
		KeyRotationGracePeriod: DefaultKeyRotationGracePeriod,
		EnableRequestSigning:   true,
		EvaluationJitter: EvaluationJitterConfig{
			Enabled: false,
			MinMs:   DefaultEvaluationJitterMinMs,
			MaxMs:   DefaultEvaluationJitterMaxMs,
		},
		BootstrapVerification: BootstrapVerificationConfig{
			Enabled:   true,
			MaxAge:    DefaultBootstrapMaxAge,
			OnFailure: DefaultBootstrapOnFailure,
		},
	}
}

// Validate validates the options.
func (o *Options) Validate() error {
	if o.APIKey == "" {
		return NewError(ErrConfigMissingRequired, "API key is required")
	}

	if len(o.APIKey) < 10 {
		return NewError(ErrAuthInvalidKey, "API key is too short")
	}

	// Validate secondary API key if provided
	if o.SecondaryAPIKey != "" && len(o.SecondaryAPIKey) < 10 {
		return NewError(ErrAuthInvalidKey, "secondary API key is too short")
	}

	if o.BaseURL == "" {
		o.BaseURL = DefaultBaseURL
	}

	if o.PollingInterval < time.Second {
		return NewError(ErrConfigInvalidInterval, "Polling interval must be at least 1 second")
	}

	if o.Timeout <= 0 {
		o.Timeout = DefaultTimeout
	}

	if o.Retries < 0 {
		o.Retries = 0
	}

	if o.CacheTTL <= 0 {
		o.CacheTTL = DefaultCacheTTL
	}

	if o.KeyRotationGracePeriod <= 0 {
		o.KeyRotationGracePeriod = DefaultKeyRotationGracePeriod
	}

	return nil
}

// OptionFunc is a function that modifies Options.
type OptionFunc func(*Options)

// WithBaseURL sets the base URL.
func WithBaseURL(url string) OptionFunc {
	return func(o *Options) {
		o.BaseURL = url
	}
}

// WithPollingInterval sets the polling interval.
func WithPollingInterval(d time.Duration) OptionFunc {
	return func(o *Options) {
		o.PollingInterval = d
	}
}

// WithPollingDisabled disables polling.
func WithPollingDisabled() OptionFunc {
	return func(o *Options) {
		o.EnablePolling = false
	}
}

// WithCacheTTL sets the cache TTL.
func WithCacheTTL(d time.Duration) OptionFunc {
	return func(o *Options) {
		o.CacheTTL = d
	}
}

// WithCacheDisabled disables caching.
func WithCacheDisabled() OptionFunc {
	return func(o *Options) {
		o.CacheEnabled = false
	}
}

// WithOffline enables offline mode.
func WithOffline() OptionFunc {
	return func(o *Options) {
		o.Offline = true
	}
}

// WithTimeout sets the HTTP timeout.
func WithTimeout(d time.Duration) OptionFunc {
	return func(o *Options) {
		o.Timeout = d
	}
}

// WithRetries sets the number of retries.
func WithRetries(n int) OptionFunc {
	return func(o *Options) {
		o.Retries = n
	}
}

// WithBootstrap sets bootstrap values.
func WithBootstrap(values map[string]any) OptionFunc {
	return func(o *Options) {
		o.Bootstrap = values
	}
}

// WithDebug enables debug logging.
func WithDebug() OptionFunc {
	return func(o *Options) {
		o.Debug = true
	}
}

// WithLogger sets a custom logger.
func WithLogger(logger Logger) OptionFunc {
	return func(o *Options) {
		o.Logger = logger
	}
}

// WithOnReady sets the ready callback.
func WithOnReady(fn func()) OptionFunc {
	return func(o *Options) {
		o.OnReady = fn
	}
}

// WithOnError sets the error callback.
func WithOnError(fn func(error)) OptionFunc {
	return func(o *Options) {
		o.OnError = fn
	}
}

// WithOnUpdate sets the update callback.
func WithOnUpdate(fn func([]FlagState)) OptionFunc {
	return func(o *Options) {
		o.OnUpdate = fn
	}
}

// WithOnUsageUpdate sets the callback for usage metrics updates.
// This callback is invoked when usage metrics are received from API responses,
// providing visibility into API usage, rate limits, and subscription status.
func WithOnUsageUpdate(fn func(*UsageMetrics)) OptionFunc {
	return func(o *Options) {
		o.OnUsageUpdate = fn
	}
}

// WithOnSubscriptionError sets the callback for subscription errors.
// This callback is invoked when the subscription is suspended or has other issues.
func WithOnSubscriptionError(fn func(message string)) OptionFunc {
	return func(o *Options) {
		o.OnSubscriptionError = fn
	}
}

// WithOnConnectionLimitError sets the callback for connection limit errors.
// This callback is invoked when the streaming connection limit is reached.
func WithOnConnectionLimitError(fn func()) OptionFunc {
	return func(o *Options) {
		o.OnConnectionLimitError = fn
	}
}

// WithLocalPort enables local development mode with the specified port.
func WithLocalPort(port int) OptionFunc {
	return func(o *Options) {
		o.LocalPort = port
	}
}

// WithSecondaryAPIKey sets a secondary API key for key rotation.
func WithSecondaryAPIKey(key string) OptionFunc {
	return func(o *Options) {
		o.SecondaryAPIKey = key
	}
}

// WithKeyRotationGracePeriod sets the key rotation grace period.
func WithKeyRotationGracePeriod(d time.Duration) OptionFunc {
	return func(o *Options) {
		o.KeyRotationGracePeriod = d
	}
}

// WithStrictPIIMode enables strict PII detection mode.
// When enabled, PII detection returns errors instead of warnings.
func WithStrictPIIMode() OptionFunc {
	return func(o *Options) {
		o.StrictPIIMode = true
	}
}

// WithRequestSigning enables or disables HMAC-SHA256 request signing.
func WithRequestSigning(enabled bool) OptionFunc {
	return func(o *Options) {
		o.EnableRequestSigning = enabled
	}
}

// WithCacheEncryption enables AES-256-GCM encryption for cached data.
func WithCacheEncryption() OptionFunc {
	return func(o *Options) {
		o.EnableCacheEncryption = true
	}
}

// WithPersistEvents enables crash-resilient event persistence.
// When enabled, events are written to disk before being queued for sending.
func WithPersistEvents(enabled bool) OptionFunc {
	return func(o *Options) {
		o.PersistEvents = enabled
	}
}

// WithEventStoragePath sets the directory for event storage files.
func WithEventStoragePath(path string) OptionFunc {
	return func(o *Options) {
		o.EventStoragePath = path
	}
}

// WithMaxPersistedEvents sets the maximum number of events to persist.
func WithMaxPersistedEvents(max int) OptionFunc {
	return func(o *Options) {
		o.MaxPersistedEvents = max
	}
}

// WithPersistenceFlushInterval sets the interval between disk writes for event persistence.
func WithPersistenceFlushInterval(interval time.Duration) OptionFunc {
	return func(o *Options) {
		o.PersistenceFlushInterval = interval
	}
}

// WithEvaluationJitter configures evaluation jitter for cache timing attack protection.
// When enabled, a random delay between minMs and maxMs is added at the start of each flag evaluation.
func WithEvaluationJitter(enabled bool, minMs, maxMs int) OptionFunc {
	return func(o *Options) {
		o.EvaluationJitter = EvaluationJitterConfig{
			Enabled: enabled,
			MinMs:   minMs,
			MaxMs:   maxMs,
		}
	}
}

// WithBootstrapVerification configures bootstrap signature verification.
// When enabled, bootstrap data with signatures will be verified using HMAC-SHA256.
// Parameters:
//   - enabled: whether to perform signature verification (default: true)
//   - maxAge: maximum age of bootstrap data (default: 24 hours)
//   - onFailure: behavior when verification fails - "warn", "error", or "ignore" (default: "warn")
func WithBootstrapVerification(enabled bool, maxAge time.Duration, onFailure string) OptionFunc {
	return func(o *Options) {
		o.BootstrapVerification = BootstrapVerificationConfig{
			Enabled:   enabled,
			MaxAge:    maxAge,
			OnFailure: onFailure,
		}
	}
}

// WithSignedBootstrap sets bootstrap values with HMAC signature verification.
// This takes precedence over WithBootstrap if both are set.
func WithSignedBootstrap(config *BootstrapConfig) OptionFunc {
	return func(o *Options) {
		o.BootstrapWithSignature = config
	}
}

// WithErrorSanitization enables error message sanitization to prevent information leakage.
// When enabled, sensitive information like file paths, IP addresses, API keys, and
// connection strings are redacted from error messages.
func WithErrorSanitization(enabled bool) OptionFunc {
	return func(o *Options) {
		o.ErrorSanitization.Enabled = enabled
	}
}

// WithErrorSanitizationConfig sets the full error sanitization configuration.
// Use this for more control over sanitization behavior, including preserving
// original messages for debugging.
func WithErrorSanitizationConfig(config ErrorSanitizationConfig) OptionFunc {
	return func(o *Options) {
		o.ErrorSanitization = config
	}
}
