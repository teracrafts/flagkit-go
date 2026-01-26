package internal

import (
	"fmt"
	"sync"
	"time"
)

// Event represents an analytics event.
type Event struct {
	ID            string                 `json:"id,omitempty"`
	Type          string                 `json:"type"`
	Timestamp     string                 `json:"timestamp"`
	SessionID     string                 `json:"sessionId"`
	EnvironmentID string                 `json:"environmentId"`
	SDKVersion    string                 `json:"sdkVersion"`
	Data          map[string]any `json:"data,omitempty"`
	Context       map[string]any `json:"context,omitempty"`
}

// EventPersister is the interface for event persistence.
type EventPersister interface {
	Persist(event PersistedEvent) error
	MarkSending(eventIDs []string) error
	MarkSent(eventIDs []string) error
	MarkFailed(eventIDs []string) error
	Recover() ([]PersistedEvent, error)
	Flush() error
}

// PersistedEvent represents an event stored on disk.
type PersistedEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      map[string]any `json:"data,omitempty"`
	Context   map[string]any `json:"context,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Status    string                 `json:"status"`
	SentAt    int64                  `json:"sentAt,omitempty"`
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

	// Persistence support
	persister      EventPersister
	persistEnabled bool
}

// EventQueueOptions contains options for creating an event queue.
type EventQueueOptions struct {
	HTTPClient     *HTTPClient
	SessionID      string
	EnvironmentID  string
	SDKVersion     string
	Logger         Logger
	Config         *EventQueueConfig
	Persister      EventPersister
	PersistEnabled bool
}

// NewEventQueue creates a new event queue.
func NewEventQueue(opts *EventQueueOptions) *EventQueue {
	config := opts.Config
	if config == nil {
		config = DefaultEventQueueConfig()
	}

	eq := &EventQueue{
		config:         config,
		events:         make([]Event, 0, config.MaxSize),
		httpClient:     opts.HTTPClient,
		sessionID:      opts.SessionID,
		environmentID:  opts.EnvironmentID,
		sdkVersion:     opts.SDKVersion,
		logger:         opts.Logger,
		stopCh:         make(chan struct{}),
		flushCh:        make(chan struct{}, 1),
		persister:      opts.Persister,
		persistEnabled: opts.PersistEnabled,
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
func (eq *EventQueue) Track(eventType string, data map[string]any) {
	eq.mu.Lock()
	defer eq.mu.Unlock()

	if len(eq.events) >= eq.config.MaxSize {
		if eq.logger != nil {
			eq.logger.Warn("Event queue full, dropping event", "type", eventType)
		}
		return
	}

	eventID := eq.generateEventID()
	now := time.Now().UTC()

	event := Event{
		ID:            eventID,
		Type:          eventType,
		Timestamp:     now.Format(time.RFC3339),
		SessionID:     eq.sessionID,
		EnvironmentID: eq.environmentID,
		SDKVersion:    eq.sdkVersion,
		Data:          data,
	}

	// Persist event before adding to queue (crash-safe)
	if eq.persistEnabled && eq.persister != nil {
		persistedEvent := PersistedEvent{
			ID:        eventID,
			Type:      eventType,
			Data:      data,
			Timestamp: now.UnixMilli(),
			Status:    "pending",
		}
		if err := eq.persister.Persist(persistedEvent); err != nil {
			if eq.logger != nil {
				eq.logger.Warn("Failed to persist event", "error", err.Error(), "eventId", eventID)
			}
			// Continue anyway - event will still be in memory
		}
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

// generateEventID generates a unique event ID.
func (eq *EventQueue) generateEventID() string {
	return fmt.Sprintf("evt_%d_%s", time.Now().UnixNano(), eq.sessionID[:8])
}

// TrackWithContext adds an event with context to the queue.
func (eq *EventQueue) TrackWithContext(eventType string, data map[string]any, ctx *EvaluationContext) {
	eq.mu.Lock()
	defer eq.mu.Unlock()

	if len(eq.events) >= eq.config.MaxSize {
		return
	}

	var contextMap map[string]any
	if ctx != nil {
		contextMap = ctx.StripPrivateAttributes().ToMap()
	}

	eventID := eq.generateEventID()
	now := time.Now().UTC()

	event := Event{
		ID:            eventID,
		Type:          eventType,
		Timestamp:     now.Format(time.RFC3339),
		SessionID:     eq.sessionID,
		EnvironmentID: eq.environmentID,
		SDKVersion:    eq.sdkVersion,
		Data:          data,
		Context:       contextMap,
	}

	// Persist event before adding to queue (crash-safe)
	if eq.persistEnabled && eq.persister != nil {
		persistedEvent := PersistedEvent{
			ID:        eventID,
			Type:      eventType,
			Data:      data,
			Context:   contextMap,
			Timestamp: now.UnixMilli(),
			Status:    "pending",
		}
		if err := eq.persister.Persist(persistedEvent); err != nil {
			if eq.logger != nil {
				eq.logger.Warn("Failed to persist event", "error", err.Error(), "eventId", eventID)
			}
		}
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

	// Collect event IDs for persistence tracking
	var eventIDs []string
	if eq.persistEnabled && eq.persister != nil {
		eventIDs = make([]string, len(events))
		for i, e := range events {
			eventIDs[i] = e.ID
		}
		// Mark as sending before send attempt
		if err := eq.persister.MarkSending(eventIDs); err != nil {
			if eq.logger != nil {
				eq.logger.Warn("Failed to mark events as sending", "error", err.Error())
			}
		}
	}

	payload := map[string]any{
		"events": events,
	}

	_, err := eq.httpClient.Post("/sdk/events/batch", payload)
	if err != nil {
		if eq.logger != nil {
			eq.logger.Warn("Failed to send events", "error", err.Error(), "count", len(events))
		}
		// Mark events as failed if persistence is enabled
		if eq.persistEnabled && eq.persister != nil && len(eventIDs) > 0 {
			if markErr := eq.persister.MarkFailed(eventIDs); markErr != nil {
				if eq.logger != nil {
					eq.logger.Warn("Failed to mark events as failed", "error", markErr.Error())
				}
			}
		}
		return
	}

	// Mark events as sent on success
	if eq.persistEnabled && eq.persister != nil && len(eventIDs) > 0 {
		if err := eq.persister.MarkSent(eventIDs); err != nil {
			if eq.logger != nil {
				eq.logger.Warn("Failed to mark events as sent", "error", err.Error())
			}
		}
	}
}

// RecoverEvents recovers pending events from persistence on startup.
func (eq *EventQueue) RecoverEvents() error {
	if !eq.persistEnabled || eq.persister == nil {
		return nil
	}

	recovered, err := eq.persister.Recover()
	if err != nil {
		return err
	}

	if len(recovered) == 0 {
		return nil
	}

	eq.mu.Lock()
	defer eq.mu.Unlock()

	// Add recovered events to the queue with priority
	for _, pe := range recovered {
		if len(eq.events) >= eq.config.MaxSize {
			if eq.logger != nil {
				eq.logger.Warn("Event queue full during recovery, some events dropped")
			}
			break
		}

		event := Event{
			ID:        pe.ID,
			Type:      pe.Type,
			Timestamp: time.UnixMilli(pe.Timestamp).UTC().Format(time.RFC3339),
			SessionID: eq.sessionID,
			SDKVersion: eq.sdkVersion,
			Data:      pe.Data,
			Context:   pe.Context,
		}
		// Insert at the beginning (priority)
		eq.events = append([]Event{event}, eq.events...)
	}

	if eq.logger != nil {
		eq.logger.Info("Recovered persisted events", "count", len(recovered))
	}

	return nil
}

// SetPersister sets the event persister.
func (eq *EventQueue) SetPersister(persister EventPersister, enabled bool) {
	eq.mu.Lock()
	defer eq.mu.Unlock()
	eq.persister = persister
	eq.persistEnabled = enabled
}
