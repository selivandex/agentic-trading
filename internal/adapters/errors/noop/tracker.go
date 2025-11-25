package noop

import (
	"context"

	"prometheus/pkg/errors"
)

// Tracker is a no-op implementation of the error tracker
// Used when error tracking is disabled or for testing
type Tracker struct{}

// New creates a new no-op tracker
func New() *Tracker {
	return &Tracker{}
}

// CaptureError does nothing
func (t *Tracker) CaptureError(ctx context.Context, err error, tags map[string]string) error {
	return nil
}

// CaptureMessage does nothing
func (t *Tracker) CaptureMessage(ctx context.Context, message string, level errors.Level, tags map[string]string) error {
	return nil
}

// SetUser does nothing
func (t *Tracker) SetUser(ctx context.Context, userID string, email string, username string) {
}

// AddBreadcrumb does nothing
func (t *Tracker) AddBreadcrumb(ctx context.Context, message string, category string, level errors.Level, data map[string]interface{}) {
}

// Flush does nothing
func (t *Tracker) Flush(ctx context.Context) error {
	return nil
}
