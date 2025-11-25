package exchange_account

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for exchange account data access
type Repository interface {
	Create(ctx context.Context, account *ExchangeAccount) error
	GetByID(ctx context.Context, id uuid.UUID) (*ExchangeAccount, error)
	GetByUser(ctx context.Context, userID uuid.UUID) ([]*ExchangeAccount, error)
	GetActiveByUser(ctx context.Context, userID uuid.UUID) ([]*ExchangeAccount, error)
	Update(ctx context.Context, account *ExchangeAccount) error
	UpdateLastSync(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}
