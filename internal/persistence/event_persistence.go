package persistence

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/teracrafts/flagkit-go/internal/core"
	"github.com/teracrafts/flagkit-go/internal/types"
)

// Logger is an alias for the types.Logger interface.
type Logger = types.Logger

// eventCounter is used to generate unique event IDs
var eventCounter uint64

// EventStatus represents the status of a persisted event.
type EventStatus string

const (
	// EventStatusPending indicates the event is pending to be sent.
	EventStatusPending EventStatus = "pending"
	// EventStatusSending indicates the event is currently being sent.
	EventStatusSending EventStatus = "sending"
	// EventStatusSent indicates the event was successfully sent.
	EventStatusSent EventStatus = "sent"
	// EventStatusFailed indicates the event failed to send after max retries.
	EventStatusFailed EventStatus = "failed"
)

// PersistedEvent represents an event stored on disk.
type PersistedEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      map[string]any `json:"data,omitempty"`
	Context   map[string]any `json:"context,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Status    EventStatus            `json:"status"`
	SentAt    int64                  `json:"sentAt,omitempty"`
}

// EventPersistence handles crash-resilient event persistence using write-ahead logging.
type EventPersistence struct {
	storagePath   string
	maxEvents     int
	flushInterval time.Duration
	logger        Logger

	buffer      []PersistedEvent
	bufferSize  int
	currentFile string
	mu          sync.Mutex

	stopCh  chan struct{}
	running bool
}

// EventPersistenceConfig contains configuration for event persistence.
type EventPersistenceConfig struct {
	StoragePath   string
	MaxEvents     int
	FlushInterval time.Duration
	BufferSize    int
	Logger        Logger
}

// DefaultEventPersistenceConfig returns the default event persistence configuration.
func DefaultEventPersistenceConfig() *EventPersistenceConfig {
	return &EventPersistenceConfig{
		StoragePath:   os.TempDir(),
		MaxEvents:     10000,
		FlushInterval: time.Second,
		BufferSize:    100,
	}
}

// NewEventPersistence creates a new event persistence handler.
func NewEventPersistence(storagePath string, maxEvents int, flushInterval time.Duration, logger Logger) (*EventPersistence, error) {
	if storagePath == "" {
		storagePath = os.TempDir()
	}

	// Ensure storage directory exists
	if err := os.MkdirAll(storagePath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	if maxEvents <= 0 {
		maxEvents = 10000
	}

	if flushInterval <= 0 {
		flushInterval = time.Second
	}

	ep := &EventPersistence{
		storagePath:   storagePath,
		maxEvents:     maxEvents,
		flushInterval: flushInterval,
		logger:        logger,
		buffer:        make([]PersistedEvent, 0, 100),
		bufferSize:    100,
		stopCh:        make(chan struct{}),
	}

	// Generate current file name
	ep.currentFile = ep.generateFileName()

	return ep, nil
}

// Start starts the background flush loop.
func (ep *EventPersistence) Start() {
	ep.mu.Lock()
	if ep.running {
		ep.mu.Unlock()
		return
	}
	ep.running = true
	ep.stopCh = make(chan struct{})
	ep.mu.Unlock()

	go ep.run()
}

// Stop stops the background flush loop.
func (ep *EventPersistence) Stop() {
	ep.mu.Lock()
	if !ep.running {
		ep.mu.Unlock()
		return
	}
	ep.running = false
	close(ep.stopCh)
	ep.mu.Unlock()
}

// Persist adds an event to the buffer and flushes if the buffer is full.
func (ep *EventPersistence) Persist(event PersistedEvent) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	// Set defaults if not provided
	if event.ID == "" {
		event.ID = GenerateEventID()
	}
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().UnixMilli()
	}
	if event.Status == "" {
		event.Status = EventStatusPending
	}

	ep.buffer = append(ep.buffer, event)

	// Flush if buffer is full
	if len(ep.buffer) >= ep.bufferSize {
		return ep.flushLocked()
	}

	return nil
}

// Flush writes buffered events to disk with file locking.
func (ep *EventPersistence) Flush() error {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	return ep.flushLocked()
}

// flushLocked writes buffered events to disk (must be called with lock held).
func (ep *EventPersistence) flushLocked() error {
	if len(ep.buffer) == 0 {
		return nil
	}

	lockPath := filepath.Join(ep.storagePath, "flagkit-events.lock")

	// Acquire file lock
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		ep.logWarn("Failed to open lock file", "error", err)
		return fmt.Errorf("failed to open lock file: %w", err)
	}
	defer ep.closeFile(lockFile)

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		ep.logWarn("Failed to acquire file lock", "error", err)
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer ep.releaseLock(int(lockFile.Fd()))

	// Open or create current log file
	filePath := filepath.Join(ep.storagePath, ep.currentFile)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		ep.logWarn("Failed to open event file", "error", err)
		return fmt.Errorf("failed to open event file: %w", err)
	}
	defer ep.closeFile(file)

	// Write events in JSON Lines format
	for _, event := range ep.buffer {
		data, err := json.Marshal(event)
		if err != nil {
			ep.logWarn("Failed to marshal event", "error", err, "eventId", event.ID)
			continue
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			ep.logWarn("Failed to write event", "error", err, "eventId", event.ID)
			return fmt.Errorf("failed to write event: %w", err)
		}
	}

	// Sync to disk
	if err := file.Sync(); err != nil {
		ep.logWarn("Failed to sync file", "error", err)
		return fmt.Errorf("failed to sync file: %w", err)
	}

	ep.logDebug("Flushed events to disk", "count", len(ep.buffer))
	ep.buffer = ep.buffer[:0]

	return nil
}

// MarkSent marks the specified events as sent.
func (ep *EventPersistence) MarkSent(eventIDs []string) error {
	if len(eventIDs) == 0 {
		return nil
	}

	ep.mu.Lock()
	defer ep.mu.Unlock()

	lockPath := filepath.Join(ep.storagePath, "flagkit-events.lock")

	// Acquire file lock
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("failed to open lock file: %w", err)
	}
	defer ep.closeFile(lockFile)

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer ep.releaseLock(int(lockFile.Fd()))

	// Create status update entries
	filePath := filepath.Join(ep.storagePath, ep.currentFile)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open event file: %w", err)
	}
	defer ep.closeFile(file)

	sentAt := time.Now().UnixMilli()
	for _, id := range eventIDs {
		update := map[string]any{
			"id":     id,
			"status": EventStatusSent,
			"sentAt": sentAt,
		}
		data, err := json.Marshal(update)
		if err != nil {
			continue
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			ep.logWarn("Failed to write event status update", "error", err, "eventId", id)
		}
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	ep.logDebug("Marked events as sent", "count", len(eventIDs))
	return nil
}

// MarkSending marks the specified events as currently being sent.
func (ep *EventPersistence) MarkSending(eventIDs []string) error {
	if len(eventIDs) == 0 {
		return nil
	}

	ep.mu.Lock()
	defer ep.mu.Unlock()

	lockPath := filepath.Join(ep.storagePath, "flagkit-events.lock")

	// Acquire file lock
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("failed to open lock file: %w", err)
	}
	defer ep.closeFile(lockFile)

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer ep.releaseLock(int(lockFile.Fd()))

	filePath := filepath.Join(ep.storagePath, ep.currentFile)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open event file: %w", err)
	}
	defer ep.closeFile(file)

	for _, id := range eventIDs {
		update := map[string]any{
			"id":     id,
			"status": EventStatusSending,
		}
		data, err := json.Marshal(update)
		if err != nil {
			continue
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			ep.logWarn("Failed to write event status update", "error", err, "eventId", id)
		}
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}

// MarkFailed marks the specified events as failed to send.
func (ep *EventPersistence) MarkFailed(eventIDs []string) error {
	if len(eventIDs) == 0 {
		return nil
	}

	ep.mu.Lock()
	defer ep.mu.Unlock()

	lockPath := filepath.Join(ep.storagePath, "flagkit-events.lock")

	// Acquire file lock
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("failed to open lock file: %w", err)
	}
	defer ep.closeFile(lockFile)

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer ep.releaseLock(int(lockFile.Fd()))

	filePath := filepath.Join(ep.storagePath, ep.currentFile)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open event file: %w", err)
	}
	defer ep.closeFile(file)

	for _, id := range eventIDs {
		update := map[string]any{
			"id":     id,
			"status": EventStatusFailed,
		}
		data, err := json.Marshal(update)
		if err != nil {
			continue
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			ep.logWarn("Failed to write event status update", "error", err, "eventId", id)
		}
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}

// Recover recovers pending events from disk on startup.
func (ep *EventPersistence) Recover() ([]PersistedEvent, error) {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	lockPath := filepath.Join(ep.storagePath, "flagkit-events.lock")

	// Acquire file lock
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}
	defer ep.closeFile(lockFile)

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer ep.releaseLock(int(lockFile.Fd()))

	// Find all event files
	pattern := filepath.Join(ep.storagePath, "flagkit-events-*.jsonl")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find event files: %w", err)
	}

	// Map to track the latest status for each event ID
	eventMap := make(map[string]PersistedEvent)

	// Read all event files
	for _, filePath := range files {
		if err := ep.readEventsFromFile(filePath, eventMap); err != nil {
			ep.logWarn("Failed to read event file", "file", filePath, "error", err)
			continue
		}
	}

	// Collect pending and sending events (sending = crashed mid-send)
	var pendingEvents []PersistedEvent
	for _, event := range eventMap {
		if event.Status == EventStatusPending || event.Status == EventStatusSending {
			// Reset sending events to pending
			event.Status = EventStatusPending
			pendingEvents = append(pendingEvents, event)
		}
	}

	ep.logInfo("Recovered pending events", "count", len(pendingEvents))
	return pendingEvents, nil
}

// readEventsFromFile reads events from a single file into the event map.
func (ep *EventPersistence) readEventsFromFile(filePath string, eventMap map[string]PersistedEvent) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer ep.closeFile(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event PersistedEvent
		if err := json.Unmarshal(line, &event); err != nil {
			// Try parsing as status update
			var update struct {
				ID     string      `json:"id"`
				Status EventStatus `json:"status"`
				SentAt int64       `json:"sentAt,omitempty"`
			}
			if err := json.Unmarshal(line, &update); err != nil {
				continue
			}
			// Update existing event status
			if existing, ok := eventMap[update.ID]; ok {
				existing.Status = update.Status
				existing.SentAt = update.SentAt
				eventMap[update.ID] = existing
			}
			continue
		}

		// Full event entry
		if event.ID != "" {
			eventMap[event.ID] = event
		}
	}

	return scanner.Err()
}

// Cleanup removes old sent/failed events and compacts event files.
func (ep *EventPersistence) Cleanup() error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	lockPath := filepath.Join(ep.storagePath, "flagkit-events.lock")

	// Acquire file lock
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("failed to open lock file: %w", err)
	}
	defer ep.closeFile(lockFile)

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer ep.releaseLock(int(lockFile.Fd()))

	// Find all event files
	pattern := filepath.Join(ep.storagePath, "flagkit-events-*.jsonl")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to find event files: %w", err)
	}

	// Collect all events and their final states
	eventMap := make(map[string]PersistedEvent)
	for _, filePath := range files {
		if err := ep.readEventsFromFile(filePath, eventMap); err != nil {
			continue
		}
	}

	// Filter out sent and failed events, keep only pending
	var pendingEvents []PersistedEvent
	for _, event := range eventMap {
		if event.Status == EventStatusPending || event.Status == EventStatusSending {
			pendingEvents = append(pendingEvents, event)
		}
	}

	// Remove old files
	for _, filePath := range files {
		if err := os.Remove(filePath); err != nil {
			ep.logWarn("Failed to remove old event file", "file", filePath, "error", err)
		}
	}

	// Generate new file name
	ep.currentFile = ep.generateFileName()

	// Write pending events to new file
	if len(pendingEvents) > 0 {
		filePath := filepath.Join(ep.storagePath, ep.currentFile)
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return fmt.Errorf("failed to create new event file: %w", err)
		}
		defer ep.closeFile(file)

		for _, event := range pendingEvents {
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			if _, err := file.Write(append(data, '\n')); err != nil {
				ep.logWarn("Failed to write event during cleanup", "error", err, "eventId", event.ID)
			}
		}

		if err := file.Sync(); err != nil {
			return fmt.Errorf("failed to sync file: %w", err)
		}
	}

	ep.logInfo("Cleaned up event files", "pendingCount", len(pendingEvents), "removedFiles", len(files))
	return nil
}

// Close flushes remaining events and cleans up resources.
func (ep *EventPersistence) Close() error {
	ep.Stop()

	// Final flush
	if err := ep.Flush(); err != nil {
		ep.logWarn("Failed to flush on close", "error", err)
	}

	return nil
}

// GetBufferSize returns the current number of events in the buffer.
func (ep *EventPersistence) GetBufferSize() int {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	return len(ep.buffer)
}

// GetStoragePath returns the storage path.
func (ep *EventPersistence) GetStoragePath() string {
	return ep.storagePath
}

// run is the background flush loop.
func (ep *EventPersistence) run() {
	ticker := time.NewTicker(ep.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ep.stopCh:
			return
		case <-ticker.C:
			if err := ep.Flush(); err != nil {
				ep.logWarn("Background flush failed", "error", err)
			}
		}
	}
}

// generateFileName generates a unique file name for the event log.
func (ep *EventPersistence) generateFileName() string {
	timestamp := time.Now().UnixMilli()
	random := generateRandomString(8)
	return fmt.Sprintf("flagkit-events-%d-%s.jsonl", timestamp, random)
}

// GenerateEventID generates a unique event ID.
func GenerateEventID() string {
	count := atomic.AddUint64(&eventCounter, 1)
	return fmt.Sprintf("evt_%d_%s", count, generateRandomString(8))
}

// generateRandomString generates a cryptographically secure random string.
func generateRandomString(length int) string {
	bytes := make([]byte, (length+1)/2)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based if crypto/rand fails
		return fmt.Sprintf("%x", time.Now().UnixNano())[:length]
	}
	return hex.EncodeToString(bytes)[:length]
}

// Logging helpers
func (ep *EventPersistence) logDebug(msg string, keysAndValues ...any) {
	if ep.logger != nil {
		ep.logger.Debug(msg, keysAndValues...)
	}
}

func (ep *EventPersistence) logInfo(msg string, keysAndValues ...any) {
	if ep.logger != nil {
		ep.logger.Info(msg, keysAndValues...)
	}
}

func (ep *EventPersistence) logWarn(msg string, keysAndValues ...any) {
	if ep.logger != nil {
		ep.logger.Warn(msg, keysAndValues...)
	}
}

// closeFile closes a file and logs any error.
func (ep *EventPersistence) closeFile(f *os.File) {
	if err := f.Close(); err != nil {
		ep.logWarn("Failed to close file", "error", err)
	}
}

// releaseLock releases a file lock and logs any error.
func (ep *EventPersistence) releaseLock(fd int) {
	if err := syscall.Flock(fd, syscall.LOCK_UN); err != nil {
		ep.logWarn("Failed to release file lock", "error", err)
	}
}

// EventPersisterAdapter adapts EventPersistence to the internal.EventPersister interface.
type EventPersisterAdapter struct {
	ep *EventPersistence
}

// NewEventPersisterAdapter creates a new adapter for the internal EventPersister interface.
func NewEventPersisterAdapter(ep *EventPersistence) *EventPersisterAdapter {
	return &EventPersisterAdapter{ep: ep}
}

// Persist persists an event using the internal PersistedEvent type.
func (a *EventPersisterAdapter) Persist(event core.PersistedEvent) error {
	return a.ep.Persist(PersistedEvent{
		ID:        event.ID,
		Type:      event.Type,
		Data:      event.Data,
		Context:   event.Context,
		Timestamp: event.Timestamp,
		Status:    EventStatus(event.Status),
		SentAt:    event.SentAt,
	})
}

// MarkSending marks events as sending.
func (a *EventPersisterAdapter) MarkSending(eventIDs []string) error {
	return a.ep.MarkSending(eventIDs)
}

// MarkSent marks events as sent.
func (a *EventPersisterAdapter) MarkSent(eventIDs []string) error {
	return a.ep.MarkSent(eventIDs)
}

// MarkFailed marks events as failed.
func (a *EventPersisterAdapter) MarkFailed(eventIDs []string) error {
	return a.ep.MarkFailed(eventIDs)
}

// Recover recovers pending events.
func (a *EventPersisterAdapter) Recover() ([]core.PersistedEvent, error) {
	events, err := a.ep.Recover()
	if err != nil {
		return nil, err
	}

	result := make([]core.PersistedEvent, len(events))
	for i, e := range events {
		result[i] = core.PersistedEvent{
			ID:        e.ID,
			Type:      e.Type,
			Data:      e.Data,
			Context:   e.Context,
			Timestamp: e.Timestamp,
			Status:    string(e.Status),
			SentAt:    e.SentAt,
		}
	}
	return result, nil
}

// Flush flushes the persistence buffer.
func (a *EventPersisterAdapter) Flush() error {
	return a.ep.Flush()
}
