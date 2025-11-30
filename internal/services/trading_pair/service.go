package trading_pair

import (
	"context"

	"prometheus/internal/domain/trading_pair"
	"prometheus/internal/domain/user"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Service handles trading pair business logic
// Provides abstraction over trading pair repository
type Service struct {
	tradingPairRepo trading_pair.Repository
	userRepo        user.Repository
	log             *logger.Logger
}

// NewService creates a new trading pair service
func NewService(
	tradingPairRepo trading_pair.Repository,
	userRepo user.Repository,
	log *logger.Logger,
) *Service {
	return &Service{
		tradingPairRepo: tradingPairRepo,
		userRepo:        userRepo,
		log:             log,
	}
}

// UserWithPair represents a user and their trading pair configuration
type UserWithPair struct {
	User        *user.User
	TradingPair *trading_pair.TradingPair
}

// GetUsersMonitoringSymbol returns all active users monitoring a specific symbol
// with their trading pair configurations
// This is used by opportunity consumer to find users interested in a signal
func (s *Service) GetUsersMonitoringSymbol(ctx context.Context, symbol string) ([]*UserWithPair, error) {
	// Get all trading pairs for this symbol
	pairs, err := s.tradingPairRepo.GetActiveBySymbol(ctx, symbol)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get trading pairs")
	}

	if len(pairs) == 0 {
		s.log.Debugw("No users monitoring symbol", "symbol", symbol)
		return []*UserWithPair{}, nil
	}

	// Get users for each pair and filter by active + circuit breaker
	result := make([]*UserWithPair, 0, len(pairs))
	for _, pair := range pairs {
		usr, err := s.userRepo.GetByID(ctx, pair.UserID)
		if err != nil {
			s.log.Errorw("Failed to get user for trading pair",
				"user_id", pair.UserID,
				"symbol", symbol,
				"error", err,
			)
			continue
		}

		// Only include active users with circuit breaker on
		if !usr.IsActive || !usr.Settings.CircuitBreakerOn {
			s.log.Debugw("Skipping user (inactive or circuit breaker off)",
				"user_id", usr.ID,
				"is_active", usr.IsActive,
				"circuit_breaker_on", usr.Settings.CircuitBreakerOn,
			)
			continue
		}

		result = append(result, &UserWithPair{
			User:        usr,
			TradingPair: pair,
		})
	}

	s.log.Infow("Found users monitoring symbol",
		"symbol", symbol,
		"total_pairs", len(pairs),
		"active_users", len(result),
	)

	return result, nil
}

// GetByID retrieves a trading pair by ID
func (s *Service) GetByID(ctx context.Context, symbol string, userID string) (*trading_pair.TradingPair, error) {
	// TODO: Implement if needed
	return nil, errors.New("not implemented")
}
