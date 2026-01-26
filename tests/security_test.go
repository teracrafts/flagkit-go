package flagkit

import (
	"os"
	"strings"
	"testing"
	"time"
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
		data := map[string]any{
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
		data := map[string]any{
			"user": map[string]any{
				"email": "user@example.com",
				"phone": "123-456-7890",
			},
			"settings": map[string]any{
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
		data := map[string]any{
			"profile": map[string]any{
				"contact": map[string]any{
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
		data := map[string]any{
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
		data := map[string]any{
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
		data := map[string]any{
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
		data := map[string]any{"email": "test@example.com"}
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

func TestCheckForPotentialPII(t *testing.T) {
	t.Run("returns empty for nil data", func(t *testing.T) {
		result := CheckForPotentialPII(nil, "context")
		if result.HasPII {
			t.Error("expected no PII for nil data")
		}
	})

	t.Run("detects PII in context data", func(t *testing.T) {
		data := map[string]any{
			"email": "user@example.com",
			"userId": "user-123",
		}
		result := CheckForPotentialPII(data, "context")
		if !result.HasPII {
			t.Error("expected PII to be detected")
		}
		if !contains(result.Fields, "email") {
			t.Error("expected 'email' in PII fields")
		}
		if !strings.Contains(result.Message, "privateAttributes") {
			t.Error("expected message to mention privateAttributes for context")
		}
	})

	t.Run("detects PII in event data", func(t *testing.T) {
		data := map[string]any{
			"phone": "123-456-7890",
		}
		result := CheckForPotentialPII(data, "event")
		if !result.HasPII {
			t.Error("expected PII to be detected")
		}
		if !strings.Contains(result.Message, "removing sensitive data") {
			t.Error("expected message to mention removing sensitive data for events")
		}
	})
}

func TestCheckPIIWithStrictMode(t *testing.T) {
	t.Run("returns nil when no PII", func(t *testing.T) {
		data := map[string]any{"userId": "user-123"}
		err := CheckPIIWithStrictMode(data, "context", true, nil)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("returns error in strict mode when PII detected", func(t *testing.T) {
		data := map[string]any{"email": "user@example.com"}
		err := CheckPIIWithStrictMode(data, "context", true, nil)
		if err == nil {
			t.Error("expected error in strict mode")
		}
		fkErr, ok := err.(*FlagKitError)
		if !ok {
			t.Error("expected FlagKitError")
		}
		if fkErr.Code != ErrSecurityPIIDetected {
			t.Errorf("expected ErrSecurityPIIDetected, got %s", fkErr.Code)
		}
	})

	t.Run("logs warning when not in strict mode", func(t *testing.T) {
		logger := &mockLogger{}
		data := map[string]any{"email": "user@example.com"}
		err := CheckPIIWithStrictMode(data, "context", false, logger)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(logger.warnings) != 1 {
			t.Errorf("expected 1 warning, got %d", len(logger.warnings))
		}
	})
}

func TestIsProductionEnvironment(t *testing.T) {
	// Save original env values
	originalGoEnv := os.Getenv("GO_ENV")
	originalAppEnv := os.Getenv("APP_ENV")
	defer func() {
		_ = os.Setenv("GO_ENV", originalGoEnv)
		_ = os.Setenv("APP_ENV", originalAppEnv)
	}()

	t.Run("returns false when no env set", func(t *testing.T) {
		_ = os.Unsetenv("GO_ENV")
		_ = os.Unsetenv("APP_ENV")
		if IsProductionEnvironment() {
			t.Error("expected false when no env set")
		}
	})

	t.Run("returns true for GO_ENV=production", func(t *testing.T) {
		_ = os.Setenv("GO_ENV", "production")
		_ = os.Unsetenv("APP_ENV")
		if !IsProductionEnvironment() {
			t.Error("expected true for GO_ENV=production")
		}
	})

	t.Run("returns true for APP_ENV=production", func(t *testing.T) {
		_ = os.Unsetenv("GO_ENV")
		_ = os.Setenv("APP_ENV", "production")
		if !IsProductionEnvironment() {
			t.Error("expected true for APP_ENV=production")
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		_ = os.Setenv("GO_ENV", "PRODUCTION")
		_ = os.Unsetenv("APP_ENV")
		if !IsProductionEnvironment() {
			t.Error("expected true for GO_ENV=PRODUCTION (case insensitive)")
		}
	})
}

func TestValidateLocalPort(t *testing.T) {
	// Save original env
	originalGoEnv := os.Getenv("GO_ENV")
	originalAppEnv := os.Getenv("APP_ENV")
	defer func() {
		_ = os.Setenv("GO_ENV", originalGoEnv)
		_ = os.Setenv("APP_ENV", originalAppEnv)
	}()

	t.Run("allows localPort in non-production", func(t *testing.T) {
		_ = os.Setenv("GO_ENV", "development")
		_ = os.Unsetenv("APP_ENV")
		err := ValidateLocalPort(8080)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("rejects localPort in production", func(t *testing.T) {
		_ = os.Setenv("GO_ENV", "production")
		_ = os.Unsetenv("APP_ENV")
		err := ValidateLocalPort(8080)
		if err == nil {
			t.Error("expected error for localPort in production")
		}
		fkErr, ok := err.(*FlagKitError)
		if !ok {
			t.Error("expected FlagKitError")
		}
		if fkErr.Code != ErrSecurityLocalPortInProduction {
			t.Errorf("expected ErrSecurityLocalPortInProduction, got %s", fkErr.Code)
		}
	})

	t.Run("allows zero localPort in production", func(t *testing.T) {
		_ = os.Setenv("GO_ENV", "production")
		_ = os.Unsetenv("APP_ENV")
		err := ValidateLocalPort(0)
		if err != nil {
			t.Errorf("expected no error for zero localPort, got %v", err)
		}
	})
}

func TestGetKeyID(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"sdk_abcdefghijk", "sdk_abcd"},
		{"srv_12345678901234", "srv_1234"},
		{"short", "short"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := GetKeyID(tt.key)
			if result != tt.expected {
				t.Errorf("GetKeyID(%q) = %q, want %q", tt.key, result, tt.expected)
			}
		})
	}
}

func TestGenerateHMACSHA256(t *testing.T) {
	t.Run("generates consistent signature", func(t *testing.T) {
		message := "test message"
		key := "secret_key"

		sig1 := GenerateHMACSHA256(message, key)
		sig2 := GenerateHMACSHA256(message, key)

		if sig1 != sig2 {
			t.Error("expected consistent signatures")
		}
	})

	t.Run("different messages produce different signatures", func(t *testing.T) {
		key := "secret_key"

		sig1 := GenerateHMACSHA256("message1", key)
		sig2 := GenerateHMACSHA256("message2", key)

		if sig1 == sig2 {
			t.Error("expected different signatures for different messages")
		}
	})

	t.Run("different keys produce different signatures", func(t *testing.T) {
		message := "test message"

		sig1 := GenerateHMACSHA256(message, "key1")
		sig2 := GenerateHMACSHA256(message, "key2")

		if sig1 == sig2 {
			t.Error("expected different signatures for different keys")
		}
	})
}

func TestCreateRequestSignature(t *testing.T) {
	body := `{"key":"value"}`
	apiKey := "sdk_test_api_key_12345"

	sig := CreateRequestSignature(body, apiKey)

	if sig.Signature == "" {
		t.Error("expected non-empty signature")
	}
	if sig.Timestamp == 0 {
		t.Error("expected non-zero timestamp")
	}
	if sig.KeyID != "sdk_test" {
		t.Errorf("expected keyID 'sdk_test', got '%s'", sig.KeyID)
	}
}

func TestVerifyRequestSignature(t *testing.T) {
	body := `{"key":"value"}`
	apiKey := "sdk_test_api_key_12345"

	t.Run("verifies valid signature", func(t *testing.T) {
		sig := CreateRequestSignature(body, apiKey)
		valid := VerifyRequestSignature(body, sig.Signature, sig.Timestamp, apiKey, 0)
		if !valid {
			t.Error("expected signature to be valid")
		}
	})

	t.Run("rejects tampered body", func(t *testing.T) {
		sig := CreateRequestSignature(body, apiKey)
		valid := VerifyRequestSignature(`{"key":"tampered"}`, sig.Signature, sig.Timestamp, apiKey, 0)
		if valid {
			t.Error("expected signature to be invalid for tampered body")
		}
	})

	t.Run("rejects wrong key", func(t *testing.T) {
		sig := CreateRequestSignature(body, apiKey)
		valid := VerifyRequestSignature(body, sig.Signature, sig.Timestamp, "wrong_key", 0)
		if valid {
			t.Error("expected signature to be invalid for wrong key")
		}
	})

	t.Run("rejects expired signature", func(t *testing.T) {
		oldTimestamp := time.Now().Add(-10 * time.Minute).UnixMilli()
		message := "test message"
		signature := GenerateHMACSHA256(message, apiKey)

		valid := VerifyRequestSignature(message, signature, oldTimestamp, apiKey, 300000)
		if valid {
			t.Error("expected signature to be invalid for expired timestamp")
		}
	})
}

func TestSignPayload(t *testing.T) {
	data := "test data"
	apiKey := "sdk_test_api_key_12345"

	t.Run("creates signed payload", func(t *testing.T) {
		payload := SignPayload(data, apiKey, 0)

		if payload.Data != data {
			t.Errorf("expected data '%s', got '%v'", data, payload.Data)
		}
		if payload.Signature == "" {
			t.Error("expected non-empty signature")
		}
		if payload.Timestamp == 0 {
			t.Error("expected non-zero timestamp")
		}
		if payload.KeyID != "sdk_test" {
			t.Errorf("expected keyID 'sdk_test', got '%s'", payload.KeyID)
		}
	})

	t.Run("uses provided timestamp", func(t *testing.T) {
		customTimestamp := int64(1234567890)
		payload := SignPayload(data, apiKey, customTimestamp)

		if payload.Timestamp != customTimestamp {
			t.Errorf("expected timestamp %d, got %d", customTimestamp, payload.Timestamp)
		}
	})
}

func TestVerifySignedPayload(t *testing.T) {
	data := "test data"
	apiKey := "sdk_test_api_key_12345"

	t.Run("verifies valid payload", func(t *testing.T) {
		payload := SignPayload(data, apiKey, 0)
		valid := VerifySignedPayload(payload, apiKey, 0)
		if !valid {
			t.Error("expected payload to be valid")
		}
	})

	t.Run("rejects wrong key", func(t *testing.T) {
		payload := SignPayload(data, apiKey, 0)
		valid := VerifySignedPayload(payload, "wrong_key_12345678", 0)
		if valid {
			t.Error("expected payload to be invalid for wrong key")
		}
	})

	t.Run("rejects expired payload", func(t *testing.T) {
		oldTimestamp := time.Now().Add(-10 * time.Minute).UnixMilli()
		payload := SignPayload(data, apiKey, oldTimestamp)
		valid := VerifySignedPayload(payload, apiKey, 300000)
		if valid {
			t.Error("expected payload to be invalid for expired timestamp")
		}
	})
}

// Tests for strict PII mode integration in client methods

func TestClient_StrictPIIModeEnforcement(t *testing.T) {
	// Note: These tests verify that the client methods properly enforce
	// strict PII mode when configured. We use offline mode to avoid
	// network calls during testing.

	t.Run("Identify returns error in strict mode when PII detected", func(t *testing.T) {
		client, err := NewClient(
			"sdk_test_api_key_12345",
			WithOffline(),
			WithStrictPIIMode(),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		defer func() { _ = client.Close() }()

		// Attributes containing PII
		attrs := map[string]any{
			"email":  "user@example.com",
			"userId": "user-123",
		}

		err = client.Identify("user-123", attrs)
		if err == nil {
			t.Error("expected error when PII detected in strict mode")
		}

		fkErr, ok := err.(*FlagKitError)
		if !ok {
			t.Errorf("expected FlagKitError, got %T", err)
		} else if fkErr.Code != ErrSecurityPIIDetected {
			t.Errorf("expected ErrSecurityPIIDetected, got %s", fkErr.Code)
		}
	})

	t.Run("Identify succeeds without PII in strict mode", func(t *testing.T) {
		client, err := NewClient(
			"sdk_test_api_key_12345",
			WithOffline(),
			WithStrictPIIMode(),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		defer func() { _ = client.Close() }()

		// Safe attributes without PII
		attrs := map[string]any{
			"plan":    "premium",
			"company": "Acme Inc",
		}

		err = client.Identify("user-123", attrs)
		if err != nil {
			t.Errorf("expected no error for safe data, got %v", err)
		}
	})

	t.Run("Identify warns but succeeds when not in strict mode", func(t *testing.T) {
		logger := &mockLogger{}
		client, err := NewClient(
			"sdk_test_api_key_12345",
			WithOffline(),
			WithLogger(logger),
			// No WithStrictPIIMode() - not strict
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		defer func() { _ = client.Close() }()

		attrs := map[string]any{
			"email": "user@example.com",
		}

		err = client.Identify("user-123", attrs)
		if err != nil {
			t.Errorf("expected no error in non-strict mode, got %v", err)
		}

		// Should have logged a warning
		if len(logger.warnings) == 0 {
			t.Error("expected warning to be logged")
		}
	})

	t.Run("Track returns error in strict mode when PII detected", func(t *testing.T) {
		client, err := NewClient(
			"sdk_test_api_key_12345",
			WithOffline(),
			WithStrictPIIMode(),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		defer func() { _ = client.Close() }()

		// Event data containing PII
		eventData := map[string]any{
			"phone":       "123-456-7890",
			"productId":   "prod-123",
		}

		err = client.Track("purchase", eventData)
		if err == nil {
			t.Error("expected error when PII detected in strict mode")
		}

		fkErr, ok := err.(*FlagKitError)
		if !ok {
			t.Errorf("expected FlagKitError, got %T", err)
		} else if fkErr.Code != ErrSecurityPIIDetected {
			t.Errorf("expected ErrSecurityPIIDetected, got %s", fkErr.Code)
		}
	})

	t.Run("Track succeeds without PII in strict mode", func(t *testing.T) {
		client, err := NewClient(
			"sdk_test_api_key_12345",
			WithOffline(),
			WithStrictPIIMode(),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		defer func() { _ = client.Close() }()

		// Safe event data
		eventData := map[string]any{
			"productId": "prod-123",
			"quantity":  2,
			"action":    "purchase",
		}

		err = client.Track("purchase", eventData)
		if err != nil {
			t.Errorf("expected no error for safe data, got %v", err)
		}
	})

	t.Run("Track warns but succeeds when not in strict mode", func(t *testing.T) {
		logger := &mockLogger{}
		client, err := NewClient(
			"sdk_test_api_key_12345",
			WithOffline(),
			WithLogger(logger),
			// No WithStrictPIIMode() - not strict
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		defer func() { _ = client.Close() }()

		eventData := map[string]any{
			"creditCard": "4111111111111111",
		}

		err = client.Track("payment", eventData)
		if err != nil {
			t.Errorf("expected no error in non-strict mode, got %v", err)
		}

		// Should have logged a warning
		if len(logger.warnings) == 0 {
			t.Error("expected warning to be logged")
		}
	})

	t.Run("SetContext returns error in strict mode when PII detected", func(t *testing.T) {
		client, err := NewClient(
			"sdk_test_api_key_12345",
			WithOffline(),
			WithStrictPIIMode(),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		defer func() { _ = client.Close() }()

		ctx := NewContext("user-123")
		ctx.WithCustom("ssn", "123-45-6789")
		ctx.WithCustom("plan", "premium")

		err = client.SetContext(ctx)
		if err == nil {
			t.Error("expected error when PII detected in strict mode")
		}

		fkErr, ok := err.(*FlagKitError)
		if !ok {
			t.Errorf("expected FlagKitError, got %T", err)
		} else if fkErr.Code != ErrSecurityPIIDetected {
			t.Errorf("expected ErrSecurityPIIDetected, got %s", fkErr.Code)
		}
	})

	t.Run("SetContext succeeds without PII in strict mode", func(t *testing.T) {
		client, err := NewClient(
			"sdk_test_api_key_12345",
			WithOffline(),
			WithStrictPIIMode(),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		defer func() { _ = client.Close() }()

		ctx := NewContext("user-123")
		ctx.WithCustom("plan", "premium")
		ctx.WithCustom("company", "Acme Inc")

		err = client.SetContext(ctx)
		if err != nil {
			t.Errorf("expected no error for safe data, got %v", err)
		}
	})

	t.Run("SetContext warns but succeeds when not in strict mode", func(t *testing.T) {
		logger := &mockLogger{}
		client, err := NewClient(
			"sdk_test_api_key_12345",
			WithOffline(),
			WithLogger(logger),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		defer func() { _ = client.Close() }()

		ctx := NewContext("user-123")
		ctx.WithCustom("password", "secret123")

		err = client.SetContext(ctx)
		if err != nil {
			t.Errorf("expected no error in non-strict mode, got %v", err)
		}

		// Should have logged a warning
		if len(logger.warnings) == 0 {
			t.Error("expected warning to be logged")
		}
	})

	t.Run("SetContext skips PII check when privateAttributes set", func(t *testing.T) {
		client, err := NewClient(
			"sdk_test_api_key_12345",
			WithOffline(),
			WithStrictPIIMode(),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		defer func() { _ = client.Close() }()

		ctx := NewContext("user-123")
		ctx.WithCustom("email", "user@example.com")
		ctx.WithPrivateAttribute("email") // Mark as private, so PII check should be skipped

		err = client.SetContext(ctx)
		if err != nil {
			t.Errorf("expected no error when privateAttributes set, got %v", err)
		}
	})

	t.Run("Track with nil data succeeds in strict mode", func(t *testing.T) {
		client, err := NewClient(
			"sdk_test_api_key_12345",
			WithOffline(),
			WithStrictPIIMode(),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		defer func() { _ = client.Close() }()

		err = client.Track("page_view")
		if err != nil {
			t.Errorf("expected no error for nil data, got %v", err)
		}
	})

	t.Run("SetContext with nil context succeeds in strict mode", func(t *testing.T) {
		client, err := NewClient(
			"sdk_test_api_key_12345",
			WithOffline(),
			WithStrictPIIMode(),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		defer func() { _ = client.Close() }()

		err = client.SetContext(nil)
		if err != nil {
			t.Errorf("expected no error for nil context, got %v", err)
		}
	})

	t.Run("Identify without attributes succeeds in strict mode", func(t *testing.T) {
		client, err := NewClient(
			"sdk_test_api_key_12345",
			WithOffline(),
			WithStrictPIIMode(),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		defer func() { _ = client.Close() }()

		err = client.Identify("user-123")
		if err != nil {
			t.Errorf("expected no error for identify without attrs, got %v", err)
		}
	})
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

func (l *mockLogger) Debug(msg string, fields ...any) {
	l.debugs = append(l.debugs, msg)
}

func (l *mockLogger) Info(msg string, fields ...any) {
	l.infos = append(l.infos, msg)
}

func (l *mockLogger) Warn(msg string, fields ...any) {
	l.warnings = append(l.warnings, msg)
}

func (l *mockLogger) Error(msg string, fields ...any) {
	l.errors = append(l.errors, msg)
}
