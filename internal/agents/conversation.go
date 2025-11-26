package agents

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/genai"

	"prometheus/pkg/errors"
)

// Message represents a single message in the conversation history
type Message struct {
	Role       string                 `json:"role"` // "user", "assistant", "tool", "system"
	Content    string                 `json:"content"`
	ToolCalls  []ToolCall             `json:"tool_calls,omitempty"`
	ToolCallID string                 `json:"tool_call_id,omitempty"` // For tool response messages
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Tokens     int                    `json:"tokens,omitempty"` // Estimated token count
}

// ToolCall represents a function/tool call request
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ConversationManager manages conversation history with token tracking and compression
type ConversationManager struct {
	history       []Message
	systemPrompt  string
	maxTokens     int
	currentTokens int
	compressionOn bool
	turnCount     int
}

// NewConversationManager creates a new conversation manager
func NewConversationManager(systemPrompt string, maxTokens int) *ConversationManager {
	if maxTokens <= 0 {
		maxTokens = 50000 // Default: 50k tokens
	}

	return &ConversationManager{
		history:       make([]Message, 0, 50),
		systemPrompt:  systemPrompt,
		maxTokens:     maxTokens,
		currentTokens: estimateTokens(systemPrompt),
		compressionOn: true,
	}
}

// AddUserMessage adds a user message to the conversation
func (cm *ConversationManager) AddUserMessage(content string) {
	msg := Message{
		Role:      "user",
		Content:   content,
		Timestamp: time.Now(),
		Tokens:    estimateTokens(content),
	}
	cm.history = append(cm.history, msg)
	cm.currentTokens += msg.Tokens
	cm.turnCount++
}

// AddAssistantMessage adds an assistant message to the conversation
func (cm *ConversationManager) AddAssistantMessage(content string, toolCalls []ToolCall) {
	msg := Message{
		Role:      "assistant",
		Content:   content,
		ToolCalls: toolCalls,
		Timestamp: time.Now(),
		Tokens:    estimateTokens(content) + estimateToolCallsTokens(toolCalls),
	}
	cm.history = append(cm.history, msg)
	cm.currentTokens += msg.Tokens
}

// AddToolResult adds a tool execution result to the conversation
func (cm *ConversationManager) AddToolResult(toolCallID, toolName string, result interface{}) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return errors.Wrap(err, "failed to marshal tool result")
	}

	content := string(resultJSON)
	msg := Message{
		Role:       "tool",
		Content:    content,
		ToolCallID: toolCallID,
		Metadata: map[string]interface{}{
			"tool_name": toolName,
		},
		Timestamp: time.Now(),
		Tokens:    estimateTokens(content),
	}

	cm.history = append(cm.history, msg)
	cm.currentTokens += msg.Tokens

	// Check if we need compression
	if cm.compressionOn && cm.currentTokens > cm.maxTokens {
		cm.Compress()
	}

	return nil
}

// GetHistory returns the full conversation history
func (cm *ConversationManager) GetHistory() []Message {
	return cm.history
}

// GetHistoryForADK converts conversation history to ADK format (genai.Content)
func (cm *ConversationManager) GetHistoryForADK() []*genai.Content {
	contents := make([]*genai.Content, 0, len(cm.history)+1)

	// Add system prompt as first message
	if cm.systemPrompt != "" {
		contents = append(contents, &genai.Content{
			Role: "system",
			Parts: []*genai.Part{
				{Text: cm.systemPrompt},
			},
		})
	}

	// Convert messages to ADK format
	for _, msg := range cm.history {
		content := &genai.Content{
			Role:  convertRoleToADK(msg.Role),
			Parts: []*genai.Part{},
		}

		// Add text content
		if msg.Content != "" {
			content.Parts = append(content.Parts, &genai.Part{
				Text: msg.Content,
			})
		}

		// Add tool calls (function calls in ADK)
		for _, tc := range msg.ToolCalls {
			content.Parts = append(content.Parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					Name: tc.Name,
					Args: tc.Arguments,
				},
			})
		}

		// Add tool response
		if msg.Role == "tool" && msg.ToolCallID != "" {
			// Tool results are represented as FunctionResponse
			var resultMap map[string]interface{}
			if err := json.Unmarshal([]byte(msg.Content), &resultMap); err != nil {
				// If not JSON, wrap as text
				resultMap = map[string]interface{}{"result": msg.Content}
			}

			toolName := ""
			if tn, ok := msg.Metadata["tool_name"].(string); ok {
				toolName = tn
			}

			content.Parts = append(content.Parts, &genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					Name:     toolName,
					Response: resultMap,
				},
			})
		}

		contents = append(contents, content)
	}

	return contents
}

// GetLastMessage returns the last message in the conversation
func (cm *ConversationManager) GetLastMessage() *Message {
	if len(cm.history) == 0 {
		return nil
	}
	return &cm.history[len(cm.history)-1]
}

// GetTokenCount returns the current estimated token count
func (cm *ConversationManager) GetTokenCount() int {
	return cm.currentTokens
}

// GetTurnCount returns the number of conversation turns
func (cm *ConversationManager) GetTurnCount() int {
	return cm.turnCount
}

// IsComplete checks if the conversation is complete (last message has no tool calls)
func (cm *ConversationManager) IsComplete() bool {
	if len(cm.history) == 0 {
		return false
	}
	lastMsg := cm.history[len(cm.history)-1]
	return lastMsg.Role == "assistant" && len(lastMsg.ToolCalls) == 0
}

// Compress compresses the conversation history to reduce token count
// Strategy: Keep system prompt + last N turns, compress middle turns
func (cm *ConversationManager) Compress() {
	if len(cm.history) <= 10 {
		return // Too short to compress
	}

	// Keep last 10 messages
	keepLast := 10
	if len(cm.history) < keepLast {
		keepLast = len(cm.history)
	}

	// Compress messages in the middle (before last 10)
	compressFrom := 0
	compressTo := len(cm.history) - keepLast

	if compressTo <= compressFrom {
		return // Nothing to compress
	}

	// Create summary of compressed turns
	summary := cm.summarizeMessages(cm.history[compressFrom:compressTo])

	// Build new history
	newHistory := make([]Message, 0, keepLast+1)

	// Add summary as a single message
	summaryMsg := Message{
		Role:      "assistant",
		Content:   summary,
		Timestamp: time.Now(),
		Tokens:    estimateTokens(summary),
		Metadata: map[string]interface{}{
			"compressed": true,
			"turns":      compressTo - compressFrom,
		},
	}
	newHistory = append(newHistory, summaryMsg)

	// Add last N messages
	newHistory = append(newHistory, cm.history[compressTo:]...)

	// Update history and recalculate tokens
	cm.history = newHistory
	cm.recalculateTokens()
}

// summarizeMessages creates a concise summary of multiple messages
func (cm *ConversationManager) summarizeMessages(messages []Message) string {
	var summary strings.Builder
	summary.WriteString("[COMPRESSED HISTORY]\n")
	summary.WriteString(fmt.Sprintf("Summary of %d conversation turns:\n\n", len(messages)))

	toolCallsSummary := make(map[string]int)
	keyFindings := []string{}

	for _, msg := range messages {
		if msg.Role == "assistant" {
			// Extract key statements (non-empty content)
			if msg.Content != "" && len(msg.Content) > 20 {
				// Take first 100 chars as key finding
				finding := msg.Content
				if len(finding) > 100 {
					finding = finding[:100] + "..."
				}
				keyFindings = append(keyFindings, finding)
			}

			// Count tool calls
			for _, tc := range msg.ToolCalls {
				toolCallsSummary[tc.Name]++
			}
		}
	}

	// Write tool calls summary
	if len(toolCallsSummary) > 0 {
		summary.WriteString("Tools used:\n")
		for toolName, count := range toolCallsSummary {
			summary.WriteString(fmt.Sprintf("- %s: %d times\n", toolName, count))
		}
		summary.WriteString("\n")
	}

	// Write key findings
	if len(keyFindings) > 0 {
		summary.WriteString("Key observations:\n")
		for i, finding := range keyFindings {
			if i >= 5 { // Limit to 5 key findings
				break
			}
			summary.WriteString(fmt.Sprintf("- %s\n", finding))
		}
	}

	return summary.String()
}

// recalculateTokens recalculates the total token count
func (cm *ConversationManager) recalculateTokens() {
	total := estimateTokens(cm.systemPrompt)
	for _, msg := range cm.history {
		total += msg.Tokens
	}
	cm.currentTokens = total
}

// Clear clears the conversation history
func (cm *ConversationManager) Clear() {
	cm.history = make([]Message, 0, 50)
	cm.currentTokens = estimateTokens(cm.systemPrompt)
	cm.turnCount = 0
}

// SetCompression enables or disables conversation compression
func (cm *ConversationManager) SetCompression(enabled bool) {
	cm.compressionOn = enabled
}

// convertRoleToADK converts our role format to ADK role format
func convertRoleToADK(role string) string {
	switch role {
	case "user":
		return "user"
	case "assistant":
		return "model"
	case "system":
		return "system"
	case "tool":
		return "tool"
	default:
		return "user"
	}
}

// estimateTokens provides a rough estimate of token count
// Rule of thumb: ~4 characters per token for English text
func estimateTokens(text string) int {
	if text == "" {
		return 0
	}
	// Rough estimation: 1 token ≈ 4 characters
	return len(text) / 4
}

// estimateToolCallsTokens estimates tokens for tool calls
func estimateToolCallsTokens(toolCalls []ToolCall) int {
	total := 0
	for _, tc := range toolCalls {
		// Function name
		total += len(tc.Name) / 4
		// Arguments (serialize to estimate)
		if argsJSON, err := json.Marshal(tc.Arguments); err == nil {
			total += len(argsJSON) / 4
		}
	}
	return total
}
