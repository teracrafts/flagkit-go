# FlagKit Go SDK

Official Go SDK for [FlagKit](https://flagkit.dev) - Feature flag management made simple.

## Installation

```bash
go get github.com/flagkit/flagkit-go
```

## Requirements

- Go 1.21+

## Quick Start

```go
package main

import (
    "log"
    "github.com/flagkit/flagkit-go"
)

func main() {
    // Initialize the SDK
    client, err := flagkit.Initialize("sdk_your_api_key")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Identify a user
    client.Identify("user-123", map[string]interface{}{
        "email": "user@example.com",
        "plan":  "premium",
    })

    // Evaluate flags
    darkMode := client.GetBooleanValue("dark-mode", false)
    welcomeMsg := client.GetStringValue("welcome-message", "Hello!")
    maxItems := client.GetIntValue("max-items", 10)
    config := client.GetJSONValue("feature-config", map[string]interface{}{})

    // Get full evaluation details
    result := client.Evaluate("dark-mode")
    log.Printf("Value: %v, Reason: %s", result.Value, result.Reason)

    // Track custom events
    client.Track("button_clicked", map[string]interface{}{"button": "signup"})
}
```

## Features

- **Type-safe evaluation** - Boolean, string, number, and JSON flag types
- **Local caching** - Fast evaluations with configurable TTL
- **Background polling** - Automatic flag updates
- **Event tracking** - Analytics with batching
- **Resilient** - Circuit breaker, retry with backoff, offline support
- **Thread-safe** - Safe for concurrent use

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
    flagkit.WithBootstrap(map[string]interface{}{
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
config := client.GetJSONValue("config", map[string]interface{}{"enabled": false})

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
client.Identify("user-123", map[string]interface{}{"email": "user@example.com"})

// Reset to anonymous
client.Reset()

// Pass context to evaluation
enabled := client.GetBooleanValue("feature-flag", false, ctx)
```

### Event Tracking

```go
// Track custom event
client.Track("purchase", map[string]interface{}{
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

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `WithBaseURL` | string | `https://api.flagkit.dev/api/v1` | API base URL |
| `WithPollingInterval` | Duration | 30s | Polling interval |
| `WithPollingDisabled` | - | - | Disable background polling |
| `WithCacheTTL` | Duration | 5m | Cache TTL |
| `WithCacheDisabled` | - | - | Disable local caching |
| `WithOffline` | - | - | Offline mode |
| `WithTimeout` | Duration | 5s | HTTP request timeout |
| `WithRetries` | int | 3 | Number of retry attempts |
| `WithBootstrap` | map | {} | Initial flag values |
| `WithDebug` | - | - | Enable debug logging |
| `WithLogger` | Logger | - | Custom logger |
| `WithOnReady` | func() | - | Ready callback |
| `WithOnError` | func(error) | - | Error callback |
| `WithOnUpdate` | func([]FlagState) | - | Update callback |

## Testing

```go
// Use offline mode with bootstrap values
client, _ := flagkit.NewClient("sdk_test",
    flagkit.WithOffline(),
    flagkit.WithBootstrap(map[string]interface{}{
        "feature-flag": true,
    }),
)
client.Initialize()

// Or mock the HTTP client
```

## License

MIT License - see [LICENSE](LICENSE) for details.
