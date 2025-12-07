package core

import (
	"fmt"
	"log"
	"os"
)

// DefaultLogger is a simple logger implementation
type DefaultLogger struct {
	logger *log.Logger
}

// NewDefaultLogger creates a new default logger
func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{
		logger: log.New(os.Stdout, "[BeaconAuth] ", log.LstdFlags),
	}
}

// Debug logs a debug message
func (l *DefaultLogger) Debug(msg string, fields ...interface{}) {
	l.logger.Printf("[DEBUG] %s %v", msg, fields)
}

// Info logs an info message
func (l *DefaultLogger) Info(msg string, fields ...interface{}) {
	l.logger.Printf("[INFO] %s %v", msg, fields)
}

// Warn logs a warning message
func (l *DefaultLogger) Warn(msg string, fields ...interface{}) {
	l.logger.Printf("[WARN] %s %v", msg, fields)
}

// Error logs an error message
func (l *DefaultLogger) Error(msg string, fields ...interface{}) {
	l.logger.Printf("[ERROR] %s %v", msg, fields)
}

// NoopLogger is a logger that does nothing
type NoopLogger struct{}

// NewNoopLogger creates a new noop logger
func NewNoopLogger() *NoopLogger {
	return &NoopLogger{}
}

// Debug does nothing
func (l *NoopLogger) Debug(msg string, fields ...interface{}) {}

// Info does nothing
func (l *NoopLogger) Info(msg string, fields ...interface{}) {}

// Warn does nothing
func (l *NoopLogger) Warn(msg string, fields ...interface{}) {}

// Error does nothing
func (l *NoopLogger) Error(msg string, fields ...interface{}) {}

// LogLevel represents the logging level
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", l)
	}
}
