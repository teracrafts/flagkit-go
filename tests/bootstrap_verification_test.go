package tests

import (
	"testing"
	"time"

	. "github.com/flagkit/flagkit-go"
)

func TestCanonicalizeObject(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		result, err := CanonicalizeObject(map[string]any{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "{}" {
			t.Errorf("expected '{}', got '%s'", result)
		}
	})

	t.Run("nil map", func(t *testing.T) {
		result, err := CanonicalizeObject(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "{}" {
			t.Errorf("expected '{}', got '%s'", result)
		}
	})

	t.Run("simple values", func(t *testing.T) {
		obj := map[string]any{
			"bool":   true,
			"string": "hello",
			"number": float64(42),
		}
		result, err := CanonicalizeObject(obj)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Keys should be sorted alphabetically
		expected := `{"bool":true,"number":42,"string":"hello"}`
		if result != expected {
			t.Errorf("expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("sorted keys", func(t *testing.T) {
		obj := map[string]any{
			"zebra": 1,
			"apple": 2,
			"mango": 3,
		}
		result, err := CanonicalizeObject(obj)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := `{"apple":2,"mango":3,"zebra":1}`
		if result != expected {
			t.Errorf("expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("nested objects", func(t *testing.T) {
		obj := map[string]any{
			"outer": map[string]any{
				"inner": "value",
			},
		}
		result, err := CanonicalizeObject(obj)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := `{"outer":{"inner":"value"}}`
		if result != expected {
			t.Errorf("expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("arrays", func(t *testing.T) {
		obj := map[string]any{
			"list": []any{1, 2, 3},
		}
		result, err := CanonicalizeObject(obj)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := `{"list":[1,2,3]}`
		if result != expected {
			t.Errorf("expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("null values", func(t *testing.T) {
		obj := map[string]any{
			"null_val": nil,
		}
		result, err := CanonicalizeObject(obj)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := `{"null_val":null}`
		if result != expected {
			t.Errorf("expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("string escaping", func(t *testing.T) {
		obj := map[string]any{
			"special": "hello\nworld\"quote",
		}
		result, err := CanonicalizeObject(obj)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := `{"special":"hello\nworld\"quote"}`
		if result != expected {
			t.Errorf("expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("deterministic output", func(t *testing.T) {
		obj := map[string]any{
			"c": 3,
			"a": 1,
			"b": 2,
		}

		// Run multiple times to ensure consistent output
		var results []string
		for i := 0; i < 10; i++ {
			result, err := CanonicalizeObject(obj)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			results = append(results, result)
		}

		for i := 1; i < len(results); i++ {
			if results[i] != results[0] {
				t.Errorf("output not deterministic: '%s' vs '%s'", results[0], results[i])
			}
		}
	})
}

func TestVerifyBootstrapSignature(t *testing.T) {
	apiKey := "sdk_test_api_key_12345"

	t.Run("valid signature accepted", func(t *testing.T) {
		// Create signed bootstrap
		flags := map[string]any{
			"feature_a": true,
			"feature_b": "enabled",
			"feature_c": float64(42),
		}

		bootstrap, err := CreateBootstrapSignature(flags, apiKey)
		if err != nil {
			t.Fatalf("failed to create bootstrap signature: %v", err)
		}

		config := BootstrapVerificationConfig{
			Enabled:   true,
			MaxAge:    24 * time.Hour,
			OnFailure: "error",
		}

		valid, err := VerifyBootstrapSignature(*bootstrap, apiKey, config)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !valid {
			t.Error("expected signature to be valid")
		}
	})

	t.Run("invalid signature rejected", func(t *testing.T) {
		bootstrap := BootstrapConfig{
			Flags: map[string]any{
				"feature": true,
			},
			Signature: "invalid_signature_12345678901234567890123456789012345678901234567890123456",
			Timestamp: time.Now().UnixMilli(),
		}

		config := BootstrapVerificationConfig{
			Enabled:   true,
			MaxAge:    24 * time.Hour,
			OnFailure: "error",
		}

		valid, err := VerifyBootstrapSignature(bootstrap, apiKey, config)
		if valid {
			t.Error("expected signature to be invalid")
		}
		if err == nil {
			t.Error("expected error for invalid signature")
		}

		fkErr, ok := err.(*FlagKitError)
		if !ok {
			t.Errorf("expected FlagKitError, got %T", err)
		} else if fkErr.Code != ErrSecuritySignatureInvalid {
			t.Errorf("expected ErrSecuritySignatureInvalid, got %s", fkErr.Code)
		}
	})

	t.Run("tampered flags rejected", func(t *testing.T) {
		// Create valid signature
		bootstrap, _ := CreateBootstrapSignature(map[string]any{"feature": true}, apiKey)

		// Tamper with flags
		bootstrap.Flags["feature"] = false

		config := BootstrapVerificationConfig{
			Enabled:   true,
			MaxAge:    24 * time.Hour,
			OnFailure: "error",
		}

		valid, err := VerifyBootstrapSignature(*bootstrap, apiKey, config)
		if valid {
			t.Error("expected tampered signature to be invalid")
		}
		if err == nil {
			t.Error("expected error for tampered flags")
		}
	})

	t.Run("expired timestamp rejected", func(t *testing.T) {
		flags := map[string]any{"feature": true}
		oldTimestamp := time.Now().Add(-48 * time.Hour).UnixMilli()

		// Create signature with old timestamp
		canonical, _ := CanonicalizeObject(flags)
		message := string(rune(oldTimestamp)) + "." + canonical
		_ = message // Used in manual signature creation

		bootstrap := BootstrapConfig{
			Flags:     flags,
			Timestamp: oldTimestamp,
		}
		// Create proper signature for the old timestamp
		signedBootstrap, _ := CreateBootstrapSignature(flags, apiKey)
		bootstrap.Signature = signedBootstrap.Signature
		bootstrap.Timestamp = oldTimestamp // Override to old timestamp

		config := BootstrapVerificationConfig{
			Enabled:   true,
			MaxAge:    24 * time.Hour,
			OnFailure: "error",
		}

		valid, err := VerifyBootstrapSignature(bootstrap, apiKey, config)
		if valid {
			t.Error("expected expired timestamp to be invalid")
		}
		if err == nil {
			t.Error("expected error for expired timestamp")
		}
	})

	t.Run("verification disabled always valid", func(t *testing.T) {
		bootstrap := BootstrapConfig{
			Flags: map[string]any{
				"feature": true,
			},
			Signature: "completely_invalid_signature",
			Timestamp: time.Now().UnixMilli(),
		}

		config := BootstrapVerificationConfig{
			Enabled:   false,
			MaxAge:    24 * time.Hour,
			OnFailure: "error",
		}

		valid, err := VerifyBootstrapSignature(bootstrap, apiKey, config)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !valid {
			t.Error("expected valid when verification disabled")
		}
	})

	t.Run("empty signature skips verification (legacy)", func(t *testing.T) {
		bootstrap := BootstrapConfig{
			Flags: map[string]any{
				"feature": true,
			},
			Signature: "",
			Timestamp: 0,
		}

		config := BootstrapVerificationConfig{
			Enabled:   true,
			MaxAge:    24 * time.Hour,
			OnFailure: "error",
		}

		valid, err := VerifyBootstrapSignature(bootstrap, apiKey, config)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !valid {
			t.Error("expected valid for empty signature (legacy format)")
		}
	})

	t.Run("wrong API key rejected", func(t *testing.T) {
		bootstrap, _ := CreateBootstrapSignature(map[string]any{"feature": true}, apiKey)

		config := BootstrapVerificationConfig{
			Enabled:   true,
			MaxAge:    24 * time.Hour,
			OnFailure: "error",
		}

		// Verify with different key
		valid, err := VerifyBootstrapSignature(*bootstrap, "different_api_key_12345", config)
		if valid {
			t.Error("expected invalid for wrong API key")
		}
		if err == nil {
			t.Error("expected error for wrong API key")
		}
	})

	t.Run("future timestamp rejected", func(t *testing.T) {
		flags := map[string]any{"feature": true}
		futureTimestamp := time.Now().Add(10 * time.Minute).UnixMilli()

		canonical, _ := CanonicalizeObject(flags)
		message := string(rune(futureTimestamp)) + "." + canonical
		signature := GenerateHMACSHA256(message, apiKey)

		bootstrap := BootstrapConfig{
			Flags:     flags,
			Signature: signature,
			Timestamp: futureTimestamp,
		}

		config := BootstrapVerificationConfig{
			Enabled:   true,
			MaxAge:    24 * time.Hour,
			OnFailure: "error",
		}

		valid, err := VerifyBootstrapSignature(bootstrap, apiKey, config)
		if valid {
			t.Error("expected invalid for future timestamp")
		}
		if err == nil {
			t.Error("expected error for future timestamp")
		}
	})
}

func TestCreateBootstrapSignature(t *testing.T) {
	apiKey := "sdk_test_api_key_12345"

	t.Run("creates valid signed bootstrap", func(t *testing.T) {
		flags := map[string]any{
			"feature_a": true,
			"feature_b": "value",
		}

		bootstrap, err := CreateBootstrapSignature(flags, apiKey)
		if err != nil {
			t.Fatalf("failed to create bootstrap: %v", err)
		}

		if bootstrap.Flags == nil {
			t.Error("expected flags to be set")
		}
		if bootstrap.Signature == "" {
			t.Error("expected signature to be set")
		}
		if bootstrap.Timestamp == 0 {
			t.Error("expected timestamp to be set")
		}

		// Verify the signature is valid
		config := BootstrapVerificationConfig{
			Enabled:   true,
			MaxAge:    24 * time.Hour,
			OnFailure: "error",
		}
		valid, err := VerifyBootstrapSignature(*bootstrap, apiKey, config)
		if err != nil {
			t.Errorf("verification failed: %v", err)
		}
		if !valid {
			t.Error("expected created signature to be valid")
		}
	})

	t.Run("timestamp is current", func(t *testing.T) {
		before := time.Now().UnixMilli()
		bootstrap, _ := CreateBootstrapSignature(map[string]any{}, apiKey)
		after := time.Now().UnixMilli()

		if bootstrap.Timestamp < before || bootstrap.Timestamp > after {
			t.Errorf("timestamp %d not in range [%d, %d]", bootstrap.Timestamp, before, after)
		}
	})
}

func TestBootstrapVerificationClientIntegration(t *testing.T) {
	apiKey := "sdk_test_api_key_12345"

	t.Run("raw map format works (legacy)", func(t *testing.T) {
		client, err := NewClient(
			apiKey,
			WithOffline(),
			WithBootstrap(map[string]any{
				"feature_legacy": true,
			}),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		defer func() { _ = client.Close() }()

		value := client.GetBooleanValue("feature_legacy", false)
		if !value {
			t.Error("expected legacy bootstrap value to be applied")
		}
	})

	t.Run("signed bootstrap with valid signature", func(t *testing.T) {
		bootstrap, _ := CreateBootstrapSignature(map[string]any{
			"feature_signed": true,
		}, apiKey)

		client, err := NewClient(
			apiKey,
			WithOffline(),
			WithSignedBootstrap(bootstrap),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		defer func() { _ = client.Close() }()

		value := client.GetBooleanValue("feature_signed", false)
		if !value {
			t.Error("expected signed bootstrap value to be applied")
		}
	})

	t.Run("signed bootstrap with invalid signature - warn mode", func(t *testing.T) {
		logger := &mockLogger{}
		bootstrap := &BootstrapConfig{
			Flags: map[string]any{
				"feature_invalid": true,
			},
			Signature: "invalid_signature_1234567890abcdef1234567890abcdef1234567890abcdef12345678",
			Timestamp: time.Now().UnixMilli(),
		}

		client, err := NewClient(
			apiKey,
			WithOffline(),
			WithSignedBootstrap(bootstrap),
			WithBootstrapVerification(true, 24*time.Hour, "warn"),
			WithLogger(logger),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		defer func() { _ = client.Close() }()

		// Should still apply values in warn mode
		value := client.GetBooleanValue("feature_invalid", false)
		if !value {
			t.Error("expected bootstrap value to be applied in warn mode")
		}

		// Should have logged a warning
		if len(logger.warnings) == 0 {
			t.Error("expected warning to be logged")
		}
	})

	t.Run("signed bootstrap with invalid signature - error mode", func(t *testing.T) {
		bootstrap := &BootstrapConfig{
			Flags: map[string]any{
				"feature_error": true,
			},
			Signature: "invalid_signature_1234567890abcdef1234567890abcdef1234567890abcdef12345678",
			Timestamp: time.Now().UnixMilli(),
		}

		var capturedError error
		client, err := NewClient(
			apiKey,
			WithOffline(),
			WithSignedBootstrap(bootstrap),
			WithBootstrapVerification(true, 24*time.Hour, "error"),
			WithOnError(func(e error) {
				capturedError = e
			}),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		defer func() { _ = client.Close() }()

		// Should NOT apply values in error mode
		value := client.GetBooleanValue("feature_error", false)
		if value {
			t.Error("expected bootstrap value to NOT be applied in error mode")
		}

		// Should have captured error
		if capturedError == nil {
			t.Error("expected error to be captured")
		}
	})

	t.Run("signed bootstrap with invalid signature - ignore mode", func(t *testing.T) {
		logger := &mockLogger{}
		bootstrap := &BootstrapConfig{
			Flags: map[string]any{
				"feature_ignore": true,
			},
			Signature: "invalid_signature_1234567890abcdef1234567890abcdef1234567890abcdef12345678",
			Timestamp: time.Now().UnixMilli(),
		}

		client, err := NewClient(
			apiKey,
			WithOffline(),
			WithSignedBootstrap(bootstrap),
			WithBootstrapVerification(true, 24*time.Hour, "ignore"),
			WithLogger(logger),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		defer func() { _ = client.Close() }()

		// Should apply values in ignore mode
		value := client.GetBooleanValue("feature_ignore", false)
		if !value {
			t.Error("expected bootstrap value to be applied in ignore mode")
		}

		// Should NOT have logged any warnings about verification
		for _, w := range logger.warnings {
			if contains([]string{w}, "signature") {
				t.Error("expected no warning about signature in ignore mode")
			}
		}
	})

	t.Run("signed bootstrap takes precedence over raw bootstrap", func(t *testing.T) {
		signedBootstrap, _ := CreateBootstrapSignature(map[string]any{
			"feature": "signed_value",
		}, apiKey)

		client, err := NewClient(
			apiKey,
			WithOffline(),
			WithBootstrap(map[string]any{
				"feature": "raw_value",
			}),
			WithSignedBootstrap(signedBootstrap),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		defer func() { _ = client.Close() }()

		value := client.GetStringValue("feature", "default")
		if value != "signed_value" {
			t.Errorf("expected signed bootstrap to take precedence, got '%s'", value)
		}
	})

	t.Run("verification disabled accepts invalid signature", func(t *testing.T) {
		bootstrap := &BootstrapConfig{
			Flags: map[string]any{
				"feature_no_verify": true,
			},
			Signature: "invalid_signature",
			Timestamp: time.Now().UnixMilli(),
		}

		client, err := NewClient(
			apiKey,
			WithOffline(),
			WithSignedBootstrap(bootstrap),
			WithBootstrapVerification(false, 0, "error"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		defer func() { _ = client.Close() }()

		value := client.GetBooleanValue("feature_no_verify", false)
		if !value {
			t.Error("expected bootstrap value to be applied when verification disabled")
		}
	})
}

func TestWithBootstrapVerification(t *testing.T) {
	t.Run("sets verification config", func(t *testing.T) {
		opts := DefaultOptions("sdk_test_key_12345")
		WithBootstrapVerification(true, 12*time.Hour, "error")(opts)

		if !opts.BootstrapVerification.Enabled {
			t.Error("expected Enabled to be true")
		}
		if opts.BootstrapVerification.MaxAge != 12*time.Hour {
			t.Errorf("expected MaxAge to be 12h, got %v", opts.BootstrapVerification.MaxAge)
		}
		if opts.BootstrapVerification.OnFailure != "error" {
			t.Errorf("expected OnFailure to be 'error', got '%s'", opts.BootstrapVerification.OnFailure)
		}
	})

	t.Run("disables verification", func(t *testing.T) {
		opts := DefaultOptions("sdk_test_key_12345")
		WithBootstrapVerification(false, 0, "ignore")(opts)

		if opts.BootstrapVerification.Enabled {
			t.Error("expected Enabled to be false")
		}
	})
}

func TestWithSignedBootstrap(t *testing.T) {
	t.Run("sets signed bootstrap", func(t *testing.T) {
		bootstrap := &BootstrapConfig{
			Flags: map[string]any{
				"test": true,
			},
			Signature: "test_sig",
			Timestamp: 12345,
		}

		opts := DefaultOptions("sdk_test_key_12345")
		WithSignedBootstrap(bootstrap)(opts)

		if opts.BootstrapWithSignature == nil {
			t.Error("expected BootstrapWithSignature to be set")
		}
		if opts.BootstrapWithSignature.Signature != "test_sig" {
			t.Error("expected signature to match")
		}
	})
}
