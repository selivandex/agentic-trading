package sentry

import (
	"context"
	"time"

	"github.com/getsentry/sentry-go"

	"prometheus/pkg/errors"
)

// Tracker implements error tracking via Sentry
type Tracker struct {
	hub *sentry.Hub
}

// New creates a new Sentry tracker
func New(dsn string, environment string) (*Tracker, error) {
	err := sentry.Init(sentry.ClientOptions{
		Dsn:         dsn,
		Environment: environment,
	})
	if err != nil {
		return nil, err
	}

	return &Tracker{
		hub: sentry.CurrentHub(),
	}, nil
}

// CaptureError sends an error to Sentry
func (t *Tracker) CaptureError(ctx context.Context, err error, tags map[string]string) error {
	hub := t.hub.Clone()

	// Configure scope with tags and user
	hub.ConfigureScope(func(scope *sentry.Scope) {
		// Add tags
		for k, v := range tags {
			scope.SetTag(k, v)
		}

		// Add user context if available
		if userID, ok := ctx.Value("user_id").(string); ok {
			scope.SetUser(sentry.User{ID: userID})
		}
	})

	hub.CaptureException(err)
	return nil
}

// CaptureMessage sends a message to Sentry
func (t *Tracker) CaptureMessage(ctx context.Context, message string, level errors.Level, tags map[string]string) error {
	hub := t.hub.Clone()

	// Convert level
	sentryLevel := t.convertLevel(level)

	// Configure scope with tags and level
	hub.ConfigureScope(func(scope *sentry.Scope) {
		// Add tags
		for k, v := range tags {
			scope.SetTag(k, v)
		}
		scope.SetLevel(sentryLevel)
	})

	hub.CaptureMessage(message)
	return nil
}

// SetUser associates the current context with a user
func (t *Tracker) SetUser(ctx context.Context, userID string, email string, username string) {
	t.hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{
			ID:       userID,
			Email:    email,
			Username: username,
		})
	})
}

// AddBreadcrumb adds a breadcrumb to track user actions
func (t *Tracker) AddBreadcrumb(ctx context.Context, message string, category string, level errors.Level, data map[string]interface{}) {
	sentryLevel := t.convertLevel(level)

	t.hub.AddBreadcrumb(&sentry.Breadcrumb{
		Message:  message,
		Category: category,
		Level:    sentryLevel,
		Data:     data,
	}, &sentry.BreadcrumbHint{})
}

// Flush waits for all pending events to be sent
func (t *Tracker) Flush(ctx context.Context) error {
	if sentry.Flush(2 * time.Second) {
		return nil
	}
	return nil // Sentry.Flush returns bool, not error
}

// convertLevel converts our level to Sentry level
func (t *Tracker) convertLevel(level errors.Level) sentry.Level {
	switch level {
	case errors.LevelDebug:
		return sentry.LevelDebug
	case errors.LevelInfo:
		return sentry.LevelInfo
	case errors.LevelWarning:
		return sentry.LevelWarning
	case errors.LevelError:
		return sentry.LevelError
	case errors.LevelFatal:
		return sentry.LevelFatal
	default:
		return sentry.LevelInfo
	}
}
