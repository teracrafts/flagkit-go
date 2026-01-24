// Package internal contains internal implementation details for the FlagKit SDK.
package internal

// Logger defines the interface for logging.
// This mirrors the public Logger interface to avoid import cycles.
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// FlagType represents the type of a flag value.
type FlagType string

const (
	FlagTypeBoolean FlagType = "boolean"
	FlagTypeString  FlagType = "string"
	FlagTypeNumber  FlagType = "number"
	FlagTypeJSON    FlagType = "json"
)

// FlagState represents the state of a feature flag.
// This mirrors the public FlagState to avoid import cycles.
type FlagState struct {
	Key          string      `json:"key"`
	Value        interface{} `json:"value"`
	Enabled      bool        `json:"enabled"`
	Version      int         `json:"version"`
	FlagType     FlagType    `json:"flagType"`
	LastModified string      `json:"lastModified"`
}

// ErrorCode represents a FlagKit error code.
type ErrorCode string

// Error codes (duplicated from public package to avoid import cycle)
const (
	ErrCircuitOpen       ErrorCode = "CIRCUIT_OPEN"
	ErrNetworkError      ErrorCode = "NETWORK_ERROR"
	ErrNetworkTimeout    ErrorCode = "NETWORK_TIMEOUT"
	ErrNetworkRetryLimit ErrorCode = "NETWORK_RETRY_LIMIT"
	ErrAuthUnauthorized  ErrorCode = "AUTH_UNAUTHORIZED"
	ErrAuthInvalidKey    ErrorCode = "AUTH_INVALID_KEY"
	ErrEvalFlagNotFound  ErrorCode = "EVAL_FLAG_NOT_FOUND"
)

// FlagKitError is the internal error type.
type FlagKitError struct {
	Code        ErrorCode
	Message     string
	Cause       error
	Recoverable bool
}

// Error implements the error interface.
func (e *FlagKitError) Error() string {
	if e.Cause != nil {
		return "[" + string(e.Code) + "] " + e.Message + ": " + e.Cause.Error()
	}
	return "[" + string(e.Code) + "] " + e.Message
}

// Unwrap returns the underlying cause.
func (e *FlagKitError) Unwrap() error {
	return e.Cause
}

// NewError creates a new FlagKitError.
func NewError(code ErrorCode, message string) *FlagKitError {
	return &FlagKitError{
		Code:        code,
		Message:     message,
		Recoverable: isRecoverableCode(code),
	}
}

// NewErrorWithCause creates a new FlagKitError with a cause.
func NewErrorWithCause(code ErrorCode, message string, cause error) *FlagKitError {
	return &FlagKitError{
		Code:        code,
		Message:     message,
		Cause:       cause,
		Recoverable: isRecoverableCode(code),
	}
}

// NetworkError creates a network error.
func NetworkError(code ErrorCode, message string, cause error) *FlagKitError {
	return NewErrorWithCause(code, message, cause)
}

// isRecoverableCode determines if an error code represents a recoverable error.
func isRecoverableCode(code ErrorCode) bool {
	switch code {
	case ErrNetworkError, ErrNetworkTimeout, ErrNetworkRetryLimit, ErrCircuitOpen:
		return true
	default:
		return false
	}
}

// EvaluationContext contains user and environment information for flag evaluation.
// This mirrors the public EvaluationContext to avoid import cycles.
type EvaluationContext struct {
	UserID            string                 `json:"userId,omitempty"`
	Email             string                 `json:"email,omitempty"`
	Name              string                 `json:"name,omitempty"`
	Anonymous         bool                   `json:"anonymous,omitempty"`
	Country           string                 `json:"country,omitempty"`
	DeviceType        string                 `json:"deviceType,omitempty"`
	OS                string                 `json:"os,omitempty"`
	Browser           string                 `json:"browser,omitempty"`
	Custom            map[string]interface{} `json:"custom,omitempty"`
	PrivateAttributes []string               `json:"privateAttributes,omitempty"`
}

// StripPrivateAttributes returns a copy of the context with private attributes removed.
func (c *EvaluationContext) StripPrivateAttributes() *EvaluationContext {
	stripped := &EvaluationContext{
		UserID:    c.UserID,
		Anonymous: c.Anonymous,
		Custom:    make(map[string]interface{}),
	}

	privateSet := make(map[string]bool)
	for _, attr := range c.PrivateAttributes {
		privateSet[attr] = true
	}

	if !privateSet["email"] {
		stripped.Email = c.Email
	}
	if !privateSet["name"] {
		stripped.Name = c.Name
	}
	if !privateSet["country"] {
		stripped.Country = c.Country
	}
	if !privateSet["deviceType"] {
		stripped.DeviceType = c.DeviceType
	}
	if !privateSet["os"] {
		stripped.OS = c.OS
	}
	if !privateSet["browser"] {
		stripped.Browser = c.Browser
	}

	for k, v := range c.Custom {
		if !privateSet[k] {
			stripped.Custom[k] = v
		}
	}

	return stripped
}

// ToMap converts the context to a map for serialization.
func (c *EvaluationContext) ToMap() map[string]interface{} {
	m := make(map[string]interface{})

	if c.UserID != "" {
		m["userId"] = c.UserID
	}
	if c.Email != "" {
		m["email"] = c.Email
	}
	if c.Name != "" {
		m["name"] = c.Name
	}
	if c.Anonymous {
		m["anonymous"] = c.Anonymous
	}
	if c.Country != "" {
		m["country"] = c.Country
	}
	if len(c.Custom) > 0 {
		m["custom"] = c.Custom
	}

	return m
}
