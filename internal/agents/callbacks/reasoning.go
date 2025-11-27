package callbacks

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"google.golang.org/adk/agent"
	"google.golang.org/genai"

	"prometheus/internal/domain/reasoning"
	"prometheus/pkg/logger"
)

// SaveStructuredReasoningCallback creates a callback that parses structured agent output
// and saves reasoning trace to the database for audit and debugging.
func SaveStructuredReasoningCallback(reasoningRepo reasoning.Repository) agent.AfterAgentCallback {
	return func(ctx agent.CallbackContext) (*genai.Content, error) {
		log := logger.Get().With(
			"component", "reasoning_callback",
			"agent", ctx.AgentName(),
			"session", ctx.SessionID(),
		)

		// Get agent output from conversation history
		history := ctx.ConversationHistory()
		if len(history) == 0 {
			log.Debug("No conversation history available")
			return nil, nil
		}

		// Get the last message (agent's response)
		lastMessage := history[len(history)-1]
		if lastMessage.Role != genai.RoleModel {
			log.Debug("Last message is not from model, skipping")
			return nil, nil
		}

		// Extract text content from message
		var outputText string
		for _, part := range lastMessage.Parts {
			if textPart, ok := part.(*genai.TextPart); ok {
				outputText += textPart.Text
			}
		}

		if outputText == "" {
			log.Debug("No text output from agent")
			return nil, nil
		}

		// Try to parse as structured JSON
		var structuredOutput map[string]interface{}
		if err := json.Unmarshal([]byte(outputText), &structuredOutput); err != nil {
			// Not JSON or invalid JSON - save raw output as fallback
			log.Debugf("Agent output is not structured JSON (expected for agents without OutputSchema): %v", err)
			return saveRawOutput(ctx, reasoningRepo, outputText)
		}

	// Extract reasoning steps and final output
	// Supports multiple output formats:
	// - OpportunitySynthesizer: synthesis_steps + decision + conflicts
	// - Analyst agents: reasoning_trace + final_analysis + tool_calls_summary
	// - Legacy: reasoning_steps (fallback)
	reasoningSteps, hasSteps := structuredOutput["synthesis_steps"]
	if !hasSteps {
		reasoningSteps, hasSteps = structuredOutput["reasoning_trace"]
	}
	if !hasSteps {
		reasoningSteps, hasSteps = structuredOutput["reasoning_steps"]
	}

	var finalOutput interface{}
	if decision, ok := structuredOutput["decision"]; ok {
		finalOutput = decision
	} else if analysis, ok := structuredOutput["final_analysis"]; ok {
		finalOutput = analysis
	} else {
		// No specific final output field, use entire output
		finalOutput = structuredOutput
	}

	// Extract additional metadata (tool calls, conflicts, etc.)
	metadata := make(map[string]interface{})
	if toolCalls, ok := structuredOutput["tool_calls_summary"]; ok {
		metadata["tool_calls_summary"] = toolCalls
	}
	if conflicts, ok := structuredOutput["conflicts"]; ok {
		metadata["conflicts"] = conflicts
	}

		// Marshal to JSONB for storage
		reasoningStepsJSON, err := json.Marshal(reasoningSteps)
		if err != nil {
			log.Warnf("Failed to marshal reasoning steps: %v", err)
			return saveRawOutput(ctx, reasoningRepo, outputText)
		}

		finalOutputJSON, err := json.Marshal(finalOutput)
		if err != nil {
			log.Warnf("Failed to marshal final output: %v", err)
			return saveRawOutput(ctx, reasoningRepo, outputText)
		}

		// Get agent type from temp state (set by ValidationBeforeCallback)
		agentTypeVal, _ := ctx.ReadonlyState().Get("_agent_type")
		agentType, _ := agentTypeVal.(string)
		if agentType == "" {
			agentType = ctx.AgentName() // Fallback to agent name
		}

		// Parse session ID as UUID (ADK session IDs are UUIDs)
		var sessionUUID uuid.UUID
		if ctx.SessionID() != "" {
			parsed, err := uuid.Parse(ctx.SessionID())
			if err != nil {
				log.Warnf("Failed to parse session ID as UUID: %v", err)
				sessionUUID = uuid.New() // Generate new UUID as fallback
			} else {
				sessionUUID = parsed
			}
		} else {
			sessionUUID = uuid.New()
		}

		// Parse user ID if available
		var userUUID *uuid.UUID
		if ctx.UserID() != "" {
			parsed, err := uuid.Parse(ctx.UserID())
			if err == nil {
				userUUID = &parsed
			}
		}

		// Create reasoning log entry
		entry := &reasoning.LogEntry{
			ID:             uuid.New(),
			UserID:         userUUID,
			AgentID:        ctx.AgentName(),
			AgentType:      agentType,
			SessionID:      sessionUUID.String(),
			ReasoningSteps: reasoningStepsJSON,
			Decision:       finalOutputJSON,
			CreatedAt:      time.Now(),
		}

		// Get metadata from temp state (tokens, cost, duration)
		if startTimeVal, err := ctx.ReadonlyState().Get("_temp_start_time"); err == nil {
			if startTime, ok := startTimeVal.(time.Time); ok {
				duration := time.Since(startTime)
				entry.DurationMs = int(duration.Milliseconds())
			}
		}

		// Save to database (async-safe - don't block agent execution)
		go func() {
			saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := reasoningRepo.Create(saveCtx, entry); err != nil {
				log.Errorf("Failed to save reasoning log: %v", err)
			} else {
				log.Infof("Reasoning log saved: session=%s agent=%s", entry.SessionID, entry.AgentType)
			}
		}()

		return nil, nil // Non-blocking callback
	}
}

// saveRawOutput saves raw text output as fallback when structured parsing fails
func saveRawOutput(ctx agent.CallbackContext, repo reasoning.Repository, rawOutput string) (*genai.Content, error) {
	log := logger.Get().With("component", "reasoning_callback")

	// Wrap raw output in minimal structure
	rawSteps := []map[string]interface{}{
		{
			"step":      1,
			"action":    "raw_output",
			"content":   rawOutput,
			"timestamp": time.Now().Format(time.RFC3339),
			"note":      "Agent output was not structured JSON - saved as raw text",
		},
	}

	reasoningStepsJSON, _ := json.Marshal(rawSteps)
	finalOutputJSON := []byte(rawOutput)

	// Get agent type
	agentTypeVal, _ := ctx.ReadonlyState().Get("_agent_type")
	agentType, _ := agentTypeVal.(string)
	if agentType == "" {
		agentType = ctx.AgentName()
	}

	// Parse session ID
	var sessionUUID uuid.UUID
	if ctx.SessionID() != "" {
		parsed, err := uuid.Parse(ctx.SessionID())
		if err != nil {
			sessionUUID = uuid.New()
		} else {
			sessionUUID = parsed
		}
	} else {
		sessionUUID = uuid.New()
	}

	// Parse user ID if available
	var userUUID *uuid.UUID
	if ctx.UserID() != "" {
		parsed, err := uuid.Parse(ctx.UserID())
		if err == nil {
			userUUID = &parsed
		}
	}

	entry := &reasoning.LogEntry{
		ID:             uuid.New(),
		UserID:         userUUID,
		AgentID:        ctx.AgentName(),
		AgentType:      agentType,
		SessionID:      sessionUUID.String(),
		ReasoningSteps: reasoningStepsJSON,
		Decision:       finalOutputJSON,
		CreatedAt:      time.Now(),
	}

	// Save async
	go func() {
		saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := repo.Create(saveCtx, entry); err != nil {
			log.Errorf("Failed to save raw reasoning log: %v", err)
		} else {
			log.Debugf("Raw reasoning log saved: session=%s", entry.SessionID)
		}
	}()

	return nil, nil
}
