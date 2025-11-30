package onboarding

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	telegram "prometheus/internal/adapters/telegram"
	"prometheus/internal/agents/workflows"
	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/strategy"
	"prometheus/internal/domain/user"
	strategyservice "prometheus/internal/services/strategy"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/templates"
)

// Service orchestrates the portfolio initialization workflow during onboarding
type Service struct {
	workflowFactory *workflows.Factory
	sessionService  session.Service
	templates       *templates.Registry
	userRepo        user.Repository
	exchAcctRepo    exchange_account.Repository
	strategyService StrategyService
	log             *logger.Logger
}

// NewService creates a new onboarding orchestration service
func NewService(
	workflowFactory *workflows.Factory,
	sessionService session.Service,
	tmpl *templates.Registry,
	userRepo user.Repository,
	exchAcctRepo exchange_account.Repository,
	strategyService StrategyService,
	log *logger.Logger,
) *Service {
	if tmpl == nil {
		tmpl = templates.Get()
	}

	return &Service{
		workflowFactory: workflowFactory,
		sessionService:  sessionService,
		templates:       tmpl,
		userRepo:        userRepo,
		exchAcctRepo:    exchAcctRepo,
		strategyService: strategyService,
		log:             log.With("component", "onboarding_orchestrator"),
	}
}

// StartOnboarding executes the PortfolioArchitect workflow to initialize user's portfolio
func (s *Service) StartOnboarding(ctx context.Context, onboardingSession *telegram.OnboardingSession) error {
	s.log.Infow("Starting portfolio initialization workflow",
		"user_id", onboardingSession.UserID,
		"capital", onboardingSession.Capital,
		"risk_profile", onboardingSession.RiskProfile,
	)

	// Step 1: Create strategy to track allocated capital and transactions
	strategyName := fmt.Sprintf("Portfolio %s", time.Now().Format("2006-01-02"))
	strategy, err := s.strategyService.CreateStrategy(ctx, CreateStrategyParams{
		UserID:           onboardingSession.UserID,
		Name:             strategyName,
		Description:      fmt.Sprintf("Automated portfolio with %s risk profile", onboardingSession.RiskProfile),
		AllocatedCapital: decimal.NewFromFloat(onboardingSession.Capital),
		RiskTolerance:    onboardingSession.RiskProfile,
		ReasoningLog:     nil, // Will be updated after workflow completes
	})
	if err != nil {
		return errors.Wrap(err, "failed to create strategy")
	}

	s.log.Infow("Strategy created",
		"strategy_id", strategy.ID,
		"allocated_capital", strategy.AllocatedCapital,
	)

	// Step 2: Create portfolio initialization workflow (PortfolioArchitect agent)
	workflow, err := s.workflowFactory.CreatePortfolioInitializationWorkflow()
	if err != nil {
		return errors.Wrap(err, "failed to create portfolio initialization workflow")
	}

	// Create ADK runner
	runnerInstance, err := runner.New(runner.Config{
		AppName:        fmt.Sprintf("prometheus_onboarding_%s", onboardingSession.UserID.String()),
		Agent:          workflow,
		SessionService: s.sessionService,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create ADK runner")
	}

	// Get exchange account details
	var exchangeName string
	if onboardingSession.ExchangeAccountID != nil {
		exchAcct, err := s.exchAcctRepo.GetByID(ctx, *onboardingSession.ExchangeAccountID)
		if err != nil {
			return errors.Wrap(err, "failed to get exchange account")
		}
		exchangeName = string(exchAcct.Exchange)
	} else {
		exchangeName = "demo"
	}

	// Build input prompt for PortfolioArchitect
	input, err := s.buildArchitectPrompt(onboardingSession, exchangeName)
	if err != nil {
		return errors.Wrap(err, "failed to build architect prompt")
	}

	// Run workflow
	sessionID := uuid.New().String()
	userID := onboardingSession.UserID.String()

	// Create ADK session before running workflow
	appName := fmt.Sprintf("prometheus_onboarding_%s", onboardingSession.UserID.String())
	_, err = s.sessionService.Create(ctx, &session.CreateRequest{
		AppName:   appName,
		UserID:    userID,
		SessionID: sessionID,
		State:     nil, // Empty initial state
	})
	if err != nil {
		return errors.Wrap(err, "failed to create session")
	}

	runConfig := agent.RunConfig{
		StreamingMode: agent.StreamingModeNone, // No streaming for background workflow
	}

	// Track execution
	startTime := time.Now()
	tradesExecuted := 0

	// Execute workflow and monitor events
	for event, err := range runnerInstance.Run(ctx, userID, sessionID, input, runConfig) {
		if err != nil {
			return errors.Wrap(err, "portfolio initialization workflow failed")
		}

		if event == nil || event.LLMResponse.Partial {
			continue
		}

		// Check for trade execution
		if event.LLMResponse.Content != nil {
			for _, part := range event.LLMResponse.Content.Parts {
				if part.FunctionCall != nil && (part.FunctionCall.Name == "execute_trade" || part.FunctionCall.Name == "place_order") {
					tradesExecuted++
					s.log.Infow("Portfolio allocation: trade executed",
						"user_id", onboardingSession.UserID,
						"session_id", sessionID,
						"trades_executed", tradesExecuted,
					)
				}
			}
		}

		// Check if workflow is complete
		if event.TurnComplete && event.IsFinalResponse() {
			duration := time.Since(startTime)
			s.log.Infow("Portfolio initialization complete",
				"user_id", onboardingSession.UserID,
				"session_id", sessionID,
				"trades_executed", tradesExecuted,
				"duration", duration,
			)
			break
		}
	}

	if tradesExecuted == 0 {
		s.log.Warnw("Portfolio initialization completed but no trades were executed",
			"user_id", onboardingSession.UserID,
			"session_id", sessionID,
		)
	}

	return nil
}

// buildArchitectPrompt creates the input prompt for PortfolioArchitect agent
func (s *Service) buildArchitectPrompt(onboardingSession *telegram.OnboardingSession, exchangeName string) (*genai.Content, error) {
	// Prepare data for template
	data := map[string]interface{}{
		"Capital":     onboardingSession.Capital,
		"RiskProfile": onboardingSession.RiskProfile,
		"Exchange":    exchangeName,
		"UserID":      onboardingSession.UserID.String(),
		"Timestamp":   time.Now().Format(time.RFC3339),
	}

	// Render template
	promptText, err := s.templates.Render("prompts/workflows/portfolio_initialization_input", data)
	if err != nil {
		// Fallback to simple prompt if template not found
		s.log.Warnw("Template not found, using fallback prompt", "error", err)
		promptText = fmt.Sprintf(`Design initial portfolio allocation for a new client.

Client Profile:
- Capital: $%.2f
- Risk Profile: %s
- Exchange: %s

Your Task:
1. Call analysis tools to assess current market conditions
2. Design diversified portfolio allocation based on risk profile
3. Calculate position sizes for each asset
4. Validate each position via pre_trade_check tool
5. Execute portfolio via execute_trade or place_order tool
6. Save portfolio strategy to memory

Use tools systematically. Be thorough but decisive.`,
			onboardingSession.Capital,
			onboardingSession.RiskProfile,
			exchangeName,
		)
	}

	return &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{Text: promptText},
		},
	}, nil
}
