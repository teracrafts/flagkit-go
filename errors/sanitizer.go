package errors

import (
	"regexp"
)

// ErrorSanitizationConfig configures error message sanitization to prevent information leakage.
type ErrorSanitizationConfig struct {
	// Enabled enables error message sanitization. Default: false.
	Enabled bool

	// PreserveOriginal keeps the original unsanitized message in the error's Details map
	// under the key "originalMessage". Useful for debugging while still sanitizing user-facing messages.
	PreserveOriginal bool
}

// sanitizationPattern represents a pattern to match and its replacement.
type sanitizationPattern struct {
	pattern     *regexp.Regexp
	replacement string
}

// sanitizationPatterns defines patterns for sensitive information that should be redacted.
var sanitizationPatterns = []sanitizationPattern{
	// Unix-style file paths (must come before email to avoid false positives)
	{regexp.MustCompile(`/(?:[\w.-]+/)+[\w.-]+`), "[PATH]"},
	// Windows-style file paths (supports paths with spaces)
	{regexp.MustCompile(`[A-Za-z]:\\(?:[^\\]+\\)+[^\\]*`), "[PATH]"},
	// IP addresses (IPv4)
	{regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`), "[IP]"},
	// FlagKit SDK API keys
	{regexp.MustCompile(`sdk_[a-zA-Z0-9_-]{8,}`), "sdk_[REDACTED]"},
	// FlagKit server API keys
	{regexp.MustCompile(`srv_[a-zA-Z0-9_-]{8,}`), "srv_[REDACTED]"},
	// FlagKit CLI API keys
	{regexp.MustCompile(`cli_[a-zA-Z0-9_-]{8,}`), "cli_[REDACTED]"},
	// Email addresses
	{regexp.MustCompile(`[\w.+-]+@[\w.-]+\.\w+`), "[EMAIL]"},
	// Database connection strings (postgres, mysql, mongodb, redis)
	{regexp.MustCompile(`(?i)(?:postgres|mysql|mongodb|redis)://[^\s]+`), "[CONNECTION_STRING]"},
}

// SanitizeErrorMessage removes sensitive information from an error message.
// If sanitization is disabled, the original message is returned unchanged.
func SanitizeErrorMessage(message string, config ErrorSanitizationConfig) string {
	if !config.Enabled {
		return message
	}

	result := message
	for _, sp := range sanitizationPatterns {
		result = sp.pattern.ReplaceAllString(result, sp.replacement)
	}

	return result
}

// defaultSanitizationConfig is the package-level sanitization configuration.
// It is set when a client is initialized with WithErrorSanitization.
var defaultSanitizationConfig = ErrorSanitizationConfig{
	Enabled:          false,
	PreserveOriginal: false,
}

// SetDefaultSanitizationConfig sets the default sanitization configuration.
// This is called internally when initializing the client.
func SetDefaultSanitizationConfig(config ErrorSanitizationConfig) {
	defaultSanitizationConfig = config
}

// GetDefaultSanitizationConfig returns the current default sanitization configuration.
func GetDefaultSanitizationConfig() ErrorSanitizationConfig {
	return defaultSanitizationConfig
}

// sanitizeMessage applies the default sanitization configuration to a message.
func sanitizeMessage(message string) string {
	return SanitizeErrorMessage(message, defaultSanitizationConfig)
}

// sanitizeCause sanitizes an error's message if it exists.
func sanitizeCause(cause error) error {
	if cause == nil {
		return nil
	}

	// If the cause is already a FlagKitError, it will be sanitized when created
	if _, ok := cause.(*FlagKitError); ok {
		return cause
	}

	// For other errors, wrap them in a sanitized error if sanitization is enabled
	if defaultSanitizationConfig.Enabled {
		return &sanitizedError{
			original:  cause,
			sanitized: sanitizeMessage(cause.Error()),
		}
	}

	return cause
}

// sanitizedError wraps an error with a sanitized message.
type sanitizedError struct {
	original  error
	sanitized string
}

func (e *sanitizedError) Error() string {
	return e.sanitized
}

func (e *sanitizedError) Unwrap() error {
	return e.original
}
