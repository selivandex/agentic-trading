package analysis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"prometheus/internal/agents/workflows"
	analysisservice "prometheus/internal/services/analysis"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/templates"
)

// OpportunityFinder runs market research workflow to find trading opportunities
// This worker executes the global market research workflow (8 analysts + synthesizer)
// for each monitored symbol, publishing high-quality signals to Kafka for user consumption.
type OpportunityFinder struct {
	*workers.BaseWorker
	workflow       agent.Agent                       // MarketResearchWorkflow (ADK)
	runner         *runner.Runner                    // ADK runner
	sessionService session.Service                   // Session persistence
	templates      *templates.Registry               // Template registry for workflow prompts
	preScreener    *analysisservice.PreScreener      // Pre-screening service (Phase 5 optimization)
	symbols        []string                          // List of symbols to monitor
	exchange       string              // Primary exchange for analysis
	log            *logger.Logger
	
	// Metrics (Phase 5)
	totalRuns      int64
	skippedRuns    int64
	analyzedRuns   int64
}

// NewOpportunityFinder creates a new opportunity finder using ADK workflow
func NewOpportunityFinder(
	workflowFactory *workflows.Factory,
	sessionService session.Service,
	templates *templates.Registry,
	preScreener *analysisservice.PreScreener,
	symbols []string,
	exchange string,
	interval time.Duration,
	enabled bool,
) (*OpportunityFinder, error) {
	log := logger.Get().With("component", "opportunity_finder")

	// Create market research workflow
	workflow, err := workflowFactory.CreateMarketResearchWorkflow()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create market research workflow")
	}

	// Create ADK runner
	runnerInstance, err := runner.New(runner.Config{
		AppName:        "prometheus_market_research",
		Agent:          workflow,
		SessionService: sessionService,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ADK runner")
	}

	log.Info("OpportunityFinder initialized with ADK MarketResearchWorkflow",
		"prescreener_enabled", preScreener != nil,
	)

	return &OpportunityFinder{
		BaseWorker:     workers.NewBaseWorker("opportunity_finder", interval, enabled),
		workflow:       workflow,
		runner:         runnerInstance,
		sessionService: sessionService,
		templates:      templates,
		preScreener:    preScreener,
		symbols:        symbols,
		exchange:       exchange,
		log:            log,
	}, nil
}

// Run executes one iteration of market research
// For each symbol, runs the full MarketResearchWorkflow (8 analysts + synthesizer)
func (of *OpportunityFinder) Run(ctx context.Context) error {
	of.Log().Debug("Market research: starting iteration")

	if len(of.symbols) == 0 {
		of.Log().Warn("No symbols configured for market research")
		return nil
	}

	opportunities := 0
	skipped := 0
	errors := 0

	// Run workflow for each symbol
	for _, symbol := range of.symbols {
		// Check for context cancellation (graceful shutdown)
		// Critical: ADK workflows can take minutes, check between symbols
		select {
		case <-ctx.Done():
			of.Log().Info("Market research interrupted by shutdown",
				"opportunities_found", opportunities,
				"skipped", skipped,
				"symbols_remaining", len(of.symbols)-opportunities-skipped-errors,
			)
			return ctx.Err()
		default:
		}

		// Phase 5: Pre-screening before expensive LLM analysis
		if of.preScreener != nil {
			of.totalRuns++
			
			preScreen, err := of.preScreener.ShouldAnalyze(ctx, of.exchange, symbol)
			if err != nil {
				of.Log().Warn("Pre-screening failed, proceeding with analysis",
					"symbol", symbol,
					"error", err,
				)
			} else if !preScreen.ShouldAnalyze {
				of.skippedRuns++
				skipped++
				of.Log().Debug("Skipped analysis (pre-screening)",
					"symbol", symbol,
					"reason", preScreen.SkipReason,
					"confidence", preScreen.Confidence,
					"metrics", preScreen.Metrics,
				)
				continue
			} else {
				of.analyzedRuns++
				of.Log().Debug("Passed pre-screening",
					"symbol", symbol,
					"metrics", preScreen.Metrics,
				)
			}
		}

		found, err := of.analyzeSymbol(ctx, symbol)
		if err != nil {
			of.Log().Error("Failed to analyze symbol",
				"symbol", symbol,
				"error", err,
			)
			errors++
			continue
		}

		if found {
			opportunities++
		}
	}

	// Calculate skip rate for monitoring
	skipRate := 0.0
	if of.totalRuns > 0 {
		skipRate = float64(of.skippedRuns) / float64(of.totalRuns) * 100
	}

	of.Log().Debug("Market research complete",
		"opportunities_published", opportunities,
		"skipped", skipped,
		"errors", errors,
		"total_symbols", len(of.symbols),
		"skip_rate_pct", skipRate,
	)

	return nil
}

// analyzeSymbol runs the market research workflow for a single symbol
func (of *OpportunityFinder) analyzeSymbol(ctx context.Context, symbol string) (bool, error) {
	sessionID := uuid.New().String()
	userID := "system" // Global analysis (no specific user)

	of.log.Debug("Starting market research workflow",
		"symbol", symbol,
		"exchange", of.exchange,
		"session_id", sessionID,
	)

	// Build input prompt for workflow
	input := &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{Text: of.buildResearchPrompt(symbol)},
		},
	}

	// Run workflow through ADK runner
	runConfig := agent.RunConfig{
		StreamingMode: agent.StreamingModeNone, // No streaming for background jobs
	}

	// Track if opportunity was published
	opportunityPublished := false

	// Iterate through events from the workflow
	for event, err := range of.runner.Run(ctx, userID, sessionID, input, runConfig) {
		// Check for context cancellation between workflow events
		// Critical: ADK workflows can take minutes, need to exit gracefully on shutdown
		select {
		case <-ctx.Done():
			of.log.Info("Workflow interrupted by shutdown",
				"symbol", symbol,
				"session_id", sessionID,
				"opportunity_published", opportunityPublished,
			)
			return opportunityPublished, ctx.Err()
		default:
		}

		if err != nil {
			return false, errors.Wrap(err, "market research workflow failed")
		}

		if event == nil {
			continue
		}

		// Skip partial events
		if event.LLMResponse.Partial {
			continue
		}

		// Check for tool calls (publish_opportunity)
		if event.LLMResponse.Content != nil {
			for _, part := range event.LLMResponse.Content.Parts {
				if part.FunctionCall != nil && part.FunctionCall.Name == "publish_opportunity" {
					opportunityPublished = true
					of.log.Info("Opportunity signal published",
						"symbol", symbol,
						"session_id", sessionID,
					)
				}
			}
		}

		// Check if workflow is complete
		if event.TurnComplete && event.IsFinalResponse() {
			of.log.Debug("Market research workflow complete",
				"symbol", symbol,
				"session_id", sessionID,
				"opportunity_published", opportunityPublished,
			)
			break
		}
	}

	return opportunityPublished, nil
}

// buildResearchPrompt creates the input prompt for market research workflow
// Uses template system for centralized prompt management
func (of *OpportunityFinder) buildResearchPrompt(symbol string) string {
	// Render workflow input template with context
	prompt, err := of.templates.Render("workflows/market_research_input", map[string]interface{}{
		"Symbol":    symbol,
		"Exchange":  of.exchange,
		"Timestamp": time.Now().Format(time.RFC3339),
	})

	if err != nil {
		of.log.Errorf("Failed to render workflow input template: %v", err)
		// Fallback to basic prompt if template rendering fails
		return fmt.Sprintf("Analyze %s on %s exchange for trading opportunities. Use all available analyst agents, synthesize findings, and publish if high-quality opportunity found (>65%% confidence, clear levels, R:R >2:1).", symbol, of.exchange)
	}

	return prompt
}
