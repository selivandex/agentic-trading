package strategy

import (
	"context"
	"fmt"

	"prometheus/internal/domain/limit_profile"
	"prometheus/internal/domain/strategy"
	"prometheus/internal/domain/user"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// InvestmentValidator validates investment operations against user limits and profile
type InvestmentValidator struct {
	limitProfileRepo limit_profile.Repository
	strategyRepo     strategy.Repository
	log              *logger.Logger
}

// NewInvestmentValidator creates a new investment validator
func NewInvestmentValidator(
	limitProfileRepo limit_profile.Repository,
	strategyRepo strategy.Repository,
	log *logger.Logger,
) *InvestmentValidator {
	return &InvestmentValidator{
		limitProfileRepo: limitProfileRepo,
		strategyRepo:     strategyRepo,
		log:              log.With("component", "investment_validator"),
	}
}

// ValidationResult contains the result of investment validation
type ValidationResult struct {
	Allowed         bool
	Reason          string
	MaxAllowed      float64
	CurrentExposure float64
}

// ValidateInvestment checks if user can invest the requested amount
// Validates against:
// 1. limit_profile limits (tier-based, hard limits)
// 2. user settings limits (personal risk profile, soft limits)
// 3. current portfolio exposure from strategies
func (v *InvestmentValidator) ValidateInvestment(ctx context.Context, usr *user.User, requestedCapital float64) (*ValidationResult, error) {
	v.log.Debugw("Validating investment",
		"user_id", usr.ID,
		"requested_capital", requestedCapital,
	)

	// Will check minimum after getting limit profile (tier-specific minimum)

	// Get current allocated capital from active strategies
	allocatedCapital, err := v.strategyRepo.GetTotalAllocatedCapital(ctx, usr.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get total allocated capital")
	}

	currentExposure, _ := allocatedCapital.Float64()
	totalWithNew := currentExposure + requestedCapital

	// Get active strategies count
	activeStrategies, err := v.strategyRepo.GetActiveByUserID(ctx, usr.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get active strategies")
	}

	v.log.Debugw("Current portfolio exposure",
		"current_exposure", currentExposure,
		"total_with_new", totalWithNew,
		"active_strategies_count", len(activeStrategies),
	)

	// PRIORITY 1: Check limit_profile limits (tier-based, hard limits)
	if usr.LimitProfileID != nil {
		profile, err := v.limitProfileRepo.GetByID(ctx, *usr.LimitProfileID)
		if err != nil {
			v.log.Warnw("Failed to get limit profile, falling back to user settings",
				"limit_profile_id", usr.LimitProfileID,
				"error", err,
			)
		} else {
			limits, err := profile.ParseLimits()
			if err != nil {
				v.log.Errorw("Failed to parse limit profile", "error", err)
			} else {
				// Check minimum investment amount (tier-based)
				if limits.MinInvestmentAmountUSD > 0 && requestedCapital < limits.MinInvestmentAmountUSD {
					return &ValidationResult{
						Allowed:         false,
						Reason:          fmt.Sprintf("Minimum investment for %s tier is $%.2f. You're trying to invest $%.2f.", profile.Name, limits.MinInvestmentAmountUSD, requestedCapital),
						MaxAllowed:      0,
						CurrentExposure: currentExposure,
					}, nil
				}

				// Check total exposure limit (HARD LIMIT - tier-based)
				if limits.MaxTotalExposureUSD > 0 && totalWithNew > limits.MaxTotalExposureUSD {
					return &ValidationResult{
						Allowed:         false,
						Reason:          fmt.Sprintf("Investment exceeds your plan limit. Your %s tier allows max $%.2f total exposure, you already have $%.2f invested. Upgrade your plan to invest more.", profile.Name, limits.MaxTotalExposureUSD, currentExposure),
						MaxAllowed:      limits.MaxTotalExposureUSD - currentExposure,
						CurrentExposure: currentExposure,
					}, nil
				}

				// Check if live trading is allowed
				if !limits.LiveTradingAllowed {
					return &ValidationResult{
						Allowed:         false,
						Reason:          fmt.Sprintf("Live trading is not available on %s tier. Upgrade to start live trading.", profile.Name),
						MaxAllowed:      0,
						CurrentExposure: currentExposure,
					}, nil
				}

				v.log.Debugw("Limit profile validation passed",
					"profile", profile.Name,
					"max_total_exposure", limits.MaxTotalExposureUSD,
					"total_with_new", totalWithNew,
				)
			}
		}
	}

	// PRIORITY 2: Check user settings limits (personal risk profile, soft limits)
	userLimits := usr.GetRiskTolerance()

	// Check user's personal minimum (if set and higher than tier minimum)
	if usr.Settings.MinPositionSizeUSD > 0 && requestedCapital < usr.Settings.MinPositionSizeUSD {
		return &ValidationResult{
			Allowed:         false,
			Reason:          fmt.Sprintf("Investment amount must be at least $%.2f based on your personal risk settings. Adjust settings in /settings to lower this minimum.", usr.Settings.MinPositionSizeUSD),
			MaxAllowed:      0,
			CurrentExposure: currentExposure,
		}, nil
	}

	// Fallback minimum if no limits set (safety check)
	if requestedCapital < 1.0 {
		return &ValidationResult{
			Allowed:         false,
			Reason:          "Investment amount must be at least $1",
			MaxAllowed:      0,
			CurrentExposure: 0,
		}, nil
	}

	// Check user's personal total exposure limit
	if userLimits.MaxTotalExposureUSD > 0 && totalWithNew > userLimits.MaxTotalExposureUSD {
		return &ValidationResult{
			Allowed:         false,
			Reason:          fmt.Sprintf("Investment exceeds your personal risk limit. You set max total exposure to $%.2f, current exposure is $%.2f. Adjust your risk settings in /settings to invest more.", userLimits.MaxTotalExposureUSD, currentExposure),
			MaxAllowed:      userLimits.MaxTotalExposureUSD - currentExposure,
			CurrentExposure: currentExposure,
		}, nil
	}

	// Note: MaxPositions setting applies to positions within a strategy, not number of strategies
	// Strategies manage their own position limits based on risk profile

	v.log.Infow("Investment validation passed",
		"user_id", usr.ID,
		"requested_capital", requestedCapital,
		"current_exposure", currentExposure,
		"total_with_new", totalWithNew,
	)

	// All checks passed
	return &ValidationResult{
		Allowed:         true,
		Reason:          "ok",
		MaxAllowed:      0, // Not needed when allowed
		CurrentExposure: currentExposure,
	}, nil
}
