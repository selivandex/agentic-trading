package user

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for user data access
// Implementation will be in internal/repository/postgres/user.go
type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByTelegramID(ctx context.Context, telegramID int64) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*User, error)

	// GetUsersWithScope returns users filtered by scope, search and filters
	// Scope examples: "all", "active", "inactive", "premium", "free"
	// Search: searches across firstName, lastName, email, telegramUsername
	// Filters: dynamic filters as key-value map (e.g. {"is_active": "true", "risk_level": ["CONSERVATIVE", "MODERATE"]})
	GetUsersWithScope(ctx context.Context, scope *string, search *string, filters map[string]any) ([]*User, error)

	// GetUsersScopes returns count for each scope
	// Used for tab badges in frontend
	GetUsersScopes(ctx context.Context) (map[string]int, error)
}
