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

## Architecture

The SDK is organized into clean, modular packages:

```
flagkit-go/
├── flagkit.go          # Main entry point and public API
├── exports.go          # Internal type exports for testing
├── client/             # Client implementation
├── config/             # Configuration and options
├── errors/             # Error types and sanitization
├── security/           # Security utilities (PII, signing, verification)
├── types/              # Public type definitions
├── internal/           # Private implementation details
│   ├── core/           # Cache, polling, event queue
│   ├── http/           # HTTP client, circuit breaker, retry
│   ├── persistence/    # Event persistence
│   ├── storage/        # Encrypted storage
│   └── types/          # Internal types
└── tests/              # Integration tests
```

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

## Security Features

### PII Detection

The SDK can detect and warn about potential PII (Personally Identifiable Information) in contexts and events:

```go
// Enable strict PII mode - returns errors instead of warnings
client, err := flagkit.NewClient("sdk_...",
    flagkit.WithStrictPIIMode(),
)

// Attributes containing PII will trigger an error
err = client.Identify("user-123", map[string]any{
    "email": "user@example.com",  // PII detected!
})
if err != nil {
    // Handle PII error
}

// Use private attributes to mark fields as intentionally containing PII
ctx := flagkit.NewContext("user-123").
    WithCustom("email", "user@example.com").
    WithPrivateAttribute("email")  // Marks email as intentionally private
```

### Request Signing

POST requests to the FlagKit API are signed with HMAC-SHA256 for integrity:

```go
// Enabled by default, can be disabled if needed
client, err := flagkit.NewClient("sdk_...",
    flagkit.WithRequestSigning(false),  // Disable signing
)
```

### Bootstrap Signature Verification

Verify bootstrap data integrity using HMAC signatures:

```go
// Create signed bootstrap data
bootstrap, err := flagkit.CreateBootstrapSignature(map[string]any{
    "feature-a": true,
    "feature-b": "value",
}, "sdk_your_api_key")

// Use signed bootstrap with verification
client, err := flagkit.NewClient("sdk_...",
    flagkit.WithSignedBootstrap(bootstrap),
    flagkit.WithBootstrapVerification(true, 24*time.Hour, "error"),
    // OnFailure options: "warn" (default), "error", "ignore"
)
```

### Cache Encryption

Enable AES-256-GCM encryption for cached flag data:

```go
client, err := flagkit.NewClient("sdk_...",
    flagkit.WithCacheEncryption(),
)
```

### Evaluation Jitter (Timing Attack Protection)

Add random delays to flag evaluations to prevent cache timing attacks:

```go
client, err := flagkit.NewClient("sdk_...",
    flagkit.WithEvaluationJitter(true, 5, 15),  // enabled, minMs, maxMs
)
```

### Error Sanitization

Automatically redact sensitive information from error messages:

```go
client, err := flagkit.NewClient("sdk_...",
    flagkit.WithErrorSanitization(true),
)
// Errors will have paths, IPs, API keys, and emails redacted
```

## Event Persistence

Enable crash-resilient event persistence to prevent data loss:

```go
client, err := flagkit.NewClient("sdk_...",
    flagkit.WithPersistEvents(true),
    flagkit.WithEventStoragePath("/path/to/storage"),  // Optional, defaults to temp dir
    flagkit.WithMaxPersistedEvents(10000),             // Optional, default 10000
    flagkit.WithPersistenceFlushInterval(time.Second), // Optional, default 1s
)
```

Events are written to disk before being sent, and automatically recovered on restart.

## Key Rotation

Support seamless API key rotation:

```go
client, err := flagkit.NewClient("sdk_primary_key",
    flagkit.WithSecondaryAPIKey("sdk_secondary_key"),
)
// SDK will automatically failover to secondary key on 401 errors
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

### Error Codes

| Code | Description |
|------|-------------|
| `INIT_FAILED` | SDK initialization failed |
| `INIT_TIMEOUT` | Initialization timed out |
| `INIT_ALREADY_INITIALIZED` | SDK already initialized |
| `INIT_NOT_INITIALIZED` | SDK not initialized |
| `NETWORK_ERROR` | Network request failed |
| `AUTH_INVALID_KEY` | Invalid API key |
| `SECURITY_PII_DETECTED` | PII detected in strict mode |
| `SECURITY_LOCAL_PORT_IN_PRODUCTION` | Local port used in production |
| `SECURITY_SIGNATURE_INVALID` | Bootstrap signature verification failed |

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `WithBaseURL` | string | `https://api.flagkit.dev/api/v1` | API base URL |
| `WithPollingInterval` | Duration | 30s | Polling interval |
| `WithPollingDisabled` | - | - | Disable background polling |
| `WithCacheTTL` | Duration | 5m | Cache TTL |
| `WithCacheDisabled` | - | - | Disable local caching |
| `WithCacheEncryption` | - | - | Enable AES-256-GCM cache encryption |
| `WithOffline` | - | - | Offline mode |
| `WithLocalPort` | int | 0 | Local development port (0 = production) |
| `WithTimeout` | Duration | 5s | HTTP request timeout |
| `WithRetries` | int | 3 | Number of retry attempts |
| `WithBootstrap` | map | {} | Initial flag values |
| `WithSignedBootstrap` | *BootstrapConfig | nil | Signed bootstrap values |
| `WithBootstrapVerification` | bool, Duration, string | true, 24h, "warn" | Bootstrap verification settings |
| `WithDebug` | - | - | Enable debug logging |
| `WithLogger` | Logger | - | Custom logger |
| `WithOnReady` | func() | - | Ready callback |
| `WithOnError` | func(error) | - | Error callback |
| `WithOnUpdate` | func([]FlagState) | - | Update callback |
| `WithSecondaryAPIKey` | string | "" | Secondary key for rotation |
| `WithStrictPIIMode` | - | - | Error on PII detection |
| `WithRequestSigning` | bool | true | Enable request signing |
| `WithPersistEvents` | bool | false | Enable event persistence |
| `WithEventStoragePath` | string | temp dir | Event storage directory |
| `WithMaxPersistedEvents` | int | 10000 | Max persisted events |
| `WithPersistenceFlushInterval` | Duration | 1s | Persistence flush interval |
| `WithEvaluationJitter` | bool, int, int | false, 5, 15 | Timing attack protection |
| `WithErrorSanitization` | bool | false | Redact sensitive info from errors |

## Testing

```go
// Use offline mode with bootstrap values
client, _ := flagkit.NewClient("sdk_test",
    flagkit.WithOffline(),
    flagkit.WithBootstrap(map[string]any{
        "feature-flag": true,
    }),
)
client.Initialize()

// With signed bootstrap for verification testing
bootstrap, _ := flagkit.CreateBootstrapSignature(map[string]any{
    "feature-flag": true,
}, "sdk_test")

client, _ := flagkit.NewClient("sdk_test",
    flagkit.WithOffline(),
    flagkit.WithSignedBootstrap(bootstrap),
)
```

## Thread Safety

All SDK methods are safe for concurrent use from multiple goroutines. The client uses internal synchronization to ensure thread-safe access to:

- Flag cache
- Event queue
- Context management
- Polling state

## License

MIT License - see [LICENSE](LICENSE) for details.
