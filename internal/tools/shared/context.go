package shared

import (
	"context"

	"github.com/google/uuid"
)

type contextKey struct{}

// InvocationMetadata captures request-scoped identifiers for tool telemetry.
type InvocationMetadata struct {
	UserID    uuid.UUID
	AgentID   string
	SessionID string
	Symbol    string
}

// WithInvocationMetadata injects tool invocation metadata into a context.
func WithInvocationMetadata(ctx context.Context, meta InvocationMetadata) context.Context {
	return context.WithValue(ctx, contextKey{}, meta)
}

// MetadataFromContext extracts invocation metadata if present.
func MetadataFromContext(ctx context.Context) (InvocationMetadata, bool) {
	meta, ok := ctx.Value(contextKey{}).(InvocationMetadata)
	return meta, ok
}
