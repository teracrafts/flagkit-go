package flagkit

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewError(t *testing.T) {
	err := NewError(ErrAuthInvalidKey, "invalid API key")

	assert.Equal(t, ErrAuthInvalidKey, err.Code)
	assert.Equal(t, "invalid API key", err.Message)
	assert.Nil(t, err.Cause)
	assert.False(t, err.Recoverable)
}

func TestNewErrorWithCause(t *testing.T) {
	cause := errors.New("underlying error")
	err := NewErrorWithCause(ErrNetworkTimeout, "request timed out", cause)

	assert.Equal(t, ErrNetworkTimeout, err.Code)
	assert.Equal(t, "request timed out", err.Message)
	assert.Equal(t, cause, err.Cause)
	assert.True(t, err.Recoverable)
}

func TestFlagKitErrorError(t *testing.T) {
	err := NewError(ErrAuthInvalidKey, "invalid API key")
	assert.Equal(t, "[AUTH_INVALID_KEY] invalid API key", err.Error())

	cause := errors.New("connection refused")
	errWithCause := NewErrorWithCause(ErrNetworkError, "network error", cause)
	assert.Equal(t, "[NETWORK_ERROR] network error: connection refused", errWithCause.Error())
}

func TestFlagKitErrorUnwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := NewErrorWithCause(ErrNetworkError, "network error", cause)

	assert.Equal(t, cause, err.Unwrap())
}

func TestIsRecoverable(t *testing.T) {
	tests := []struct {
		code        ErrorCode
		recoverable bool
	}{
		{ErrNetworkError, true},
		{ErrNetworkTimeout, true},
		{ErrNetworkRetryLimit, true},
		{ErrAuthInvalidKey, false},
		{ErrAuthExpiredKey, false},
		{ErrInitFailed, false},
		{ErrEvalInvalidKey, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			err := NewError(tt.code, "test error")
			assert.Equal(t, tt.recoverable, IsRecoverable(err))
		})
	}
}

func TestFlagKitErrorIs(t *testing.T) {
	err := NewError(ErrAuthInvalidKey, "invalid key")
	var fkErr *FlagKitError
	assert.True(t, errors.As(err, &fkErr))
}

func TestWithDetails(t *testing.T) {
	err := NewError(ErrEvalFlagNotFound, "flag not found").
		WithDetails(map[string]interface{}{
			"flagKey": "my-flag",
			"reason":  "not in cache",
		})

	assert.Equal(t, "my-flag", err.Details["flagKey"])
	assert.Equal(t, "not in cache", err.Details["reason"])
}
