package internal

import (
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // Normal operation
	CircuitOpen                         // Failing, reject requests
	CircuitHalfOpen                     // Testing if service recovered
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "CLOSED"
	case CircuitOpen:
		return "OPEN"
	case CircuitHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig contains circuit breaker configuration.
type CircuitBreakerConfig struct {
	FailureThreshold   int
	SuccessThreshold   int
	ResetTimeout       time.Duration
	HalfOpenMaxAllowed int
}

// DefaultCircuitBreakerConfig returns the default circuit breaker configuration.
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		FailureThreshold:   5,
		SuccessThreshold:   2,
		ResetTimeout:       30 * time.Second,
		HalfOpenMaxAllowed: 1,
	}
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	config             *CircuitBreakerConfig
	state              CircuitState
	failures           int
	successes          int
	lastFailureTime    time.Time
	halfOpenAllowed    int
	halfOpenInProgress int
	mu                 sync.Mutex
	logger             Logger
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}
	return &CircuitBreaker{
		config: config,
		state:  CircuitClosed,
	}
}

// Allow checks if a request should be allowed.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true

	case CircuitOpen:
		// Check if reset timeout has elapsed
		if time.Since(cb.lastFailureTime) >= cb.config.ResetTimeout {
			cb.transitionTo(CircuitHalfOpen)
			cb.halfOpenAllowed = cb.config.HalfOpenMaxAllowed
			cb.halfOpenInProgress = 0
		} else {
			return false
		}
		fallthrough

	case CircuitHalfOpen:
		if cb.halfOpenInProgress < cb.halfOpenAllowed {
			cb.halfOpenInProgress++
			return true
		}
		return false
	}

	return false
}

// RecordSuccess records a successful request.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitHalfOpen:
		cb.successes++
		cb.halfOpenInProgress--
		if cb.successes >= cb.config.SuccessThreshold {
			cb.transitionTo(CircuitClosed)
		}

	case CircuitClosed:
		// Reset failure count on success
		cb.failures = 0
	}
}

// RecordFailure records a failed request.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitClosed:
		cb.failures++
		if cb.failures >= cb.config.FailureThreshold {
			cb.transitionTo(CircuitOpen)
		}

	case CircuitHalfOpen:
		cb.halfOpenInProgress--
		cb.transitionTo(CircuitOpen)
	}
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitClosed
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenAllowed = 0
	cb.halfOpenInProgress = 0
}

// transitionTo transitions to a new state.
func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	oldState := cb.state
	cb.state = newState

	// Reset counters on state change
	cb.failures = 0
	cb.successes = 0

	if cb.logger != nil {
		cb.logger.Debug("Circuit breaker state change",
			"from", oldState.String(),
			"to", newState.String(),
		)
	}
}

// Stats returns circuit breaker statistics.
func (cb *CircuitBreaker) Stats() map[string]any {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return map[string]any{
		"state":                 cb.state.String(),
		"failures":              cb.failures,
		"successes":             cb.successes,
		"failure_threshold":     cb.config.FailureThreshold,
		"success_threshold":     cb.config.SuccessThreshold,
		"half_open_in_progress": cb.halfOpenInProgress,
	}
}
