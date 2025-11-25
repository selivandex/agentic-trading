package errors

import (
	"context"
)

// Tracker defines the interface for error tracking services (Sentry, Datadog, etc.)
type Tracker interface {
	// CaptureError sends an error to the tracking service
	CaptureError(ctx context.Context, err error, tags map[string]string) error

	// CaptureMessage sends a message to the tracking service
	CaptureMessage(ctx context.Context, message string, level Level, tags map[string]string) error

	// SetUser associates the current context with a user
	SetUser(ctx context.Context, userID string, email string, username string)

	// AddBreadcrumb adds a breadcrumb to track user actions
	AddBreadcrumb(ctx context.Context, message string, category string, level Level, data map[string]interface{})

	// Flush waits for all pending events to be sent
	Flush(ctx context.Context) error
}

// Level represents the severity level of an error or message
type Level string

const (
	LevelDebug   Level = "debug"
	LevelInfo    Level = "info"
	LevelWarning Level = "warning"
	LevelError   Level = "error"
	LevelFatal   Level = "fatal"
)

// String returns the string representation of the level
func (l Level) String() string {
	return string(l)
}
