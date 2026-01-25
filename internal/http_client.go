package internal

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// SDKVersion should be set by the main package.
var SDKVersion = "1.0.0"

// defaultBaseURL is the internal base URL for the FlagKit API.
const defaultBaseURL = "https://api.flagkit.dev/api/v1"

// HTTPClient handles HTTP communication with the FlagKit API.
type HTTPClient struct {
	baseURL                string
	apiKey                 string
	secondaryAPIKey        string
	currentAPIKey          string
	keyRotationTimestamp   *time.Time
	keyRotationGracePeriod time.Duration
	enableRequestSigning   bool
	timeout                time.Duration
	client                 *http.Client
	retry                  *RetryConfig
	circuitBreaker         *CircuitBreaker
	logger                 Logger
	mu                     sync.RWMutex
}

// HTTPClientConfig contains HTTP client configuration.
type HTTPClientConfig struct {
	APIKey                 string
	SecondaryAPIKey        string
	KeyRotationGracePeriod time.Duration
	EnableRequestSigning   bool
	Timeout                time.Duration
	Retry                  *RetryConfig
	CircuitBreaker         *CircuitBreakerConfig
	Logger                 Logger
	LocalPort              int
}

// HTTPResponse represents an HTTP response.
type HTTPResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Data       any
}

// NewHTTPClient creates a new HTTP client.
func NewHTTPClient(config *HTTPClientConfig) *HTTPClient {
	baseURL := defaultBaseURL
	if config.LocalPort > 0 {
		baseURL = fmt.Sprintf("http://localhost:%d/api/v1", config.LocalPort)
	}

	gracePeriod := config.KeyRotationGracePeriod
	if gracePeriod == 0 {
		gracePeriod = 5 * time.Minute
	}

	client := &HTTPClient{
		baseURL:                baseURL,
		apiKey:                 config.APIKey,
		secondaryAPIKey:        config.SecondaryAPIKey,
		currentAPIKey:          config.APIKey,
		keyRotationGracePeriod: gracePeriod,
		enableRequestSigning:   config.EnableRequestSigning,
		timeout:                config.Timeout,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		logger: config.Logger,
	}

	if config.Retry != nil {
		client.retry = config.Retry
	} else {
		client.retry = DefaultRetryConfig()
	}

	if config.CircuitBreaker != nil {
		client.circuitBreaker = NewCircuitBreaker(config.CircuitBreaker)
	} else {
		client.circuitBreaker = NewCircuitBreaker(DefaultCircuitBreakerConfig())
	}

	return client
}

// GetActiveAPIKey returns the currently active API key.
func (c *HTTPClient) GetActiveAPIKey() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentAPIKey
}

// GetKeyID returns the first 8 characters of the current API key.
func (c *HTTPClient) GetKeyID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.currentAPIKey) < 8 {
		return c.currentAPIKey
	}
	return c.currentAPIKey[:8]
}

// IsInKeyRotation returns true if key rotation is currently active.
func (c *HTTPClient) IsInKeyRotation() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.keyRotationTimestamp == nil {
		return false
	}

	elapsed := time.Since(*c.keyRotationTimestamp)
	return elapsed < c.keyRotationGracePeriod
}

// rotateToSecondaryKey attempts to rotate to the secondary API key.
// Returns true if rotation was successful.
func (c *HTTPClient) rotateToSecondaryKey() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.secondaryAPIKey == "" {
		return false
	}

	if c.currentAPIKey == c.secondaryAPIKey {
		// Already using secondary key
		return false
	}

	if c.logger != nil {
		c.logger.Info("Rotating to secondary API key due to authentication failure")
	}

	c.currentAPIKey = c.secondaryAPIKey
	now := time.Now()
	c.keyRotationTimestamp = &now
	return true
}

// generateHMACSHA256 generates an HMAC-SHA256 signature.
func generateHMACSHA256(message, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// createRequestSignature creates signature headers for a request body.
func (c *HTTPClient) createRequestSignature(body []byte) (signature string, timestamp int64, keyID string) {
	c.mu.RLock()
	apiKey := c.currentAPIKey
	c.mu.RUnlock()

	timestamp = time.Now().UnixMilli()
	message := strconv.FormatInt(timestamp, 10) + "." + string(body)
	signature = generateHMACSHA256(message, apiKey)

	if len(apiKey) >= 8 {
		keyID = apiKey[:8]
	} else {
		keyID = apiKey
	}

	return signature, timestamp, keyID
}

// Get performs a GET request.
func (c *HTTPClient) Get(path string) (*HTTPResponse, error) {
	return c.request(context.Background(), http.MethodGet, path, nil)
}

// GetWithContext performs a GET request with context.
func (c *HTTPClient) GetWithContext(ctx context.Context, path string) (*HTTPResponse, error) {
	return c.request(ctx, http.MethodGet, path, nil)
}

// Post performs a POST request with automatic signing.
func (c *HTTPClient) Post(path string, body any) (*HTTPResponse, error) {
	return c.postWithKeyRotation(context.Background(), path, body)
}

// PostWithContext performs a POST request with context and automatic signing.
func (c *HTTPClient) PostWithContext(ctx context.Context, path string, body any) (*HTTPResponse, error) {
	return c.postWithKeyRotation(ctx, path, body)
}

// postWithKeyRotation performs a POST request with key rotation support.
func (c *HTTPClient) postWithKeyRotation(ctx context.Context, path string, body any) (*HTTPResponse, error) {
	resp, err := c.request(ctx, http.MethodPost, path, body)

	// Handle 401 errors with key rotation
	if err != nil {
		if fkErr, ok := err.(*FlagKitError); ok {
			if fkErr.Code == ErrAuthUnauthorized || fkErr.Code == ErrAuthInvalidKey {
				if c.secondaryAPIKey != "" && c.rotateToSecondaryKey() {
					if c.logger != nil {
						c.logger.Debug("Retrying request with secondary API key")
					}
					return c.request(ctx, http.MethodPost, path, body)
				}
			}
		}
	}

	return resp, err
}

// request performs an HTTP request with retry and circuit breaker.
func (c *HTTPClient) request(ctx context.Context, method, path string, body any) (*HTTPResponse, error) {
	// Check circuit breaker
	if !c.circuitBreaker.Allow() {
		return nil, NewError(ErrCircuitOpen, "circuit breaker is open")
	}

	var lastErr error

	for attempt := 1; attempt <= c.retry.MaxAttempts; attempt++ {
		resp, err := c.doRequest(ctx, method, path, body)
		if err == nil {
			c.circuitBreaker.RecordSuccess()
			return resp, nil
		}

		lastErr = err

		// Check if error is retryable
		if !c.isRetryable(err) {
			c.circuitBreaker.RecordFailure()
			return nil, err
		}

		// Don't retry if we've exhausted attempts
		if attempt >= c.retry.MaxAttempts {
			break
		}

		// Calculate backoff
		delay := CalculateBackoff(attempt, c.retry)

		if c.logger != nil {
			c.logger.Debug("Retrying request",
				"attempt", attempt,
				"max_attempts", c.retry.MaxAttempts,
				"delay", delay,
				"error", err.Error(),
			)
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}

	c.circuitBreaker.RecordFailure()
	return nil, NetworkError(ErrNetworkRetryLimit, "max retries exceeded", lastErr)
}

// doRequest performs a single HTTP request.
func (c *HTTPClient) doRequest(ctx context.Context, method, path string, body any) (*HTTPResponse, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	var jsonBody []byte
	if body != nil {
		var err error
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return nil, NewErrorWithCause(ErrNetworkError, "failed to marshal request body", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, NewErrorWithCause(ErrNetworkError, "failed to create request", err)
	}

	// Get current API key
	c.mu.RLock()
	currentKey := c.currentAPIKey
	c.mu.RUnlock()

	// Set headers
	req.Header.Set("X-API-Key", currentKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("FlagKit-Go/%s", SDKVersion))
	req.Header.Set("X-FlagKit-SDK-Version", SDKVersion)
	req.Header.Set("X-FlagKit-SDK-Language", "go")

	// Add request signing for POST requests
	if method == http.MethodPost && c.enableRequestSigning && len(jsonBody) > 0 {
		signature, timestamp, keyID := c.createRequestSignature(jsonBody)
		req.Header.Set("X-Signature", signature)
		req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
		req.Header.Set("X-Key-Id", keyID)
	}

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, NewErrorWithCause(ErrNetworkError, "request failed", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewErrorWithCause(ErrNetworkError, "failed to read response body", err)
	}

	response := &HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       respBody,
	}

	// Parse JSON response
	if len(respBody) > 0 {
		var data any
		if err := json.Unmarshal(respBody, &data); err == nil {
			response.Data = data
		}
	}

	// Handle error status codes
	if resp.StatusCode >= 400 {
		return response, c.handleErrorResponse(resp.StatusCode, respBody)
	}

	return response, nil
}

// handleErrorResponse converts HTTP error responses to FlagKitErrors.
func (c *HTTPClient) handleErrorResponse(statusCode int, body []byte) error {
	message := string(body)
	if message == "" {
		message = http.StatusText(statusCode)
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return NewError(ErrAuthUnauthorized, message)
	case http.StatusForbidden:
		return NewError(ErrAuthInvalidKey, message)
	case http.StatusNotFound:
		return NewError(ErrEvalFlagNotFound, message)
	case http.StatusTooManyRequests:
		return NewError(ErrNetworkRetryLimit, message)
	case http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout:
		return NewError(ErrNetworkError, message)
	default:
		return NewError(ErrNetworkError, fmt.Sprintf("HTTP %d: %s", statusCode, message))
	}
}

// isRetryable checks if an error is retryable.
func (c *HTTPClient) isRetryable(err error) bool {
	if fkErr, ok := err.(*FlagKitError); ok {
		switch fkErr.Code {
		case ErrNetworkError, ErrNetworkTimeout:
			return true
		}
	}
	return false
}

// Close closes the HTTP client.
func (c *HTTPClient) Close() error {
	c.client.CloseIdleConnections()
	return nil
}
