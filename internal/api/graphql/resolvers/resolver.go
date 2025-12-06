package resolvers

import (
	agentService "prometheus/internal/services/agent"
	authService "prometheus/internal/services/auth"
	fundWatchlistService "prometheus/internal/services/fundwatchlist"
	strategyService "prometheus/internal/services/strategy"
	userService "prometheus/internal/services/user"
	"prometheus/pkg/logger"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

// Resolver is the root resolver following Clean Architecture principles
// Uses services layer, not repositories directly
type Resolver struct {
	// Services (not repositories!)
	AuthService          *authService.Service
	UserService          *userService.Service
	StrategyService      *strategyService.Service
	FundWatchlistService *fundWatchlistService.Service
	AgentService         *agentService.Service
	Log                  *logger.Logger
}
