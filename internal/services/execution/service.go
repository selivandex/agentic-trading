package execution

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/order"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// TradingPlan represents an approved trading plan ready for execution.
type TradingPlan struct {
	UserID     uuid.UUID
	StrategyID *uuid.UUID // Optional: link to strategy
	Symbol     string
	Side       order.OrderSide
	Amount     decimal.Decimal
	OrderType  order.OrderType
	Price      decimal.Decimal // For limit orders
	StopPrice  decimal.Decimal // For stop orders
	Exchange   string
	Urgency    UrgencyLevel
	TimeWindow time.Duration
}

// UrgencyLevel defines execution urgency.
type UrgencyLevel string

const (
	UrgencyLow    UrgencyLevel = "low"    // Can wait, optimize for fees
	UrgencyMedium UrgencyLevel = "medium" // Normal execution
	UrgencyHigh   UrgencyLevel = "high"   // Fast execution needed
)

// ExecutionResult contains execution outcome.
type ExecutionResult struct {
	OrderID      string
	Status       string
	FilledQty    decimal.Decimal
	AvgPrice     decimal.Decimal
	Fee          decimal.Decimal
	Slippage     decimal.Decimal
	Duration     time.Duration
	ExecutedAt   time.Time
	ErrorMessage string
}

// VenueInfo contains exchange liquidity info.
type VenueInfo struct {
	Exchange        string
	Spread          decimal.Decimal
	DepthBid        decimal.Decimal
	DepthAsk        decimal.Decimal
	VolumePerMinute decimal.Decimal
	FeeMaker        decimal.Decimal
	FeeTaker        decimal.Decimal
	Latency         time.Duration
}

// ExecutionService handles order execution with venue selection and slicing.
type ExecutionService struct {
	orderService        *order.Service
	exchangeAccountRepo exchange_account.Repository
	log                 *logger.Logger
}

// NewExecutionService creates a new execution service.
func NewExecutionService(orderService *order.Service, exchangeAccountRepo exchange_account.Repository) *ExecutionService {
	return &ExecutionService{
		orderService:        orderService,
		exchangeAccountRepo: exchangeAccountRepo,
		log:                 logger.Get().With("component", "execution_service"),
	}
}

// Execute places an order with optimal execution strategy.
func (s *ExecutionService) Execute(ctx context.Context, plan TradingPlan) (*ExecutionResult, error) {
	s.log.Infow("Executing trading plan",
		"user_id", plan.UserID,
		"symbol", plan.Symbol,
		"side", plan.Side,
		"amount", plan.Amount,
	)

	startTime := time.Now()

	// Resolve exchange account for user
	// For MVP: use first active exchange account
	accounts, err := s.exchangeAccountRepo.GetActiveByUser(ctx, plan.UserID)
	if err != nil {
		return &ExecutionResult{
			Status:       "failed",
			ErrorMessage: "failed to get exchange accounts",
			Duration:     time.Since(startTime),
		}, errors.Wrap(err, "failed to get exchange accounts")
	}

	// GetActiveByUser already filters by IsActive, just take first one
	var exchangeAccountID uuid.UUID
	if len(accounts) > 0 {
		exchangeAccountID = accounts[0].ID
	}

	if exchangeAccountID == uuid.Nil {
		return &ExecutionResult{
			Status:       "failed",
			ErrorMessage: "no active exchange account found",
			Duration:     time.Since(startTime),
		}, errors.New("no active exchange account found")
	}

	s.log.Debugw("Resolved exchange account for order execution",
		"user_id", plan.UserID,
		"exchange_account_id", exchangeAccountID,
	)

	// For now, use simple execution via order service
	// TODO: Add venue selection, slicing, TWAP logic in future iterations
	orderParams := order.PlaceParams{
		UserID:            plan.UserID,
		StrategyID:        plan.StrategyID, // From trading plan
		ExchangeAccountID: exchangeAccountID,
		Symbol:            plan.Symbol,
		MarketType:        "spot", // TODO: Parse from symbol or plan
		Side:              plan.Side,
		Type:              plan.OrderType,
		Amount:            plan.Amount,
		Price:             plan.Price,
		StopPrice:         plan.StopPrice,
		ReduceOnly:        false,
		AgentID:           "execution_service",
		Reasoning:         "Automated execution via ExecutionService",
	}

	// Place order
	createdOrder, err := s.orderService.Place(ctx, orderParams)
	if err != nil {
		return &ExecutionResult{
			Status:       "failed",
			ErrorMessage: err.Error(),
			Duration:     time.Since(startTime),
		}, errors.Wrap(err, "failed to place order")
	}

	s.log.Infow("Order placed successfully",
		"order_id", createdOrder.ID,
		"status", createdOrder.Status,
	)

	return &ExecutionResult{
		OrderID:    createdOrder.ID.String(),
		Status:     string(createdOrder.Status),
		FilledQty:  createdOrder.FilledAmount,
		AvgPrice:   createdOrder.AvgFillPrice,
		Fee:        createdOrder.Fee,
		Duration:   time.Since(startTime),
		ExecutedAt: createdOrder.CreatedAt,
	}, nil
}

// selectBestVenue selects optimal exchange for execution (algorithmic).
func (s *ExecutionService) selectBestVenue(ctx context.Context, symbol string, venues []VenueInfo) string {
	// TODO: Implement algorithmic venue selection based on:
	// - Liquidity depth (prefer depth >5x order size)
	// - Spread (<10bps preferred)
	// - Fees (lower is better)
	// - Latency (<200ms preferred)

	// For now, return first venue
	if len(venues) > 0 {
		return venues[0].Exchange
	}
	return ""
}

// selectOrderType selects order type based on urgency and liquidity (algorithmic).
func (s *ExecutionService) selectOrderType(urgency UrgencyLevel, liquidity VenueInfo) order.OrderType {
	// Decision matrix:
	// High urgency + good liquidity → Market
	// Medium urgency + good liquidity → Aggressive limit
	// Low urgency + good liquidity → Limit post-only
	// Any urgency + low liquidity → Iceberg/TWAP

	switch urgency {
	case UrgencyHigh:
		return order.OrderTypeMarket
	case UrgencyMedium:
		return order.OrderTypeLimit
	case UrgencyLow:
		return order.OrderTypeLimit
	default:
		return order.OrderTypeLimit
	}
}

// shouldSlice determines if order needs slicing (algorithmic).
func (s *ExecutionService) shouldSlice(orderSize, volumePerMinute decimal.Decimal) bool {
	// Slice if order >20% of 1-min volume (to minimize market impact)
	threshold := volumePerMinute.Mul(decimal.NewFromFloat(0.20))
	return orderSize.GreaterThan(threshold)
}
