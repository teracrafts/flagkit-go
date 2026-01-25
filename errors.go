package flagkit

import (
	"fmt"
)

// ErrorCode represents a FlagKit error code.
type ErrorCode string

// Error codes
const (
	// Initialization errors
	ErrInitFailed             ErrorCode = "INIT_FAILED"
	ErrInitTimeout            ErrorCode = "INIT_TIMEOUT"
	ErrInitAlreadyInitialized ErrorCode = "INIT_ALREADY_INITIALIZED"
	ErrInitNotInitialized     ErrorCode = "INIT_NOT_INITIALIZED"

	// Authentication errors
	ErrAuthInvalidKey   ErrorCode = "AUTH_INVALID_KEY"
	ErrAuthExpiredKey   ErrorCode = "AUTH_EXPIRED_KEY"
	ErrAuthMissingKey   ErrorCode = "AUTH_MISSING_KEY"
	ErrAuthUnauthorized ErrorCode = "AUTH_UNAUTHORIZED"

	// Network errors
	ErrNetworkError      ErrorCode = "NETWORK_ERROR"
	ErrNetworkTimeout    ErrorCode = "NETWORK_TIMEOUT"
	ErrNetworkRetryLimit ErrorCode = "NETWORK_RETRY_LIMIT"

	// Evaluation errors
	ErrEvalFlagNotFound  ErrorCode = "EVAL_FLAG_NOT_FOUND"
	ErrEvalTypeMismatch  ErrorCode = "EVAL_TYPE_MISMATCH"
	ErrEvalInvalidKey    ErrorCode = "EVAL_INVALID_KEY"
	ErrEvalInvalidValue  ErrorCode = "EVAL_INVALID_VALUE"
	ErrEvalDisabled      ErrorCode = "EVAL_DISABLED"
	ErrEvalError         ErrorCode = "EVAL_ERROR"
	ErrEvalContextError  ErrorCode = "EVAL_CONTEXT_ERROR"
	ErrEvalDefaultUsed   ErrorCode = "EVAL_DEFAULT_USED"
	ErrEvalStaleValue    ErrorCode = "EVAL_STALE_VALUE"
	ErrEvalCacheMiss     ErrorCode = "EVAL_CACHE_MISS"
	ErrEvalNetworkError  ErrorCode = "EVAL_NETWORK_ERROR"
	ErrEvalParseError    ErrorCode = "EVAL_PARSE_ERROR"
	ErrEvalTimeoutError  ErrorCode = "EVAL_TIMEOUT_ERROR"

	// Cache errors
	ErrCacheReadError    ErrorCode = "CACHE_READ_ERROR"
	ErrCacheWriteError   ErrorCode = "CACHE_WRITE_ERROR"
	ErrCacheInvalidData  ErrorCode = "CACHE_INVALID_DATA"
	ErrCacheExpired      ErrorCode = "CACHE_EXPIRED"
	ErrCacheStorageError ErrorCode = "CACHE_STORAGE_ERROR"

	// Event errors
	ErrEventQueueFull    ErrorCode = "EVENT_QUEUE_FULL"
	ErrEventInvalidType  ErrorCode = "EVENT_INVALID_TYPE"
	ErrEventInvalidData  ErrorCode = "EVENT_INVALID_DATA"
	ErrEventSendFailed   ErrorCode = "EVENT_SEND_FAILED"
	ErrEventFlushFailed  ErrorCode = "EVENT_FLUSH_FAILED"
	ErrEventFlushTimeout ErrorCode = "EVENT_FLUSH_TIMEOUT"

	// Circuit breaker errors
	ErrCircuitOpen ErrorCode = "CIRCUIT_OPEN"

	// Configuration errors
	ErrConfigInvalidURL      ErrorCode = "CONFIG_INVALID_URL"
	ErrConfigInvalidInterval ErrorCode = "CONFIG_INVALID_INTERVAL"
	ErrConfigMissingRequired ErrorCode = "CONFIG_MISSING_REQUIRED"

	// Security errors
	ErrSecurityLocalPortInProduction ErrorCode = "SECURITY_LOCAL_PORT_IN_PRODUCTION"
	ErrSecurityPIIDetected           ErrorCode = "SECURITY_PII_DETECTED"
	ErrSecuritySignatureInvalid      ErrorCode = "SECURITY_SIGNATURE_INVALID"
	ErrSecurityEncryptionFailed      ErrorCode = "SECURITY_ENCRYPTION_FAILED"
	ErrSecurityDecryptionFailed      ErrorCode = "SECURITY_DECRYPTION_FAILED"
)

// FlagKitError is the base error type for all FlagKit errors.
type FlagKitError struct {
	Code        ErrorCode
	Message     string
	Cause       error
	Recoverable bool
	Details     map[string]interface{}
}

// Error implements the error interface.
func (e *FlagKitError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause.
func (e *FlagKitError) Unwrap() error {
	return e.Cause
}

// IsRecoverable returns whether this error is recoverable.
// This method implements the internal.RecoverableError interface.
func (e *FlagKitError) IsRecoverable() bool {
	return e.Recoverable
}

// NewError creates a new FlagKitError.
func NewError(code ErrorCode, message string) *FlagKitError {
	return &FlagKitError{
		Code:        code,
		Message:     message,
		Recoverable: isRecoverable(code),
		Details:     make(map[string]interface{}),
	}
}

// NewErrorWithCause creates a new FlagKitError with a cause.
func NewErrorWithCause(code ErrorCode, message string, cause error) *FlagKitError {
	return &FlagKitError{
		Code:        code,
		Message:     message,
		Cause:       cause,
		Recoverable: isRecoverable(code),
		Details:     make(map[string]interface{}),
	}
}

// WithDetails adds details to the error.
func (e *FlagKitError) WithDetails(details map[string]interface{}) *FlagKitError {
	for k, v := range details {
		e.Details[k] = v
	}
	return e
}

// isRecoverable determines if an error code represents a recoverable error.
func isRecoverable(code ErrorCode) bool {
	switch code {
	case ErrNetworkError, ErrNetworkTimeout, ErrNetworkRetryLimit,
		ErrCircuitOpen, ErrCacheExpired, ErrEvalStaleValue,
		ErrEvalCacheMiss, ErrEvalNetworkError, ErrEventSendFailed:
		return true
	default:
		return false
	}
}

// IsRecoverable checks if the error is recoverable.
func IsRecoverable(err error) bool {
	if fkErr, ok := err.(*FlagKitError); ok {
		return fkErr.Recoverable
	}
	return false
}

// InitializationError creates an initialization error.
func InitializationError(code ErrorCode, message string) *FlagKitError {
	return NewError(code, message)
}

// AuthenticationError creates an authentication error.
func AuthenticationError(code ErrorCode, message string) *FlagKitError {
	return NewError(code, message)
}

// NetworkError creates a network error.
func NetworkError(code ErrorCode, message string, cause error) *FlagKitError {
	return NewErrorWithCause(code, message, cause)
}

// EvaluationError creates an evaluation error.
func EvaluationError(code ErrorCode, message string) *FlagKitError {
	return NewError(code, message)
}

// SecurityError creates a security error.
func SecurityError(code ErrorCode, message string) *FlagKitError {
	err := NewError(code, message)
	err.Recoverable = false
	return err
}
