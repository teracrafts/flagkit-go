package errors_test

import (
	"testing"

	"github.com/teracrafts/flagkit-go/config"
	. "github.com/teracrafts/flagkit-go/errors"
)

func TestWithErrorSanitization(t *testing.T) {
	opts := config.DefaultOptions("sdk_test_api_key_12345678")

	config.WithErrorSanitization(true)(opts)

	if !opts.ErrorSanitization.Enabled {
		t.Error("Expected ErrorSanitization.Enabled to be true")
	}
}

func TestWithErrorSanitizationConfig(t *testing.T) {
	opts := config.DefaultOptions("sdk_test_api_key_12345678")

	cfg := ErrorSanitizationConfig{
		Enabled:          true,
		PreserveOriginal: true,
	}
	config.WithErrorSanitizationConfig(cfg)(opts)

	if !opts.ErrorSanitization.Enabled {
		t.Error("Expected ErrorSanitization.Enabled to be true")
	}
	if !opts.ErrorSanitization.PreserveOriginal {
		t.Error("Expected ErrorSanitization.PreserveOriginal to be true")
	}
}
