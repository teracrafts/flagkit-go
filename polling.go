package flagkit

import (
	"time"

	"github.com/flagkit/flagkit-go/internal/core"
)

// PollingConfig contains polling configuration.
type PollingConfig struct {
	Interval          time.Duration
	Jitter            time.Duration
	BackoffMultiplier float64
	MaxInterval       time.Duration
}

// DefaultPollingConfig returns the default polling configuration.
func DefaultPollingConfig() *PollingConfig {
	cfg := core.DefaultPollingConfig()
	return &PollingConfig{
		Interval:          cfg.Interval,
		Jitter:            cfg.Jitter,
		BackoffMultiplier: cfg.BackoffMultiplier,
		MaxInterval:       cfg.MaxInterval,
	}
}

// PollingManager manages background polling for flag updates.
type PollingManager = core.PollingManager

// NewPollingManager creates a new polling manager.
func NewPollingManager(onPoll func(), config *PollingConfig, logger Logger) *PollingManager {
	var coreConfig *core.PollingConfig
	if config != nil {
		coreConfig = &core.PollingConfig{
			Interval:          config.Interval,
			Jitter:            config.Jitter,
			BackoffMultiplier: config.BackoffMultiplier,
			MaxInterval:       config.MaxInterval,
		}
	}
	return core.NewPollingManager(onPoll, coreConfig, logger)
}
