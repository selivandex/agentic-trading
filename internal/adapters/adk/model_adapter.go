package adk

import (
	"context"
	"encoding/json"
	"iter"
	"strings"

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
		return m.streamingGenerateContent(ctx, req)
	}
	return m.nonStreamingGenerateContent(ctx, req)
}

// nonStreamingGenerateContent handles non-streaming LLM calls
func (m *ModelAdapter) nonStreamingGenerateContent(
	ctx context.Context,
	req *model.LLMRequest,
) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		// Convert ADK request to our ChatRequest
		chatReq := m.convertToChatRequest(req)

		m.log.Debug("Calling LLM (non-streaming)", "messages", len(chatReq.Messages), "tools", len(chatReq.Tools))

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

// streamingGenerateContent handles streaming LLM calls
func (m *ModelAdapter) streamingGenerateContent(
	ctx context.Context,
	req *model.LLMRequest,
) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		chatReq := m.convertToChatRequest(req)

		m.log.Debug("Calling LLM (streaming)", "messages", len(chatReq.Messages), "tools", len(chatReq.Tools))

		// Get streaming response from provider
		chunkCh, errCh := m.provider.ChatStream(ctx, chatReq)

		aggregator := NewStreamAggregator()

		for {
			select {
			case chunk, ok := <-chunkCh:
				if !ok {
					// Stream completed - send final response with full token counts
					finalResp := aggregator.GetFinal()
					finalResp.TurnComplete = true
					m.log.Debug("Stream completed", "tokens", finalResp.UsageMetadata.TotalTokenCount)
					yield(finalResp, nil)
					return
				}

				// Convert chunk to ADK response
				partialResp := m.convertStreamChunkToADKResponse(&chunk)
				partialResp.Partial = true
				aggregator.AddChunk(partialResp)

				if !yield(partialResp, nil) {
					m.log.Debug("Stream cancelled by client")
					return
				}

			case err := <-errCh:
				m.log.Error("Stream error", "error", err)
				yield(nil, errors.Wrap(err, "streaming failed"))
				return

			case <-ctx.Done():
				m.log.Warn("Stream cancelled by context")
				yield(nil, ctx.Err())
				return
			}
		}
	}
}

// convertToChatRequest converts ADK request to our format.
func (m *ModelAdapter) convertToChatRequest(req *model.LLMRequest) ai.ChatRequest {
	chatReq := ai.ChatRequest{
		Model:       m.modelName,
		Messages:    []ai.Message{},        // Initialize as empty slice (never nil)
		Tools:       []ai.ToolDefinition{}, // Initialize as empty slice (never nil)
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

// StreamAggregator aggregates streaming chunks into a complete response
type StreamAggregator struct {
	textParts      []string
	promptTokens   int32
	responseTokens int32
	finishReason   genai.FinishReason
}

// NewStreamAggregator creates a new stream aggregator
func NewStreamAggregator() *StreamAggregator {
	return &StreamAggregator{
		textParts:    make([]string, 0, 10),
		finishReason: genai.FinishReasonStop,
	}
}

// AddChunk adds a streaming chunk to the aggregator
func (a *StreamAggregator) AddChunk(resp *model.LLMResponse) {
	if resp.Content != nil {
		for _, part := range resp.Content.Parts {
			if part.Text != "" {
				a.textParts = append(a.textParts, part.Text)
			}
		}
	}

	if resp.UsageMetadata != nil {
		a.promptTokens = resp.UsageMetadata.PromptTokenCount
		a.responseTokens += resp.UsageMetadata.CandidatesTokenCount
	}

	if resp.FinishReason != genai.FinishReasonUnspecified {
		a.finishReason = resp.FinishReason
	}
}

// GetFinal returns the aggregated final response
func (a *StreamAggregator) GetFinal() *model.LLMResponse {
	// Combine all text parts
	combinedText := strings.Join(a.textParts, "")

	return &model.LLMResponse{
		Content: &genai.Content{
			Role: "model",
			Parts: []*genai.Part{
				{Text: combinedText},
			},
		},
		FinishReason: a.finishReason,
		UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     a.promptTokens,
			CandidatesTokenCount: a.responseTokens,
			TotalTokenCount:      a.promptTokens + a.responseTokens,
		},
		TurnComplete: true,
	}
}

// convertStreamChunkToADKResponse converts a streaming chunk to ADK response
func (m *ModelAdapter) convertStreamChunkToADKResponse(chunk *ai.ChatStreamChunk) *model.LLMResponse {
	resp := &model.LLMResponse{
		Partial: true,
	}

	// Use first choice (streaming typically has one choice)
	if len(chunk.Choices) == 0 {
		return resp
	}

	choice := chunk.Choices[0]

	// Convert delta content
	if choice.Delta.Content != "" {
		resp.Content = &genai.Content{
			Role: "model",
			Parts: []*genai.Part{
				{Text: choice.Delta.Content},
			},
		}
	}

	// Handle tool calls in delta
	if len(choice.Delta.ToolCalls) > 0 {
		if resp.Content == nil {
			resp.Content = &genai.Content{Role: "model", Parts: []*genai.Part{}}
		}
		for _, tc := range choice.Delta.ToolCalls {
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				m.log.Warn("Failed to parse tool call arguments in stream", "error", err)
				continue
			}
			resp.Content.Parts = append(resp.Content.Parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					Name: tc.Function.Name,
					Args: args,
				},
			})
		}
	}

	// Set finish reason if present
	if choice.FinishReason != "" {
		switch choice.FinishReason {
		case ai.FinishReasonStop:
			resp.FinishReason = genai.FinishReasonStop
		case ai.FinishReasonLength:
			resp.FinishReason = genai.FinishReasonMaxTokens
		default:
			resp.FinishReason = genai.FinishReasonStop
		}
	}

	// Set token usage if available (usually only in final chunk)
	if chunk.Usage != nil {
		resp.UsageMetadata = &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     int32(chunk.Usage.PromptTokens),
			CandidatesTokenCount: int32(chunk.Usage.CompletionTokens),
			TotalTokenCount:      int32(chunk.Usage.TotalTokens),
		}
	}

	return resp
}

// Ensure ModelAdapter implements model.LLM
var _ model.LLM = (*ModelAdapter)(nil)
