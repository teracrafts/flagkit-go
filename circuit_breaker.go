package flagkit

import (
	"time"

	"github.com/flagkit/flagkit-go/internal/http"
)

// CircuitState represents the state of a circuit breaker.
type CircuitState = http.CircuitState

const (
	CircuitClosed   = http.CircuitClosed
	CircuitOpen     = http.CircuitOpen
	CircuitHalfOpen = http.CircuitHalfOpen
)

// CircuitBreakerConfig contains circuit breaker configuration.
type CircuitBreakerConfig struct {
	FailureThreshold   int
	SuccessThreshold   int
	ResetTimeout       time.Duration
	HalfOpenMaxAllowed int
}

// DefaultCircuitBreakerConfig returns the default circuit breaker configuration.
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	cfg := http.DefaultCircuitBreakerConfig()
	return &CircuitBreakerConfig{
		FailureThreshold:   cfg.FailureThreshold,
		SuccessThreshold:   cfg.SuccessThreshold,
		ResetTimeout:       cfg.ResetTimeout,
		HalfOpenMaxAllowed: cfg.HalfOpenMaxAllowed,
	}
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker = http.CircuitBreaker

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	var httpConfig *http.CircuitBreakerConfig
	if config != nil {
		httpConfig = &http.CircuitBreakerConfig{
			FailureThreshold:   config.FailureThreshold,
			SuccessThreshold:   config.SuccessThreshold,
			ResetTimeout:       config.ResetTimeout,
			HalfOpenMaxAllowed: config.HalfOpenMaxAllowed,
		}
	}
	return http.NewCircuitBreaker(httpConfig)
}
