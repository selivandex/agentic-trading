package limits

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/limit_profile"
	"prometheus/internal/domain/user"
	pkgerrors "prometheus/pkg/errors"
)

// Service handles limit validation logic
type Service struct {
	limitProfileRepo    limit_profile.Repository
	exchangeAccountRepo exchange_account.Repository
	userRepo            user.Repository
}

// NewService creates a new limits service
func NewService(
	limitProfileRepo limit_profile.Repository,
	exchangeAccountRepo exchange_account.Repository,
	userRepo user.Repository,
) *Service {
	return &Service{
		limitProfileRepo:    limitProfileRepo,
		exchangeAccountRepo: exchangeAccountRepo,
		userRepo:            userRepo,
	}
}

// CheckExchangeAccountLimit checks if user can create a new exchange account
// Returns error if limit is exceeded
func (s *Service) CheckExchangeAccountLimit(ctx context.Context, userID uuid.UUID) error {
	// Get user
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to get user")
	}

	// Get user's limit profile
	if u.LimitProfileID == nil {
		return fmt.Errorf("user has no limit profile assigned")
	}

	profile, err := s.limitProfileRepo.GetByID(ctx, *u.LimitProfileID)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to get limit profile")
	}

	// Parse limits
	limits, err := profile.ParseLimits()
	if err != nil {
		return pkgerrors.Wrap(err, "failed to parse limits")
	}

	// Get current exchange accounts count
	accounts, err := s.exchangeAccountRepo.GetByUser(ctx, userID)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to get user exchange accounts")
	}

	currentCount := len(accounts)

	// Check limit
	if currentCount >= limits.ExchangesCount {
		return fmt.Errorf("exchange account limit exceeded: you have %d/%d exchanges (upgrade your plan to add more)",
			currentCount, limits.ExchangesCount)
	}

	return nil
}

// GetUserLimits retrieves the limits for a user
func (s *Service) GetUserLimits(ctx context.Context, userID uuid.UUID) (*limit_profile.Limits, error) {
	// Get user
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to get user")
	}

	// Get user's limit profile
	if u.LimitProfileID == nil {
		// Return default free tier if no profile assigned
		freeLimits := limit_profile.FreeTierLimits()
		return &freeLimits, nil
	}

	profile, err := s.limitProfileRepo.GetByID(ctx, *u.LimitProfileID)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to get limit profile")
	}

	// Parse limits
	limits, err := profile.ParseLimits()
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to parse limits")
	}

	return limits, nil
}

// CheckPositionLimit checks if user can open a new position
func (s *Service) CheckPositionLimit(ctx context.Context, userID uuid.UUID, currentPositions int) error {
	limits, err := s.GetUserLimits(ctx, userID)
	if err != nil {
		return err
	}

	if currentPositions >= limits.ActivePositions {
		return fmt.Errorf("position limit exceeded: you have %d/%d positions (upgrade your plan to open more)",
			currentPositions, limits.ActivePositions)
	}

	return nil
}

// CheckDailyTradesLimit checks if user can execute more trades today
func (s *Service) CheckDailyTradesLimit(ctx context.Context, userID uuid.UUID, todayTrades int) error {
	limits, err := s.GetUserLimits(ctx, userID)
	if err != nil {
		return err
	}

	// -1 means unlimited
	if limits.DailyTradesCount == -1 {
		return nil
	}

	if todayTrades >= limits.DailyTradesCount {
		return fmt.Errorf("daily trades limit exceeded: you have %d/%d trades today (upgrade your plan for more)",
			todayTrades, limits.DailyTradesCount)
	}

	return nil
}

// CheckMonthlyTradesLimit checks if user can execute more trades this month
func (s *Service) CheckMonthlyTradesLimit(ctx context.Context, userID uuid.UUID, monthTrades int) error {
	limits, err := s.GetUserLimits(ctx, userID)
	if err != nil {
		return err
	}

	// -1 means unlimited
	if limits.MonthlyTradesCount == -1 {
		return nil
	}

	if monthTrades >= limits.MonthlyTradesCount {
		return fmt.Errorf("monthly trades limit exceeded: you have %d/%d trades this month (upgrade your plan)",
			monthTrades, limits.MonthlyTradesCount)
	}

	return nil
}

// CheckAIRequestsLimit checks if user can make more AI requests this month
func (s *Service) CheckAIRequestsLimit(ctx context.Context, userID uuid.UUID, monthlyRequests int) error {
	limits, err := s.GetUserLimits(ctx, userID)
	if err != nil {
		return err
	}

	if monthlyRequests >= limits.MonthlyAIRequests {
		return fmt.Errorf("AI requests limit exceeded: you have %d/%d requests this month (upgrade for more)",
			monthlyRequests, limits.MonthlyAIRequests)
	}

	return nil
}

// CheckFeatureAccess checks if user has access to a specific feature
func (s *Service) CheckFeatureAccess(ctx context.Context, userID uuid.UUID, feature string) (bool, error) {
	limits, err := s.GetUserLimits(ctx, userID)
	if err != nil {
		return false, err
	}

	switch feature {
	case "live_trading":
		return limits.LiveTradingAllowed, nil
	case "paper_trading":
		return limits.PaperTradingAllowed, nil
	case "backtesting":
		return limits.BacktestingAllowed, nil
	case "advanced_agents":
		return limits.AdvancedAgentsAccess, nil
	case "custom_agents":
		return limits.CustomAgentsAllowed, nil
	case "advanced_order_types":
		return limits.AdvancedOrderTypes, nil
	case "webhooks":
		return limits.WebhooksAllowed, nil
	case "api_access":
		return limits.APIAccessAllowed, nil
	case "realtime_data":
		return limits.RealtimeDataAccess, nil
	case "advanced_charts":
		return limits.AdvancedCharts, nil
	case "custom_indicators":
		return limits.CustomIndicators, nil
	case "portfolio_analytics":
		return limits.PortfolioAnalytics, nil
	case "risk_management_tools":
		return limits.RiskManagementTools, nil
	case "custom_reports":
		return limits.CustomReportsAllowed, nil
	case "priority_support":
		return limits.PrioritySupport, nil
	case "dedicated_support":
		return limits.DedicatedSupport, nil
	case "priority_execution":
		return limits.PriorityExecution, nil
	case "advanced_ai_models":
		return limits.AdvancedAIModels, nil
	default:
		return false, fmt.Errorf("unknown feature: %s", feature)
	}
}

// AssignLimitProfile assigns a limit profile to a user
func (s *Service) AssignLimitProfile(ctx context.Context, userID uuid.UUID, profileName string) error {
	// Get user
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to get user")
	}

	// Get limit profile by name
	profile, err := s.limitProfileRepo.GetByName(ctx, profileName)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to get limit profile")
	}

	// Assign profile to user
	u.LimitProfileID = &profile.ID

	// Update user
	err = s.userRepo.Update(ctx, u)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to update user")
	}

	return nil
}

// GetLimitUsage returns current usage vs limits for a user
func (s *Service) GetLimitUsage(ctx context.Context, userID uuid.UUID) (*LimitUsage, error) {
	limits, err := s.GetUserLimits(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get current exchange accounts count
	accounts, err := s.exchangeAccountRepo.GetByUser(ctx, userID)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to get exchange accounts")
	}

	usage := &LimitUsage{
		Limits:                limits,
		ExchangesUsed:         len(accounts),
		ExchangesLimit:        limits.ExchangesCount,
		ExchangesPercentage:   calculatePercentage(len(accounts), limits.ExchangesCount),
		ActivePositionsLimit:  limits.ActivePositions,
		DailyTradesLimit:      limits.DailyTradesCount,
		MonthlyTradesLimit:    limits.MonthlyTradesCount,
		MonthlyAIRequestLimit: limits.MonthlyAIRequests,
	}

	return usage, nil
}

// LimitUsage holds current usage information
type LimitUsage struct {
	Limits *limit_profile.Limits

	// Exchanges
	ExchangesUsed       int
	ExchangesLimit      int
	ExchangesPercentage float64

	// Positions
	ActivePositionsUsed  int
	ActivePositionsLimit int

	// Trades
	DailyTradesUsed    int
	DailyTradesLimit   int
	MonthlyTradesUsed  int
	MonthlyTradesLimit int

	// AI
	MonthlyAIRequestsUsed int
	MonthlyAIRequestLimit int
}

// calculatePercentage calculates usage percentage
func calculatePercentage(used, limit int) float64 {
	if limit == 0 || limit == -1 {
		return 0
	}
	return (float64(used) / float64(limit)) * 100
}
