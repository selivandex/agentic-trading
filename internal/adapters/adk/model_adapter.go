package adk

import (
	"context"
	"encoding/json"
	"iter"

	"google.golang.org/adk/model"
	"google.golang.org/genai"

	"prometheus/internal/adapters/ai"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// ModelAdapter adapts our AI ChatProvider to ADK's model.LLM interface.
type ModelAdapter struct {
	provider  ai.ChatProvider
	modelName string
	log       *logger.Logger
}

// NewModelAdapter creates a new ADK model adapter.
func NewModelAdapter(provider ai.ChatProvider, modelName string) *ModelAdapter {
	return &ModelAdapter{
		provider:  provider,
		modelName: modelName,
		log:       logger.Get().With("component", "model_adapter", "model", modelName),
	}
}

// Name returns the model name.
func (m *ModelAdapter) Name() string {
	return m.modelName
}

// GenerateContent implements the ADK model.LLM interface.
func (m *ModelAdapter) GenerateContent(
	ctx context.Context,
	req *model.LLMRequest,
	stream bool,
) iter.Seq2[*model.LLMResponse, error] {
	if stream {
		// Streaming not implemented yet
		return func(yield func(*model.LLMResponse, error) bool) {
			yield(nil, errors.Wrap(errors.ErrNotImplemented, "streaming not implemented"))
		}
	}

	// Non-streaming implementation
	return func(yield func(*model.LLMResponse, error) bool) {
		// Convert ADK request to our ChatRequest
		chatReq := m.convertToChatRequest(req)

		m.log.Debug("Calling LLM", "messages", len(chatReq.Messages), "tools", len(chatReq.Tools))

		// Call our provider
		resp, err := m.provider.Chat(ctx, chatReq)
		if err != nil {
			m.log.Error("LLM call failed", "error", err)
			yield(nil, errors.Wrap(err, "chat provider failed"))
			return
		}

		m.log.Debug("LLM response received",
			"choices", len(resp.Choices),
			"tokens", resp.Usage.TotalTokens,
		)

		// Convert response back to ADK format
		adkResp := m.convertToADKResponse(resp)
		yield(adkResp, nil)
	}
}

// convertToChatRequest converts ADK request to our format.
func (m *ModelAdapter) convertToChatRequest(req *model.LLMRequest) ai.ChatRequest {
	chatReq := ai.ChatRequest{
		Model:       m.modelName,
		MaxTokens:   4096,
		Temperature: 0.7,
	}

	// Convert messages from genai.Content
	for _, content := range req.Contents {
		chatMsg := ai.Message{}

		// Determine role
		switch content.Role {
		case "user":
			chatMsg.Role = ai.RoleUser
		case "model":
			chatMsg.Role = ai.RoleAssistant
		case "system":
			chatMsg.Role = ai.RoleSystem
		case "function", "tool":
			chatMsg.Role = ai.RoleTool
		default:
			chatMsg.Role = ai.RoleUser
		}

		// Extract text content from Parts
		for _, part := range content.Parts {
			if part.Text != "" {
				if chatMsg.Content != "" {
					chatMsg.Content += "\n"
				}
				chatMsg.Content += part.Text
			}
			// Handle function responses if needed
			if part.FunctionResponse != nil {
				chatMsg.Role = ai.RoleTool
				chatMsg.ToolCallID = part.FunctionResponse.Name
				if respData, err := json.Marshal(part.FunctionResponse.Response); err == nil {
					chatMsg.Content = string(respData)
				}
			}
		}

		chatReq.Messages = append(chatReq.Messages, chatMsg)
	}

	// Convert tools from map to our format
	if req.Tools != nil {
		for toolName, toolData := range req.Tools {
			// toolData is any - try to extract description
			desc := ""
			if m, ok := toolData.(map[string]interface{}); ok {
				if d, ok := m["description"].(string); ok {
					desc = d
				}
			}

			chatReq.Tools = append(chatReq.Tools, ai.ToolDefinition{
				Type: "function",
				Function: ai.FunctionDefinition{
					Name:        toolName,
					Description: desc,
					Parameters: map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{},
						"required":   []string{},
					},
				},
			})
		}
	}

	return chatReq
}

// convertToADKResponse converts our response to ADK format.
func (m *ModelAdapter) convertToADKResponse(resp *ai.ChatResponse) *model.LLMResponse {
	adkResp := &model.LLMResponse{}

	if len(resp.Choices) == 0 {
		adkResp.FinishReason = genai.FinishReasonOther
		adkResp.ErrorMessage = "no choices in response"
		return adkResp
	}

	// Use first choice
	choice := resp.Choices[0]

	// Create content
	content := &genai.Content{
		Role:  "model",
		Parts: []*genai.Part{},
	}

	// Add text content
	if choice.Message.Content != "" {
		content.Parts = append(content.Parts, &genai.Part{
			Text: choice.Message.Content,
		})
	}

	// Add tool calls as function calls
	for _, tc := range choice.Message.ToolCalls {
		// Parse arguments as map
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			m.log.Warn("Failed to parse tool call arguments", "error", err)
			continue
		}

		content.Parts = append(content.Parts, &genai.Part{
			FunctionCall: &genai.FunctionCall{
				Name: tc.Function.Name,
				Args: args,
			},
		})
	}

	adkResp.Content = content

	// Set finish reason
	switch choice.FinishReason {
	case ai.FinishReasonStop:
		adkResp.FinishReason = genai.FinishReasonStop
	case ai.FinishReasonLength:
		adkResp.FinishReason = genai.FinishReasonMaxTokens
	case ai.FinishReasonToolCalls:
		adkResp.FinishReason = genai.FinishReasonStop
	default:
		adkResp.FinishReason = genai.FinishReasonStop
	}

	// Set usage metadata
	adkResp.UsageMetadata = &genai.GenerateContentResponseUsageMetadata{
		PromptTokenCount:     int32(resp.Usage.PromptTokens),
		CandidatesTokenCount: int32(resp.Usage.CompletionTokens),
		TotalTokenCount:      int32(resp.Usage.TotalTokens),
	}

	adkResp.TurnComplete = true

	return adkResp
}

// Ensure ModelAdapter implements model.LLM
var _ model.LLM = (*ModelAdapter)(nil)
