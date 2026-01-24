package flagkit

import (
	"strings"
	"testing"
)

func TestIsPotentialPIIField(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		expected bool
	}{
		// Email fields
		{"email lowercase", "email", true},
		{"userEmail camelCase", "userEmail", true},
		{"EMAIL uppercase", "EMAIL", true},

		// Phone fields
		{"phone", "phone", true},
		{"phoneNumber", "phoneNumber", true},
		{"mobile", "mobile", true},
		{"telephone", "telephone", true},

		// SSN fields
		{"ssn", "ssn", true},
		{"socialSecurity", "socialSecurity", true},
		{"social_security", "social_security", true},

		// Credit card fields
		{"creditCard", "creditCard", true},
		{"credit_card", "credit_card", true},
		{"cardNumber", "cardNumber", true},
		{"cvv", "cvv", true},

		// Auth fields
		{"password", "password", true},
		{"secret", "secret", true},
		{"apiKey", "apiKey", true},
		{"accessToken", "accessToken", true},
		{"refreshToken", "refreshToken", true},

		// Address fields
		{"address", "address", true},
		{"street", "street", true},
		{"zipCode", "zipCode", true},
		{"postalCode", "postalCode", true},

		// Safe fields
		{"userId", "userId", false},
		{"plan", "plan", false},
		{"country", "country", false},
		{"featureEnabled", "featureEnabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPotentialPIIField(tt.field)
			if result != tt.expected {
				t.Errorf("IsPotentialPIIField(%q) = %v, want %v", tt.field, result, tt.expected)
			}
		})
	}
}

func TestDetectPotentialPII(t *testing.T) {
	t.Run("detects PII in flat objects", func(t *testing.T) {
		data := map[string]interface{}{
			"userId": "user-123",
			"email":  "user@example.com",
			"plan":   "premium",
		}

		piiFields := DetectPotentialPII(data, "")

		if !contains(piiFields, "email") {
			t.Error("expected 'email' in PII fields")
		}
		if contains(piiFields, "userId") {
			t.Error("'userId' should not be in PII fields")
		}
		if contains(piiFields, "plan") {
			t.Error("'plan' should not be in PII fields")
		}
	})

	t.Run("detects PII in nested objects", func(t *testing.T) {
		data := map[string]interface{}{
			"user": map[string]interface{}{
				"email": "user@example.com",
				"phone": "123-456-7890",
			},
			"settings": map[string]interface{}{
				"darkMode": true,
			},
		}

		piiFields := DetectPotentialPII(data, "")

		if !contains(piiFields, "user.email") {
			t.Error("expected 'user.email' in PII fields")
		}
		if !contains(piiFields, "user.phone") {
			t.Error("expected 'user.phone' in PII fields")
		}
		if contains(piiFields, "settings.darkMode") {
			t.Error("'settings.darkMode' should not be in PII fields")
		}
	})

	t.Run("handles deeply nested objects", func(t *testing.T) {
		data := map[string]interface{}{
			"profile": map[string]interface{}{
				"contact": map[string]interface{}{
					"primaryEmail": "user@example.com",
				},
			},
		}

		piiFields := DetectPotentialPII(data, "")

		if !contains(piiFields, "profile.contact.primaryEmail") {
			t.Error("expected 'profile.contact.primaryEmail' in PII fields")
		}
	})

	t.Run("returns empty for safe data", func(t *testing.T) {
		data := map[string]interface{}{
			"userId": "user-123",
			"plan":   "premium",
		}

		piiFields := DetectPotentialPII(data, "")

		if len(piiFields) != 0 {
			t.Errorf("expected empty PII fields, got %v", piiFields)
		}
	})
}

func TestWarnIfPotentialPII(t *testing.T) {
	t.Run("logs warning when PII detected", func(t *testing.T) {
		logger := &mockLogger{}
		data := map[string]interface{}{
			"email": "user@example.com",
			"phone": "123-456-7890",
		}

		WarnIfPotentialPII(data, "context", logger)

		if len(logger.warnings) != 1 {
			t.Errorf("expected 1 warning, got %d", len(logger.warnings))
		}
		if !strings.Contains(logger.warnings[0], "Potential PII detected") {
			t.Error("warning should contain 'Potential PII detected'")
		}
	})

	t.Run("does not log when no PII", func(t *testing.T) {
		logger := &mockLogger{}
		data := map[string]interface{}{
			"userId": "user-123",
			"plan":   "premium",
		}

		WarnIfPotentialPII(data, "context", logger)

		if len(logger.warnings) != 0 {
			t.Errorf("expected no warnings, got %d", len(logger.warnings))
		}
	})

	t.Run("handles nil data", func(t *testing.T) {
		logger := &mockLogger{}
		WarnIfPotentialPII(nil, "event", logger)

		if len(logger.warnings) != 0 {
			t.Errorf("expected no warnings for nil data, got %d", len(logger.warnings))
		}
	})

	t.Run("handles nil logger", func(t *testing.T) {
		data := map[string]interface{}{"email": "test@example.com"}
		// Should not panic
		WarnIfPotentialPII(data, "event", nil)
	})
}

func TestIsServerKey(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"srv_abc123", true},
		{"srv_", true},
		{"sdk_abc123", false},
		{"cli_abc123", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := IsServerKey(tt.key)
			if result != tt.expected {
				t.Errorf("IsServerKey(%q) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestIsClientKey(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"sdk_abc123", true},
		{"cli_abc123", true},
		{"srv_abc123", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := IsClientKey(tt.key)
			if result != tt.expected {
				t.Errorf("IsClientKey(%q) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestDefaultSecurityConfig(t *testing.T) {
	config := DefaultSecurityConfig()

	if !config.WarnOnServerKeyInBrowser {
		t.Error("WarnOnServerKeyInBrowser should be true by default")
	}
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Mock logger for testing
type mockLogger struct {
	debugs   []string
	infos    []string
	warnings []string
	errors   []string
}

func (l *mockLogger) Debug(msg string, fields ...interface{}) {
	l.debugs = append(l.debugs, msg)
}

func (l *mockLogger) Info(msg string, fields ...interface{}) {
	l.infos = append(l.infos, msg)
}

func (l *mockLogger) Warn(msg string, fields ...interface{}) {
	l.warnings = append(l.warnings, msg)
}

func (l *mockLogger) Error(msg string, fields ...interface{}) {
	l.errors = append(l.errors, msg)
}
