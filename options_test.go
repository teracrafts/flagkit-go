package flagkit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions("sdk_test_key")

	assert.Equal(t, "sdk_test_key", opts.APIKey)
	assert.Equal(t, "https://api.flagkit.dev/api/v1", opts.BaseURL)
	assert.Equal(t, 30*time.Second, opts.PollingInterval)
	assert.Equal(t, 5*time.Minute, opts.CacheTTL)
	assert.Equal(t, 5*time.Second, opts.Timeout)
	assert.Equal(t, 3, opts.Retries)
	assert.True(t, opts.EnablePolling)
	assert.False(t, opts.Offline)
	assert.False(t, opts.Debug)
}

func TestOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    *Options
		wantErr bool
	}{
		{
			name:    "valid options",
			opts:    DefaultOptions("sdk_test_key"),
			wantErr: false,
		},
		{
			name: "empty api key",
			opts: &Options{
				APIKey:  "",
				BaseURL: "https://api.flagkit.dev/api/v1",
			},
			wantErr: true,
		},
		{
			name: "invalid api key prefix",
			opts: &Options{
				APIKey:  "invalid_key",
				BaseURL: "https://api.flagkit.dev/api/v1",
			},
			wantErr: true,
		},
		{
			name: "empty base url",
			opts: &Options{
				APIKey:  "sdk_test_key",
				BaseURL: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWithBaseURL(t *testing.T) {
	opts := DefaultOptions("sdk_test_key")
	WithBaseURL("https://custom.api.com")(opts)

	assert.Equal(t, "https://custom.api.com", opts.BaseURL)
}

func TestWithPollingInterval(t *testing.T) {
	opts := DefaultOptions("sdk_test_key")
	WithPollingInterval(60 * time.Second)(opts)

	assert.Equal(t, 60*time.Second, opts.PollingInterval)
}

func TestWithPollingDisabled(t *testing.T) {
	opts := DefaultOptions("sdk_test_key")
	WithPollingDisabled()(opts)

	assert.False(t, opts.EnablePolling)
}

func TestWithCacheTTL(t *testing.T) {
	opts := DefaultOptions("sdk_test_key")
	WithCacheTTL(10 * time.Minute)(opts)

	assert.Equal(t, 10*time.Minute, opts.CacheTTL)
}

func TestWithCacheDisabled(t *testing.T) {
	opts := DefaultOptions("sdk_test_key")
	WithCacheDisabled()(opts)

	assert.False(t, opts.CacheEnabled)
}

func TestWithOffline(t *testing.T) {
	opts := DefaultOptions("sdk_test_key")
	WithOffline()(opts)

	assert.True(t, opts.Offline)
}

func TestWithTimeout(t *testing.T) {
	opts := DefaultOptions("sdk_test_key")
	WithTimeout(10 * time.Second)(opts)

	assert.Equal(t, 10*time.Second, opts.Timeout)
}

func TestWithRetries(t *testing.T) {
	opts := DefaultOptions("sdk_test_key")
	WithRetries(5)(opts)

	assert.Equal(t, 5, opts.Retries)
}

func TestWithBootstrap(t *testing.T) {
	opts := DefaultOptions("sdk_test_key")
	bootstrap := map[string]interface{}{
		"feature-flag": true,
		"variant":      "A",
	}
	WithBootstrap(bootstrap)(opts)

	assert.Equal(t, true, opts.Bootstrap["feature-flag"])
	assert.Equal(t, "A", opts.Bootstrap["variant"])
}

func TestWithDebug(t *testing.T) {
	opts := DefaultOptions("sdk_test_key")
	WithDebug()(opts)

	assert.True(t, opts.Debug)
}

func TestWithLogger(t *testing.T) {
	opts := DefaultOptions("sdk_test_key")
	logger := &NullLogger{}
	WithLogger(logger)(opts)

	assert.Equal(t, logger, opts.Logger)
}

func TestWithCallbacks(t *testing.T) {
	opts := DefaultOptions("sdk_test_key")

	readyCalled := false
	errorCalled := false
	updateCalled := false

	WithOnReady(func() { readyCalled = true })(opts)
	WithOnError(func(err error) { errorCalled = true })(opts)
	WithOnUpdate(func(flags []FlagState) { updateCalled = true })(opts)

	opts.OnReady()
	opts.OnError(nil)
	opts.OnUpdate(nil)

	assert.True(t, readyCalled)
	assert.True(t, errorCalled)
	assert.True(t, updateCalled)
}
