package flagkit

import (
	"github.com/flagkit/flagkit-go/internal/types"
)

// Logger defines the interface for logging.
type Logger = types.Logger

// DefaultLogger is the default logger implementation.
type DefaultLogger = types.DefaultLogger

// NewDefaultLogger creates a new default logger.
func NewDefaultLogger(debug bool) *DefaultLogger {
	return types.NewDefaultLogger(debug)
}

// NullLogger is a logger that discards all messages.
type NullLogger = types.NullLogger
