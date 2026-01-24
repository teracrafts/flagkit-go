package internal

import (
	"math"
	"math/rand"
	"time"
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
	return &RetryConfig{
		MaxAttempts:       3,
		BaseDelay:         time.Second,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            100 * time.Millisecond,
	}
}

// CalculateBackoff calculates the backoff delay for a retry attempt.
func CalculateBackoff(attempt int, config *RetryConfig) time.Duration {
	// Exponential backoff: baseDelay * (multiplier ^ (attempt - 1))
	exponentialDelay := float64(config.BaseDelay) * math.Pow(config.BackoffMultiplier, float64(attempt-1))

	// Cap at maxDelay
	delay := time.Duration(math.Min(exponentialDelay, float64(config.MaxDelay)))

	// Add jitter
	jitter := time.Duration(rand.Float64() * float64(config.Jitter))

	return delay + jitter
}

// WithRetry executes a function with retry logic.
func WithRetry[T any](fn func() (T, error), config *RetryConfig) (T, error) {
	var zero T
	var lastErr error

	if config == nil {
		config = DefaultRetryConfig()
	}

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return zero, err
		}

		// Don't wait after the last attempt
		if attempt >= config.MaxAttempts {
			break
		}

		delay := CalculateBackoff(attempt, config)
		time.Sleep(delay)
	}

	return zero, lastErr
}

// RecoverableError is an interface for errors that can be checked for recoverability.
type RecoverableError interface {
	error
	IsRecoverable() bool
}

// isRetryableError checks if an error should be retried.
func isRetryableError(err error) bool {
	// Check internal FlagKitError
	if fkErr, ok := err.(*FlagKitError); ok {
		return fkErr.Recoverable
	}
	// Check for RecoverableError interface (for public package errors)
	if re, ok := err.(RecoverableError); ok {
		return re.IsRecoverable()
	}
	return false
}
