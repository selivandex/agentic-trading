package position

import (
	"context"
	"fmt"
	"time"

	"strings"

	"github.com/google/uuid"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"prometheus/internal/domain/position"
	eventspb "prometheus/internal/events/proto"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// AgentEventHandler handles position events that require LLM-based decision making
// Uses PositionGuardian agent for complex analysis and decisions
type AgentEventHandler struct {
	posRepo        position.Repository
	agent          agent.Agent // PositionGuardian agent
	sessionService session.Service
	log            *logger.Logger
	timeout        time.Duration
}

// NewAgentEventHandler creates a new agent event handler
func NewAgentEventHandler(
	posRepo position.Repository,
	positionGuardianAgent agent.Agent,
	sessionService session.Service,
	log *logger.Logger,
	timeout time.Duration,
) *AgentEventHandler {
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	return &AgentEventHandler{
		posRepo:        posRepo,
		agent:          positionGuardianAgent,
		sessionService: sessionService,
		log:            log,
		timeout:        timeout,
	}
}

// HandleStopApproaching handles stop approaching event - asks agent if should exit early
func (h *AgentEventHandler) HandleStopApproaching(ctx context.Context, event *eventspb.StopApproachingEvent) error {
	h.log.Infow("Agent evaluating stop approaching event",
		"position_id", event.PositionId,
		"symbol", event.Symbol,
		"distance_pct", event.DistancePercent,
	)

	// Parse position ID
	positionID, err := uuid.Parse(event.PositionId)
	if err != nil {
		return errors.Wrap(err, "invalid position ID")
	}

	// Get position
	pos, err := h.posRepo.GetByID(ctx, positionID)
	if err != nil {
		return errors.Wrap(err, "failed to get position")
	}

	if pos == nil || pos.Status == position.PositionClosed {
		h.log.Infow("Position not found or already closed", "position_id", event.PositionId)
		return nil
	}

	// Build context for agent
	prompt := h.buildStopApproachingPrompt(event, pos)

	// Run agent with timeout
	decision, err := h.runAgent(ctx, pos.UserID.String(), prompt)
	if err != nil {
		h.log.Error("Agent evaluation failed for stop approaching",
			"position_id", event.PositionId,
			"error", err,
		)
		// Fallback: do nothing, let it hit stop if needed
		return errors.Wrap(err, "agent evaluation failed")
	}

	// Execute decision
	return h.executeDecision(ctx, pos, decision, "stop_approaching")
}

// HandleTargetApproaching handles target approaching event - asks agent if should take profit or let run
func (h *AgentEventHandler) HandleTargetApproaching(ctx context.Context, event *eventspb.TargetApproachingEvent) error {
	h.log.Infow("Agent evaluating target approaching event",
		"position_id", event.PositionId,
		"symbol", event.Symbol,
		"distance_pct", event.DistancePercent,
		"pnl_pct", event.UnrealizedPnlPercent,
	)

	// Parse position ID
	positionID, err := uuid.Parse(event.PositionId)
	if err != nil {
		return errors.Wrap(err, "invalid position ID")
	}

	// Get position
	pos, err := h.posRepo.GetByID(ctx, positionID)
	if err != nil {
		return errors.Wrap(err, "failed to get position")
	}

	if pos == nil || pos.Status == position.PositionClosed {
		h.log.Infow("Position not found or already closed", "position_id", event.PositionId)
		return nil
	}

	// Build context for agent
	prompt := h.buildTargetApproachingPrompt(event, pos)

	// Run agent with timeout
	decision, err := h.runAgent(ctx, pos.UserID.String(), prompt)
	if err != nil {
		h.log.Error("Agent evaluation failed for target approaching",
			"position_id", event.PositionId,
			"error", err,
		)
		// Fallback: hold position
		return errors.Wrap(err, "agent evaluation failed")
	}

	// Execute decision
	return h.executeDecision(ctx, pos, decision, "target_approaching")
}

// HandleProfitMilestone handles profit milestone event - asks agent if should trail stop
func (h *AgentEventHandler) HandleProfitMilestone(ctx context.Context, event *eventspb.ProfitMilestoneEvent) error {
	h.log.Infow("Agent evaluating profit milestone event",
		"position_id", event.PositionId,
		"symbol", event.Symbol,
		"milestone", event.Milestone,
		"pnl_pct", event.UnrealizedPnlPercent,
	)

	// Parse position ID
	positionID, err := uuid.Parse(event.PositionId)
	if err != nil {
		return errors.Wrap(err, "invalid position ID")
	}

	// Get position
	pos, err := h.posRepo.GetByID(ctx, positionID)
	if err != nil {
		return errors.Wrap(err, "failed to get position")
	}

	if pos == nil || pos.Status == position.PositionClosed {
		h.log.Infow("Position not found or already closed", "position_id", event.PositionId)
		return nil
	}

	// Build context for agent
	prompt := h.buildProfitMilestonePrompt(event, pos)

	// Run agent with timeout
	decision, err := h.runAgent(ctx, pos.UserID.String(), prompt)
	if err != nil {
		h.log.Error("Agent evaluation failed for profit milestone",
			"position_id", event.PositionId,
			"error", err,
		)
		// Fallback: hold position
		return errors.Wrap(err, "agent evaluation failed")
	}

	// Execute decision
	return h.executeDecision(ctx, pos, decision, "profit_milestone")
}

// HandleThesisInvalidation handles thesis invalidation event - asks agent if should exit
func (h *AgentEventHandler) HandleThesisInvalidation(ctx context.Context, event *eventspb.ThesisInvalidationEvent) error {
	h.log.Warn("Agent evaluating thesis invalidation event",
		"position_id", event.PositionId,
		"symbol", event.Symbol,
		"reason", event.InvalidationReason,
		"confidence", event.Confidence,
	)

	// Parse position ID
	positionID, err := uuid.Parse(event.PositionId)
	if err != nil {
		return errors.Wrap(err, "invalid position ID")
	}

	// Get position
	pos, err := h.posRepo.GetByID(ctx, positionID)
	if err != nil {
		return errors.Wrap(err, "failed to get position")
	}

	if pos == nil || pos.Status == position.PositionClosed {
		h.log.Infow("Position not found or already closed", "position_id", event.PositionId)
		return nil
	}

	// Build context for agent
	prompt := h.buildThesisInvalidationPrompt(event, pos)

	// Run agent with timeout
	decision, err := h.runAgent(ctx, pos.UserID.String(), prompt)
	if err != nil {
		h.log.Errorw("Agent evaluation failed for thesis invalidation",
			"position_id", event.PositionId,
			"error", err,
		)
		// Fallback: close position (thesis invalidated = serious)
		h.log.Warnw("Fallback: closing position due to thesis invalidation", "position_id", event.PositionId)
		posID, _ := uuid.Parse(event.PositionId)
		// Close with current price and PnL from position
		closeErr := h.posRepo.Close(ctx, posID, pos.CurrentPrice, pos.UnrealizedPnL)
		if closeErr != nil {
			return errors.Wrap(closeErr, "fallback close failed")
		}
		return errors.Wrap(err, "agent evaluation failed, position closed as fallback")
	}

	// Execute decision
	return h.executeDecision(ctx, pos, decision, "thesis_invalidation")
}

// HandleTimeDecay handles time decay event - asks agent if should close stale position
func (h *AgentEventHandler) HandleTimeDecay(ctx context.Context, event *eventspb.TimeDecayEvent) error {
	h.log.Infow("Agent evaluating time decay event",
		"position_id", event.PositionId,
		"symbol", event.Symbol,
		"duration_hours", event.DurationHours,
		"pnl_pct", event.CurrentPnlPercent,
	)

	// Parse position ID
	positionID, err := uuid.Parse(event.PositionId)
	if err != nil {
		return errors.Wrap(err, "invalid position ID")
	}

	// Get position
	pos, err := h.posRepo.GetByID(ctx, positionID)
	if err != nil {
		return errors.Wrap(err, "failed to get position")
	}

	if pos == nil || pos.Status == position.PositionClosed {
		h.log.Infow("Position not found or already closed", "position_id", event.PositionId)
		return nil
	}

	// Build context for agent
	prompt := h.buildTimeDecayPrompt(event, pos)

	// Run agent with timeout
	decision, err := h.runAgent(ctx, pos.UserID.String(), prompt)
	if err != nil {
		h.log.Errorw("Agent evaluation failed for time decay",
			"position_id", event.PositionId,
			"error", err,
		)
		// Fallback: hold position
		return errors.Wrap(err, "agent evaluation failed")
	}

	// Execute decision
	return h.executeDecision(ctx, pos, decision, "time_decay")
}

// runAgent runs the PositionGuardian agent with the given prompt
func (h *AgentEventHandler) runAgent(ctx context.Context, userID string, prompt string) (string, error) {
	// Create timeout context
	agentCtx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	// Create ADK runner
	runnerInstance, err := runner.New(runner.Config{
		AppName:        fmt.Sprintf("position_guardian_%s", userID),
		Agent:          h.agent,
		SessionService: h.sessionService,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to create runner")
	}

	// Build input
	input := &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{Text: prompt},
		},
	}

	// Run agent
	sessionID := uuid.New().String()
	runConfig := agent.RunConfig{
		StreamingMode: agent.StreamingModeNone,
	}

	var response string
	for event, err := range runnerInstance.Run(agentCtx, userID, sessionID, input, runConfig) {
		if err != nil {
			return "", errors.Wrap(err, "agent run failed")
		}

		if event == nil || event.LLMResponse.Partial {
			continue
		}

		// Extract response text
		if event.LLMResponse.Content != nil {
			for _, part := range event.LLMResponse.Content.Parts {
				if part.Text != "" {
					response = part.Text
				}
			}
		}

		// Check if complete
		if event.TurnComplete && event.IsFinalResponse() {
			break
		}
	}

	if response == "" {
		return "", errors.New("no response from agent")
	}

	return response, nil
}

// executeDecision executes the agent's decision
func (h *AgentEventHandler) executeDecision(ctx context.Context, pos *position.Position, decision string, eventType string) error {
	h.log.Infow("Executing agent decision",
		"position_id", pos.ID,
		"symbol", pos.Symbol,
		"decision", decision,
		"event_type", eventType,
	)

	// Parse decision (simple parsing - in production you'd want structured output)
	// Expected decisions: HOLD, EXIT, TRAIL_STOP, TRIM
	// For now, simple implementation

	switch {
	case containsAction(decision, "EXIT"):
		h.log.Infow("Agent decided to EXIT position", "position_id", pos.ID)
		// Close position with current price and PnL
		err := h.posRepo.Close(ctx, pos.ID, pos.CurrentPrice, pos.UnrealizedPnL)
		return err

	case containsAction(decision, "HOLD"):
		h.log.Infow("Agent decided to HOLD position", "position_id", pos.ID)
		return nil

	case containsAction(decision, "TRAIL_STOP"):
		h.log.Infow("Agent decided to TRAIL_STOP (not implemented yet)", "position_id", pos.ID)
		// TODO: Implement stop loss trailing
		return nil

	case containsAction(decision, "TRIM"):
		h.log.Infow("Agent decided to TRIM position (not implemented yet)", "position_id", pos.ID)
		// TODO: Implement partial position closing
		return nil

	default:
		h.log.Warnw("Unknown agent decision, defaulting to HOLD",
			"position_id", pos.ID,
			"decision", decision,
		)
		return nil
	}
}

// Helper functions to build prompts for different event types

func (h *AgentEventHandler) buildStopApproachingPrompt(event *eventspb.StopApproachingEvent, pos *position.Position) string {
	return fmt.Sprintf(`# Position Event: Stop Approaching

## Event Details
- Position ID: %s
- Symbol: %s
- Current Price: %.2f
- Stop Loss: %.2f
- Distance to Stop: %.2f%%
- Side: %s

## Your Task
Price is approaching stop loss. Should we:
1. EXIT now to avoid slippage?
2. HOLD and let stop trigger naturally?

Consider:
- Market conditions
- Volatility
- Slippage risk
- Thesis validity

Respond with: EXIT or HOLD + reasoning.`,
		event.PositionId,
		event.Symbol,
		event.CurrentPrice,
		event.StopLossPrice,
		event.DistancePercent,
		event.Side,
	)
}

func (h *AgentEventHandler) buildTargetApproachingPrompt(event *eventspb.TargetApproachingEvent, pos *position.Position) string {
	return fmt.Sprintf(`# Position Event: Target Approaching

## Event Details
- Position ID: %s
- Symbol: %s
- Current Price: %.2f
- Take Profit: %.2f
- Distance to Target: %.2f%%
- Current PnL: %.2f%%
- Side: %s

## Your Task
Price is approaching take profit. Should we:
1. EXIT now and lock profit?
2. HOLD and let target trigger?
3. TRAIL_STOP to let winners run?

Consider:
- Momentum strength
- Resistance levels
- Risk of reversal

Respond with: EXIT, HOLD, or TRAIL_STOP + reasoning.`,
		event.PositionId,
		event.Symbol,
		event.CurrentPrice,
		event.TakeProfitPrice,
		event.DistancePercent,
		event.UnrealizedPnlPercent,
		event.Side,
	)
}

func (h *AgentEventHandler) buildProfitMilestonePrompt(event *eventspb.ProfitMilestoneEvent, pos *position.Position) string {
	return fmt.Sprintf(`# Position Event: Profit Milestone

## Event Details
- Position ID: %s
- Symbol: %s
- Milestone: +%d%%
- Current PnL: %.2f%%
- Entry: %.2f
- Current: %.2f

## Your Task
Position reached +%d%% profit. Should we:
1. TRAIL_STOP to breakeven (protect capital)?
2. TRIM (take partial profit)?
3. HOLD (let it run)?

Consider:
- Milestone significance
- Momentum
- Risk management

Respond with: TRAIL_STOP, TRIM, or HOLD + reasoning.`,
		event.PositionId,
		event.Symbol,
		event.Milestone,
		event.UnrealizedPnlPercent,
		event.EntryPrice,
		event.CurrentPrice,
		event.Milestone,
	)
}

func (h *AgentEventHandler) buildThesisInvalidationPrompt(event *eventspb.ThesisInvalidationEvent, pos *position.Position) string {
	return fmt.Sprintf(`# Position Event: Thesis Invalidation

## Event Details
- Position ID: %s
- Symbol: %s
- Invalidation Reason: %s
- Confidence: %.2f
- Original Thesis: %s
- Current Price: %.2f

## Your Task
Trade thesis may be invalidated. Should we:
1. EXIT immediately (thesis broken)?
2. HOLD (thesis still valid)?

Consider:
- Invalidation confidence
- Alternative scenarios
- Risk/reward if wrong

This is SERIOUS. Be decisive.

Respond with: EXIT or HOLD + reasoning.`,
		event.PositionId,
		event.Symbol,
		event.InvalidationReason,
		event.Confidence,
		event.OriginalThesis,
		event.CurrentPrice,
	)
}

func (h *AgentEventHandler) buildTimeDecayPrompt(event *eventspb.TimeDecayEvent, pos *position.Position) string {
	return fmt.Sprintf(`# Position Event: Time Decay

## Event Details
- Position ID: %s
- Symbol: %s
- Duration: %d hours (expected: %d hours)
- Current PnL: %.2f%%
- Timeframe: %s

## Your Task
Position held longer than expected. Should we:
1. EXIT (cut stale position)?
2. HOLD (still valid)?

Consider:
- Current PnL
- Market changed?
- Opportunity cost

Respond with: EXIT or HOLD + reasoning.`,
		event.PositionId,
		event.Symbol,
		event.DurationHours,
		event.ExpectedDurationHours,
		event.CurrentPnlPercent,
		event.Timeframe,
	)
}

// containsAction checks if decision text contains an action keyword
func containsAction(text, action string) bool {
	// Simple string matching - in production use structured output parsing
	if len(text) < len(action) {
		return false
	}
	return strings.HasPrefix(text, action) || strings.HasPrefix(text, action+":")
}
