package flagkit

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewPollingManager(t *testing.T) {
	pollCount := int32(0)
	pm := NewPollingManager(func() {
		atomic.AddInt32(&pollCount, 1)
	}, &PollingConfig{
		Interval: 100 * time.Millisecond,
	}, &NullLogger{})

	assert.NotNil(t, pm)
	assert.False(t, pm.IsActive())
}

func TestPollingManagerStartStop(t *testing.T) {
	pollCount := int32(0)
	pm := NewPollingManager(func() {
		atomic.AddInt32(&pollCount, 1)
	}, &PollingConfig{
		Interval: 50 * time.Millisecond,
		Jitter:   0,
	}, &NullLogger{})

	pm.Start()
	assert.True(t, pm.IsActive())

	// Wait for a couple of polls
	time.Sleep(150 * time.Millisecond)

	pm.Stop()
	assert.False(t, pm.IsActive())
	assert.GreaterOrEqual(t, atomic.LoadInt32(&pollCount), int32(2))
}

func TestPollingManagerDoubleStart(t *testing.T) {
	pm := NewPollingManager(func() {}, &PollingConfig{
		Interval: time.Second,
	}, &NullLogger{})

	pm.Start()
	pm.Start() // Should not start a second goroutine
	assert.True(t, pm.IsActive())

	pm.Stop()
}

func TestPollingManagerOnSuccess(t *testing.T) {
	pm := NewPollingManager(func() {}, &PollingConfig{
		Interval:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		MaxInterval:       time.Second,
	}, &NullLogger{})

	// Simulate errors to increase interval
	pm.OnError()
	pm.OnError()

	currentInterval := pm.GetCurrentInterval()
	assert.Greater(t, currentInterval, 100*time.Millisecond)

	// Success should reset interval
	pm.OnSuccess()
	assert.Equal(t, 100*time.Millisecond, pm.GetCurrentInterval())
}

func TestPollingManagerOnError(t *testing.T) {
	pm := NewPollingManager(func() {}, &PollingConfig{
		Interval:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		MaxInterval:       time.Second,
	}, &NullLogger{})

	pm.OnError()
	assert.Equal(t, 200*time.Millisecond, pm.GetCurrentInterval())

	pm.OnError()
	assert.Equal(t, 400*time.Millisecond, pm.GetCurrentInterval())
}

func TestPollingManagerMaxInterval(t *testing.T) {
	pm := NewPollingManager(func() {}, &PollingConfig{
		Interval:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		MaxInterval:       500 * time.Millisecond,
	}, &NullLogger{})

	// Multiple errors should cap at max interval
	for i := 0; i < 10; i++ {
		pm.OnError()
	}

	assert.Equal(t, 500*time.Millisecond, pm.GetCurrentInterval())
}

func TestPollingManagerReset(t *testing.T) {
	pm := NewPollingManager(func() {}, &PollingConfig{
		Interval:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		MaxInterval:       time.Second,
	}, &NullLogger{})

	pm.OnError()
	pm.OnError()

	pm.Reset()
	assert.Equal(t, 100*time.Millisecond, pm.GetCurrentInterval())
}

func TestPollingManagerPollNow(t *testing.T) {
	pollCount := int32(0)
	pm := NewPollingManager(func() {
		atomic.AddInt32(&pollCount, 1)
	}, &PollingConfig{
		Interval: time.Hour, // Long interval so regular polling won't trigger
	}, &NullLogger{})

	pm.Start()
	time.Sleep(10 * time.Millisecond)

	pm.PollNow()
	time.Sleep(10 * time.Millisecond)

	pm.Stop()
	assert.Equal(t, int32(1), atomic.LoadInt32(&pollCount))
}

func TestDefaultPollingConfig(t *testing.T) {
	config := DefaultPollingConfig()

	assert.Equal(t, 30*time.Second, config.Interval)
	assert.Equal(t, time.Second, config.Jitter)
	assert.Equal(t, 2.0, config.BackoffMultiplier)
	assert.Equal(t, 5*time.Minute, config.MaxInterval)
}
