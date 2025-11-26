package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	adksession "google.golang.org/adk/session"
	"google.golang.org/genai"

	"prometheus/internal/adapters/ai"
	"prometheus/internal/adapters/config"
	"prometheus/internal/domain/memory"
	"prometheus/internal/domain/reasoning"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// ExecutionInput contains input parameters for agent execution
type ExecutionInput struct {
	UserID     uuid.UUID
	Symbol     string
	MarketType string
	Context    map[string]interface{} // Additional context data
	MaxTokens  int                    // Max tokens for this execution (0 = use default)
	Timeout    time.Duration          // Execution timeout (0 = use default)
}

// ExecutionOutput contains the result of agent execution
type ExecutionOutput struct {
	AgentType   AgentType
	Result      map[string]interface{} // Structured output extracted from reasoning
	RawResponse string                 // Raw final response from agent
	Confidence  float64                // Confidence score (0-1)

	// Metrics
	TokensUsed    int
	InputTokens   int
	OutputTokens  int
	CostUSD       float64
	Duration      time.Duration
	ToolCallCount int
	TurnCount     int

	// Trace
	ReasoningTrace []Message // Full conversation history
	SessionID      string    // Unique session identifier
}

// AgentRunner executes agents with full CoT logging, cost tracking, and memory integration
type AgentRunner struct {
	agent          agent.Agent
	runner         *runner.Runner
	agentType      AgentType
	agentConfig    AgentConfig
	modelInfo      *ai.ModelInfo
	runtimeConfig  config.AgentsConfig
	sessionService adksession.Service

	memoryService *memory.Service
	reasoningRepo reasoning.Repository
	costTracker   *CostTracker

	log *logger.Logger
}

// NewAgentRunner creates a new agent runner
func NewAgentRunner(
	ag agent.Agent,
	agentType AgentType,
	agentConfig AgentConfig,
	modelInfo *ai.ModelInfo,
	runtimeConfig config.AgentsConfig,
	sessionService adksession.Service,
	memoryService *memory.Service,
	reasoningRepo reasoning.Repository,
	costTracker *CostTracker,
) (*AgentRunner, error) {
	// Use provided session service or fallback to in-memory
	if sessionService == nil {
		sessionService = adksession.InMemoryService()
	}

	// Create ADK runner for this agent
	runnerInstance, err := runner.New(runner.Config{
		AppName:        fmt.Sprintf("prometheus_%s", agentType),
		Agent:          ag,
		SessionService: sessionService,
		// Note: We use our own memory and reasoning services, not ADK's built-in ones
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ADK runner")
	}

	return &AgentRunner{
		agent:          ag,
		runner:         runnerInstance,
		agentType:      agentType,
		agentConfig:    agentConfig,
		modelInfo:      modelInfo,
		runtimeConfig:  runtimeConfig,
		sessionService: sessionService,
		memoryService:  memoryService,
		reasoningRepo:  reasoningRepo,
		costTracker:    costTracker,
		log:            logger.Get().With("component", "agent_runner", "agent", agentType),
	}, nil
}

// Execute runs the agent with full conversation tracking and logging
func (e *AgentRunner) Execute(ctx context.Context, input ExecutionInput) (*ExecutionOutput, error) {
	startTime := time.Now()
	sessionID := uuid.New().String()

	e.log.Infof("Starting agent execution: session=%s user=%s symbol=%s",
		sessionID, input.UserID, input.Symbol)

	// 1. Setup execution context with timeout
	execCtx := ctx
	timeout := input.Timeout
	if timeout == 0 {
		// Use agent-specific timeout if configured
		if e.agentConfig.TotalTimeout > 0 {
			timeout = e.agentConfig.TotalTimeout
		} else {
			// Fallback to runtime config default
			timeout = e.runtimeConfig.ExecutionTimeout
		}
	}

	if timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// 2. Load agent memory and build context (only if enabled)
	memoryContext := ""
	if e.runtimeConfig.EnableMemory {
		var err error
		memoryContext, err = e.loadMemoryContext(ctx, input.UserID)
		if err != nil {
			e.log.Warnf("Failed to load memory context: %v", err)
			// Continue without memory
		}
	}

	// 3. Initialize conversation manager
	maxTokens := input.MaxTokens
	if maxTokens == 0 {
		maxTokens = e.runtimeConfig.MaxTokens
	}

	systemPrompt := e.buildSystemPrompt(memoryContext, input)
	conversation := NewConversationManager(systemPrompt, maxTokens)
	conversation.SetCompression(e.runtimeConfig.EnableCompression)

	// 4. Add initial user message
	userMessage := e.buildUserMessage(input)
	conversation.AddUserMessage(userMessage)

	// 5. Execute agent loop
	// Note: ADK handles the loop internally, but we need to track each turn
	output, err := e.executeAgentLoop(execCtx, conversation, sessionID)
	if err != nil {
		return nil, errors.Wrap(err, "agent execution failed")
	}

	// 6. Calculate metrics
	duration := time.Since(startTime)
	output.Duration = duration
	output.SessionID = sessionID
	output.AgentType = e.agentType

	// 7. Record cost
	if e.costTracker != nil && e.modelInfo != nil {
		output.CostUSD = e.costTracker.RecordUsage(
			e.modelInfo,
			output.InputTokens,
			output.OutputTokens,
		)
	}

	// 8. Save reasoning trace to database
	if err := e.saveReasoningTrace(ctx, input.UserID, output); err != nil {
		e.log.Errorf("Failed to save reasoning trace: %v", err)
		// Don't fail execution on logging error
	}

	// 9. Update agent memory with learnings (async)
	go func() {
		if err := e.updateAgentMemory(context.Background(), input.UserID, output); err != nil {
			e.log.Errorf("Failed to update agent memory: %v", err)
		}
	}()

	e.log.Infof("Agent execution complete: session=%s duration=%v tokens=%d cost=$%.4f tools=%d",
		sessionID, duration, output.TokensUsed, output.CostUSD, output.ToolCallCount)

	return output, nil
}

// executeAgentLoop runs the agent with ADK, tracking each conversation turn
func (e *AgentRunner) executeAgentLoop(
	ctx context.Context,
	conversation *ConversationManager,
	sessionID string,
) (*ExecutionOutput, error) {

	// Get the last user message (initial prompt)
	lastMsg := conversation.GetLastMessage()
	if lastMsg == nil || lastMsg.Role != "user" {
		return nil, errors.New("conversation must start with user message")
	}

	// Convert to genai.Content
	userContent := &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{Text: lastMsg.Content},
		},
	}

	// Use a fixed userID for now (should be passed from input in real impl)
	userID := "system"

	e.log.Infof("Starting agent execution: session=%s", sessionID)

	// Run the agent through ADK runner
	// This will handle the full conversation loop automatically
	toolCallCount := 0
	totalInputTokens := 0
	totalOutputTokens := 0
	var finalResponse *adksession.Event

	// Run config for agent execution
	runConfig := agent.RunConfig{
		StreamingMode:             agent.StreamingModeSSE, // Enable streaming
		SaveInputBlobsAsArtifacts: false,                  // Can enable later with artifact service
	}

	// Iterate through events from the agent
	for event, err := range e.runner.Run(ctx, userID, sessionID, userContent, runConfig) {
		if err != nil {
			return nil, errors.Wrap(err, "agent execution failed")
		}

		if event == nil {
			continue
		}

		// Handle partial events (streaming chunks)
		if event.LLMResponse.Partial {
			e.log.Debugf("Partial event received from streaming")
			// TODO: Send partial updates to streaming channel if needed
			// For now, we just skip partial events and wait for complete ones
			continue
		}

		e.log.Debugf("Agent event: author=%s turn_complete=%v", event.Author, event.TurnComplete)

		// Track token usage from response
		if event.UsageMetadata != nil {
			totalInputTokens += int(event.UsageMetadata.PromptTokenCount)
			totalOutputTokens += int(event.UsageMetadata.CandidatesTokenCount)
		}

		// Process event content
		if event.LLMResponse.Content != nil {
			assistantContent := ""
			toolCalls := []ToolCall{}

			// Extract all content from parts
			for _, part := range event.LLMResponse.Content.Parts {
				// Text content
				if part.Text != "" {
					assistantContent += part.Text
				}

				// Function calls (tool calls)
				if part.FunctionCall != nil {
					toolCallCount++
					toolCalls = append(toolCalls, ToolCall{
						ID:        fmt.Sprintf("call_%d", toolCallCount),
						Name:      part.FunctionCall.Name,
						Arguments: part.FunctionCall.Args,
					})
					e.log.Debugf("Tool call: %s(%v)", part.FunctionCall.Name, part.FunctionCall.Args)
				}

				// Function responses (tool results)
				if part.FunctionResponse != nil {
					e.log.Debugf("Tool result: %s", part.FunctionResponse.Name)

					// Add tool result to conversation
					err := conversation.AddToolResult(
						fmt.Sprintf("call_%d", toolCallCount),
						part.FunctionResponse.Name,
						part.FunctionResponse.Response,
					)
					if err != nil {
						e.log.Warnf("Failed to add tool result to conversation: %v", err)
					}
				}
			}

			// Add assistant message to conversation if from our agent
			if event.Author == e.agent.Name() && (assistantContent != "" || len(toolCalls) > 0) {
				conversation.AddAssistantMessage(assistantContent, toolCalls)
			}
		}

		// Check if this is the final response
		if event.TurnComplete && event.IsFinalResponse() {
			finalResponse = event
			e.log.Info("Agent completed execution")
			break
		}
	}

	// Build output
	output := &ExecutionOutput{
		ReasoningTrace: conversation.GetHistory(),
		TurnCount:      conversation.GetTurnCount(),
		TokensUsed:     totalInputTokens + totalOutputTokens,
		InputTokens:    totalInputTokens,
		OutputTokens:   totalOutputTokens,
		ToolCallCount:  toolCallCount,
	}

	// Extract final response
	if finalResponse != nil && finalResponse.LLMResponse.Content != nil {
		rawResponse := ""
		for _, part := range finalResponse.LLMResponse.Content.Parts {
			if part.Text != "" {
				rawResponse += part.Text
			}
		}
		output.RawResponse = rawResponse

		// Try to extract structured output from the response
		structured, err := ExtractStructuredOutput(rawResponse)
		if err == nil {
			output.Result = structured

			// Extract confidence if present
			if conf, ok := structured["confidence"].(float64); ok {
				output.Confidence = conf
			}
		} else {
			// Fallback: wrap raw response
			output.Result = map[string]interface{}{
				"raw_response": rawResponse,
			}
			output.Confidence = 0.5
		}
	} else {
		// No final response - execution may have been interrupted
		output.RawResponse = "No final response received"
		output.Result = map[string]interface{}{
			"error": "agent did not provide final response",
		}
		output.Confidence = 0.0
	}

	return output, nil
}

// buildSystemPrompt constructs the system prompt with memory context
func (e *AgentRunner) buildSystemPrompt(memoryContext string, input ExecutionInput) string {
	// The system prompt should already be in the agent's instruction
	// But we can enhance it with memory context

	if memoryContext == "" {
		return "" // Agent already has instruction
	}

	// Add memory context as additional system information
	return fmt.Sprintf(`# MEMORY CONTEXT

%s

# CURRENT TASK
Analyze %s (%s) for user.

---
`, memoryContext, input.Symbol, input.MarketType)
}

// buildUserMessage constructs the initial user message
func (e *AgentRunner) buildUserMessage(input ExecutionInput) string {
	contextJSON := ""
	if len(input.Context) > 0 {
		if data, err := json.Marshal(input.Context); err == nil {
			contextJSON = string(data)
		}
	}

	message := fmt.Sprintf(`Analyze %s (%s market).

Please perform comprehensive analysis using available tools.
Think step-by-step and explain your reasoning.`,
		input.Symbol,
		input.MarketType,
	)

	if contextJSON != "" {
		message += fmt.Sprintf("\n\nAdditional context:\n%s", contextJSON)
	}

	return message
}

// loadMemoryContext loads relevant memories for the agent
func (e *AgentRunner) loadMemoryContext(ctx context.Context, userID uuid.UUID) (string, error) {
	if e.memoryService == nil {
		return "", nil
	}

	// Get recent memories for this user + agent type
	memories, err := e.memoryService.GetRecentByAgent(ctx, userID, string(e.agentType), 5)
	if err != nil {
		e.log.Warnf("Failed to load recent memories: %v", err)
		return "", nil // Don't fail execution on memory load error
	}

	if len(memories) == 0 {
		return "", nil
	}

	// Format memories into context string
	var contextBuilder strings.Builder
	contextBuilder.WriteString("## Past Experience & Learnings\n\n")

	for i, mem := range memories {
		contextBuilder.WriteString(fmt.Sprintf("### Memory %d (%s ago)\n", i+1, formatTimeAgo(mem.CreatedAt)))
		contextBuilder.WriteString(fmt.Sprintf("Type: %s\n", mem.Type))

		if mem.Symbol != "" {
			contextBuilder.WriteString(fmt.Sprintf("Symbol: %s\n", mem.Symbol))
		}

		// Truncate content if too long
		content := mem.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		contextBuilder.WriteString(fmt.Sprintf("Content: %s\n\n", content))
	}

	return contextBuilder.String(), nil
}

// saveReasoningTrace saves the full conversation to the database
func (e *AgentRunner) saveReasoningTrace(
	ctx context.Context,
	userID uuid.UUID,
	output *ExecutionOutput,
) error {
	if e.reasoningRepo == nil {
		return nil
	}

	// Convert reasoning trace to steps
	steps := make([]reasoning.Step, 0, len(output.ReasoningTrace))
	for i, msg := range output.ReasoningTrace {
		step := reasoning.Step{
			Step:      i + 1,
			Timestamp: msg.Timestamp,
		}

		switch msg.Role {
		case "user":
			step.Action = "user_input"
			step.Content = msg.Content

		case "assistant":
			if len(msg.ToolCalls) > 0 {
				step.Action = "tool_call"
				// Record tool calls
				for _, tc := range msg.ToolCalls {
					step.Tool = tc.Name
					step.Input = tc.Arguments
					break // Just first tool for now
				}
			} else {
				step.Action = "thinking"
				step.Content = msg.Content
			}

		case "tool":
			step.Action = "tool_result"
			if toolName, ok := msg.Metadata["tool_name"].(string); ok {
				step.Tool = toolName
			}
			// Parse tool result
			var resultMap map[string]interface{}
			if err := json.Unmarshal([]byte(msg.Content), &resultMap); err == nil {
				step.Output = resultMap
			}
		}

		steps = append(steps, step)
	}

	// Marshal steps to JSON
	stepsJSON, err := json.Marshal(steps)
	if err != nil {
		return errors.Wrap(err, "marshal reasoning steps")
	}

	// Marshal decision to JSON
	decisionJSON, err := json.Marshal(output.Result)
	if err != nil {
		return errors.Wrap(err, "marshal decision")
	}

	// Create log entry
	entry := &reasoning.LogEntry{
		ID:             uuid.New(),
		UserID:         userID,
		AgentID:        string(e.agentType),
		SessionID:      output.SessionID,
		ReasoningSteps: stepsJSON,
		Decision:       decisionJSON,
		Confidence:     output.Confidence,
		TokensUsed:     output.TokensUsed,
		CostUSD:        output.CostUSD,
		DurationMs:     int(output.Duration.Milliseconds()),
		ToolCallsCount: output.ToolCallCount,
		CreatedAt:      time.Now(),
	}

	return e.reasoningRepo.Create(ctx, entry)
}

// updateAgentMemory updates the agent's long-term memory with learnings
func (e *AgentRunner) updateAgentMemory(
	ctx context.Context,
	userID uuid.UUID,
	output *ExecutionOutput,
) error {
	if e.memoryService == nil {
		return nil
	}

	// Extract insights from the agent's reasoning
	// We look for high-confidence patterns or explicit learnings
	if output.Confidence < 0.6 {
		// Don't store low-confidence outputs as learnings
		return nil
	}

	// Create memory entry from the final result
	contentJSON, err := json.Marshal(output.Result)
	if err != nil {
		return errors.Wrap(err, "marshal output result")
	}

	// Generate embedding for semantic search
	// Note: Embedding generation is handled by tools (save_analysis, save_insight)
	// which have access to EmbeddingService via Deps. Here we store without embedding,
	// and it can be generated asynchronously by a background worker if needed.
	var embedding pgvector.Vector

	mem := &memory.Memory{
		ID:         uuid.New(),
		UserID:     userID,
		AgentID:    string(e.agentType),
		SessionID:  output.SessionID,
		Type:       memory.MemoryDecision,
		Content:    string(contentJSON),
		Embedding:  embedding,
		Importance: output.Confidence,
		CreatedAt:  time.Now(),
	}

	return e.memoryService.Store(ctx, mem)
}

// formatTimeAgo formats a time as "X hours ago", "X days ago", etc.
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		mins := int(duration.Minutes())
		return fmt.Sprintf("%d min ago", mins)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	}
}

// ExtractStructuredOutput extracts structured data from agent's final response
func ExtractStructuredOutput(response string) (map[string]interface{}, error) {
	// Try to find JSON in the response
	// Agents should end with a JSON block containing structured output

	// Look for JSON block (between { and })
	start := -1
	braceCount := 0

	for i, ch := range response {
		if ch == '{' {
			if start == -1 {
				start = i
			}
			braceCount++
		} else if ch == '}' {
			braceCount--
			if braceCount == 0 && start != -1 {
				// Found complete JSON block
				jsonStr := response[start : i+1]
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
					return result, nil
				}
			}
		}
	}

	// If no JSON found, return response as plain text
	return map[string]interface{}{
		"raw_response": response,
		"error":        "no structured output found",
	}, nil
}
