package types

import (
	"fmt"
	"log"
	"os"
)

// Logger defines the interface for logging.
type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
}

// DefaultLogger is the default logger implementation.
type DefaultLogger struct {
	debug  bool
	logger *log.Logger
}

// NewDefaultLogger creates a new default logger.
func NewDefaultLogger(debug bool) *DefaultLogger {
	return &DefaultLogger{
		debug:  debug,
		logger: log.New(os.Stdout, "[flagkit] ", log.LstdFlags),
	}
}

func (l *DefaultLogger) formatMessage(level, msg string, keysAndValues ...any) string {
	if len(keysAndValues) == 0 {
		return fmt.Sprintf("%s %s", level, msg)
	}

	pairs := ""
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			pairs += fmt.Sprintf(" %v=%v", keysAndValues[i], keysAndValues[i+1])
		}
	}
	return fmt.Sprintf("%s %s%s", level, msg, pairs)
}

// Debug logs a debug message.
func (l *DefaultLogger) Debug(msg string, keysAndValues ...any) {
	if l.debug {
		l.logger.Println(l.formatMessage("DEBUG", msg, keysAndValues...))
	}
}

// Info logs an info message.
func (l *DefaultLogger) Info(msg string, keysAndValues ...any) {
	l.logger.Println(l.formatMessage("INFO", msg, keysAndValues...))
}

// Warn logs a warning message.
func (l *DefaultLogger) Warn(msg string, keysAndValues ...any) {
	l.logger.Println(l.formatMessage("WARN", msg, keysAndValues...))
}

// Error logs an error message.
func (l *DefaultLogger) Error(msg string, keysAndValues ...any) {
	l.logger.Println(l.formatMessage("ERROR", msg, keysAndValues...))
}

// NullLogger is a logger that discards all messages.
type NullLogger struct{}

// Debug does nothing.
func (l *NullLogger) Debug(msg string, keysAndValues ...any) {}

// Info does nothing.
func (l *NullLogger) Info(msg string, keysAndValues ...any) {}

// Warn does nothing.
func (l *NullLogger) Warn(msg string, keysAndValues ...any) {}

// Error does nothing.
func (l *NullLogger) Error(msg string, keysAndValues ...any) {}
