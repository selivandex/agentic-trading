package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"prometheus/pkg/errors"
)

const claudeAPIURL = "https://api.anthropic.com/v1/messages"

// Ensure ClaudeProvider implements ChatProvider
var _ ChatProvider = (*ClaudeProvider)(nil)

// Chat sends a chat completion request to Claude API.
func (p *ClaudeProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if p.apiKey == "" {
		return nil, errors.Wrap(errors.ErrInvalidInput, "claude API key not configured")
	}

	// Convert to Claude API format
	claudeReq := p.convertToClaude(req)

	// Marshal request
	body, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, errors.Wrap(err, "marshal claude request")
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", claudeAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "create HTTP request")
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	client := &http.Client{Timeout: p.timeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "send claude request")
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read claude response")
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			return nil, errors.Wrapf(errors.ErrExternal, "claude API error (%d): %s - %s",
				resp.StatusCode, errResp.Error.Type, errResp.Error.Message)
		}
		return nil, errors.Wrapf(errors.ErrExternal, "claude API error (%d): %s",
			resp.StatusCode, string(respBody))
	}

	// Parse response
	var claudeResp claudeResponse
	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal claude response")
	}

	// Convert back to our format
	return p.convertFromClaude(&claudeResp), nil
}

// ChatStream implements streaming (not implemented yet - returns error).
func (p *ClaudeProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatStreamChunk, <-chan error) {
	errCh := make(chan error, 1)
	errCh <- errors.Wrap(errors.ErrNotImplemented, "claude streaming not yet implemented")
	close(errCh)
	return nil, errCh
}

// Claude API types
type claudeRequest struct {
	Model       string          `json:"model"`
	Messages    []claudeMessage `json:"messages"`
	System      string          `json:"system,omitempty"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Tools       []claudeTool    `json:"tools,omitempty"`
}

type claudeMessage struct {
	Role    string      `json:"role"`    // "user" or "assistant"
	Content interface{} `json:"content"` // string or []claudeContent
}

type claudeContent struct {
	Type      string                 `json:"type"` // "text", "tool_use", "tool_result"
	Text      string                 `json:"text,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Content   interface{}            `json:"content,omitempty"` // For tool_result
	ToolUseID string                 `json:"tool_use_id,omitempty"`
}

type claudeTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type claudeResponse struct {
	ID           string          `json:"id"`
	Type         string          `json:"type"`
	Role         string          `json:"role"`
	Content      []claudeContent `json:"content"`
	Model        string          `json:"model"`
	StopReason   string          `json:"stop_reason"`
	StopSequence string          `json:"stop_sequence,omitempty"`
	Usage        claudeUsage     `json:"usage"`
}

type claudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// convertToClaude converts our request format to Claude's format.
func (p *ClaudeProvider) convertToClaude(req ChatRequest) claudeRequest {
	claudeReq := claudeRequest{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	if claudeReq.MaxTokens == 0 {
		claudeReq.MaxTokens = 4096 // Default
	}

	// Separate system message
	var systemPrompt string
	var messages []Message
	for _, msg := range req.Messages {
		if msg.Role == RoleSystem {
			systemPrompt = msg.Content
		} else {
			messages = append(messages, msg)
		}
	}
	claudeReq.System = systemPrompt

	// Convert messages
	for _, msg := range messages {
		claudeMsg := claudeMessage{Role: string(msg.Role)}

		// Handle different message types
		if msg.Role == RoleTool {
			// Tool result
			claudeMsg.Role = "user"
			claudeMsg.Content = []claudeContent{{
				Type:      "tool_result",
				ToolUseID: msg.ToolCallID,
				Content:   msg.Content,
			}}
		} else if len(msg.ToolCalls) > 0 {
			// Assistant with tool calls
			contents := []claudeContent{}
			if msg.Content != "" {
				contents = append(contents, claudeContent{
					Type: "text",
					Text: msg.Content,
				})
			}
			for _, tc := range msg.ToolCalls {
				var input map[string]interface{}
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &input) // Ignore unmarshal errors
				contents = append(contents, claudeContent{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: input,
				})
			}
			claudeMsg.Content = contents
		} else {
			// Simple text message
			claudeMsg.Content = msg.Content
		}

		claudeReq.Messages = append(claudeReq.Messages, claudeMsg)
	}

	// Convert tools
	for _, tool := range req.Tools {
		claudeReq.Tools = append(claudeReq.Tools, claudeTool{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			InputSchema: tool.Function.Parameters,
		})
	}

	return claudeReq
}

// convertFromClaude converts Claude's response to our format.
func (p *ClaudeProvider) convertFromClaude(resp *claudeResponse) *ChatResponse {
	chatResp := &ChatResponse{
		ID:    resp.ID,
		Model: resp.Model,
		Usage: Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}

	// Build message from content blocks
	msg := Message{Role: MessageRole(resp.Role)}
	var textParts []string
	var toolCalls []ToolCall

	for _, content := range resp.Content {
		switch content.Type {
		case "text":
			textParts = append(textParts, content.Text)
		case "tool_use":
			argsBytes, _ := json.Marshal(content.Input)
			toolCalls = append(toolCalls, ToolCall{
				ID:   content.ID,
				Type: "function",
				Function: FunctionCall{
					Name:      content.Name,
					Arguments: string(argsBytes),
				},
			})
		}
	}

	msg.Content = joinTextParts(textParts)
	msg.ToolCalls = toolCalls

	// Determine finish reason
	finishReason := FinishReasonStop
	switch resp.StopReason {
	case "end_turn":
		finishReason = FinishReasonStop
	case "max_tokens":
		finishReason = FinishReasonLength
	case "tool_use":
		finishReason = FinishReasonToolCalls
	case "stop_sequence":
		finishReason = FinishReasonStop
	}

	chatResp.Choices = []Choice{{
		Index:        0,
		Message:      msg,
		FinishReason: finishReason,
	}}

	return chatResp
}

func joinTextParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += "\n"
		}
		result += part
	}
	return result
}
