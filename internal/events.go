package internal

import (
	"sync"
	"time"
)

// Event represents an analytics event.
type Event struct {
	Type          string                 `json:"type"`
	Timestamp     string                 `json:"timestamp"`
	SessionID     string                 `json:"sessionId"`
	EnvironmentID string                 `json:"environmentId"`
	SDKVersion    string                 `json:"sdkVersion"`
	Data          map[string]interface{} `json:"data,omitempty"`
	Context       map[string]interface{} `json:"context,omitempty"`
}

// EventQueueConfig contains event queue configuration.
type EventQueueConfig struct {
	MaxSize       int
	FlushInterval time.Duration
	BatchSize     int
}

// DefaultEventQueueConfig returns the default event queue configuration.
func DefaultEventQueueConfig() *EventQueueConfig {
	return &EventQueueConfig{
		MaxSize:       1000,
		FlushInterval: 30 * time.Second,
		BatchSize:     10,
	}
}

// EventQueue manages analytics events with batching.
type EventQueue struct {
	config        *EventQueueConfig
	events        []Event
	httpClient    *HTTPClient
	sessionID     string
	environmentID string
	sdkVersion    string
	logger        Logger
	running       bool
	stopCh        chan struct{}
	flushCh       chan struct{}
	mu            sync.Mutex
}

// EventQueueOptions contains options for creating an event queue.
type EventQueueOptions struct {
	HTTPClient    *HTTPClient
	SessionID     string
	EnvironmentID string
	SDKVersion    string
	Logger        Logger
	Config        *EventQueueConfig
}

// NewEventQueue creates a new event queue.
func NewEventQueue(opts *EventQueueOptions) *EventQueue {
	config := opts.Config
	if config == nil {
		config = DefaultEventQueueConfig()
	}

	eq := &EventQueue{
		config:        config,
		events:        make([]Event, 0, config.MaxSize),
		httpClient:    opts.HTTPClient,
		sessionID:     opts.SessionID,
		environmentID: opts.EnvironmentID,
		sdkVersion:    opts.SDKVersion,
		logger:        opts.Logger,
		stopCh:        make(chan struct{}),
		flushCh:       make(chan struct{}, 1),
	}

	return eq
}

// Start starts the background flush loop.
func (eq *EventQueue) Start() {
	eq.mu.Lock()
	if eq.running {
		eq.mu.Unlock()
		return
	}
	eq.running = true
	eq.stopCh = make(chan struct{})
	eq.mu.Unlock()

	go eq.run()
}

// Stop stops the event queue and flushes remaining events.
func (eq *EventQueue) Stop() {
	eq.mu.Lock()
	if !eq.running {
		eq.mu.Unlock()
		return
	}
	eq.running = false
	close(eq.stopCh)
	eq.mu.Unlock()

	// Final flush
	eq.Flush()
}

// SetEnvironmentID sets the environment ID.
func (eq *EventQueue) SetEnvironmentID(id string) {
	eq.mu.Lock()
	defer eq.mu.Unlock()
	eq.environmentID = id
}

// Track adds an event to the queue.
func (eq *EventQueue) Track(eventType string, data map[string]interface{}) {
	eq.mu.Lock()
	defer eq.mu.Unlock()

	if len(eq.events) >= eq.config.MaxSize {
		if eq.logger != nil {
			eq.logger.Warn("Event queue full, dropping event", "type", eventType)
		}
		return
	}

	event := Event{
		Type:          eventType,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		SessionID:     eq.sessionID,
		EnvironmentID: eq.environmentID,
		SDKVersion:    eq.sdkVersion,
		Data:          data,
	}

	eq.events = append(eq.events, event)

	if eq.logger != nil {
		eq.logger.Debug("Event tracked", "type", eventType, "queue_size", len(eq.events))
	}

	// Trigger flush if batch size reached
	if len(eq.events) >= eq.config.BatchSize {
		select {
		case eq.flushCh <- struct{}{}:
		default:
		}
	}
}

// TrackWithContext adds an event with context to the queue.
func (eq *EventQueue) TrackWithContext(eventType string, data map[string]interface{}, ctx *EvaluationContext) {
	eq.mu.Lock()
	defer eq.mu.Unlock()

	if len(eq.events) >= eq.config.MaxSize {
		return
	}

	var contextMap map[string]interface{}
	if ctx != nil {
		contextMap = ctx.StripPrivateAttributes().ToMap()
	}

	event := Event{
		Type:          eventType,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		SessionID:     eq.sessionID,
		EnvironmentID: eq.environmentID,
		SDKVersion:    eq.sdkVersion,
		Data:          data,
		Context:       contextMap,
	}

	eq.events = append(eq.events, event)

	if len(eq.events) >= eq.config.BatchSize {
		select {
		case eq.flushCh <- struct{}{}:
		default:
		}
	}
}

// Flush sends all queued events to the server.
func (eq *EventQueue) Flush() {
	eq.mu.Lock()
	if len(eq.events) == 0 {
		eq.mu.Unlock()
		return
	}

	// Copy events and clear queue
	events := make([]Event, len(eq.events))
	copy(events, eq.events)
	eq.events = eq.events[:0]
	eq.mu.Unlock()

	if eq.logger != nil {
		eq.logger.Debug("Flushing events", "count", len(events))
	}

	eq.sendEvents(events)
}

// QueueSize returns the number of queued events.
func (eq *EventQueue) QueueSize() int {
	eq.mu.Lock()
	defer eq.mu.Unlock()
	return len(eq.events)
}

// run is the background flush loop.
func (eq *EventQueue) run() {
	ticker := time.NewTicker(eq.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-eq.stopCh:
			return
		case <-ticker.C:
			eq.Flush()
		case <-eq.flushCh:
			eq.Flush()
		}
	}
}

// sendEvents sends events to the server.
func (eq *EventQueue) sendEvents(events []Event) {
	if eq.httpClient == nil {
		return
	}

	payload := map[string]interface{}{
		"events": events,
	}

	_, err := eq.httpClient.Post("/sdk/events/batch", payload)
	if err != nil {
		if eq.logger != nil {
			eq.logger.Warn("Failed to send events", "error", err.Error(), "count", len(events))
		}
		// Could implement retry logic here
	}
}
