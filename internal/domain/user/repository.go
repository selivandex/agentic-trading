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
}
