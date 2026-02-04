package core

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/flagkit/flagkit-go/internal/types"
)

// StreamingState represents the connection state.
type StreamingState string

const (
	StreamingStateDisconnected StreamingState = "disconnected"
	StreamingStateConnecting   StreamingState = "connecting"
	StreamingStateConnected    StreamingState = "connected"
	StreamingStateReconnecting StreamingState = "reconnecting"
	StreamingStateFailed       StreamingState = "failed"
)

// StreamTokenResponse is the response from the stream token endpoint.
type StreamTokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expiresIn"`
}

// StreamErrorCode represents SSE error codes from server.
type StreamErrorCode string

const (
	// StreamErrorTokenInvalid indicates the token is invalid and re-authentication is needed.
	StreamErrorTokenInvalid StreamErrorCode = "TOKEN_INVALID"
	// StreamErrorTokenExpired indicates the token has expired and needs refresh.
	StreamErrorTokenExpired StreamErrorCode = "TOKEN_EXPIRED"
	// StreamErrorSubscriptionSuspended indicates the subscription is suspended.
	StreamErrorSubscriptionSuspended StreamErrorCode = "SUBSCRIPTION_SUSPENDED"
	// StreamErrorConnectionLimit indicates the connection limit has been reached.
	StreamErrorConnectionLimit StreamErrorCode = "CONNECTION_LIMIT"
	// StreamErrorStreamingUnavailable indicates streaming is not available.
	StreamErrorStreamingUnavailable StreamErrorCode = "STREAMING_UNAVAILABLE"
)

// StreamErrorData represents SSE error event data from server.
type StreamErrorData struct {
	Code    StreamErrorCode `json:"code"`
	Message string          `json:"message"`
}

// StreamingConfig contains streaming configuration.
type StreamingConfig struct {
	ReconnectInterval    time.Duration
	MaxReconnectAttempts int
	HeartbeatInterval    time.Duration
}

// DefaultStreamingConfig returns the default streaming configuration.
func DefaultStreamingConfig() *StreamingConfig {
	return &StreamingConfig{
		ReconnectInterval:    3 * time.Second,
		MaxReconnectAttempts: 3,
		HeartbeatInterval:    30 * time.Second,
	}
}

// StreamingManager manages Server-Sent Events (SSE) connection for real-time flag updates.
//
// Security: Uses token exchange pattern to avoid exposing API keys in URLs.
// 1. Fetches short-lived token via POST with API key in header
// 2. Connects to SSE endpoint with disposable token in URL
type StreamingManager struct {
	baseURL                  string
	getAPIKey                func() string
	config                   *StreamingConfig
	onFlagUpdate             func(flag *types.FlagState)
	onFlagDelete             func(key string)
	onFlagsReset             func(flags []*types.FlagState)
	onFallbackToPolling      func()
	onSubscriptionError      func(message string)
	onConnectionLimitError   func()
	logger                   Logger

	state               StreamingState
	consecutiveFailures int
	lastHeartbeat       time.Time
	client              *http.Client
	cancelFunc          context.CancelFunc
	tokenRefreshTimer   *time.Timer
	heartbeatTimer      *time.Timer
	retryTimer          *time.Timer
	mu                  sync.Mutex
}

// NewStreamingManager creates a new streaming manager.
func NewStreamingManager(
	baseURL string,
	getAPIKey func() string,
	config *StreamingConfig,
	onFlagUpdate func(flag *types.FlagState),
	onFlagDelete func(key string),
	onFlagsReset func(flags []*types.FlagState),
	onFallbackToPolling func(),
	onSubscriptionError func(message string),
	onConnectionLimitError func(),
	logger Logger,
) *StreamingManager {
	if config == nil {
		config = DefaultStreamingConfig()
	}
	return &StreamingManager{
		baseURL:                baseURL,
		getAPIKey:              getAPIKey,
		config:                 config,
		onFlagUpdate:           onFlagUpdate,
		onFlagDelete:           onFlagDelete,
		onFlagsReset:           onFlagsReset,
		onFallbackToPolling:    onFallbackToPolling,
		onSubscriptionError:    onSubscriptionError,
		onConnectionLimitError: onConnectionLimitError,
		logger:                 logger,
		state:                  StreamingStateDisconnected,
		client:                 &http.Client{Timeout: 0}, // No timeout for SSE
	}
}

// GetState returns the current connection state.
func (sm *StreamingManager) GetState() StreamingState {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.state
}

// IsConnected returns true if streaming is connected.
func (sm *StreamingManager) IsConnected() bool {
	return sm.GetState() == StreamingStateConnected
}

// Connect starts the streaming connection.
func (sm *StreamingManager) Connect() {
	sm.mu.Lock()
	if sm.state == StreamingStateConnected || sm.state == StreamingStateConnecting {
		sm.mu.Unlock()
		return
	}
	sm.state = StreamingStateConnecting
	sm.mu.Unlock()

	go sm.initiateConnection()
}

// Disconnect stops the streaming connection.
func (sm *StreamingManager) Disconnect() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.cleanup()
	sm.state = StreamingStateDisconnected
	sm.consecutiveFailures = 0

	if sm.logger != nil {
		sm.logger.Debug("Streaming disconnected")
	}
}

// RetryConnection retries the streaming connection.
func (sm *StreamingManager) RetryConnection() {
	sm.mu.Lock()
	if sm.state == StreamingStateConnected || sm.state == StreamingStateConnecting {
		sm.mu.Unlock()
		return
	}
	sm.consecutiveFailures = 0
	sm.mu.Unlock()

	sm.Connect()
}

// initiateConnection fetches a token and establishes the SSE connection.
func (sm *StreamingManager) initiateConnection() {
	// Step 1: Fetch short-lived stream token
	tokenResponse, err := sm.fetchStreamToken()
	if err != nil {
		if sm.logger != nil {
			sm.logger.Error("Failed to fetch stream token", "error", err)
		}
		sm.handleConnectionFailure()
		return
	}

	// Step 2: Schedule token refresh at 80% of TTL
	sm.scheduleTokenRefresh(time.Duration(float64(tokenResponse.ExpiresIn)*0.8) * time.Second)

	// Step 3: Create SSE connection with token
	sm.createConnection(tokenResponse.Token)
}

// fetchStreamToken fetches a short-lived token from the API.
func (sm *StreamingManager) fetchStreamToken() (*StreamTokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/sdk/stream/token", sm.baseURL)

	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader("{}"))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", sm.getAPIKey())

	resp, err := sm.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch stream token: %d", resp.StatusCode)
	}

	var tokenResponse StreamTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, err
	}

	return &tokenResponse, nil
}

// scheduleTokenRefresh schedules a token refresh before expiry.
func (sm *StreamingManager) scheduleTokenRefresh(delay time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.tokenRefreshTimer != nil {
		sm.tokenRefreshTimer.Stop()
	}

	sm.tokenRefreshTimer = time.AfterFunc(delay, func() {
		tokenResponse, err := sm.fetchStreamToken()
		if err != nil {
			if sm.logger != nil {
				sm.logger.Warn("Failed to refresh stream token, reconnecting", "error", err)
			}
			sm.Disconnect()
			sm.Connect()
			return
		}

		sm.scheduleTokenRefresh(time.Duration(float64(tokenResponse.ExpiresIn)*0.8) * time.Second)
	})
}

// createConnection creates the SSE connection with the token.
func (sm *StreamingManager) createConnection(token string) {
	streamURL := fmt.Sprintf("%s/sdk/stream?token=%s", sm.baseURL, token)

	ctx, cancel := context.WithCancel(context.Background())
	sm.mu.Lock()
	sm.cancelFunc = cancel
	sm.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		if sm.logger != nil {
			sm.logger.Error("Failed to create SSE request", "error", err)
		}
		sm.handleConnectionFailure()
		return
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := sm.client.Do(req)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return // Normal cancellation
		}
		if sm.logger != nil {
			sm.logger.Error("Failed to connect to SSE", "error", err)
		}
		sm.handleConnectionFailure()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if sm.logger != nil {
			sm.logger.Error("SSE connection failed", "status", resp.StatusCode)
		}
		sm.handleConnectionFailure()
		return
	}

	sm.handleOpen()
	sm.readEvents(resp.Body, ctx)
}

// handleOpen handles successful connection.
func (sm *StreamingManager) handleOpen() {
	sm.mu.Lock()
	sm.state = StreamingStateConnected
	sm.consecutiveFailures = 0
	sm.lastHeartbeat = time.Now()
	sm.mu.Unlock()

	sm.startHeartbeatMonitor()

	if sm.logger != nil {
		sm.logger.Info("Streaming connected")
	}
}

// readEvents reads and processes SSE events.
func (sm *StreamingManager) readEvents(body io.Reader, ctx context.Context) {
	reader := bufio.NewReader(body)
	var eventType string
	var dataBuilder strings.Builder

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if ctx.Err() == context.Canceled {
				return
			}
			if sm.GetState() == StreamingStateConnected {
				sm.handleConnectionFailure()
			}
			return
		}

		line = strings.TrimSpace(line)

		// Empty line = end of event
		if line == "" {
			if eventType != "" && dataBuilder.Len() > 0 {
				sm.processEvent(eventType, dataBuilder.String())
				eventType = ""
				dataBuilder.Reset()
			}
			continue
		}

		// Parse SSE format
		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(line[6:])
		} else if strings.HasPrefix(line, "data:") {
			dataBuilder.WriteString(strings.TrimSpace(line[5:]))
		}
	}
}

// processEvent processes a parsed SSE event.
func (sm *StreamingManager) processEvent(eventType, data string) {
	switch eventType {
	case "flag_updated":
		var flag types.FlagState
		if err := json.Unmarshal([]byte(data), &flag); err == nil {
			sm.onFlagUpdate(&flag)
		}
	case "flag_deleted":
		var deleteData struct {
			Key string `json:"key"`
		}
		if err := json.Unmarshal([]byte(data), &deleteData); err == nil {
			sm.onFlagDelete(deleteData.Key)
		}
	case "flags_reset":
		var flags []*types.FlagState
		if err := json.Unmarshal([]byte(data), &flags); err == nil {
			sm.onFlagsReset(flags)
		}
	case "heartbeat":
		sm.mu.Lock()
		sm.lastHeartbeat = time.Now()
		sm.mu.Unlock()
	case "error":
		sm.handleStreamError(data)
	}
}

// handleStreamError handles SSE error events from the server.
// These are application-level errors sent as SSE events, not connection errors.
//
// Error codes:
// - TOKEN_INVALID: Re-authenticate completely
// - TOKEN_EXPIRED: Refresh token and reconnect
// - SUBSCRIPTION_SUSPENDED: Notify user, fall back to cached values
// - CONNECTION_LIMIT: Implement backoff or close other connections
// - STREAMING_UNAVAILABLE: Fall back to polling
func (sm *StreamingManager) handleStreamError(data string) {
	var errorData StreamErrorData
	if err := json.Unmarshal([]byte(data), &errorData); err != nil {
		if sm.logger != nil {
			sm.logger.Debug("Failed to parse stream error event", "error", err)
		}
		return
	}

	if sm.logger != nil {
		sm.logger.Warn("SSE error event received", "code", errorData.Code, "message", errorData.Message)
	}

	switch errorData.Code {
	case StreamErrorTokenExpired:
		// Token expired, refresh and reconnect
		if sm.logger != nil {
			sm.logger.Info("Stream token expired, refreshing...")
		}
		sm.Disconnect()
		sm.Connect() // Will fetch new token

	case StreamErrorTokenInvalid:
		// Token is invalid, need full re-authentication
		if sm.logger != nil {
			sm.logger.Error("Stream token invalid, re-authenticating...")
		}
		sm.Disconnect()
		sm.Connect() // Will fetch new token

	case StreamErrorSubscriptionSuspended:
		// Subscription issue - notify and fall back
		if sm.logger != nil {
			sm.logger.Error("Subscription suspended", "message", errorData.Message)
		}
		if sm.onSubscriptionError != nil {
			sm.onSubscriptionError(errorData.Message)
		}
		sm.mu.Lock()
		sm.cleanup()
		sm.state = StreamingStateFailed
		sm.mu.Unlock()
		sm.onFallbackToPolling()

	case StreamErrorConnectionLimit:
		// Too many connections - implement backoff
		if sm.logger != nil {
			sm.logger.Warn("Connection limit reached, backing off...")
		}
		if sm.onConnectionLimitError != nil {
			sm.onConnectionLimitError()
		}
		sm.handleConnectionFailure()

	case StreamErrorStreamingUnavailable:
		// Streaming not available - fall back to polling
		if sm.logger != nil {
			sm.logger.Warn("Streaming service unavailable, falling back to polling")
		}
		sm.mu.Lock()
		sm.cleanup()
		sm.state = StreamingStateFailed
		sm.mu.Unlock()
		sm.onFallbackToPolling()

	default:
		if sm.logger != nil {
			sm.logger.Warn("Unknown stream error code", "code", errorData.Code)
		}
		sm.handleConnectionFailure()
	}
}

// handleConnectionFailure handles connection failure.
func (sm *StreamingManager) handleConnectionFailure() {
	sm.mu.Lock()
	sm.cleanup()
	sm.consecutiveFailures++
	failures := sm.consecutiveFailures
	maxAttempts := sm.config.MaxReconnectAttempts
	sm.mu.Unlock()

	if failures >= maxAttempts {
		sm.mu.Lock()
		sm.state = StreamingStateFailed
		sm.mu.Unlock()

		if sm.logger != nil {
			sm.logger.Warn("Streaming failed, falling back to polling", "failures", failures)
		}
		sm.onFallbackToPolling()
		sm.scheduleStreamingRetry()
	} else {
		sm.mu.Lock()
		sm.state = StreamingStateReconnecting
		sm.mu.Unlock()
		sm.scheduleReconnect()
	}
}

// scheduleReconnect schedules a reconnection attempt.
func (sm *StreamingManager) scheduleReconnect() {
	delay := sm.getReconnectDelay()
	if sm.logger != nil {
		sm.logger.Debug("Scheduling reconnect", "delay", delay, "attempt", sm.consecutiveFailures)
	}

	time.AfterFunc(delay, func() {
		sm.Connect()
	})
}

// getReconnectDelay calculates reconnection delay with exponential backoff.
func (sm *StreamingManager) getReconnectDelay() time.Duration {
	sm.mu.Lock()
	baseDelay := sm.config.ReconnectInterval
	failures := sm.consecutiveFailures
	sm.mu.Unlock()

	backoff := math.Pow(2, float64(failures-1))
	delay := time.Duration(float64(baseDelay) * backoff)

	// Cap at 30 seconds
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}
	return delay
}

// scheduleStreamingRetry schedules retry after fallback to polling.
func (sm *StreamingManager) scheduleStreamingRetry() {
	sm.mu.Lock()
	if sm.retryTimer != nil {
		sm.retryTimer.Stop()
	}
	sm.retryTimer = time.AfterFunc(5*time.Minute, func() {
		if sm.logger != nil {
			sm.logger.Info("Retrying streaming connection")
		}
		sm.RetryConnection()
	})
	sm.mu.Unlock()
}

// startHeartbeatMonitor starts monitoring heartbeats.
func (sm *StreamingManager) startHeartbeatMonitor() {
	sm.stopHeartbeatMonitor()

	checkInterval := time.Duration(float64(sm.config.HeartbeatInterval) * 1.5)

	sm.mu.Lock()
	sm.heartbeatTimer = time.AfterFunc(checkInterval, func() {
		sm.mu.Lock()
		timeSince := time.Since(sm.lastHeartbeat)
		threshold := sm.config.HeartbeatInterval * 2
		sm.mu.Unlock()

		if timeSince > threshold {
			if sm.logger != nil {
				sm.logger.Warn("Heartbeat timeout, reconnecting", "timeSince", timeSince)
			}
			sm.handleConnectionFailure()
		} else {
			sm.startHeartbeatMonitor()
		}
	})
	sm.mu.Unlock()
}

// stopHeartbeatMonitor stops the heartbeat monitor.
func (sm *StreamingManager) stopHeartbeatMonitor() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.heartbeatTimer != nil {
		sm.heartbeatTimer.Stop()
		sm.heartbeatTimer = nil
	}
}

// cleanup cleans up resources.
func (sm *StreamingManager) cleanup() {
	if sm.cancelFunc != nil {
		sm.cancelFunc()
		sm.cancelFunc = nil
	}
	if sm.tokenRefreshTimer != nil {
		sm.tokenRefreshTimer.Stop()
		sm.tokenRefreshTimer = nil
	}
	if sm.heartbeatTimer != nil {
		sm.heartbeatTimer.Stop()
		sm.heartbeatTimer = nil
	}
	if sm.retryTimer != nil {
		sm.retryTimer.Stop()
		sm.retryTimer = nil
	}
}
