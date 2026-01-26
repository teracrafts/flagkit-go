package flagkit

import (
	"testing"
	"time"
)

func TestEvaluationJitter_DisabledByDefault(t *testing.T) {
	// Create client with default options (jitter disabled)
	client, err := NewClient("sdk_test_api_key_1234567890",
		WithOffline(),
		WithPollingDisabled(),
		WithBootstrap(map[string]any{
			"test-flag": true,
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() { _ = client.Close() }()

	// Verify jitter is disabled by default
	if client.options.EvaluationJitter.Enabled {
		t.Error("Expected evaluation jitter to be disabled by default")
	}

	// Measure evaluation time without jitter
	start := time.Now()
	_ = client.GetBooleanValue("test-flag", false)
	elapsed := time.Since(start)

	// Without jitter, evaluation should be very fast (< 5ms)
	if elapsed >= 5*time.Millisecond {
		t.Errorf("Expected evaluation without jitter to be fast, got %v", elapsed)
	}
}

func TestEvaluationJitter_AppliedWhenEnabled(t *testing.T) {
	minMs := 10
	maxMs := 20

	// Create client with jitter enabled
	client, err := NewClient("sdk_test_api_key_1234567890",
		WithOffline(),
		WithPollingDisabled(),
		WithBootstrap(map[string]any{
			"test-flag": true,
		}),
		WithEvaluationJitter(true, minMs, maxMs),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() { _ = client.Close() }()

	// Verify jitter is enabled
	if !client.options.EvaluationJitter.Enabled {
		t.Error("Expected evaluation jitter to be enabled")
	}
	if client.options.EvaluationJitter.MinMs != minMs {
		t.Errorf("Expected MinMs to be %d, got %d", minMs, client.options.EvaluationJitter.MinMs)
	}
	if client.options.EvaluationJitter.MaxMs != maxMs {
		t.Errorf("Expected MaxMs to be %d, got %d", maxMs, client.options.EvaluationJitter.MaxMs)
	}

	// Measure evaluation time with jitter
	start := time.Now()
	_ = client.GetBooleanValue("test-flag", false)
	elapsed := time.Since(start)

	// With jitter enabled, evaluation should take at least minMs
	if elapsed < time.Duration(minMs)*time.Millisecond {
		t.Errorf("Expected evaluation with jitter to take at least %dms, got %v", minMs, elapsed)
	}
}

func TestEvaluationJitter_TimingWithinRange(t *testing.T) {
	minMs := 15
	maxMs := 25

	// Create client with jitter enabled
	client, err := NewClient("sdk_test_api_key_1234567890",
		WithOffline(),
		WithPollingDisabled(),
		WithBootstrap(map[string]any{
			"test-flag": "value",
		}),
		WithEvaluationJitter(true, minMs, maxMs),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() { _ = client.Close() }()

	// Run multiple evaluations and check timing is within expected range
	// Allow some tolerance for scheduling jitter
	toleranceMs := 5
	numTests := 10

	for i := 0; i < numTests; i++ {
		start := time.Now()
		_ = client.GetStringValue("test-flag", "default")
		elapsed := time.Since(start)

		// Check lower bound (should be >= minMs)
		if elapsed < time.Duration(minMs)*time.Millisecond {
			t.Errorf("Iteration %d: Expected evaluation to take at least %dms, got %v", i, minMs, elapsed)
		}

		// Check upper bound with tolerance (should be <= maxMs + tolerance)
		maxExpected := time.Duration(maxMs+toleranceMs) * time.Millisecond
		if elapsed > maxExpected {
			t.Errorf("Iteration %d: Expected evaluation to take at most %v, got %v", i, maxExpected, elapsed)
		}
	}
}

func TestWithEvaluationJitter_OptionWorks(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		minMs   int
		maxMs   int
	}{
		{
			name:    "enabled with custom values",
			enabled: true,
			minMs:   10,
			maxMs:   20,
		},
		{
			name:    "disabled explicitly",
			enabled: false,
			minMs:   5,
			maxMs:   15,
		},
		{
			name:    "enabled with zero values",
			enabled: true,
			minMs:   0,
			maxMs:   0,
		},
		{
			name:    "enabled with same min and max",
			enabled: true,
			minMs:   10,
			maxMs:   10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient("sdk_test_api_key_1234567890",
				WithOffline(),
				WithPollingDisabled(),
				WithEvaluationJitter(tt.enabled, tt.minMs, tt.maxMs),
			)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			defer func() { _ = client.Close() }()

			if client.options.EvaluationJitter.Enabled != tt.enabled {
				t.Errorf("Expected Enabled to be %v, got %v", tt.enabled, client.options.EvaluationJitter.Enabled)
			}
			if client.options.EvaluationJitter.MinMs != tt.minMs {
				t.Errorf("Expected MinMs to be %d, got %d", tt.minMs, client.options.EvaluationJitter.MinMs)
			}
			if client.options.EvaluationJitter.MaxMs != tt.maxMs {
				t.Errorf("Expected MaxMs to be %d, got %d", tt.maxMs, client.options.EvaluationJitter.MaxMs)
			}
		})
	}
}

func TestEvaluationJitter_DefaultValues(t *testing.T) {
	opts := DefaultOptions("sdk_test_api_key_1234567890")

	// Check default values
	if opts.EvaluationJitter.Enabled != false {
		t.Error("Expected default Enabled to be false")
	}
	if opts.EvaluationJitter.MinMs != DefaultEvaluationJitterMinMs {
		t.Errorf("Expected default MinMs to be %d, got %d", DefaultEvaluationJitterMinMs, opts.EvaluationJitter.MinMs)
	}
	if opts.EvaluationJitter.MaxMs != DefaultEvaluationJitterMaxMs {
		t.Errorf("Expected default MaxMs to be %d, got %d", DefaultEvaluationJitterMaxMs, opts.EvaluationJitter.MaxMs)
	}
}

func TestEvaluationJitter_AllEvaluationMethods(t *testing.T) {
	minMs := 10
	maxMs := 15

	// Create client with jitter enabled
	client, err := NewClient("sdk_test_api_key_1234567890",
		WithOffline(),
		WithPollingDisabled(),
		WithBootstrap(map[string]any{
			"bool-flag":   true,
			"string-flag": "hello",
			"number-flag": 42.0,
			"int-flag":    123.0,
			"json-flag":   map[string]any{"key": "value"},
		}),
		WithEvaluationJitter(true, minMs, maxMs),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() { _ = client.Close() }()

	// Test all evaluation methods apply jitter
	testCases := []struct {
		name     string
		evaluate func()
	}{
		{
			name: "GetBooleanValue",
			evaluate: func() {
				client.GetBooleanValue("bool-flag", false)
			},
		},
		{
			name: "GetStringValue",
			evaluate: func() {
				client.GetStringValue("string-flag", "default")
			},
		},
		{
			name: "GetNumberValue",
			evaluate: func() {
				client.GetNumberValue("number-flag", 0.0)
			},
		},
		{
			name: "GetIntValue",
			evaluate: func() {
				client.GetIntValue("int-flag", 0)
			},
		},
		{
			name: "GetJSONValue",
			evaluate: func() {
				client.GetJSONValue("json-flag", nil)
			},
		},
		{
			name: "Evaluate",
			evaluate: func() {
				client.Evaluate("bool-flag")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			tc.evaluate()
			elapsed := time.Since(start)

			if elapsed < time.Duration(minMs)*time.Millisecond {
				t.Errorf("Expected %s to take at least %dms with jitter, got %v", tc.name, minMs, elapsed)
			}
		})
	}
}
