package resolvers

import (
	fundWatchlistService "prometheus/internal/services/fund_watchlist"
	strategyService "prometheus/internal/services/strategy"
	userService "prometheus/internal/services/user"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

// Resolver is the root resolver following Clean Architecture principles
// Uses services layer, not repositories directly
type Resolver struct {
	// Services (not repositories!)
	UserService          *userService.Service
	StrategyService      *strategyService.Service
	FundWatchlistService *fundWatchlistService.Service
}
