package flagkit

import (
	"github.com/flagkit/flagkit-go/internal/errors"
)

// ErrorCode represents a FlagKit error code.
type ErrorCode = errors.ErrorCode

// Error codes
const (
	// Initialization errors
	ErrInitFailed             = errors.ErrInitFailed
	ErrInitTimeout            = errors.ErrInitTimeout
	ErrInitAlreadyInitialized = errors.ErrInitAlreadyInitialized
	ErrInitNotInitialized     = errors.ErrInitNotInitialized

	// Authentication errors
	ErrAuthInvalidKey   = errors.ErrAuthInvalidKey
	ErrAuthExpiredKey   = errors.ErrAuthExpiredKey
	ErrAuthMissingKey   = errors.ErrAuthMissingKey
	ErrAuthUnauthorized = errors.ErrAuthUnauthorized

	// Network errors
	ErrNetworkError      = errors.ErrNetworkError
	ErrNetworkTimeout    = errors.ErrNetworkTimeout
	ErrNetworkRetryLimit = errors.ErrNetworkRetryLimit

	// Evaluation errors
	ErrEvalFlagNotFound  = errors.ErrEvalFlagNotFound
	ErrEvalTypeMismatch  = errors.ErrEvalTypeMismatch
	ErrEvalInvalidKey    = errors.ErrEvalInvalidKey
	ErrEvalInvalidValue  = errors.ErrEvalInvalidValue
	ErrEvalDisabled      = errors.ErrEvalDisabled
	ErrEvalError         = errors.ErrEvalError
	ErrEvalContextError  = errors.ErrEvalContextError
	ErrEvalDefaultUsed   = errors.ErrEvalDefaultUsed
	ErrEvalStaleValue    = errors.ErrEvalStaleValue
	ErrEvalCacheMiss     = errors.ErrEvalCacheMiss
	ErrEvalNetworkError  = errors.ErrEvalNetworkError
	ErrEvalParseError    = errors.ErrEvalParseError
	ErrEvalTimeoutError  = errors.ErrEvalTimeoutError

	// Cache errors
	ErrCacheReadError    = errors.ErrCacheReadError
	ErrCacheWriteError   = errors.ErrCacheWriteError
	ErrCacheInvalidData  = errors.ErrCacheInvalidData
	ErrCacheExpired      = errors.ErrCacheExpired
	ErrCacheStorageError = errors.ErrCacheStorageError

	// Event errors
	ErrEventQueueFull    = errors.ErrEventQueueFull
	ErrEventInvalidType  = errors.ErrEventInvalidType
	ErrEventInvalidData  = errors.ErrEventInvalidData
	ErrEventSendFailed   = errors.ErrEventSendFailed
	ErrEventFlushFailed  = errors.ErrEventFlushFailed
	ErrEventFlushTimeout = errors.ErrEventFlushTimeout

	// Circuit breaker errors
	ErrCircuitOpen = errors.ErrCircuitOpen

	// Configuration errors
	ErrConfigInvalidURL      = errors.ErrConfigInvalidURL
	ErrConfigInvalidInterval = errors.ErrConfigInvalidInterval
	ErrConfigMissingRequired = errors.ErrConfigMissingRequired
)

// FlagKitError is the base error type for all FlagKit errors.
type FlagKitError = errors.FlagKitError

// NewError creates a new FlagKitError.
func NewError(code ErrorCode, message string) *FlagKitError {
	return errors.NewError(code, message)
}

// NewErrorWithCause creates a new FlagKitError with a cause.
func NewErrorWithCause(code ErrorCode, message string, cause error) *FlagKitError {
	return errors.NewErrorWithCause(code, message, cause)
}

// IsRecoverable checks if the error is recoverable.
func IsRecoverable(err error) bool {
	return errors.IsRecoverable(err)
}

// InitializationError creates an initialization error.
func InitializationError(code ErrorCode, message string) *FlagKitError {
	return errors.InitializationError(code, message)
}

// AuthenticationError creates an authentication error.
func AuthenticationError(code ErrorCode, message string) *FlagKitError {
	return errors.AuthenticationError(code, message)
}

// NetworkError creates a network error.
func NetworkError(code ErrorCode, message string, cause error) *FlagKitError {
	return errors.NetworkError(code, message, cause)
}

// EvaluationError creates an evaluation error.
func EvaluationError(code ErrorCode, message string) *FlagKitError {
	return errors.EvaluationError(code, message)
}
