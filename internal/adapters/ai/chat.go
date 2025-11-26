package ai

import "context"

// ChatProvider extends Provider with actual LLM chat completion capabilities.
type ChatProvider interface {
	Provider

	// Chat sends a chat completion request with tool calling support.
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// ChatStream sends a chat completion request with streaming.
	ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatStreamChunk, <-chan error)
}

// ChatRequest represents a chat completion request.
type ChatRequest struct {
	Model       string
	Messages    []Message
	Tools       []ToolDefinition
	Temperature float64
	MaxTokens   int
	TopP        float64
	Stream      bool
}

// Message represents a single message in the conversation.
type Message struct {
	Role       MessageRole
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string // For tool responses
	Name       string // For function/tool messages
}

// MessageRole defines the role of a message sender.
type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

// ToolDefinition describes a tool/function that the model can call.
type ToolDefinition struct {
	Type     string             `json:"type"` // "function"
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition describes a callable function.
type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON schema
}

// ChatResponse represents the response from a chat completion.
type ChatResponse struct {
	ID      string
	Model   string
	Choices []Choice
	Usage   Usage
}

// Choice represents a single completion choice.
type Choice struct {
	Index        int
	Message      Message
	FinishReason FinishReason
}

// FinishReason indicates why the model stopped generating.
type FinishReason string

const (
	FinishReasonStop      FinishReason = "stop"
	FinishReasonLength    FinishReason = "length"
	FinishReasonToolCalls FinishReason = "tool_calls"
	FinishReasonError     FinishReason = "error"
)

// ToolCall represents a tool invocation request from the model.
type ToolCall struct {
	ID       string
	Type     string // "function"
	Function FunctionCall
}

// FunctionCall represents a function call from the model.
type FunctionCall struct {
	Name      string
	Arguments string // JSON-encoded arguments
}

// Usage tracks token consumption.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// ChatStreamChunk represents a single chunk in a streaming response.
type ChatStreamChunk struct {
	ID      string
	Model   string
	Choices []StreamChoice
	Usage   *Usage // Only present in final chunk
}

// StreamChoice represents a streaming choice.
type StreamChoice struct {
	Index        int
	Delta        MessageDelta
	FinishReason FinishReason
}

// MessageDelta represents incremental message content in streaming.
type MessageDelta struct {
	Role      MessageRole
	Content   string
	ToolCalls []ToolCall
}
