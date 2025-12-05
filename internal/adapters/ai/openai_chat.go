package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"prometheus/pkg/errors"
)

const openaiAPIURL = "https://api.openai.com/v1/chat/completions"

// Ensure OpenAIProvider implements ChatProvider
var _ ChatProvider = (*OpenAIProvider)(nil)

// Chat sends a chat completion request to OpenAI API.
func (p *OpenAIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if p.apiKey == "" {
		return nil, errors.Wrap(errors.ErrInvalidInput, "openai API key not configured")
	}

	// Wait for rate limiter
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return nil, &RateLimitError{
			Provider: ProviderNameOpenAI,
			Limit:    p.rateLimiter.Limit(),
			Err:      err,
		}
	}

	// Convert to OpenAI format (reuse DeepSeek types since they're compatible)
	openAIReq := openAIRequest{
		Model:       req.Model,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		TopP:        req.TopP,
		Stream:      req.Stream,
	}

	if openAIReq.MaxTokens == 0 {
		openAIReq.MaxTokens = 4096
	}

	// Convert messages
	for _, msg := range req.Messages {
		openAIMsg := openAIMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
			Name:    msg.Name,
		}

		for _, tc := range msg.ToolCalls {
			openAIMsg.ToolCalls = append(openAIMsg.ToolCalls, openAIToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: openAIFunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}

		if msg.ToolCallID != "" {
			openAIMsg.ToolCallID = msg.ToolCallID
		}

		openAIReq.Messages = append(openAIReq.Messages, openAIMsg)
	}

	// Convert tools
	for _, tool := range req.Tools {
		openAIReq.Tools = append(openAIReq.Tools, openAITool{
			Type: tool.Type,
			Function: openAIFunctionDef{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			},
		})
	}

	// Marshal request
	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, errors.Wrap(err, "marshal openai request")
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", openaiAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "create HTTP request")
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Send request
	client := &http.Client{Timeout: p.timeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "send openai request")
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read openai response")
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			return nil, errors.Wrapf(errors.ErrExternal, "openai API error (%d): %s - %s",
				resp.StatusCode, errResp.Error.Type, errResp.Error.Message)
		}
		return nil, errors.Wrapf(errors.ErrExternal, "openai API error (%d): %s",
			resp.StatusCode, string(respBody))
	}

	// Parse response
	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal openai response")
	}

	// Convert to our format
	chatResp := &ChatResponse{
		ID:    openAIResp.ID,
		Model: openAIResp.Model,
		Usage: Usage{
			PromptTokens:     openAIResp.Usage.PromptTokens,
			CompletionTokens: openAIResp.Usage.CompletionTokens,
			TotalTokens:      openAIResp.Usage.TotalTokens,
		},
	}

	for _, choice := range openAIResp.Choices {
		msg := Message{
			Role:    MessageRole(choice.Message.Role),
			Content: choice.Message.Content,
			Name:    choice.Message.Name,
		}

		for _, tc := range choice.Message.ToolCalls {
			msg.ToolCalls = append(msg.ToolCalls, ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}

		finishReason := FinishReasonStop
		switch choice.FinishReason {
		case "stop":
			finishReason = FinishReasonStop
		case "length":
			finishReason = FinishReasonLength
		case "tool_calls", "function_call":
			finishReason = FinishReasonToolCalls
		default:
			finishReason = FinishReasonStop
		}

		chatResp.Choices = append(chatResp.Choices, Choice{
			Index:        choice.Index,
			Message:      msg,
			FinishReason: finishReason,
		})
	}

	return chatResp, nil
}

// ChatStream implements streaming (not implemented yet - returns error).
func (p *OpenAIProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatStreamChunk, <-chan error) {
	errCh := make(chan error, 1)
	errCh <- errors.Wrap(errors.ErrNotImplemented, "openai streaming not yet implemented")
	close(errCh)
	return nil, errCh
}
