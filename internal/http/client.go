// Package http contains HTTP client code for FlagKit SDK.
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/flagkit/flagkit-go/internal/errors"
	"github.com/flagkit/flagkit-go/internal/types"
)

// SDKVersion should be set by the main package.
var SDKVersion = "1.0.0"

// Client handles HTTP communication with the FlagKit API.
type Client struct {
	baseURL        string
	apiKey         string
	timeout        time.Duration
	client         *http.Client
	retry          *RetryConfig
	circuitBreaker *CircuitBreaker
	logger         types.Logger
}

// ClientConfig contains HTTP client configuration.
type ClientConfig struct {
	BaseURL        string
	APIKey         string
	Timeout        time.Duration
	Retry          *RetryConfig
	CircuitBreaker *CircuitBreakerConfig
	Logger         types.Logger
}

// Response represents an HTTP response.
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Data       interface{}
}

// NewClient creates a new HTTP client.
func NewClient(config *ClientConfig) *Client {
	client := &Client{
		baseURL: config.BaseURL,
		apiKey:  config.APIKey,
		timeout: config.Timeout,
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

// Get performs a GET request.
func (c *Client) Get(path string) (*Response, error) {
	return c.request(context.Background(), http.MethodGet, path, nil)
}

// GetWithContext performs a GET request with context.
func (c *Client) GetWithContext(ctx context.Context, path string) (*Response, error) {
	return c.request(ctx, http.MethodGet, path, nil)
}

// Post performs a POST request.
func (c *Client) Post(path string, body interface{}) (*Response, error) {
	return c.request(context.Background(), http.MethodPost, path, body)
}

// PostWithContext performs a POST request with context.
func (c *Client) PostWithContext(ctx context.Context, path string, body interface{}) (*Response, error) {
	return c.request(ctx, http.MethodPost, path, body)
}

// request performs an HTTP request with retry and circuit breaker.
func (c *Client) request(ctx context.Context, method, path string, body interface{}) (*Response, error) {
	// Check circuit breaker
	if !c.circuitBreaker.Allow() {
		return nil, errors.NewError(errors.ErrCircuitOpen, "circuit breaker is open")
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
	return nil, errors.NetworkError(errors.ErrNetworkRetryLimit, "max retries exceeded", lastErr)
}

// doRequest performs a single HTTP request.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*Response, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, errors.NewErrorWithCause(errors.ErrNetworkError, "failed to marshal request body", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, errors.NewErrorWithCause(errors.ErrNetworkError, "failed to create request", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("FlagKit-Go/%s", SDKVersion))

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.NewErrorWithCause(errors.ErrNetworkError, "request failed", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewErrorWithCause(errors.ErrNetworkError, "failed to read response body", err)
	}

	response := &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       respBody,
	}

	// Parse JSON response
	if len(respBody) > 0 {
		var data interface{}
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
func (c *Client) handleErrorResponse(statusCode int, body []byte) error {
	message := string(body)
	if message == "" {
		message = http.StatusText(statusCode)
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return errors.NewError(errors.ErrAuthUnauthorized, message)
	case http.StatusForbidden:
		return errors.NewError(errors.ErrAuthInvalidKey, message)
	case http.StatusNotFound:
		return errors.NewError(errors.ErrEvalFlagNotFound, message)
	case http.StatusTooManyRequests:
		return errors.NewError(errors.ErrNetworkRetryLimit, message)
	case http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout:
		return errors.NewError(errors.ErrNetworkError, message)
	default:
		return errors.NewError(errors.ErrNetworkError, fmt.Sprintf("HTTP %d: %s", statusCode, message))
	}
}

// isRetryable checks if an error is retryable.
func (c *Client) isRetryable(err error) bool {
	if fkErr, ok := err.(*errors.FlagKitError); ok {
		switch fkErr.Code {
		case errors.ErrNetworkError, errors.ErrNetworkTimeout:
			return true
		}
	}
	return false
}

// Close closes the HTTP client.
func (c *Client) Close() error {
	c.client.CloseIdleConnections()
	return nil
}
