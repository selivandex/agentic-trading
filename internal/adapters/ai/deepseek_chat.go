package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"prometheus/pkg/errors"
)

const deepseekAPIURL = "https://api.deepseek.com/v1/chat/completions"

// Ensure DeepSeekProvider implements ChatProvider
var _ ChatProvider = (*DeepSeekProvider)(nil)

// Chat sends a chat completion request to DeepSeek API.
func (p *DeepSeekProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if p.apiKey == "" {
		return nil, errors.Wrap(errors.ErrInvalidInput, "deepseek API key not configured")
	}

	// Wait for rate limiter (will be no-op for DeepSeek by default)
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return nil, &RateLimitError{
			Provider: ProviderNameDeepSeek,
			Limit:    p.rateLimiter.Limit(),
			Err:      err,
		}
	}

	// Convert to OpenAI-compatible format (DeepSeek uses OpenAI format)
	openAIReq := p.convertToOpenAI(req)

	// Marshal request
	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, errors.Wrap(err, "marshal deepseek request")
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", deepseekAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "create HTTP request")
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Send request
	client := &http.Client{Timeout: p.timeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "send deepseek request")
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read deepseek response")
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
			return nil, errors.Wrapf(errors.ErrExternal, "deepseek API error (%d): %s - %s",
				resp.StatusCode, errResp.Error.Type, errResp.Error.Message)
		}
		return nil, errors.Wrapf(errors.ErrExternal, "deepseek API error (%d): %s",
			resp.StatusCode, string(respBody))
	}

	// Parse response
	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal deepseek response")
	}

	// Convert back to our format
	return p.convertFromOpenAI(&openAIResp), nil
}

// ChatStream implements streaming (not implemented yet - returns error).
func (p *DeepSeekProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatStreamChunk, <-chan error) {
	errCh := make(chan error, 1)
	errCh <- errors.Wrap(errors.ErrNotImplemented, "deepseek streaming not yet implemented")
	close(errCh)
	return nil, errCh
}

// OpenAI-compatible request/response types
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Tools       []openAITool    `json:"tools,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	Name       string           `json:"name,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIFunctionCall `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAITool struct {
	Type     string            `json:"type"`
	Function openAIFunctionDef `json:"function"`
}

type openAIFunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type openAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
}

type openAIChoice struct {
	Index        int           `json:"index"`
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// convertToOpenAI converts our request format to OpenAI format.
func (p *DeepSeekProvider) convertToOpenAI(req ChatRequest) openAIRequest {
	openAIReq := openAIRequest{
		Model:       req.Model,
		Messages:    []openAIMessage{}, // Initialize as empty slice (never nil)
		Tools:       []openAITool{},    // Initialize as empty slice (never nil)
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		TopP:        req.TopP,
		Stream:      req.Stream,
	}

	if openAIReq.MaxTokens == 0 {
		openAIReq.MaxTokens = 4096 // Default
	}

	// Convert messages
	for _, msg := range req.Messages {
		openAIMsg := openAIMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
			Name:    msg.Name,
		}

		// Convert tool calls
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

		// Tool response
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

	return openAIReq
}

// convertFromOpenAI converts OpenAI response to our format.
func (p *DeepSeekProvider) convertFromOpenAI(resp *openAIResponse) *ChatResponse {
	chatResp := &ChatResponse{
		ID:    resp.ID,
		Model: resp.Model,
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	// Convert choices
	for _, choice := range resp.Choices {
		msg := Message{
			Role:    MessageRole(choice.Message.Role),
			Content: choice.Message.Content,
			Name:    choice.Message.Name,
		}

		// Convert tool calls
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

		// Determine finish reason
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

	return chatResp
}
