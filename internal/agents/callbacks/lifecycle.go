package callbacks

import (
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/genai"

	"prometheus/internal/agents/state"
	"prometheus/pkg/logger"
)

// CostTracker interface to avoid import cycle with agents package
type CostTracker interface {
	ExceededDailyLimit(userID string) bool
}

// CostTrackingBeforeCallback tracks execution start time and validates budget
func CostTrackingBeforeCallback(tracker CostTracker) agent.BeforeAgentCallback {
	return func(ctx agent.CallbackContext) (*genai.Content, error) {
		// Store start time in temporary state using helper
		state.SetTempStartTime(ctx.State(), time.Now())

		// Check if user has budget
		userID := ctx.UserID()
		if tracker != nil && tracker.ExceededDailyLimit(userID) {
			logger.Get().Warnf("User %s exceeded daily cost limit", userID)
			return genai.NewContentFromText(
				"Daily cost limit exceeded. Please try again tomorrow or contact support.",
				genai.RoleModel,
			), nil
		}

		// Update user last activity
		state.SetUserLastActivity(ctx.State(), time.Now())

		return nil, nil
	}
}

// ValidationBeforeCallback validates user permissions and system state
func ValidationBeforeCallback() agent.BeforeAgentCallback {
	return func(ctx agent.CallbackContext) (*genai.Content, error) {
		log := logger.Get().With(
			"agent", ctx.AgentName(),
			"user", ctx.UserID(),
			"session", ctx.SessionID(),
		)

		log.Infof("Agent %s started for user %s", ctx.AgentName(), ctx.UserID())

		// Check app-level maintenance mode using helper
		if inMaintenance, _ := state.GetAppMaintenanceMode(ctx.ReadonlyState()); inMaintenance {
			log.Warn("System in maintenance mode")
			return genai.NewContentFromText(
				"System is currently in maintenance mode. Please try again later.",
				genai.RoleModel,
			), nil
		}

		// Check if trading is globally enabled
		if tradingEnabled, _ := state.GetAppTradingEnabled(ctx.ReadonlyState()); !tradingEnabled {
			log.Warn("Trading is globally disabled")
			return genai.NewContentFromText(
				"Trading is currently disabled. Analysis mode only.",
				genai.RoleModel,
			), nil
		}

		return nil, nil
	}
}

// MetricsAfterCallback records execution metrics
func MetricsAfterCallback(statsRepo interface{}) agent.AfterAgentCallback {
	return func(ctx agent.CallbackContext) (*genai.Content, error) {
		log := logger.Get().With("agent", ctx.AgentName())

		// Get start time from temporary state
		startTimeVal, err := ctx.ReadonlyState().Get("_temp_start_time")
		if err != nil {
			log.Warnf("Start time not found in state: %v", err)
			return nil, nil
		}

		startTime, ok := startTimeVal.(time.Time)
		if !ok {
			log.Warn("Invalid start time type in state")
			return nil, nil
		}

		duration := time.Since(startTime)

		// Record metrics
		if statsRepo != nil {
			// TODO: Record to stats repository when interface is defined
			log.Debugf("Agent execution took %v", duration)
		}

		log.Infof("Agent %s completed in %v", ctx.AgentName(), duration)
		return nil, nil
	}
}

// ReasoningLogAfterCallback saves reasoning trace
func ReasoningLogAfterCallback() agent.AfterAgentCallback {
	return func(ctx agent.CallbackContext) (*genai.Content, error) {
		log := logger.Get().With("agent", ctx.AgentName())

		// TODO: Save reasoning trace to repository
		// This will be implemented when we have access to the full conversation history

		log.Debugf("Reasoning trace saved for session %s", ctx.SessionID())
		return nil, nil
	}
}
