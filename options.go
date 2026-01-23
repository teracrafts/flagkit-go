package flagkit

import (
	"time"
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

	// Offline mode disables network requests.
	Offline bool

	// Timeout is the HTTP request timeout.
	Timeout time.Duration

	// Retries is the number of retry attempts for failed requests.
	Retries int

	// Bootstrap provides initial flag values.
	Bootstrap map[string]interface{}

	// Debug enables debug logging.
	Debug bool

	// IsLocal enables local development mode (uses localhost:8200).
	IsLocal bool

	// Logger is a custom logger implementation.
	Logger Logger

	// OnReady is called when the SDK is ready.
	OnReady func()

	// OnError is called when an error occurs.
	OnError func(error)

	// OnUpdate is called when flags are updated.
	OnUpdate func([]FlagState)
}

// DefaultOptions returns options with default values.
func DefaultOptions(apiKey string) *Options {
	return &Options{
		APIKey:          apiKey,
		BaseURL:         DefaultBaseURL,
		PollingInterval: DefaultPollingInterval,
		EnablePolling:   true,
		CacheEnabled:    true,
		CacheTTL:        DefaultCacheTTL,
		Offline:         false,
		Timeout:         DefaultTimeout,
		Retries:         DefaultRetries,
		Bootstrap:       make(map[string]interface{}),
		Debug:           false,
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
func WithBootstrap(values map[string]interface{}) OptionFunc {
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

// WithIsLocal enables local development mode (uses localhost:8200).
func WithIsLocal() OptionFunc {
	return func(o *Options) {
		o.IsLocal = true
	}
}
