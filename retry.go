package flagkit

import (
	"time"

	"github.com/flagkit/flagkit-go/internal/http"
)

// RetryConfig contains retry configuration.
type RetryConfig struct {
	MaxAttempts       int
	BaseDelay         time.Duration
	MaxDelay          time.Duration
	BackoffMultiplier float64
	Jitter            time.Duration
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() *RetryConfig {
	cfg := http.DefaultRetryConfig()
	return &RetryConfig{
		MaxAttempts:       cfg.MaxAttempts,
		BaseDelay:         cfg.BaseDelay,
		MaxDelay:          cfg.MaxDelay,
		BackoffMultiplier: cfg.BackoffMultiplier,
		Jitter:            cfg.Jitter,
	}
}

// calculateBackoff calculates the backoff delay for a retry attempt.
func calculateBackoff(attempt int, config *RetryConfig) time.Duration {
	return http.CalculateBackoff(attempt, &http.RetryConfig{
		MaxAttempts:       config.MaxAttempts,
		BaseDelay:         config.BaseDelay,
		MaxDelay:          config.MaxDelay,
		BackoffMultiplier: config.BackoffMultiplier,
		Jitter:            config.Jitter,
	})
}

// WithRetry executes a function with retry logic.
func WithRetry[T any](fn func() (T, error), config *RetryConfig) (T, error) {
	var httpConfig *http.RetryConfig
	if config != nil {
		httpConfig = &http.RetryConfig{
			MaxAttempts:       config.MaxAttempts,
			BaseDelay:         config.BaseDelay,
			MaxDelay:          config.MaxDelay,
			BackoffMultiplier: config.BackoffMultiplier,
			Jitter:            config.Jitter,
		}
	}
	return http.WithRetry(fn, httpConfig)
}
