package core

import (
	"math/rand"
	"sync"
	"time"

	"github.com/teracrafts/flagkit-go/internal/types"
)

// Logger is an alias for the types.Logger interface.
type Logger = types.Logger

// PollingConfig contains polling configuration.
type PollingConfig struct {
	Interval          time.Duration
	Jitter            time.Duration
	BackoffMultiplier float64
	MaxInterval       time.Duration
}

// DefaultPollingConfig returns the default polling configuration.
func DefaultPollingConfig() *PollingConfig {
	return &PollingConfig{
		Interval:          30 * time.Second,
		Jitter:            time.Second,
		BackoffMultiplier: 2.0,
		MaxInterval:       5 * time.Minute,
	}
}

// PollingManager manages background polling for flag updates.
type PollingManager struct {
	config            *PollingConfig
	onPoll            func()
	logger            Logger
	currentInterval   time.Duration
	consecutiveErrors int
	running           bool
	stopCh            chan struct{}
	mu                sync.Mutex
}

// NewPollingManager creates a new polling manager.
func NewPollingManager(onPoll func(), config *PollingConfig, logger Logger) *PollingManager {
	if config == nil {
		config = DefaultPollingConfig()
	}
	return &PollingManager{
		config:          config,
		onPoll:          onPoll,
		logger:          logger,
		currentInterval: config.Interval,
		stopCh:          make(chan struct{}),
	}
}

// Start starts the polling loop.
func (pm *PollingManager) Start() {
	pm.mu.Lock()
	if pm.running {
		pm.mu.Unlock()
		return
	}
	pm.running = true
	pm.stopCh = make(chan struct{})
	pm.mu.Unlock()

	if pm.logger != nil {
		pm.logger.Debug("Polling started", "interval", pm.currentInterval)
	}

	go pm.run()
}

// Stop stops the polling loop.
func (pm *PollingManager) Stop() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.running {
		return
	}

	pm.running = false
	close(pm.stopCh)

	if pm.logger != nil {
		pm.logger.Debug("Polling stopped")
	}
}

// IsActive returns whether polling is active.
func (pm *PollingManager) IsActive() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.running
}

// GetCurrentInterval returns the current polling interval.
func (pm *PollingManager) GetCurrentInterval() time.Duration {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.currentInterval
}

// OnSuccess handles a successful poll.
func (pm *PollingManager) OnSuccess() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.consecutiveErrors = 0
	pm.currentInterval = pm.config.Interval
}

// OnError handles a failed poll.
func (pm *PollingManager) OnError() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.consecutiveErrors++
	newInterval := time.Duration(float64(pm.currentInterval) * pm.config.BackoffMultiplier)
	if newInterval > pm.config.MaxInterval {
		newInterval = pm.config.MaxInterval
	}
	pm.currentInterval = newInterval

	if pm.logger != nil {
		pm.logger.Debug("Polling backoff",
			"interval", pm.currentInterval,
			"consecutive_errors", pm.consecutiveErrors,
		)
	}
}

// Reset resets the polling manager.
func (pm *PollingManager) Reset() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.consecutiveErrors = 0
	pm.currentInterval = pm.config.Interval
}

// PollNow forces an immediate poll.
func (pm *PollingManager) PollNow() {
	pm.poll()
}

// run is the main polling loop.
func (pm *PollingManager) run() {
	for {
		delay := pm.getNextDelay()

		select {
		case <-pm.stopCh:
			return
		case <-time.After(delay):
			pm.poll()
		}
	}
}

// poll executes a single poll.
func (pm *PollingManager) poll() {
	pm.mu.Lock()
	if !pm.running {
		pm.mu.Unlock()
		return
	}
	pm.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			if pm.logger != nil {
				pm.logger.Error("Poll panic recovered", "error", r)
			}
			pm.OnError()
		}
	}()

	pm.onPoll()
}

// getNextDelay calculates the next poll delay with jitter.
func (pm *PollingManager) getNextDelay() time.Duration {
	pm.mu.Lock()
	interval := pm.currentInterval
	jitter := pm.config.Jitter
	pm.mu.Unlock()

	jitterAmount := time.Duration(rand.Float64() * float64(jitter))
	return interval + jitterAmount
}
