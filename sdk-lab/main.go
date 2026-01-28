// FlagKit Go SDK Lab
//
// Internal verification script for SDK functionality.
// Run with: go run ./sdk-lab
package main

import (
	"fmt"
	"os"

	flagkit "github.com/flagkit/flagkit-go"
)

const (
	pass = "\033[32m[PASS]\033[0m"
	fail = "\033[31m[FAIL]\033[0m"
)

func main() {
	fmt.Println("=== FlagKit Go SDK Lab ===\n")

	passed := 0
	failed := 0

	// Test 1: Initialization with offline mode + bootstrap
	fmt.Println("Testing initialization...")
	client, err := flagkit.Initialize(
		"sdk_lab_test_key",
		flagkit.WithOffline(),
		flagkit.WithBootstrap(map[string]any{
			"lab-bool":   true,
			"lab-string": "Hello Lab",
			"lab-number": float64(42),
			"lab-json":   map[string]any{"nested": true, "count": float64(100)},
		}),
	)
	if err != nil {
		fmt.Printf("%s Initialization - %v\n", fail, err)
		failed++
		os.Exit(1)
	}
	defer flagkit.Shutdown()

	client.WaitForReady()
	if client.IsReady() {
		fmt.Printf("%s Initialization\n", pass)
		passed++
	} else {
		fmt.Printf("%s Initialization - client not ready\n", fail)
		failed++
	}

	// Test 2: Boolean flag evaluation
	fmt.Println("\nTesting flag evaluation...")
	boolValue := client.GetBooleanValue("lab-bool", false)
	if boolValue == true {
		fmt.Printf("%s Boolean flag evaluation\n", pass)
		passed++
	} else {
		fmt.Printf("%s Boolean flag - expected true, got %v\n", fail, boolValue)
		failed++
	}

	// Test 3: String flag evaluation
	stringValue := client.GetStringValue("lab-string", "")
	if stringValue == "Hello Lab" {
		fmt.Printf("%s String flag evaluation\n", pass)
		passed++
	} else {
		fmt.Printf("%s String flag - expected \"Hello Lab\", got \"%s\"\n", fail, stringValue)
		failed++
	}

	// Test 4: Number flag evaluation
	numberValue := client.GetNumberValue("lab-number", 0)
	if numberValue == 42 {
		fmt.Printf("%s Number flag evaluation\n", pass)
		passed++
	} else {
		fmt.Printf("%s Number flag - expected 42, got %v\n", fail, numberValue)
		failed++
	}

	// Test 5: JSON flag evaluation
	jsonValue := client.GetJSONValue("lab-json", map[string]any{"nested": false, "count": float64(0)})
	nested, _ := jsonValue["nested"].(bool)
	count, _ := jsonValue["count"].(float64)
	if nested == true && count == 100 {
		fmt.Printf("%s JSON flag evaluation\n", pass)
		passed++
	} else {
		fmt.Printf("%s JSON flag - unexpected value: %v\n", fail, jsonValue)
		failed++
	}

	// Test 6: Default value for missing flag
	missingValue := client.GetBooleanValue("non-existent", true)
	if missingValue == true {
		fmt.Printf("%s Default value for missing flag\n", pass)
		passed++
	} else {
		fmt.Printf("%s Missing flag - expected default true, got %v\n", fail, missingValue)
		failed++
	}

	// Test 7: Context management - identify
	fmt.Println("\nTesting context management...")
	err = client.Identify("lab-user-123", map[string]any{"plan": "premium", "country": "US"})
	if err != nil {
		fmt.Printf("%s identify() - %v\n", fail, err)
		failed++
	} else {
		context := client.GetContext()
		if context != nil && context.UserID == "lab-user-123" {
			fmt.Printf("%s Identify()\n", pass)
			passed++
		} else {
			fmt.Printf("%s Identify() - context not set correctly\n", fail)
			failed++
		}
	}

	// Test 8: Context management - GetContext
	context := client.GetContext()
	if context != nil && context.Custom != nil {
		if plan, ok := context.Custom["plan"].(string); ok && plan == "premium" {
			fmt.Printf("%s GetContext()\n", pass)
			passed++
		} else {
			fmt.Printf("%s GetContext() - custom attributes missing\n", fail)
			failed++
		}
	} else {
		fmt.Printf("%s GetContext() - context is nil\n", fail)
		failed++
	}

	// Test 9: Context management - reset
	client.Reset()
	resetContext := client.GetContext()
	if resetContext == nil || resetContext.UserID == "" {
		fmt.Printf("%s Reset()\n", pass)
		passed++
	} else {
		fmt.Printf("%s Reset() - context not cleared\n", fail)
		failed++
	}

	// Test 10: Event tracking
	fmt.Println("\nTesting event tracking...")
	err = client.Track("lab_verification", map[string]any{"sdk": "go", "version": "1.0.0"})
	if err != nil {
		fmt.Printf("%s Track() - %v\n", fail, err)
		failed++
	} else {
		fmt.Printf("%s Track()\n", pass)
		passed++
	}

	// Test 11: Flush (offline mode - no-op but should not panic)
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("%s Flush() - panic: %v\n", fail, r)
				failed++
			}
		}()
		client.Flush()
		fmt.Printf("%s Flush()\n", pass)
		passed++
	}()

	// Test 12: Cleanup
	fmt.Println("\nTesting cleanup...")
	err = client.Close()
	if err != nil {
		fmt.Printf("%s Close() - %v\n", fail, err)
		failed++
	} else {
		fmt.Printf("%s Close()\n", pass)
		passed++
	}

	// Summary
	fmt.Println("\n" + "========================================")
	fmt.Printf("Results: %d passed, %d failed\n", passed, failed)
	fmt.Println("========================================")

	if failed > 0 {
		fmt.Println("\n\033[31mSome verifications failed!\033[0m")
		os.Exit(1)
	} else {
		fmt.Println("\n\033[32mAll verifications passed!\033[0m")
		os.Exit(0)
	}
}
