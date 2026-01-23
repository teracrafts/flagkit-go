package flagkit

import (
	"time"

	"github.com/flagkit/flagkit-go/internal/http"
)

// HTTPClient handles HTTP communication with the FlagKit API.
type HTTPClient = http.Client

// HTTPClientConfig contains HTTP client configuration.
type HTTPClientConfig struct {
	APIKey         string
	Timeout        time.Duration
	Retry          *RetryConfig
	CircuitBreaker *CircuitBreakerConfig
	Logger         Logger
}

// HTTPResponse represents an HTTP response.
type HTTPResponse = http.Response

// NewHTTPClient creates a new HTTP client.
func NewHTTPClient(config *HTTPClientConfig) *HTTPClient {
	var retryConfig *http.RetryConfig
	if config.Retry != nil {
		retryConfig = &http.RetryConfig{
			MaxAttempts:       config.Retry.MaxAttempts,
			BaseDelay:         config.Retry.BaseDelay,
			MaxDelay:          config.Retry.MaxDelay,
			BackoffMultiplier: config.Retry.BackoffMultiplier,
			Jitter:            config.Retry.Jitter,
		}
	}

	var cbConfig *http.CircuitBreakerConfig
	if config.CircuitBreaker != nil {
		cbConfig = &http.CircuitBreakerConfig{
			FailureThreshold:   config.CircuitBreaker.FailureThreshold,
			SuccessThreshold:   config.CircuitBreaker.SuccessThreshold,
			ResetTimeout:       config.CircuitBreaker.ResetTimeout,
			HalfOpenMaxAllowed: config.CircuitBreaker.HalfOpenMaxAllowed,
		}
	}

	return http.NewClient(&http.ClientConfig{
		APIKey:         config.APIKey,
		Timeout:        config.Timeout,
		Retry:          retryConfig,
		CircuitBreaker: cbConfig,
		Logger:         config.Logger,
	})
}
