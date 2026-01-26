package tests

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/flagkit/flagkit-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventPersistence(t *testing.T) {
	tempDir := t.TempDir()

	ep, err := NewEventPersistence(tempDir, 10000, time.Second, &NullLogger{})
	require.NoError(t, err)
	require.NotNil(t, ep)
	defer func() { _ = ep.Close() }()

	assert.Equal(t, tempDir, ep.GetStoragePath())
	assert.Equal(t, 0, ep.GetBufferSize())
}

func TestEventPersistence_DefaultStoragePath(t *testing.T) {
	ep, err := NewEventPersistence("", 10000, time.Second, nil)
	require.NoError(t, err)
	require.NotNil(t, ep)
	defer func() { _ = ep.Close() }()

	assert.Equal(t, os.TempDir(), ep.GetStoragePath())
}

func TestEventPersistence_Persist(t *testing.T) {
	tempDir := t.TempDir()

	ep, err := NewEventPersistence(tempDir, 10000, time.Second, &NullLogger{})
	require.NoError(t, err)
	defer func() { _ = ep.Close() }()

	event := PersistedEvent{
		ID:        "evt_test123",
		Type:      "test.event",
		Timestamp: time.Now().UnixMilli(),
		Status:    EventStatusPending,
		Data:      map[string]any{"key": "value"},
	}

	err = ep.Persist(event)
	require.NoError(t, err)

	assert.Equal(t, 1, ep.GetBufferSize())
}

func TestEventPersistence_FlushBuffer(t *testing.T) {
	tempDir := t.TempDir()

	ep, err := NewEventPersistence(tempDir, 10000, time.Second, &NullLogger{})
	require.NoError(t, err)
	defer func() { _ = ep.Close() }()

	// Add an event
	event := PersistedEvent{
		ID:        "evt_flush1",
		Type:      "test.event",
		Timestamp: time.Now().UnixMilli(),
		Status:    EventStatusPending,
	}
	err = ep.Persist(event)
	require.NoError(t, err)
	assert.Equal(t, 1, ep.GetBufferSize())

	// Flush
	err = ep.Flush()
	require.NoError(t, err)
	assert.Equal(t, 0, ep.GetBufferSize())

	// Verify file exists
	pattern := filepath.Join(tempDir, "flagkit-events-*.jsonl")
	files, err := filepath.Glob(pattern)
	require.NoError(t, err)
	assert.Len(t, files, 1)
}

func TestEventPersistence_ManyEvents(t *testing.T) {
	tempDir := t.TempDir()

	// Create event persistence with small flush interval
	ep, err := NewEventPersistence(tempDir, 10000, 50*time.Millisecond, &NullLogger{})
	require.NoError(t, err)
	defer func() { _ = ep.Close() }()

	// Start background flush
	ep.Start()

	// Add many events
	numEvents := 20
	for i := 0; i < numEvents; i++ {
		event := PersistedEvent{
			ID:        GenerateEventID(),
			Type:      "test.event",
			Timestamp: time.Now().UnixMilli(),
			Status:    EventStatusPending,
		}
		err := ep.Persist(event)
		require.NoError(t, err)
	}

	// Wait for background flush
	time.Sleep(100 * time.Millisecond)
	ep.Stop()

	// Buffer should have been flushed
	assert.Equal(t, 0, ep.GetBufferSize())

	// Verify all events were persisted
	recovered, err := ep.Recover()
	require.NoError(t, err)
	assert.Equal(t, numEvents, len(recovered))
}

func TestEventPersistence_Recover(t *testing.T) {
	tempDir := t.TempDir()

	// First persistence session - add events
	ep1, err := NewEventPersistence(tempDir, 10000, time.Second, &NullLogger{})
	require.NoError(t, err)

	events := []PersistedEvent{
		{ID: "evt_recover1", Type: "test.event1", Timestamp: time.Now().UnixMilli(), Status: EventStatusPending},
		{ID: "evt_recover2", Type: "test.event2", Timestamp: time.Now().UnixMilli(), Status: EventStatusPending},
		{ID: "evt_recover3", Type: "test.event3", Timestamp: time.Now().UnixMilli(), Status: EventStatusPending},
	}

	for _, e := range events {
		err := ep1.Persist(e)
		require.NoError(t, err)
	}
	err = ep1.Flush()
	require.NoError(t, err)
	_ = ep1.Close()

	// Second persistence session - recover events
	ep2, err := NewEventPersistence(tempDir, 10000, time.Second, &NullLogger{})
	require.NoError(t, err)
	defer func() { _ = ep2.Close() }()

	recovered, err := ep2.Recover()
	require.NoError(t, err)
	assert.Len(t, recovered, 3)

	// Verify recovered events
	eventIDs := make(map[string]bool)
	for _, e := range recovered {
		eventIDs[e.ID] = true
		assert.Equal(t, EventStatusPending, e.Status)
	}
	assert.True(t, eventIDs["evt_recover1"])
	assert.True(t, eventIDs["evt_recover2"])
	assert.True(t, eventIDs["evt_recover3"])
}

func TestEventPersistence_RecoverSendingEvents(t *testing.T) {
	tempDir := t.TempDir()

	// Create events in "sending" state (simulating crash during send)
	ep1, err := NewEventPersistence(tempDir, 10000, time.Second, &NullLogger{})
	require.NoError(t, err)

	event := PersistedEvent{
		ID:        "evt_sending1",
		Type:      "test.event",
		Timestamp: time.Now().UnixMilli(),
		Status:    EventStatusPending,
	}
	err = ep1.Persist(event)
	require.NoError(t, err)
	err = ep1.Flush()
	require.NoError(t, err)

	// Mark as sending
	err = ep1.MarkSending([]string{"evt_sending1"})
	require.NoError(t, err)
	_ = ep1.Close()

	// Recover - sending events should be reset to pending
	ep2, err := NewEventPersistence(tempDir, 10000, time.Second, &NullLogger{})
	require.NoError(t, err)
	defer func() { _ = ep2.Close() }()

	recovered, err := ep2.Recover()
	require.NoError(t, err)
	assert.Len(t, recovered, 1)
	assert.Equal(t, EventStatusPending, recovered[0].Status)
}

func TestEventPersistence_MarkSent(t *testing.T) {
	tempDir := t.TempDir()

	ep, err := NewEventPersistence(tempDir, 10000, time.Second, &NullLogger{})
	require.NoError(t, err)
	defer func() { _ = ep.Close() }()

	// Add and flush events
	events := []PersistedEvent{
		{ID: "evt_sent1", Type: "test.event", Timestamp: time.Now().UnixMilli(), Status: EventStatusPending},
		{ID: "evt_sent2", Type: "test.event", Timestamp: time.Now().UnixMilli(), Status: EventStatusPending},
	}
	for _, e := range events {
		_ = ep.Persist(e)
	}
	_ = ep.Flush()

	// Mark as sent
	err = ep.MarkSent([]string{"evt_sent1"})
	require.NoError(t, err)

	// Recover - only pending event should be recovered
	recovered, err := ep.Recover()
	require.NoError(t, err)
	assert.Len(t, recovered, 1)
	assert.Equal(t, "evt_sent2", recovered[0].ID)
}

func TestEventPersistence_Cleanup(t *testing.T) {
	tempDir := t.TempDir()

	ep, err := NewEventPersistence(tempDir, 10000, time.Second, &NullLogger{})
	require.NoError(t, err)
	defer func() { _ = ep.Close() }()

	// Add events
	events := []PersistedEvent{
		{ID: "evt_cleanup1", Type: "test.event", Timestamp: time.Now().UnixMilli(), Status: EventStatusPending},
		{ID: "evt_cleanup2", Type: "test.event", Timestamp: time.Now().UnixMilli(), Status: EventStatusPending},
		{ID: "evt_cleanup3", Type: "test.event", Timestamp: time.Now().UnixMilli(), Status: EventStatusPending},
	}
	for _, e := range events {
		_ = ep.Persist(e)
	}
	_ = ep.Flush()

	// Mark some as sent
	err = ep.MarkSent([]string{"evt_cleanup1", "evt_cleanup2"})
	require.NoError(t, err)

	// Cleanup
	err = ep.Cleanup()
	require.NoError(t, err)

	// Recover - only pending event should remain
	recovered, err := ep.Recover()
	require.NoError(t, err)
	assert.Len(t, recovered, 1)
	assert.Equal(t, "evt_cleanup3", recovered[0].ID)
}

func TestEventPersistence_FileLocking(t *testing.T) {
	tempDir := t.TempDir()

	ep, err := NewEventPersistence(tempDir, 10000, time.Second, &NullLogger{})
	require.NoError(t, err)
	defer func() { _ = ep.Close() }()

	// Test concurrent access - verify no errors and at least some events were persisted
	var wg sync.WaitGroup
	numGoroutines := 10
	eventsPerGoroutine := 10
	var errorCount int32

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				event := PersistedEvent{
					ID:        GenerateEventID(),
					Type:      "test.concurrent",
					Timestamp: time.Now().UnixMilli(),
					Status:    EventStatusPending,
				}
				if err := ep.Persist(event); err != nil {
					atomic.AddInt32(&errorCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()
	err = ep.Flush()
	require.NoError(t, err)

	// Verify no errors occurred
	assert.Equal(t, int32(0), atomic.LoadInt32(&errorCount))

	// Verify all events were written
	recovered, err := ep.Recover()
	require.NoError(t, err)
	// All events should be recovered since we wait for all goroutines to finish before flushing
	assert.Equal(t, numGoroutines*eventsPerGoroutine, len(recovered))
}

func TestEventPersistence_ConcurrentFlush(t *testing.T) {
	tempDir := t.TempDir()

	ep, err := NewEventPersistence(tempDir, 10000, 100*time.Millisecond, &NullLogger{})
	require.NoError(t, err)
	defer func() { _ = ep.Close() }()

	var wg sync.WaitGroup
	totalEvents := 60

	// Start background flush loop
	ep.Start()

	// Add events concurrently
	wg.Add(3)
	for i := 0; i < 3; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				event := PersistedEvent{
					ID:        GenerateEventID(),
					Type:      "test.concurrent.flush",
					Timestamp: time.Now().UnixMilli(),
					Status:    EventStatusPending,
				}
				_ = ep.Persist(event)
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	wg.Wait()

	// Ensure all buffered events are flushed
	err = ep.Flush()
	require.NoError(t, err)

	ep.Stop()

	// Verify all events were persisted
	recovered, err := ep.Recover()
	require.NoError(t, err)
	assert.Equal(t, totalEvents, len(recovered))
}

func TestEventPersistence_StartStop(t *testing.T) {
	tempDir := t.TempDir()

	ep, err := NewEventPersistence(tempDir, 10000, 50*time.Millisecond, &NullLogger{})
	require.NoError(t, err)
	defer func() { _ = ep.Close() }()

	ep.Start()

	// Add events
	for i := 0; i < 5; i++ {
		event := PersistedEvent{
			ID:        GenerateEventID(),
			Type:      "test.startstop",
			Timestamp: time.Now().UnixMilli(),
			Status:    EventStatusPending,
		}
		err := ep.Persist(event)
		require.NoError(t, err)
	}

	// Wait for background flush
	time.Sleep(100 * time.Millisecond)

	ep.Stop()

	// Buffer should be empty after stop
	assert.Equal(t, 0, ep.GetBufferSize())
}

func TestEventPersistence_EmptyRecovery(t *testing.T) {
	tempDir := t.TempDir()

	ep, err := NewEventPersistence(tempDir, 10000, time.Second, &NullLogger{})
	require.NoError(t, err)
	defer func() { _ = ep.Close() }()

	recovered, err := ep.Recover()
	require.NoError(t, err)
	assert.Empty(t, recovered)
}

func TestEventPersistence_EventWithContext(t *testing.T) {
	tempDir := t.TempDir()

	ep, err := NewEventPersistence(tempDir, 10000, time.Second, &NullLogger{})
	require.NoError(t, err)
	defer func() { _ = ep.Close() }()

	event := PersistedEvent{
		ID:        "evt_ctx1",
		Type:      "test.withcontext",
		Timestamp: time.Now().UnixMilli(),
		Status:    EventStatusPending,
		Data:      map[string]any{"action": "click"},
		Context:   map[string]any{"userId": "user123", "country": "US"},
	}

	err = ep.Persist(event)
	require.NoError(t, err)
	err = ep.Flush()
	require.NoError(t, err)

	recovered, err := ep.Recover()
	require.NoError(t, err)
	require.Len(t, recovered, 1)

	assert.Equal(t, "evt_ctx1", recovered[0].ID)
	assert.Equal(t, "test.withcontext", recovered[0].Type)
	assert.Equal(t, "click", recovered[0].Data["action"])
	assert.Equal(t, "user123", recovered[0].Context["userId"])
}

func TestEventPersistence_MarkFailed(t *testing.T) {
	tempDir := t.TempDir()

	ep, err := NewEventPersistence(tempDir, 10000, time.Second, &NullLogger{})
	require.NoError(t, err)
	defer func() { _ = ep.Close() }()

	// Add event
	event := PersistedEvent{
		ID:        "evt_fail1",
		Type:      "test.event",
		Timestamp: time.Now().UnixMilli(),
		Status:    EventStatusPending,
	}
	err = ep.Persist(event)
	require.NoError(t, err)
	err = ep.Flush()
	require.NoError(t, err)

	// Mark as failed
	err = ep.MarkFailed([]string{"evt_fail1"})
	require.NoError(t, err)

	// Recover - failed events should not be recovered
	recovered, err := ep.Recover()
	require.NoError(t, err)
	assert.Empty(t, recovered)
}

func TestEventPersisterAdapter(t *testing.T) {
	tempDir := t.TempDir()

	ep, err := NewEventPersistence(tempDir, 10000, time.Second, &NullLogger{})
	require.NoError(t, err)
	defer func() { _ = ep.Close() }()

	adapter := NewEventPersisterAdapter(ep)
	require.NotNil(t, adapter)

	// Test through adapter interface
	err = adapter.Flush()
	require.NoError(t, err)

	recovered, err := adapter.Recover()
	require.NoError(t, err)
	assert.Empty(t, recovered)
}

func TestDefaultEventPersistenceConfig(t *testing.T) {
	config := DefaultEventPersistenceConfig()

	assert.Equal(t, os.TempDir(), config.StoragePath)
	assert.Equal(t, 10000, config.MaxEvents)
	assert.Equal(t, time.Second, config.FlushInterval)
	assert.Equal(t, 100, config.BufferSize)
}

func TestWithPersistEventsOption(t *testing.T) {
	opts := DefaultOptions("sdk_test_key_12345")
	WithPersistEvents(true)(opts)

	assert.True(t, opts.PersistEvents)
}

func TestWithEventStoragePathOption(t *testing.T) {
	opts := DefaultOptions("sdk_test_key_12345")
	WithEventStoragePath("/custom/path")(opts)

	assert.Equal(t, "/custom/path", opts.EventStoragePath)
}

func TestWithMaxPersistedEventsOption(t *testing.T) {
	opts := DefaultOptions("sdk_test_key_12345")
	WithMaxPersistedEvents(5000)(opts)

	assert.Equal(t, 5000, opts.MaxPersistedEvents)
}

func TestWithPersistenceFlushIntervalOption(t *testing.T) {
	opts := DefaultOptions("sdk_test_key_12345")
	WithPersistenceFlushInterval(2 * time.Second)(opts)

	assert.Equal(t, 2*time.Second, opts.PersistenceFlushInterval)
}
