package order

import (
	"context"

	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Service contains business logic for orders.
type Service struct {
	repo Repository
	log  *logger.Logger
}

// NewService constructs an order service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, log: logger.Get()}
}

// Place creates a new order in pending status.
func (s *Service) Place(ctx context.Context, input PlaceParams) (*Order, error) {
	if err := input.validate(); err != nil {
		return nil, err
	}

	order := &Order{
		ID:                uuid.New(),
		UserID:            input.UserID,
		TradingPairID:     input.TradingPairID,
		ExchangeAccountID: input.ExchangeAccountID,
		Symbol:            input.Symbol,
		MarketType:        input.MarketType,
		Side:              input.Side,
		Type:              input.Type,
		Status:            OrderStatusPending,
		Price:             input.Price,
		Amount:            input.Amount,
		StopPrice:         input.StopPrice,
		ReduceOnly:        input.ReduceOnly,
		AgentID:           input.AgentID,
		Reasoning:         input.Reasoning,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, order); err != nil {
		return nil, errors.Wrap(err, "place order")
	}
	return order, nil
}

// UpdateStatus updates an order's status and fill information.
func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, status OrderStatus, filledAmount, avgPrice decimal.Decimal) error {
	if id == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if !status.Valid() {
		return errors.ErrInvalidInput
	}
	if err := s.repo.UpdateStatus(ctx, id, status, filledAmount, avgPrice); err != nil {
		return errors.Wrap(err, "update order status")
	}
	return nil
}

// Cancel cancels an order by ID.
func (s *Service) Cancel(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if err := s.repo.Cancel(ctx, id); err != nil {
		return errors.Wrap(err, "cancel order")
	}
	return nil
}

// GetByID returns a single order.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Order, error) {
	if id == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}
	o, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "get order")
	}
	return o, nil
}

// GetOpenByUser fetches open orders for a user.
func (s *Service) GetOpenByUser(ctx context.Context, userID uuid.UUID) ([]*Order, error) {
	if userID == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}
	orders, err := s.repo.GetOpenByUser(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "get open orders")
	}
	return orders, nil
}

// Update persists changes to an order.
func (s *Service) Update(ctx context.Context, order *Order) error {
	if order == nil {
		return errors.ErrInvalidInput
	}
	if order.ID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if err := s.repo.Update(ctx, order); err != nil {
		return errors.Wrap(err, "update order")
	}
	return nil
}

// Delete removes an order permanently.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "delete order")
	}
	return nil
}

// PlaceParams captures the required data to create an order.
type PlaceParams struct {
	UserID            uuid.UUID
	TradingPairID     uuid.UUID
	ExchangeAccountID uuid.UUID
	Symbol            string
	MarketType        string
	Side              OrderSide
	Type              OrderType
	Price             decimal.Decimal
	Amount            decimal.Decimal
	StopPrice         decimal.Decimal
	ReduceOnly        bool
	AgentID           string
	Reasoning         string
}

func (p PlaceParams) validate() error {
	if p.UserID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if p.TradingPairID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if p.ExchangeAccountID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if p.Symbol == "" {
		return errors.ErrInvalidInput
	}
	if !p.Side.Valid() {
		return errors.ErrInvalidInput
	}
	if !p.Type.Valid() {
		return errors.ErrInvalidInput
	}
	if p.Amount.LessThanOrEqual(decimal.Zero) {
		return errors.ErrInvalidInput
	}
	if p.Type == OrderTypeLimit && p.Price.LessThanOrEqual(decimal.Zero) {
		return errors.ErrInvalidInput
	}
	return nil
}
