package position

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/domain/position"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Service handles business logic for position management
type Service struct {
	repo position.Repository
	log  *logger.Logger
}

// NewService creates a new position service
func NewService(repo position.Repository, log *logger.Logger) *Service {
	return &Service{
		repo: repo,
		log:  log,
	}
}

// UpdateFromWebSocket updates or creates a position based on WebSocket user data event
func (s *Service) UpdateFromWebSocket(ctx context.Context, update *PositionWebSocketUpdate) error {
	// If amount is zero, position is closed
	if update.Amount.IsZero() {
		return s.ClosePosition(ctx, update.UserID, update.AccountID, update.Symbol)
	}

	// Get existing position
	positions, err := s.repo.GetOpenByUser(ctx, update.UserID)
	if err != nil {
		return errors.Wrap(err, "failed to get positions")
	}

	// Find position for this symbol and account
	var existingPos *position.Position
	for _, pos := range positions {
		if pos.Symbol == update.Symbol && pos.ExchangeAccountID == update.AccountID {
			existingPos = pos
			break
		}
	}

	if existingPos == nil {
		// Create new position
		newPos := &position.Position{
			ID:                uuid.New(),
			UserID:            update.UserID,
			ExchangeAccountID: update.AccountID,
			Symbol:            update.Symbol,
			Side:              mapPositionSide(update.Side),
			Size:              update.Amount,
			EntryPrice:        update.EntryPrice,
			CurrentPrice:      update.MarkPrice,
			UnrealizedPnL:     update.UnrealizedPnL,
			Status:            position.PositionOpen,
		}

		if err := s.repo.Create(ctx, newPos); err != nil {
			return errors.Wrap(err, "failed to create position")
		}

		s.log.Infow("Created new position",
			"position_id", newPos.ID,
			"symbol", update.Symbol,
			"user_id", update.UserID,
		)
		return nil
	}

	// Update existing position
	existingPos.Size = update.Amount
	existingPos.CurrentPrice = update.MarkPrice
	existingPos.UnrealizedPnL = update.UnrealizedPnL

	if err := s.repo.Update(ctx, existingPos); err != nil {
		return errors.Wrap(err, "failed to update position")
	}

	s.log.Debugw("Updated position",
		"position_id", existingPos.ID,
		"symbol", update.Symbol,
	)

	return nil
}

// ClosePosition closes an open position
func (s *Service) ClosePosition(ctx context.Context, userID, accountID uuid.UUID, symbol string) error {
	positions, err := s.repo.GetOpenByUser(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "failed to get positions")
	}

	// Find position for this symbol and account
	for _, pos := range positions {
		if pos.Symbol == symbol && pos.ExchangeAccountID == accountID {
			pos.Status = position.PositionClosed

			if err := s.repo.Update(ctx, pos); err != nil {
				return errors.Wrap(err, "failed to close position")
			}

			s.log.Infow("Closed position",
				"position_id", pos.ID,
				"symbol", symbol,
				"user_id", userID,
			)
			return nil
		}
	}

	s.log.Debugw("Position not found (already closed)",
		"symbol", symbol,
		"user_id", userID,
	)
	return nil // Position not found (already closed)
}

// PositionWebSocketUpdate represents a position update from WebSocket
type PositionWebSocketUpdate struct {
	UserID        uuid.UUID
	AccountID     uuid.UUID
	Symbol        string
	Side          string
	Amount        decimal.Decimal
	EntryPrice    decimal.Decimal
	MarkPrice     decimal.Decimal
	UnrealizedPnL decimal.Decimal
}

// Helper function to map position side
func mapPositionSide(side string) position.PositionSide {
	switch side {
	case "LONG":
		return position.PositionLong
	case "SHORT":
		return position.PositionShort
	default:
		return position.PositionLong
	}
}

// ParsePositionUpdateFromEvent parses UserDataPositionUpdateEvent into PositionWebSocketUpdate
func ParsePositionUpdateFromEvent(event *eventspb.UserDataPositionUpdateEvent) (*PositionWebSocketUpdate, error) {
	userID, err := uuid.Parse(event.Base.UserId)
	if err != nil {
		return nil, errors.Wrap(err, "invalid user_id")
	}

	accountID, err := uuid.Parse(event.AccountId)
	if err != nil {
		return nil, errors.Wrap(err, "invalid account_id")
	}

	amount, err := parseDecimal(event.Amount)
	if err != nil {
		return nil, errors.Wrap(err, "invalid amount")
	}

	entryPrice, _ := parseDecimal(event.EntryPrice)
	markPrice, _ := parseDecimal(event.MarkPrice)
	unrealizedPnL, _ := parseDecimal(event.UnrealizedPnl)

	return &PositionWebSocketUpdate{
		UserID:        userID,
		AccountID:     accountID,
		Symbol:        event.Symbol,
		Side:          event.Side,
		Amount:        amount,
		EntryPrice:    entryPrice,
		MarkPrice:     markPrice,
		UnrealizedPnL: unrealizedPnL,
	}, nil
}

// parseDecimal parses string to decimal
func parseDecimal(s string) (decimal.Decimal, error) {
	if s == "" || s == "0" {
		return decimal.Zero, nil
	}

	val, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, err
	}

	return val, nil
}
