package callbacks

import (
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"

	"prometheus/pkg/logger"
)

// ReasoningLogCallback logs agent's chain-of-thought reasoning to logger
// This is a lightweight version that doesn't save to DB
func ReasoningLogCallback() llmagent.AfterModelCallback {
	return func(ctx agent.CallbackContext, resp *model.LLMResponse, respErr error) (*model.LLMResponse, error) {
		if respErr != nil || resp == nil {
			return resp, respErr
		}

		log := logger.Get().With("component", "agent_reasoning")

		// Get agent name from context
		agentName := ctx.AgentName()

		// Log token usage (tracks CoT via token counts)
		if resp.UsageMetadata != nil {
			log.Debugw("Agent tokens",
				"agent", agentName,
				"input_tokens", resp.UsageMetadata.PromptTokenCount,
				"output_tokens", resp.UsageMetadata.CandidatesTokenCount,
				"total_tokens", resp.UsageMetadata.TotalTokenCount,
			)
		}

		// Log response content parts (includes thinking and function calls)
		if resp.Content != nil && len(resp.Content.Parts) > 0 {
			for _, part := range resp.Content.Parts {
				// Log function calls
				if part.FunctionCall != nil {
					log.Infow("Agent invoking tool",
						"agent", agentName,
						"tool", part.FunctionCall.Name,
					)
				}

				// Log reasoning text (CoT) - truncate if too long
				if part.Text != "" {
					textPreview := part.Text
					if len(textPreview) > 1000 {
						textPreview = textPreview[:1000] + "..."
					}
					log.Debugw("Agent thinking (CoT)",
						"agent", agentName,
						"text", textPreview,
					)
				}
			}
		}

		return resp, nil
	}
}
