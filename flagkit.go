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
