package agents

import (
	"testing"
	"time"
)

func TestConversationManager_Basic(t *testing.T) {
	// Create conversation manager
	systemPrompt := "You are a test agent."
	cm := NewConversationManager(systemPrompt, 10000)

	if cm == nil {
		t.Fatal("NewConversationManager returned nil")
	}

	// Check initial state
	if cm.systemPrompt != systemPrompt {
		t.Errorf("Expected system prompt %q, got %q", systemPrompt, cm.systemPrompt)
	}

	if cm.GetTurnCount() != 0 {
		t.Errorf("Expected 0 turns, got %d", cm.GetTurnCount())
	}

	if cm.IsComplete() {
		t.Error("Empty conversation should not be complete")
	}
}

func TestConversationManager_AddMessages(t *testing.T) {
	cm := NewConversationManager("Test agent", 10000)

	// Add user message
	cm.AddUserMessage("Analyze BTC")
	
	if cm.GetTurnCount() != 1 {
		t.Errorf("Expected 1 turn, got %d", cm.GetTurnCount())
	}

	history := cm.GetHistory()
	if len(history) != 1 {
		t.Fatalf("Expected 1 message in history, got %d", len(history))
	}

	if history[0].Role != "user" {
		t.Errorf("Expected role 'user', got %q", history[0].Role)
	}

	if history[0].Content != "Analyze BTC" {
		t.Errorf("Expected content 'Analyze BTC', got %q", history[0].Content)
	}

	// Should not be complete (needs assistant response)
	if cm.IsComplete() {
		t.Error("Conversation with only user message should not be complete")
	}
}

func TestConversationManager_ToolCalls(t *testing.T) {
	cm := NewConversationManager("Test agent", 10000)

	// User message
	cm.AddUserMessage("Get BTC price")

	// Assistant with tool call
	toolCalls := []ToolCall{
		{
			ID:   "call_1",
			Name: "get_price",
			Arguments: map[string]interface{}{
				"symbol": "BTC/USDT",
			},
		},
	}
	cm.AddAssistantMessage("Let me check the price", toolCalls)

	if cm.GetTurnCount() != 1 {
		t.Errorf("Expected 1 turn, got %d", cm.GetTurnCount())
	}

	// Should not be complete (has tool calls)
	if cm.IsComplete() {
		t.Error("Conversation with pending tool calls should not be complete")
	}

	// Add tool result
	err := cm.AddToolResult("call_1", "get_price", map[string]interface{}{
		"price": 42000.0,
		"bid":   41999.0,
		"ask":   42001.0,
	})

	if err != nil {
		t.Fatalf("AddToolResult failed: %v", err)
	}

	history := cm.GetHistory()
	if len(history) != 3 {
		t.Fatalf("Expected 3 messages (user + assistant + tool), got %d", len(history))
	}

	// Last message should be tool result
	toolMsg := history[2]
	if toolMsg.Role != "tool" {
		t.Errorf("Expected role 'tool', got %q", toolMsg.Role)
	}

	if toolMsg.ToolCallID != "call_1" {
		t.Errorf("Expected tool_call_id 'call_1', got %q", toolMsg.ToolCallID)
	}
}

func TestConversationManager_Complete(t *testing.T) {
	cm := NewConversationManager("Test agent", 10000)

	// User message
	cm.AddUserMessage("Analyze BTC")

	// Assistant final response (no tool calls)
	cm.AddAssistantMessage("BTC is at $42k, uptrend confirmed", nil)

	// Should be complete now
	if !cm.IsComplete() {
		t.Error("Conversation should be complete after assistant message with no tool calls")
	}

	lastMsg := cm.GetLastMessage()
	if lastMsg == nil {
		t.Fatal("GetLastMessage returned nil")
	}

	if lastMsg.Role != "assistant" {
		t.Errorf("Expected last role 'assistant', got %q", lastMsg.Role)
	}

	if len(lastMsg.ToolCalls) != 0 {
		t.Error("Final message should have no tool calls")
	}
}

func TestConversationManager_TokenTracking(t *testing.T) {
	cm := NewConversationManager("System prompt", 10000)

	initialTokens := cm.GetTokenCount()
	if initialTokens == 0 {
		t.Error("System prompt should have non-zero tokens")
	}

	// Add message
	cm.AddUserMessage("Short message")
	
	afterUserMsg := cm.GetTokenCount()
	if afterUserMsg <= initialTokens {
		t.Error("Token count should increase after adding message")
	}

	// Add assistant with tool call
	toolCalls := []ToolCall{
		{ID: "call_1", Name: "get_price", Arguments: map[string]interface{}{"symbol": "BTC"}},
	}
	cm.AddAssistantMessage("Checking price", toolCalls)

	afterToolCall := cm.GetTokenCount()
	if afterToolCall <= afterUserMsg {
		t.Error("Token count should increase after tool call")
	}
}

func TestConversationManager_Compression(t *testing.T) {
	cm := NewConversationManager("Test", 10000)

	// Add many messages (more than compression threshold)
	for i := 0; i < 20; i++ {
		cm.AddUserMessage("Message " + time.Now().String())
		cm.AddAssistantMessage("Response "+time.Now().String(), nil)
	}

	initialCount := len(cm.GetHistory())
	if initialCount != 40 {
		t.Fatalf("Expected 40 messages, got %d", initialCount)
	}

	// Compress
	cm.Compress()

	afterCompression := len(cm.GetHistory())
	if afterCompression >= initialCount {
		t.Errorf("Compression should reduce message count: before=%d after=%d",
			initialCount, afterCompression)
	}

	// Should keep at least last 10 + summary
	if afterCompression < 11 {
		t.Errorf("Compression should keep at least 11 messages (summary + last 10), got %d", 
			afterCompression)
	}
}

func TestExtractStructuredOutput_Valid(t *testing.T) {
	response := `Based on my analysis, here is the result:

{
  "analysis": "BTC is in uptrend",
  "confidence": 0.85,
  "recommendation": "long"
}

This completes my analysis.`

	result, err := ExtractStructuredOutput(response)
	if err != nil {
		t.Fatalf("ExtractStructuredOutput failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if analysis, ok := result["analysis"].(string); !ok || analysis != "BTC is in uptrend" {
		t.Errorf("Expected analysis 'BTC is in uptrend', got %v", result["analysis"])
	}

	if conf, ok := result["confidence"].(float64); !ok || conf != 0.85 {
		t.Errorf("Expected confidence 0.85, got %v", result["confidence"])
	}
}

func TestExtractStructuredOutput_NoJSON(t *testing.T) {
	response := "This is just plain text with no JSON"

	result, err := ExtractStructuredOutput(response)
	if err != nil {
		t.Fatalf("ExtractStructuredOutput should not error on plain text: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	// Should wrap in error response
	if _, ok := result["error"]; !ok {
		t.Error("Expected 'error' field for non-JSON response")
	}

	if raw, ok := result["raw_response"].(string); !ok || raw != response {
		t.Error("Expected raw_response to contain original text")
	}
}

func TestConversationManager_SetCompression(t *testing.T) {
	cm := NewConversationManager("Test", 10000)

	// Initially compression should be on (default)
	if !cm.compressionOn {
		t.Error("Compression should be enabled by default")
	}

	// Disable compression
	cm.SetCompression(false)
	if cm.compressionOn {
		t.Error("SetCompression(false) should disable compression")
	}

	// Enable compression
	cm.SetCompression(true)
	if !cm.compressionOn {
		t.Error("SetCompression(true) should enable compression")
	}
}

