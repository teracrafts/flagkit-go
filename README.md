# FlagKit Go SDK

Official Go SDK for [FlagKit](https://flagkit.dev) - Feature flag management made simple.

## Installation

```bash
go get github.com/teracrafts/flagkit-go
```

## Requirements

- Go 1.21+

## Quick Start

```go
package main

import (
    "log"
    "github.com/teracrafts/flagkit-go"
)

func main() {
    // Initialize the SDK
    client, err := flagkit.Initialize("sdk_your_api_key")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Identify a user
    client.Identify("user-123", map[string]any{
        "plan": "premium",
    })

    // Evaluate flags
    darkMode := client.GetBooleanValue("dark-mode", false)
    welcomeMsg := client.GetStringValue("welcome-message", "Hello!")
    maxItems := client.GetIntValue("max-items", 10)
    config := client.GetJSONValue("feature-config", map[string]any{})

    // Get full evaluation details
    result := client.Evaluate("dark-mode")
    log.Printf("Value: %v, Reason: %s", result.Value, result.Reason)

    // Track custom events
    client.Track("button_clicked", map[string]any{"button": "signup"})
}
```

## Features

- **Type-safe evaluation** - Boolean, string, number, and JSON flag types
- **Local caching** - Fast evaluations with configurable TTL and optional encryption
- **Background polling** - Automatic flag updates with jitter
- **Event tracking** - Analytics with batching and crash-resilient persistence
- **Resilient** - Circuit breaker, retry with exponential backoff, offline support
- **Thread-safe** - Safe for concurrent use
- **Security** - PII detection, request signing, bootstrap verification, timing attack protection

## API Reference

### Initialization

```go
// Using the singleton (recommended for most apps)
client, err := flagkit.Initialize("sdk_...",
    flagkit.WithPollingInterval(30 * time.Second),
    flagkit.WithCacheTTL(5 * time.Minute),
    flagkit.WithDebug(),
)

// Or create a client directly
client, err := flagkit.NewClient("sdk_...",
    flagkit.WithBaseURL("https://api.flagkit.dev/api/v1"),
    flagkit.WithOffline(),
    flagkit.WithBootstrap(map[string]any{
        "feature-flag": true,
    }),
)
if err != nil {
    log.Fatal(err)
}
err = client.Initialize()
```

### Flag Evaluation

```go
// Boolean flags
enabled := client.GetBooleanValue("feature-flag", false)

// String flags
variant := client.GetStringValue("button-text", "Click")

// Number flags (float64)
limit := client.GetNumberValue("rate-limit", 100.0)

// Integer flags
count := client.GetIntValue("max-retries", 3)

// JSON flags
config := client.GetJSONValue("config", map[string]any{"enabled": false})

// Full evaluation result
result := client.Evaluate("feature-flag")
// result.FlagKey, result.Value, result.Enabled, result.Reason, result.Version

// Evaluate all flags
allResults := client.EvaluateAll()

// Check flag existence
if client.HasFlag("my-flag") {
    // ...
}

// Get all flag keys
keys := client.GetAllFlagKeys()
```

### Context Management

```go
// Create context
ctx := flagkit.NewContext("user-123").
    WithEmail("user@example.com").
    WithCountry("US").
    WithCustom("plan", "premium").
    WithPrivateAttribute("email")

// Set global context
client.SetContext(ctx)

// Get current context
current := client.GetContext()

// Clear context
client.ClearContext()

// Identify user (shorthand)
client.Identify("user-123", map[string]any{"plan": "premium"})

// Reset to anonymous
client.Reset()

// Pass context to evaluation
enabled := client.GetBooleanValue("feature-flag", false, ctx)
```

### Event Tracking

```go
// Track custom event
client.Track("purchase", map[string]any{
    "amount":     99.99,
    "currency":   "USD",
    "product_id": "prod-123",
})

// Force flush pending events
client.Flush()
```

### Lifecycle

```go
// Check if SDK is ready
if client.IsReady() {
    // ...
}

// Wait for ready
client.WaitForReady()

// Force refresh flags from server
client.Refresh()

// Close SDK and cleanup
client.Close()

// Using singleton
flagkit.Shutdown()
```

## Security

The SDK includes built-in security features that can be enabled through configuration options, including PII detection, request signing, bootstrap signature verification, cache encryption, evaluation jitter for timing attack protection, and error sanitization.

## Error Handling

```go
client, err := flagkit.Initialize("sdk_...")
if err != nil {
    if fkErr, ok := err.(*flagkit.FlagKitError); ok {
        log.Printf("FlagKit error [%s]: %s", fkErr.Code, fkErr.Message)
        if fkErr.Recoverable {
            // Retry logic
        }
    }
    log.Fatal(err)
}
```

## Thread Safety

All SDK methods are safe for concurrent use from multiple goroutines. The client uses internal synchronization to ensure thread-safe access to:

- Flag cache
- Event queue
- Context management
- Polling state

## License

MIT License - see [LICENSE](LICENSE) for details.
