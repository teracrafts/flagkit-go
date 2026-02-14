# FlagKit Go SDK Lab

Internal verification script for the Go SDK.

## Purpose

This lab folder contains scripts to verify SDK functionality during development. It helps catch integration issues before committing changes.

## Usage

From the SDK root directory:

```bash
go run ./sdk-lab
```

Or from this directory:

```bash
go run .
```

## What it Tests

1. **Initialization** - Offline mode with bootstrap data
2. **Flag Evaluation** - Boolean, string, number, and JSON flags
3. **Default Values** - Returns defaults for missing flags
4. **Context Management** - Identify(), GetContext(), Reset()
5. **Event Tracking** - Track(), Flush()
6. **Cleanup** - Close()

## Expected Output

```
=== FlagKit Go SDK Lab ===

Testing initialization...
[PASS] Initialization

Testing flag evaluation...
[PASS] Boolean flag evaluation
[PASS] String flag evaluation
[PASS] Number flag evaluation
[PASS] JSON flag evaluation
[PASS] Default value for missing flag

Testing context management...
[PASS] Identify()
[PASS] GetContext()
[PASS] Reset()

Testing event tracking...
[PASS] Track()
[PASS] Flush()

Testing cleanup...
[PASS] Close()

========================================
Results: 12 passed, 0 failed
========================================

All verifications passed!
```

## Note

This folder uses a separate `go.mod` with a `replace` directive to reference the parent SDK module. It is not included in the distributed package.

## Mode Routing
Use `FLAGKIT_MODE` to control API target during SDK Lab runs:
- `local` -> `https://api.flagkit.on/api/v1`
- `beta` -> `https://api.beta.flagkit.dev/api/v1`
- `carbon` (default) -> `https://api.flagkit.dev/api/v1`
