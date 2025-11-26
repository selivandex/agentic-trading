package order

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

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
		return nil, fmt.Errorf("place order: %w", err)
	}
	return order, nil
}

// UpdateStatus updates an order's status and fill information.
func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, status OrderStatus, filledAmount, avgPrice decimal.Decimal) error {
	if id == uuid.Nil {
		return fmt.Errorf("update order status: id is required")
	}
	if !status.Valid() {
		return fmt.Errorf("update order status: invalid status")
	}
	if err := s.repo.UpdateStatus(ctx, id, status, filledAmount, avgPrice); err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	return nil
}

// Cancel cancels an order by ID.
func (s *Service) Cancel(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("cancel order: id is required")
	}
	if err := s.repo.Cancel(ctx, id); err != nil {
		return fmt.Errorf("cancel order: %w", err)
	}
	return nil
}

// GetByID returns a single order.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Order, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("get order: id is required")
	}
	o, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}
	return o, nil
}

// GetOpenByUser fetches open orders for a user.
func (s *Service) GetOpenByUser(ctx context.Context, userID uuid.UUID) ([]*Order, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("get open orders: user id is required")
	}
	orders, err := s.repo.GetOpenByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get open orders: %w", err)
	}
	return orders, nil
}

// Update persists changes to an order.
func (s *Service) Update(ctx context.Context, order *Order) error {
	if order == nil {
		return fmt.Errorf("update order: order is nil")
	}
	if order.ID == uuid.Nil {
		return fmt.Errorf("update order: id is required")
	}
	if err := s.repo.Update(ctx, order); err != nil {
		return fmt.Errorf("update order: %w", err)
	}
	return nil
}

// Delete removes an order permanently.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("delete order: id is required")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete order: %w", err)
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
		return fmt.Errorf("place order: user id is required")
	}
	if p.TradingPairID == uuid.Nil {
		return fmt.Errorf("place order: trading pair id is required")
	}
	if p.ExchangeAccountID == uuid.Nil {
		return fmt.Errorf("place order: exchange account id is required")
	}
	if p.Symbol == "" {
		return fmt.Errorf("place order: symbol is required")
	}
	if !p.Side.Valid() {
		return fmt.Errorf("place order: invalid side")
	}
	if !p.Type.Valid() {
		return fmt.Errorf("place order: invalid type")
	}
	if p.Amount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("place order: amount must be positive")
	}
	if p.Type == OrderTypeLimit && p.Price.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("place order: price must be set for limit orders")
	}
	return nil
}
