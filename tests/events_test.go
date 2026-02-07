package tests

import (
	"testing"
	"time"

	. "github.com/teracrafts/flagkit-go"
	"github.com/stretchr/testify/assert"
)

func TestNewEventQueue(t *testing.T) {
	eq := NewEventQueue(&EventQueueOptions{
		SessionID:  "test-session",
		SDKVersion: "1.0.0",
		Logger:     &NullLogger{},
	})

	assert.NotNil(t, eq)
	assert.Equal(t, 0, eq.QueueSize())
}

func TestEventQueueTrack(t *testing.T) {
	eq := NewEventQueue(&EventQueueOptions{
		SessionID:  "test-session",
		SDKVersion: "1.0.0",
		Logger:     &NullLogger{},
	})

	eq.Track("test_event", map[string]any{"key": "value"})

	assert.Equal(t, 1, eq.QueueSize())
}

func TestEventQueueTrackMultiple(t *testing.T) {
	eq := NewEventQueue(&EventQueueOptions{
		SessionID:  "test-session",
		SDKVersion: "1.0.0",
		Logger:     &NullLogger{},
	})

	eq.Track("event1", nil)
	eq.Track("event2", nil)
	eq.Track("event3", nil)

	assert.Equal(t, 3, eq.QueueSize())
}

func TestEventQueueMaxSize(t *testing.T) {
	eq := NewEventQueue(&EventQueueOptions{
		SessionID:  "test-session",
		SDKVersion: "1.0.0",
		Logger:     &NullLogger{},
		Config: &EventQueueConfig{
			MaxSize:       3,
			FlushInterval: time.Minute,
			BatchSize:     10,
		},
	})

	eq.Track("event1", nil)
	eq.Track("event2", nil)
	eq.Track("event3", nil)
	eq.Track("event4", nil) // Should be dropped

	assert.Equal(t, 3, eq.QueueSize())
}

func TestEventQueueSetEnvironmentID(t *testing.T) {
	eq := NewEventQueue(&EventQueueOptions{
		SessionID:  "test-session",
		SDKVersion: "1.0.0",
	})

	eq.SetEnvironmentID("env-123")

	// Track an event and verify it has the environment ID
	eq.Track("test_event", nil)
	assert.Equal(t, 1, eq.QueueSize())
}

func TestEventQueueTrackWithContext(t *testing.T) {
	eq := NewEventQueue(&EventQueueOptions{
		SessionID:  "test-session",
		SDKVersion: "1.0.0",
		Logger:     &NullLogger{},
	})

	ctx := NewContext("user-123").
		WithEmail("user@example.com").
		WithPrivateAttribute("email")

	eq.TrackWithContext("test_event", map[string]any{"action": "click"}, ctx)

	assert.Equal(t, 1, eq.QueueSize())
}

func TestEventQueueFlushEmpty(t *testing.T) {
	eq := NewEventQueue(&EventQueueOptions{
		SessionID:  "test-session",
		SDKVersion: "1.0.0",
		Logger:     &NullLogger{},
	})

	// Flushing empty queue should not error
	eq.Flush()
	assert.Equal(t, 0, eq.QueueSize())
}

func TestEventQueueFlushClearsQueue(t *testing.T) {
	eq := NewEventQueue(&EventQueueOptions{
		SessionID:  "test-session",
		SDKVersion: "1.0.0",
		Logger:     &NullLogger{},
		// No HTTP client, so events won't actually be sent
	})

	eq.Track("event1", nil)
	eq.Track("event2", nil)
	assert.Equal(t, 2, eq.QueueSize())

	eq.Flush()
	assert.Equal(t, 0, eq.QueueSize())
}

func TestEventQueueStartStop(t *testing.T) {
	eq := NewEventQueue(&EventQueueOptions{
		SessionID:  "test-session",
		SDKVersion: "1.0.0",
		Logger:     &NullLogger{},
		Config: &EventQueueConfig{
			MaxSize:       1000,
			FlushInterval: 100 * time.Millisecond,
			BatchSize:     10,
		},
	})

	eq.Start()
	eq.Track("event1", nil)

	// Give time for the background loop to start
	time.Sleep(50 * time.Millisecond)

	eq.Stop()
	// After stop, queue should be flushed
	assert.Equal(t, 0, eq.QueueSize())
}

func TestDefaultEventQueueConfig(t *testing.T) {
	config := DefaultEventQueueConfig()

	assert.Equal(t, 1000, config.MaxSize)
	assert.Equal(t, 30*time.Second, config.FlushInterval)
	assert.Equal(t, 10, config.BatchSize)
}
