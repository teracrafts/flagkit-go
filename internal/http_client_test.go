package internal

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewHTTPClient(t *testing.T) {
	t.Run("creates client with default config", func(t *testing.T) {
		client := NewHTTPClient(&HTTPClientConfig{
			APIKey:  "sdk_test_api_key_12345",
			Timeout: 5 * time.Second,
		})

		if client == nil {
			t.Error("expected client to be created")
		}

		if client.GetActiveAPIKey() != "sdk_test_api_key_12345" {
			t.Errorf("expected API key to be 'sdk_test_api_key_12345', got '%s'", client.GetActiveAPIKey())
		}
	})

	t.Run("creates client with secondary key", func(t *testing.T) {
		client := NewHTTPClient(&HTTPClientConfig{
			APIKey:          "sdk_primary_key_12345",
			SecondaryAPIKey: "sdk_secondary_key_12345",
			Timeout:         5 * time.Second,
		})

		if client.secondaryAPIKey != "sdk_secondary_key_12345" {
			t.Errorf("expected secondary API key to be set")
		}
	})

	t.Run("creates client with local port", func(t *testing.T) {
		client := NewHTTPClient(&HTTPClientConfig{
			APIKey:    "sdk_test_api_key_12345",
			Timeout:   5 * time.Second,
			LocalPort: 8080,
		})

		if client.baseURL != "http://localhost:8080/api/v1" {
			t.Errorf("expected base URL to be 'http://localhost:8080/api/v1', got '%s'", client.baseURL)
		}
	})
}

func TestHTTPClientGetKeyID(t *testing.T) {
	client := NewHTTPClient(&HTTPClientConfig{
		APIKey:  "sdk_test_api_key_12345",
		Timeout: 5 * time.Second,
	})

	keyID := client.GetKeyID()
	if keyID != "sdk_test" {
		t.Errorf("expected key ID 'sdk_test', got '%s'", keyID)
	}
}

func TestHTTPClientKeyRotation(t *testing.T) {
	t.Run("rotates to secondary key", func(t *testing.T) {
		client := NewHTTPClient(&HTTPClientConfig{
			APIKey:          "sdk_primary_key_12345",
			SecondaryAPIKey: "sdk_secondary_key_12345",
			Timeout:         5 * time.Second,
		})

		if client.GetActiveAPIKey() != "sdk_primary_key_12345" {
			t.Error("expected primary key initially")
		}

		rotated := client.rotateToSecondaryKey()
		if !rotated {
			t.Error("expected rotation to succeed")
		}

		if client.GetActiveAPIKey() != "sdk_secondary_key_12345" {
			t.Error("expected secondary key after rotation")
		}
	})

	t.Run("does not rotate when no secondary key", func(t *testing.T) {
		client := NewHTTPClient(&HTTPClientConfig{
			APIKey:  "sdk_primary_key_12345",
			Timeout: 5 * time.Second,
		})

		rotated := client.rotateToSecondaryKey()
		if rotated {
			t.Error("expected rotation to fail without secondary key")
		}
	})

	t.Run("does not rotate twice", func(t *testing.T) {
		client := NewHTTPClient(&HTTPClientConfig{
			APIKey:          "sdk_primary_key_12345",
			SecondaryAPIKey: "sdk_secondary_key_12345",
			Timeout:         5 * time.Second,
		})

		client.rotateToSecondaryKey()
		rotated := client.rotateToSecondaryKey()
		if rotated {
			t.Error("expected second rotation to fail")
		}
	})

	t.Run("tracks key rotation state", func(t *testing.T) {
		client := NewHTTPClient(&HTTPClientConfig{
			APIKey:                 "sdk_primary_key_12345",
			SecondaryAPIKey:        "sdk_secondary_key_12345",
			KeyRotationGracePeriod: 5 * time.Minute,
			Timeout:                5 * time.Second,
		})

		if client.IsInKeyRotation() {
			t.Error("expected not in key rotation initially")
		}

		client.rotateToSecondaryKey()

		if !client.IsInKeyRotation() {
			t.Error("expected to be in key rotation after rotating")
		}
	})
}

func TestHTTPClientRequestSigning(t *testing.T) {
	client := NewHTTPClient(&HTTPClientConfig{
		APIKey:               "sdk_test_api_key_12345",
		EnableRequestSigning: true,
		Timeout:              5 * time.Second,
	})

	body := []byte(`{"key":"value"}`)
	signature, timestamp, keyID := client.createRequestSignature(body)

	if signature == "" {
		t.Error("expected non-empty signature")
	}

	if timestamp == 0 {
		t.Error("expected non-zero timestamp")
	}

	if keyID != "sdk_test" {
		t.Errorf("expected key ID 'sdk_test', got '%s'", keyID)
	}
}

func TestHTTPClientPost(t *testing.T) {
	t.Run("adds signing headers when enabled", func(t *testing.T) {
		var receivedHeaders http.Header

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedHeaders = r.Header
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true}`))
		}))
		defer server.Close()

		client := NewHTTPClient(&HTTPClientConfig{
			APIKey:               "sdk_test_api_key_12345",
			EnableRequestSigning: true,
			Timeout:              5 * time.Second,
			LocalPort:            0,
		})
		// Override base URL for test
		client.baseURL = server.URL

		_, err := client.Post("/test", map[string]string{"key": "value"})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		if receivedHeaders.Get("X-Signature") == "" {
			t.Error("expected X-Signature header")
		}

		if receivedHeaders.Get("X-Timestamp") == "" {
			t.Error("expected X-Timestamp header")
		}

		if receivedHeaders.Get("X-Key-Id") == "" {
			t.Error("expected X-Key-Id header")
		}

		if receivedHeaders.Get("X-API-Key") != "sdk_test_api_key_12345" {
			t.Errorf("expected X-API-Key header to be 'sdk_test_api_key_12345', got '%s'", receivedHeaders.Get("X-API-Key"))
		}
	})

	t.Run("skips signing headers when disabled", func(t *testing.T) {
		var receivedHeaders http.Header

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedHeaders = r.Header
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true}`))
		}))
		defer server.Close()

		client := NewHTTPClient(&HTTPClientConfig{
			APIKey:               "sdk_test_api_key_12345",
			EnableRequestSigning: false,
			Timeout:              5 * time.Second,
		})
		client.baseURL = server.URL

		_, err := client.Post("/test", map[string]string{"key": "value"})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		if receivedHeaders.Get("X-Signature") != "" {
			t.Error("expected no X-Signature header when signing disabled")
		}
	})

	t.Run("rotates key on 401 error", func(t *testing.T) {
		callCount := 0

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"success":true}`))
			}
		}))
		defer server.Close()

		client := NewHTTPClient(&HTTPClientConfig{
			APIKey:          "sdk_primary_key_12345",
			SecondaryAPIKey: "sdk_secondary_key_12345",
			Timeout:         5 * time.Second,
			Retry:           &RetryConfig{MaxAttempts: 1},
		})
		client.baseURL = server.URL

		_, err := client.Post("/test", map[string]string{"key": "value"})
		if err != nil {
			t.Fatalf("request should succeed after key rotation: %v", err)
		}

		if callCount != 2 {
			t.Errorf("expected 2 calls (original + retry), got %d", callCount)
		}

		if client.GetActiveAPIKey() != "sdk_secondary_key_12345" {
			t.Error("expected client to use secondary key after rotation")
		}
	})
}

func TestHTTPClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	client := NewHTTPClient(&HTTPClientConfig{
		APIKey:  "sdk_test_api_key_12345",
		Timeout: 5 * time.Second,
		Retry:   &RetryConfig{MaxAttempts: 1},
	})
	client.baseURL = server.URL

	resp, err := client.Get("/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}
