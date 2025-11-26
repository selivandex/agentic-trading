package callbacks

import (
	"context"
	"time"

	"github.com/google/uuid"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/tool"

	"prometheus/internal/agents/state"
	"prometheus/internal/domain/stats"
	"prometheus/internal/tools"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// RiskValidationBeforeToolCallback validates dangerous operations before execution.
//
// Responsibility Separation:
//   - This callback handles CROSS-CUTTING CONCERNS: risk levels, permissions, rate limiting, circuit breakers
//   - Each tool implementation handles its OWN PARAMETER VALIDATION (required fields, types, business rules)
//   - This design avoids hardcoding tool-specific logic in generic callbacks
//
// Risk levels are defined in tools.Catalog and checked via Registry metadata.
func RiskValidationBeforeToolCallback(registry *tools.Registry, riskEngine interface{}) llmagent.BeforeToolCallback {
	return func(ctx tool.Context, t tool.Tool, args map[string]any) (map[string]any, error) {
		toolName := t.Name()
		log := logger.Get().With("component", "tool_risk_validator", "tool", toolName)

		// Get tool metadata and risk level
		meta, ok := registry.GetMetadata(toolName)
		if !ok {
			log.Warnf("Tool %s not found in registry metadata", toolName)
			return nil, nil // Allow unknown tools to proceed (backwards compatibility)
		}

		// Log risk level for all non-trivial operations
		if meta.RiskLevel != tools.RiskLevelNone {
			log.With("risk_level", meta.RiskLevel).
				Debugf("Validating tool with risk level: %s", meta.RiskLevel)
		}

		// Validate based on risk level
		// Note: Parameter validation is the responsibility of each tool implementation.
		// Callbacks handle cross-cutting concerns: risk checks, permissions, rate limiting.
		switch meta.RiskLevel {
		case tools.RiskLevelCritical:
			log.Warnf("CRITICAL RISK operation: %s - validating permissions", toolName)
			// TODO: In production:
			// - Check user permissions and role
			// - Require explicit confirmation (2FA, confirmation prompt)
			// - Validate user state (e.g., emergency_close_all: verify positions exist)
			// - Log to security audit trail

		case tools.RiskLevelHigh:
			log.Warnf("HIGH RISK operation: %s - applying strict validation", toolName)
			// TODO: In production:
			// - Additional authorization checks
			// - Stricter rate limiting
			// - Enhanced audit logging with full context

		case tools.RiskLevelMedium:
			log.Debugf("MEDIUM RISK operation: %s - standard validation", toolName)
			// TODO: In production:
			// - Standard rate limiting
			// - Risk engine checks (position limits, exposure, etc.)
			// - Circuit breaker validation
		}

		return nil, nil // Allow operation to proceed
	}
}

// AuditLogAfterToolCallback logs all tool executions
func AuditLogAfterToolCallback() llmagent.AfterToolCallback {
	return func(ctx tool.Context, t tool.Tool, args, result map[string]any, err error) (map[string]any, error) {
		toolName := t.Name()
		log := logger.Get().With(
			"component", "tool_audit",
			"tool", toolName,
			"user", ctx.UserID(),
		)

		if err != nil {
			log.Errorf("Tool %s failed: %v", toolName, err)
		} else {
			log.Infof("Tool %s executed successfully", toolName)
		}

		// Increment tool call counter using state helper
		state.IncrementToolCallCount(ctx.State())

		return result, err // Pass through unchanged
	}
}

// StatsTrackingAfterToolCallback records detailed tool usage metrics to ClickHouse
func StatsTrackingAfterToolCallback(statsRepo stats.Repository) llmagent.AfterToolCallback {
	return func(ctx tool.Context, t tool.Tool, args, result map[string]any, err error) (map[string]any, error) {
		if statsRepo == nil {
			return result, err // Pass through if no stats repo
		}

		toolName := t.Name()

		// Get start time from temporary state
		startTimeVal, stateErr := ctx.ReadonlyState().Get("_temp_tool_start_time")
		if stateErr != nil {
			// No start time recorded, skip stats
			return result, err
		}

		startTime, ok := startTimeVal.(time.Time)
		if !ok {
			return result, err
		}

		duration := time.Since(startTime)

		// Extract metadata from context/state
		userIDStr := ctx.UserID()
		sessionID := ctx.SessionID()

		// Parse user ID
		userID, parseErr := uuid.Parse(userIDStr)
		if parseErr != nil {
			logger.Get().With("tool", toolName).
				Warnf("Failed to parse user_id: %v", parseErr)
			return result, err // Skip stats on parse error
		}

		// Try to get agent_id and symbol from state
		agentID, _ := ctx.ReadonlyState().Get("_meta_agent_id")
		symbol, _ := ctx.ReadonlyState().Get("_meta_symbol")

		agentIDStr, _ := agentID.(string)
		symbolStr, _ := symbol.(string)

		// Create stats event
		usage := &stats.ToolUsageEvent{
			UserID:     userID,
			AgentID:    agentIDStr,
			ToolName:   toolName,
			Timestamp:  time.Now(),
			DurationMs: int(duration.Milliseconds()),
			Success:    err == nil,
			SessionID:  sessionID,
			Symbol:     symbolStr,
		}

		// Record asynchronously to avoid blocking tool execution
		go func() {
			if insertErr := statsRepo.InsertToolUsage(context.Background(), usage); insertErr != nil {
				logger.Get().With("tool", toolName).
					Warnf("Failed to record tool stats: %v", insertErr)
			}
		}()

		return result, err
	}
}

// RecordToolStartTimeBeforeToolCallback records execution start time for stats tracking
func RecordToolStartTimeBeforeToolCallback() llmagent.BeforeToolCallback {
	return func(ctx tool.Context, t tool.Tool, args map[string]any) (map[string]any, error) {
		// Store start time in temporary state
		ctx.State().Set("_temp_tool_start_time", time.Now())
		return nil, nil
	}
}

// ErrorHandlingCallback provides graceful error recovery for tool failures
func ErrorHandlingCallback() llmagent.AfterToolCallback {
	return func(ctx tool.Context, t tool.Tool, args, result map[string]any, err error) (map[string]any, error) {
		if err == nil {
			return result, err
		}

		toolName := t.Name()
		log := logger.Get().With("component", "tool_error_handler", "tool", toolName)

		// Log the error with context
		log.Errorf("Tool %s failed with args %v: %v", toolName, args, err)

		// Wrap error for better context
		wrappedErr := errors.Wrapf(err, "tool %s execution failed", toolName)
		return result, wrappedErr
	}
}
