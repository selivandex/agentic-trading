package state

import (
	"time"

	"google.golang.org/adk/session"
)

// State key prefixes (from ADK)
const (
	KeyPrefixApp  = "app:"  // Application-level (shared across all users)
	KeyPrefixUser = "user:" // User-level (shared across user's sessions)
	KeyPrefixTemp = "temp:" // Temporary (not persisted)
)

// ========================================
// App-Level State (shared across all users)
// ========================================

// SetAppMaintenanceMode sets system-wide maintenance mode
func SetAppMaintenanceMode(state session.State, enabled bool) error {
	return state.Set(KeyPrefixApp+"maintenance_mode", enabled)
}

// GetAppMaintenanceMode checks if system is in maintenance mode
func GetAppMaintenanceMode(state session.ReadonlyState) (bool, error) {
	val, err := state.Get(KeyPrefixApp + "maintenance_mode")
	if err != nil {
		return false, nil // Default: not in maintenance
	}
	if enabled, ok := val.(bool); ok {
		return enabled, nil
	}
	return false, nil
}

// SetAppTradingEnabled sets global trading enable/disable flag
func SetAppTradingEnabled(state session.State, enabled bool) error {
	return state.Set(KeyPrefixApp+"trading_enabled", enabled)
}

// GetAppTradingEnabled checks if trading is globally enabled
func GetAppTradingEnabled(state session.ReadonlyState) (bool, error) {
	val, err := state.Get(KeyPrefixApp + "trading_enabled")
	if err != nil {
		return true, nil // Default: trading enabled
	}
	if enabled, ok := val.(bool); ok {
		return enabled, nil
	}
	return true, nil
}

// ========================================
// User-Level State (shared across user's sessions)
// ========================================

// SetUserRiskTolerance sets user's risk tolerance level
func SetUserRiskTolerance(state session.State, level string) error {
	return state.Set(KeyPrefixUser+"risk_tolerance", level)
}

// GetUserRiskTolerance gets user's risk tolerance
func GetUserRiskTolerance(state session.ReadonlyState) (string, error) {
	val, err := state.Get(KeyPrefixUser + "risk_tolerance")
	if err != nil {
		return "moderate", nil // Default
	}
	if level, ok := val.(string); ok {
		return level, nil
	}
	return "moderate", nil
}

// SetUserTotalPnL sets user's total profit/loss
func SetUserTotalPnL(state session.State, pnl float64) error {
	return state.Set(KeyPrefixUser+"total_pnl", pnl)
}

// GetUserTotalPnL gets user's total profit/loss
func GetUserTotalPnL(state session.ReadonlyState) (float64, error) {
	val, err := state.Get(KeyPrefixUser + "total_pnl")
	if err != nil {
		return 0.0, nil
	}
	if pnl, ok := val.(float64); ok {
		return pnl, nil
	}
	return 0.0, nil
}

// SetUserLastActivity sets timestamp of user's last activity
func SetUserLastActivity(state session.State, t time.Time) error {
	return state.Set(KeyPrefixUser+"last_activity", t)
}

// GetUserLastActivity gets timestamp of user's last activity
func GetUserLastActivity(state session.ReadonlyState) (time.Time, error) {
	val, err := state.Get(KeyPrefixUser + "last_activity")
	if err != nil {
		return time.Time{}, nil
	}
	if t, ok := val.(time.Time); ok {
		return t, nil
	}
	return time.Time{}, nil
}

// SetUserDailyCost sets user's daily accumulated cost
func SetUserDailyCost(state session.State, cost float64) error {
	return state.Set(KeyPrefixUser+"daily_cost", cost)
}

// GetUserDailyCost gets user's daily accumulated cost
func GetUserDailyCost(state session.ReadonlyState) (float64, error) {
	val, err := state.Get(KeyPrefixUser + "daily_cost")
	if err != nil {
		return 0.0, nil
	}
	if cost, ok := val.(float64); ok {
		return cost, nil
	}
	return 0.0, nil
}

// ========================================
// Session-Level State (specific to current session)
// ========================================

// SetCurrentSymbol sets the currently analyzed trading pair
func SetCurrentSymbol(state session.State, symbol string) error {
	return state.Set("current_symbol", symbol)
}

// GetCurrentSymbol gets the currently analyzed trading pair
func GetCurrentSymbol(state session.ReadonlyState) (string, error) {
	val, err := state.Get("current_symbol")
	if err != nil {
		return "", err
	}
	if symbol, ok := val.(string); ok {
		return symbol, nil
	}
	return "", session.ErrStateKeyNotExist
}

// SetCurrentMarketType sets the market type being analyzed
func SetCurrentMarketType(state session.State, marketType string) error {
	return state.Set("current_market_type", marketType)
}

// GetCurrentMarketType gets the market type being analyzed
func GetCurrentMarketType(state session.ReadonlyState) (string, error) {
	val, err := state.Get("current_market_type")
	if err != nil {
		return "spot", nil // Default
	}
	if mt, ok := val.(string); ok {
		return mt, nil
	}
	return "spot", nil
}

// SetSessionStartTime sets when the session started
func SetSessionStartTime(state session.State, t time.Time) error {
	return state.Set("session_start_time", t)
}

// GetSessionStartTime gets when the session started
func GetSessionStartTime(state session.ReadonlyState) (time.Time, error) {
	val, err := state.Get("session_start_time")
	if err != nil {
		return time.Time{}, err
	}
	if t, ok := val.(time.Time); ok {
		return t, nil
	}
	return time.Time{}, session.ErrStateKeyNotExist
}

// ========================================
// Temporary State (not persisted to database)
// ========================================

// IncrementToolCallCount increments the tool call counter
func IncrementToolCallCount(state session.State) error {
	count := 0
	if val, err := state.Get(KeyPrefixTemp + "tool_call_count"); err == nil {
		if c, ok := val.(int); ok {
			count = c
		}
	}
	return state.Set(KeyPrefixTemp+"tool_call_count", count+1)
}

// GetToolCallCount gets the tool call counter
func GetToolCallCount(state session.ReadonlyState) int {
	val, err := state.Get(KeyPrefixTemp + "tool_call_count")
	if err != nil {
		return 0
	}
	if count, ok := val.(int); ok {
		return count
	}
	return 0
}

// SetTempStartTime sets temporary execution start time
func SetTempStartTime(state session.State, t time.Time) error {
	return state.Set(KeyPrefixTemp+"start_time", t)
}

// GetTempStartTime gets temporary execution start time
func GetTempStartTime(state session.ReadonlyState) (time.Time, error) {
	val, err := state.Get(KeyPrefixTemp + "start_time")
	if err != nil {
		return time.Time{}, err
	}
	if t, ok := val.(time.Time); ok {
		return t, nil
	}
	return time.Time{}, session.ErrStateKeyNotExist
}

// SetTempPromptTokens stores prompt tokens in temporary state
func SetTempPromptTokens(state session.State, tokens int) error {
	return state.Set(KeyPrefixTemp+"prompt_tokens", tokens)
}

// SetTempCompletionTokens stores completion tokens in temporary state
func SetTempCompletionTokens(state session.State, tokens int) error {
	return state.Set(KeyPrefixTemp+"completion_tokens", tokens)
}

// GetTempTokens retrieves temporary token counts
func GetTempTokens(state session.ReadonlyState) (promptTokens, completionTokens int) {
	if val, err := state.Get(KeyPrefixTemp + "prompt_tokens"); err == nil {
		if tokens, ok := val.(int); ok {
			promptTokens = tokens
		}
	}
	if val, err := state.Get(KeyPrefixTemp + "completion_tokens"); err == nil {
		if tokens, ok := val.(int); ok {
			completionTokens = tokens
		}
	}
	return promptTokens, completionTokens
}
