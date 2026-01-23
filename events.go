package flagkit

import (
	"time"

	"github.com/flagkit/flagkit-go/internal/core"
	"github.com/flagkit/flagkit-go/internal/http"
)

// Event represents an analytics event.
type Event = core.Event

// EventQueueConfig contains event queue configuration.
type EventQueueConfig struct {
	MaxSize       int
	FlushInterval time.Duration
	BatchSize     int
}

// DefaultEventQueueConfig returns the default event queue configuration.
func DefaultEventQueueConfig() *EventQueueConfig {
	cfg := core.DefaultEventQueueConfig()
	return &EventQueueConfig{
		MaxSize:       cfg.MaxSize,
		FlushInterval: cfg.FlushInterval,
		BatchSize:     cfg.BatchSize,
	}
}

// EventQueue manages analytics events with batching.
type EventQueue = core.EventQueue

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
	var config *core.EventQueueConfig
	if opts.Config != nil {
		config = &core.EventQueueConfig{
			MaxSize:       opts.Config.MaxSize,
			FlushInterval: opts.Config.FlushInterval,
			BatchSize:     opts.Config.BatchSize,
		}
	}

	// Type assertion to convert HTTPClient to http.Client
	var httpClient *http.Client
	if opts.HTTPClient != nil {
		httpClient = opts.HTTPClient
	}

	return core.NewEventQueue(&core.EventQueueOptions{
		HTTPClient:    httpClient,
		SessionID:     opts.SessionID,
		EnvironmentID: opts.EnvironmentID,
		SDKVersion:    opts.SDKVersion,
		Logger:        opts.Logger,
		Config:        config,
	})
}
