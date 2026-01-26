package tests

import (
	"testing"
	"time"

	. "github.com/flagkit/flagkit-go"
	"github.com/stretchr/testify/assert"
)

func TestCalculateBackoff(t *testing.T) {
	config := &RetryConfig{
		BaseDelay:         time.Second,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            0, // No jitter for predictable tests
	}

	// First retry (attempt=1): 1s * 2^0 = 1s
	delay := CalculateBackoff(1, config)
	assert.Equal(t, time.Second, delay)

	// Second retry (attempt=2): 1s * 2^1 = 2s
	delay = CalculateBackoff(2, config)
	assert.Equal(t, 2*time.Second, delay)

	// Third retry (attempt=3): 1s * 2^2 = 4s
	delay = CalculateBackoff(3, config)
	assert.Equal(t, 4*time.Second, delay)

	// Should cap at max delay (attempt=10: 1s * 2^9 = 512s > 30s)
	delay = CalculateBackoff(10, config)
	assert.Equal(t, 30*time.Second, delay)
}

func TestCalculateBackoffWithJitter(t *testing.T) {
	config := &RetryConfig{
		BaseDelay:         time.Second,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            500 * time.Millisecond,
	}

	delay := CalculateBackoff(1, config)
	// Should be between 1s and 1.5s
	assert.GreaterOrEqual(t, delay, time.Second)
	assert.LessOrEqual(t, delay, time.Second+500*time.Millisecond)
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, time.Second, config.BaseDelay)
	assert.Equal(t, 30*time.Second, config.MaxDelay)
	assert.Equal(t, 2.0, config.BackoffMultiplier)
}

func TestWithRetrySuccess(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:       3,
		BaseDelay:         10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	attempts := 0
	result, err := WithRetry(func() (string, error) {
		attempts++
		return "success", nil
	}, config)

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 1, attempts)
}

func TestWithRetryEventualSuccess(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:       3,
		BaseDelay:         10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	attempts := 0
	result, err := WithRetry(func() (string, error) {
		attempts++
		if attempts < 3 {
			// Return a recoverable error
			return "", NewError(ErrNetworkError, "temporary error")
		}
		return "success", nil
	}, config)

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 3, attempts)
}

func TestWithRetryExhausted(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:       3,
		BaseDelay:         10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	attempts := 0
	_, err := WithRetry(func() (string, error) {
		attempts++
		return "", NewError(ErrNetworkError, "persistent error")
	}, config)

	assert.Error(t, err)
	assert.Equal(t, 3, attempts)
}

func TestWithRetryNonRetryableError(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:       3,
		BaseDelay:         10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	attempts := 0
	_, err := WithRetry(func() (string, error) {
		attempts++
		// Non-recoverable error should not be retried
		return "", NewError(ErrAuthInvalidKey, "invalid key")
	}, config)

	assert.Error(t, err)
	assert.Equal(t, 1, attempts) // Should only attempt once
}

func TestWithRetryNilConfig(t *testing.T) {
	attempts := 0
	result, err := WithRetry(func() (string, error) {
		attempts++
		return "success", nil
	}, nil)

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 1, attempts)
}
