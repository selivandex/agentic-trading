package onboarding

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

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
	workflowFactory       *workflows.Factory
	sessionService        session.Service
	templates             *templates.Registry
	userRepo              user.Repository
	exchAcctRepo          exchange_account.Repository
	strategyService       *strategyservice.Service
	notificationPublisher NotificationPublisher
	log                   *logger.Logger
}

// NotificationPublisher interface for publishing notifications (DI for testability)
type NotificationPublisher interface {
	PublishInvestmentAccepted(ctx context.Context, userID string, telegramID int64, capital float64, riskProfile, exchange string) error
	PublishPortfolioCreated(ctx context.Context, userID, strategyID, strategyName string, telegramID int64, invested float64, positionsCount int32) error
}

// NewService creates a new onboarding orchestration service
func NewService(
	workflowFactory *workflows.Factory,
	sessionService session.Service,
	tmpl *templates.Registry,
	userRepo user.Repository,
	exchAcctRepo exchange_account.Repository,
	strategyService *strategyservice.Service,
	notificationPublisher NotificationPublisher,
	log *logger.Logger,
) *Service {
	if tmpl == nil {
		tmpl = templates.Get()
	}

	return &Service{
		workflowFactory:       workflowFactory,
		sessionService:        sessionService,
		templates:             tmpl,
		userRepo:              userRepo,
		exchAcctRepo:          exchAcctRepo,
		strategyService:       strategyService,
		notificationPublisher: notificationPublisher,
		log:                   log.With("component", "onboarding_orchestrator"),
	}
}

// StartOnboarding executes the PortfolioArchitect workflow to initialize user's portfolio
func (s *Service) StartOnboarding(ctx context.Context, onboardingSession *telegram.OnboardingSession) error {
	s.log.Infow("Starting portfolio initialization workflow",
		"user_id", onboardingSession.UserID,
		"capital", onboardingSession.Capital,
		"risk_profile", onboardingSession.RiskProfile,
	)

	// Step 1: Get or create strategy
	var createdStrategy *strategy.Strategy
	
	if onboardingSession.StrategyID != nil {
		// Use pre-created strategy from invest flow
		s.log.Infow("Using pre-created strategy",
			"strategy_id", *onboardingSession.StrategyID,
			"user_id", onboardingSession.UserID,
		)
		
		createdStrategy, err = s.strategyService.GetStrategyByID(ctx, *onboardingSession.StrategyID)
		if err != nil {
			return errors.Wrap(err, "failed to get pre-created strategy")
		}
	} else {
		// Fallback: Create strategy if not provided (backward compatibility)
		strategyName := fmt.Sprintf("Portfolio %s", time.Now().Format("2006-01-02"))

		// Map string risk profile to strategy.RiskTolerance
		var riskTolerance strategy.RiskTolerance
		switch onboardingSession.RiskProfile {
		case "conservative":
			riskTolerance = strategy.RiskConservative
		case "aggressive":
			riskTolerance = strategy.RiskAggressive
		default:
			riskTolerance = strategy.RiskModerate
		}

		createdStrategy, err = s.strategyService.CreateStrategy(ctx, strategyservice.CreateStrategyParams{
			UserID:           onboardingSession.UserID,
			Name:             strategyName,
			Description:      fmt.Sprintf("Automated portfolio with %s risk profile", onboardingSession.RiskProfile),
			AllocatedCapital: decimal.NewFromFloat(onboardingSession.Capital),
			RiskTolerance:    riskTolerance,
			ReasoningLog:     nil, // Will be updated after workflow completes
		})
		if err != nil {
			return errors.Wrap(err, "failed to create strategy")
		}

		s.log.Infow("Strategy created (fallback)",
			"strategy_id", createdStrategy.ID,
			"allocated_capital", createdStrategy.AllocatedCapital,
		)
	}

	// Get exchange name for notification (sanitize for protobuf)
	var exchangeName string
	if onboardingSession.ExchangeAccountID != nil {
		exchAcct, err := s.exchAcctRepo.GetByID(ctx, *onboardingSession.ExchangeAccountID)
		if err == nil {
			// Sanitize exchange name to ensure valid UTF-8 for protobuf
			exchangeName = sanitizeUTF8(string(exchAcct.Exchange))
		} else {
			s.log.Warnw("Failed to get exchange account", "error", err)
		}
	}
	if exchangeName == "" {
		exchangeName = "demo"
	}

	// Notify user immediately that investment is accepted
	if err := s.notificationPublisher.PublishInvestmentAccepted(
		ctx,
		onboardingSession.UserID.String(),
		onboardingSession.TelegramID,
		onboardingSession.Capital,
		sanitizeUTF8(onboardingSession.RiskProfile), // Sanitize for protobuf
		exchangeName,
	); err != nil {
		s.log.Errorw("Failed to publish investment accepted notification", "error", err)
		// Continue anyway - notification is not critical
	}

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

	// Build input prompt for PortfolioArchitect
	input, err := s.buildArchitectPrompt(onboardingSession, exchangeName)
	if err != nil {
		return errors.Wrap(err, "failed to build architect prompt")
	}

	// Run workflow
	userID := onboardingSession.UserID.String()
	appName := fmt.Sprintf("prometheus_onboarding_%s", onboardingSession.UserID.String())

	// Create ADK session before running workflow (let it generate session ID)
	createResp, err := s.sessionService.Create(ctx, &session.CreateRequest{
		AppName:   appName,
		UserID:    userID,
		SessionID: "", // Empty = auto-generate
		State:     nil,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create session")
	}

	// Use the generated session ID
	sessionID := createResp.Session.ID()

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

	// Notify user that portfolio was created successfully
	if err := s.notificationPublisher.PublishPortfolioCreated(
		ctx,
		onboardingSession.UserID.String(),
		createdStrategy.ID.String(),
		sanitizeUTF8(createdStrategy.Name), // Sanitize for protobuf
		onboardingSession.TelegramID,
		onboardingSession.Capital,
		int32(tradesExecuted), // Use actual trades count
	); err != nil {
		s.log.Errorw("Failed to publish portfolio created notification", "error", err)
		// Continue anyway - notification is not critical
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

// sanitizeUTF8 removes invalid UTF-8 characters to prevent protobuf marshaling errors
func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	// Remove invalid UTF-8 bytes
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r != utf8.RuneError {
			b.WriteRune(r)
		}
	}
	return b.String()
}
