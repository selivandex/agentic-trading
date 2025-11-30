package limit_profile

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines operations for limit profile persistence
type Repository interface {
	// Create creates a new limit profile
	Create(ctx context.Context, profile *LimitProfile) error

	// GetByID retrieves a limit profile by ID
	GetByID(ctx context.Context, id uuid.UUID) (*LimitProfile, error)

	// GetByName retrieves a limit profile by name (e.g., "free", "basic", "premium")
	GetByName(ctx context.Context, name string) (*LimitProfile, error)

	// GetAll retrieves all limit profiles
	GetAll(ctx context.Context) ([]*LimitProfile, error)

	// GetAllActive retrieves all active limit profiles
	GetAllActive(ctx context.Context) ([]*LimitProfile, error)

	// Update updates an existing limit profile
	Update(ctx context.Context, profile *LimitProfile) error

	// Delete deletes a limit profile (soft delete via is_active flag)
	Delete(ctx context.Context, id uuid.UUID) error
}
