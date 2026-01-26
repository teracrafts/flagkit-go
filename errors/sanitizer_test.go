package errors

import (
	"errors"
	"testing"
)

func TestSanitizeErrorMessage_Disabled(t *testing.T) {
	config := ErrorSanitizationConfig{
		Enabled: false,
	}

	message := "Error at /home/user/app/config.json with IP 192.168.1.100"
	result := SanitizeErrorMessage(message, config)

	if result != message {
		t.Errorf("Expected message to be unchanged when disabled, got %s", result)
	}
}

func TestSanitizeErrorMessage_UnixPaths(t *testing.T) {
	config := ErrorSanitizationConfig{
		Enabled: true,
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple unix path",
			input:    "Failed to read /home/user/config.json",
			expected: "Failed to read [PATH]",
		},
		{
			name:     "nested unix path",
			input:    "Error in /var/lib/app/data/cache/flags.json",
			expected: "Error in [PATH]",
		},
		{
			name:     "multiple unix paths",
			input:    "Copy from /src/file.txt to /dst/file.txt failed",
			expected: "Copy from [PATH] to [PATH] failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorMessage(tt.input, config)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSanitizeErrorMessage_WindowsPaths(t *testing.T) {
	config := ErrorSanitizationConfig{
		Enabled: true,
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple windows path",
			input:    `Failed to read C:\Users\Admin\config.json`,
			expected: "Failed to read [PATH]",
		},
		{
			name:     "nested windows path",
			input:    `Error in D:\Program Files\App\data\cache.txt`,
			expected: "Error in [PATH]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorMessage(tt.input, config)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSanitizeErrorMessage_IPAddresses(t *testing.T) {
	config := ErrorSanitizationConfig{
		Enabled: true,
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "localhost IP",
			input:    "Connection to 127.0.0.1 failed",
			expected: "Connection to [IP] failed",
		},
		{
			name:     "private IP",
			input:    "Server 192.168.1.100 not responding",
			expected: "Server [IP] not responding",
		},
		{
			name:     "public IP",
			input:    "Request to 8.8.8.8 timed out",
			expected: "Request to [IP] timed out",
		},
		{
			name:     "multiple IPs",
			input:    "Proxy 10.0.0.1 forwarding to 172.16.0.1",
			expected: "Proxy [IP] forwarding to [IP]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorMessage(tt.input, config)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSanitizeErrorMessage_APIKeys(t *testing.T) {
	config := ErrorSanitizationConfig{
		Enabled: true,
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "SDK key",
			input:    "Invalid key: sdk_abc123def456ghi789",
			expected: "Invalid key: sdk_[REDACTED]",
		},
		{
			name:     "server key",
			input:    "Auth failed for srv_secretkey12345678",
			expected: "Auth failed for srv_[REDACTED]",
		},
		{
			name:     "CLI key",
			input:    "CLI token cli_mytoken_abcd1234 expired",
			expected: "CLI token cli_[REDACTED] expired",
		},
		{
			name:     "multiple keys",
			input:    "Primary sdk_key123456789 and backup srv_key987654321",
			expected: "Primary sdk_[REDACTED] and backup srv_[REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorMessage(tt.input, config)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSanitizeErrorMessage_Emails(t *testing.T) {
	config := ErrorSanitizationConfig{
		Enabled: true,
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple email",
			input:    "User user@example.com not found",
			expected: "User [EMAIL] not found",
		},
		{
			name:     "email with subdomain",
			input:    "Contact admin@mail.company.org for help",
			expected: "Contact [EMAIL] for help",
		},
		{
			name:     "email with dots in local part",
			input:    "Invalid email john.doe@example.com",
			expected: "Invalid email [EMAIL]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorMessage(tt.input, config)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSanitizeErrorMessage_ConnectionStrings(t *testing.T) {
	config := ErrorSanitizationConfig{
		Enabled: true,
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "postgres connection",
			input:    "Failed to connect: postgres://user:pass@localhost:5432/db",
			expected: "Failed to connect: [CONNECTION_STRING]",
		},
		{
			name:     "mysql connection",
			input:    "MySQL error: mysql://root:secret@db.example.com/mydb",
			expected: "MySQL error: [CONNECTION_STRING]",
		},
		{
			name:     "mongodb connection",
			input:    "MongoDB: mongodb://admin:password123@cluster.mongodb.net/test",
			expected: "MongoDB: [CONNECTION_STRING]",
		},
		{
			name:     "redis connection",
			input:    "Redis unavailable: redis://default:mypassword@redis.example.com:6379",
			expected: "Redis unavailable: [CONNECTION_STRING]",
		},
		{
			name:     "case insensitive",
			input:    "Error: POSTGRES://user:pass@host/db",
			expected: "Error: [CONNECTION_STRING]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorMessage(tt.input, config)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSanitizeErrorMessage_MultiplePatterns(t *testing.T) {
	config := ErrorSanitizationConfig{
		Enabled: true,
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "path and IP",
			input:    "Error reading /etc/config.json from 192.168.1.1",
			expected: "Error reading [PATH] from [IP]",
		},
		{
			name:     "key and email",
			input:    "Key sdk_myapikey12345 belongs to user@example.com",
			expected: "Key sdk_[REDACTED] belongs to [EMAIL]",
		},
		{
			name:     "all patterns",
			input:    "Error at /app/config.json connecting to 10.0.0.1 with sdk_key123456789 for user@test.com via postgres://user:pass@db/app",
			expected: "Error at [PATH] connecting to [IP] with sdk_[REDACTED] for [EMAIL] via [CONNECTION_STRING]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorMessage(tt.input, config)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSanitizeErrorMessage_EdgeCases(t *testing.T) {
	config := ErrorSanitizationConfig{
		Enabled: true,
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no sensitive data",
			input:    "Simple error message",
			expected: "Simple error message",
		},
		{
			name:     "short key prefix (not matched)",
			input:    "Key sdk_abc not valid",
			expected: "Key sdk_abc not valid",
		},
		{
			name:     "invalid IP format (still matched by regex)",
			input:    "Value 999.999.999.999 is invalid",
			expected: "Value [IP] is invalid",
		},
		{
			name:     "single path segment (not matched)",
			input:    "File /config not found",
			expected: "File /config not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorMessage(tt.input, config)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestNewError_WithSanitization(t *testing.T) {
	// Enable sanitization
	SetDefaultSanitizationConfig(ErrorSanitizationConfig{
		Enabled:          true,
		PreserveOriginal: true,
	})
	defer SetDefaultSanitizationConfig(ErrorSanitizationConfig{Enabled: false})

	err := NewError(ErrNetworkError, "Failed to connect to 192.168.1.100")

	if err.Message != "Failed to connect to [IP]" {
		t.Errorf("Expected sanitized message, got %q", err.Message)
	}

	original, ok := err.Details["originalMessage"]
	if !ok {
		t.Error("Expected originalMessage in Details when PreserveOriginal is true")
	}
	if original != "Failed to connect to 192.168.1.100" {
		t.Errorf("Expected original message in Details, got %q", original)
	}
}

func TestNewError_WithSanitizationNoPreserve(t *testing.T) {
	// Enable sanitization without preserve
	SetDefaultSanitizationConfig(ErrorSanitizationConfig{
		Enabled:          true,
		PreserveOriginal: false,
	})
	defer SetDefaultSanitizationConfig(ErrorSanitizationConfig{Enabled: false})

	err := NewError(ErrNetworkError, "Failed to connect to 192.168.1.100")

	if err.Message != "Failed to connect to [IP]" {
		t.Errorf("Expected sanitized message, got %q", err.Message)
	}

	if _, ok := err.Details["originalMessage"]; ok {
		t.Error("Expected no originalMessage in Details when PreserveOriginal is false")
	}
}

func TestNewError_WithoutSanitization(t *testing.T) {
	// Ensure sanitization is disabled
	SetDefaultSanitizationConfig(ErrorSanitizationConfig{Enabled: false})

	err := NewError(ErrNetworkError, "Failed to connect to 192.168.1.100")

	if err.Message != "Failed to connect to 192.168.1.100" {
		t.Errorf("Expected original message when sanitization disabled, got %q", err.Message)
	}
}

func TestNewErrorWithCause_WithSanitization(t *testing.T) {
	// Enable sanitization
	SetDefaultSanitizationConfig(ErrorSanitizationConfig{
		Enabled:          true,
		PreserveOriginal: true,
	})
	defer SetDefaultSanitizationConfig(ErrorSanitizationConfig{Enabled: false})

	cause := errors.New("connection to 10.0.0.1 refused")
	err := NewErrorWithCause(ErrNetworkError, "Failed with sdk_key123456789", cause)

	if err.Message != "Failed with sdk_[REDACTED]" {
		t.Errorf("Expected sanitized message, got %q", err.Message)
	}

	// Check that cause is also sanitized
	if err.Cause.Error() != "connection to [IP] refused" {
		t.Errorf("Expected sanitized cause, got %q", err.Cause.Error())
	}
}

func TestNewErrorWithCause_FlagKitErrorCause(t *testing.T) {
	// Enable sanitization
	SetDefaultSanitizationConfig(ErrorSanitizationConfig{
		Enabled: true,
	})
	defer SetDefaultSanitizationConfig(ErrorSanitizationConfig{Enabled: false})

	// Create a FlagKitError as cause (should already be sanitized)
	cause := NewError(ErrAuthInvalidKey, "Key sdk_secretkey1234 invalid")
	err := NewErrorWithCause(ErrInitFailed, "Init failed", cause)

	// The cause should not be double-wrapped
	if _, ok := err.Cause.(*FlagKitError); !ok {
		t.Error("Expected FlagKitError cause to remain as FlagKitError")
	}
}

func TestSanitizeCause_NilCause(t *testing.T) {
	result := sanitizeCause(nil)
	if result != nil {
		t.Error("Expected nil result for nil cause")
	}
}

func TestSanitizedError_Unwrap(t *testing.T) {
	SetDefaultSanitizationConfig(ErrorSanitizationConfig{Enabled: true})
	defer SetDefaultSanitizationConfig(ErrorSanitizationConfig{Enabled: false})

	original := errors.New("original error with 192.168.1.1")
	sanitized := sanitizeCause(original)

	// Check that we can unwrap to get the original
	unwrapped := errors.Unwrap(sanitized)
	if unwrapped != original {
		t.Error("Expected Unwrap to return original error")
	}

	// Check that the sanitized error message is correct
	if sanitized.Error() != "original error with [IP]" {
		t.Errorf("Expected sanitized message, got %q", sanitized.Error())
	}
}

func TestGetSetDefaultSanitizationConfig(t *testing.T) {
	// Save original config
	original := GetDefaultSanitizationConfig()
	defer SetDefaultSanitizationConfig(original)

	// Set new config
	newConfig := ErrorSanitizationConfig{
		Enabled:          true,
		PreserveOriginal: true,
	}
	SetDefaultSanitizationConfig(newConfig)

	// Verify it was set
	got := GetDefaultSanitizationConfig()
	if got.Enabled != true || got.PreserveOriginal != true {
		t.Errorf("Expected config to be set, got %+v", got)
	}
}

// Tests for WithErrorSanitization and WithErrorSanitizationConfig are in
// sanitizer_integration_test.go (uses errors_test package to avoid circular imports)
