// Package flagkit provides a Go SDK for FlagKit feature flag management.
//
// Quick Start:
//
//	// Initialize the SDK
//	client, err := flagkit.Initialize("sdk_your_api_key")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	// Evaluate flags
//	enabled := client.GetBooleanValue("my-feature", false)
//	variant := client.GetStringValue("button-text", "Click")
//
//	// Identify user
//	client.Identify("user-123", map[string]any{"plan": "premium"})
//
//	// Track events
//	client.Track("button_clicked", map[string]any{"button": "signup"})
package flagkit

import (
	"sync"

	"github.com/teracrafts/flagkit-go/client"
	"github.com/teracrafts/flagkit-go/config"
	"github.com/teracrafts/flagkit-go/errors"
	"github.com/teracrafts/flagkit-go/security"
	"github.com/teracrafts/flagkit-go/types"
)

// Re-export commonly used types for convenience
type (
	// Client is the FlagKit SDK client.
	Client = client.Client

	// OptionFunc is a function that modifies Options.
	OptionFunc = config.OptionFunc

	// Options configures the FlagKit client.
	Options = config.Options

	// BootstrapConfig represents bootstrap flag values with optional HMAC signature verification.
	BootstrapConfig = config.BootstrapConfig

	// BootstrapVerificationConfig configures bootstrap signature verification behavior.
	BootstrapVerificationConfig = config.BootstrapVerificationConfig

	// EvaluationContext contains user and environment information for flag evaluation.
	EvaluationContext = types.EvaluationContext

	// EvaluationResult represents the result of evaluating a flag.
	EvaluationResult = types.EvaluationResult

	// FlagState represents the state of a feature flag.
	FlagState = types.FlagState

	// FlagType represents the type of a flag value.
	FlagType = types.FlagType

	// Logger defines the interface for logging.
	Logger = types.Logger

	// NullLogger is a logger that discards all output.
	NullLogger = types.NullLogger

	// FlagKitError represents an SDK error.
	FlagKitError = errors.FlagKitError

	// SecurityConfig holds security configuration options.
	SecurityConfig = security.SecurityConfig

	// PIIDetectionResult contains the result of PII detection.
	PIIDetectionResult = security.PIIDetectionResult

	// SignedPayload represents a payload with HMAC-SHA256 signature.
	SignedPayload = security.SignedPayload

	// RequestSignature contains signature information for request headers.
	RequestSignature = security.RequestSignature

	// BootstrapVerificationResult contains the result of bootstrap verification.
	BootstrapVerificationResult = security.BootstrapVerificationResult
)

// Re-export commonly used functions
var (
	// NewClient creates a new FlagKit client.
	NewClient = client.NewClient

	// DefaultOptions returns options with default values.
	DefaultOptions = config.DefaultOptions

	// NewContext creates a new EvaluationContext with the given user ID.
	NewContext = types.NewContext

	// NewAnonymousContext creates a new anonymous EvaluationContext.
	NewAnonymousContext = types.NewAnonymousContext

	// NewDefaultLogger creates a new default logger.
	NewDefaultLogger = types.NewDefaultLogger
)

// Re-export error types and functions
var (
	NewError          = errors.NewError
	NewErrorWithCause = errors.NewErrorWithCause
)

// Re-export error codes
const (
	ErrInitFailed                    = errors.ErrInitFailed
	ErrInitTimeout                   = errors.ErrInitTimeout
	ErrInitAlreadyInitialized        = errors.ErrInitAlreadyInitialized
	ErrInitNotInitialized            = errors.ErrInitNotInitialized
	ErrSecurityPIIDetected           = errors.ErrSecurityPIIDetected
	ErrSecurityLocalPortInProduction = errors.ErrSecurityLocalPortInProduction
	ErrSecuritySignatureInvalid      = errors.ErrSecuritySignatureInvalid
	ErrNetworkError                  = errors.ErrNetworkError
	ErrAuthInvalidKey                = errors.ErrAuthInvalidKey
)

// Re-export flag types
const (
	FlagTypeBoolean = types.FlagTypeBoolean
	FlagTypeString  = types.FlagTypeString
	FlagTypeNumber  = types.FlagTypeNumber
	FlagTypeJSON    = types.FlagTypeJSON
)

// Re-export evaluation jitter defaults
const (
	DefaultEvaluationJitterMinMs = config.DefaultEvaluationJitterMinMs
	DefaultEvaluationJitterMaxMs = config.DefaultEvaluationJitterMaxMs
)

// Re-export option functions
var (
	WithBaseURL               = config.WithBaseURL
	WithPollingInterval       = config.WithPollingInterval
	WithPollingDisabled       = config.WithPollingDisabled
	WithCacheTTL              = config.WithCacheTTL
	WithCacheDisabled         = config.WithCacheDisabled
	WithOffline               = config.WithOffline
	WithTimeout               = config.WithTimeout
	WithRetries               = config.WithRetries
	WithBootstrap             = config.WithBootstrap
	WithDebug                 = config.WithDebug
	WithLogger                = config.WithLogger
	WithOnReady               = config.WithOnReady
	WithOnError               = config.WithOnError
	WithOnUpdate              = config.WithOnUpdate
	WithLocalPort             = config.WithLocalPort
	WithSecondaryAPIKey       = config.WithSecondaryAPIKey
	WithStrictPIIMode         = config.WithStrictPIIMode
	WithRequestSigning        = config.WithRequestSigning
	WithCacheEncryption          = config.WithCacheEncryption
	WithPersistEvents            = config.WithPersistEvents
	WithEventStoragePath         = config.WithEventStoragePath
	WithMaxPersistedEvents       = config.WithMaxPersistedEvents
	WithPersistenceFlushInterval = config.WithPersistenceFlushInterval
	WithEvaluationJitter         = config.WithEvaluationJitter
	WithBootstrapVerification = config.WithBootstrapVerification
	WithSignedBootstrap       = config.WithSignedBootstrap
	WithErrorSanitization     = config.WithErrorSanitization
)

// Re-export security functions
var (
	CanonicalizeObject             = security.CanonicalizeObject
	CreateBootstrapSignature       = security.CreateBootstrapSignature
	VerifyBootstrapSignature       = security.VerifyBootstrapSignature
	IsPotentialPIIField            = security.IsPotentialPIIField
	DetectPotentialPII             = security.DetectPotentialPII
	WarnIfPotentialPII             = security.WarnIfPotentialPII
	IsServerKey                    = security.IsServerKey
	IsClientKey                    = security.IsClientKey
	DefaultSecurityConfig          = security.DefaultSecurityConfig
	CheckForPotentialPII           = security.CheckForPotentialPII
	CheckPIIWithStrictMode         = security.CheckPIIWithStrictMode
	IsProductionEnvironment        = security.IsProductionEnvironment
	ValidateLocalPort              = security.ValidateLocalPort
	GetKeyID                       = security.GetKeyID
	GenerateHMACSHA256             = security.GenerateHMACSHA256
	CreateRequestSignature         = security.CreateRequestSignature
	VerifyRequestSignature         = security.VerifyRequestSignature
	SignPayload                    = security.SignPayload
	VerifySignedPayload            = security.VerifySignedPayload
)

var (
	instance   *Client
	instanceMu sync.Mutex
)

// Initialize creates and initializes a singleton FlagKit client.
// This is the recommended way to use FlagKit in most applications.
func Initialize(apiKey string, opts ...OptionFunc) (*Client, error) {
	instanceMu.Lock()
	defer instanceMu.Unlock()

	if instance != nil {
		return nil, NewError(ErrInitAlreadyInitialized, "FlagKit is already initialized")
	}

	client, err := NewClient(apiKey, opts...)
	if err != nil {
		return nil, err
	}

	if err := client.Initialize(); err != nil {
		return nil, err
	}

	instance = client
	return instance, nil
}

// GetClient returns the singleton client instance.
// Returns nil if not initialized.
func GetClient() *Client {
	instanceMu.Lock()
	defer instanceMu.Unlock()
	return instance
}

// IsInitialized returns whether the SDK has been initialized.
func IsInitialized() bool {
	instanceMu.Lock()
	defer instanceMu.Unlock()
	return instance != nil
}

// Shutdown closes the singleton client and resets the instance.
func Shutdown() error {
	instanceMu.Lock()
	defer instanceMu.Unlock()

	if instance == nil {
		return nil
	}

	err := instance.Close()
	instance = nil
	return err
}

// Convenience methods that operate on the singleton instance.
// These will panic if the SDK is not initialized.

// GetBooleanValue evaluates a boolean flag using the singleton client.
func GetBooleanValue(key string, defaultValue bool) bool {
	return mustGetClient().GetBooleanValue(key, defaultValue)
}

// GetStringValue evaluates a string flag using the singleton client.
func GetStringValue(key string, defaultValue string) string {
	return mustGetClient().GetStringValue(key, defaultValue)
}

// GetNumberValue evaluates a number flag using the singleton client.
func GetNumberValue(key string, defaultValue float64) float64 {
	return mustGetClient().GetNumberValue(key, defaultValue)
}

// GetIntValue evaluates an integer flag using the singleton client.
func GetIntValue(key string, defaultValue int) int {
	return mustGetClient().GetIntValue(key, defaultValue)
}

// GetJSONValue evaluates a JSON flag using the singleton client.
func GetJSONValue(key string, defaultValue map[string]any) map[string]any {
	return mustGetClient().GetJSONValue(key, defaultValue)
}

// Evaluate evaluates a flag and returns the full result using the singleton client.
func Evaluate(key string) *EvaluationResult {
	return mustGetClient().Evaluate(key)
}

// HasFlag checks if a flag exists using the singleton client.
func HasFlag(key string) bool {
	return mustGetClient().HasFlag(key)
}

// Identify identifies a user using the singleton client.
func Identify(userID string, attributes ...map[string]any) {
	_ = mustGetClient().Identify(userID, attributes...)
}

// Reset resets to anonymous user using the singleton client.
func Reset() {
	mustGetClient().Reset()
}

// Track tracks a custom event using the singleton client.
func Track(eventType string, data ...map[string]any) {
	_ = mustGetClient().Track(eventType, data...)
}

// Flush flushes pending events using the singleton client.
func Flush() {
	mustGetClient().Flush()
}

// mustGetClient returns the singleton client or panics if not initialized.
func mustGetClient() *Client {
	client := GetClient()
	if client == nil {
		panic("FlagKit is not initialized. Call flagkit.Initialize() first.")
	}
	return client
}
