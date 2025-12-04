package user

import (
	"context"

	"github.com/google/uuid"
)

// LimitProfileAdapter adapts limit_profile.Repository to user.LimitProfileRepository interface
// This avoids circular dependency between user and limit_profile domains
type LimitProfileAdapter struct {
	getByName func(ctx context.Context, name string) (uuid.UUID, error)
}

// NewLimitProfileAdapter creates adapter with a function to get limit profile by name
func NewLimitProfileAdapter(getByName func(ctx context.Context, name string) (uuid.UUID, error)) *LimitProfileAdapter {
	return &LimitProfileAdapter{getByName: getByName}
}

// GetByName retrieves limit profile ID by name
func (a *LimitProfileAdapter) GetByName(ctx context.Context, name string) (LimitProfileInfo, error) {
	id, err := a.getByName(ctx, name)
	if err != nil {
		return LimitProfileInfo{}, err
	}
	return LimitProfileInfo{ID: id}, nil
}


