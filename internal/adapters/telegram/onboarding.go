package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/user"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/templates"
)

// OnboardingState represents the current stage of onboarding
type OnboardingState string

const (
	StateAwaitingCapital     OnboardingState = "awaiting_capital"
	StateAwaitingRiskProfile OnboardingState = "awaiting_risk_profile"
	StateAwaitingExchange    OnboardingState = "awaiting_exchange"
	StateProcessing          OnboardingState = "processing"
	StateComplete            OnboardingState = "complete"
	StateError               OnboardingState = "error"
)

// OnboardingSession stores the state of an ongoing onboarding process
type OnboardingSession struct {
	TelegramID        int64           `json:"telegram_id"`
	UserID            uuid.UUID       `json:"user_id"`
	State             OnboardingState `json:"state"`
	Capital           float64         `json:"capital"`
	RiskProfile       string          `json:"risk_profile"`
	ExchangeAccountID *uuid.UUID      `json:"exchange_account_id,omitempty"`
	PreferredAssets   []string        `json:"preferred_assets,omitempty"`
	StartedAt         time.Time       `json:"started_at"`
	LastInteractionAt time.Time       `json:"last_interaction_at"`
}

// OnboardingService manages the onboarding flow state machine
type OnboardingService struct {
	redis        *redis.Client
	bot          *Bot
	userRepo     user.Repository
	exchAcctRepo exchange_account.Repository
	orchestrator OnboardingOrchestrator
	templates    *templates.Registry
	log          *logger.Logger
}

// OnboardingOrchestrator executes the portfolio initialization workflow
type OnboardingOrchestrator interface {
	StartOnboarding(ctx context.Context, session *OnboardingSession) error
}

// NewOnboardingService creates a new onboarding service
func NewOnboardingService(
	redis *redis.Client,
	bot *Bot,
	userRepo user.Repository,
	exchAcctRepo exchange_account.Repository,
	orchestrator OnboardingOrchestrator,
	tmpl *templates.Registry,
	log *logger.Logger,
) *OnboardingService {
	if tmpl == nil {
		tmpl = templates.Get()
	}

	return &OnboardingService{
		redis:        redis,
		bot:          bot,
		userRepo:     userRepo,
		exchAcctRepo: exchAcctRepo,
		orchestrator: orchestrator,
		templates:    tmpl,
		log:          log.With("component", "onboarding_service"),
	}
}

// IsInOnboarding checks if a user is currently in onboarding
func (os *OnboardingService) IsInOnboarding(ctx context.Context, telegramID int64) (bool, error) {
	key := os.getSessionKey(telegramID)
	exists, err := os.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, errors.Wrap(err, "failed to check onboarding session")
	}
	return exists > 0, nil
}

// HandleMessage handles a message during onboarding flow
func (os *OnboardingService) HandleMessage(ctx context.Context, userID uuid.UUID, telegramID int64, text string) error {
	// Get current session
	session, err := os.getSession(ctx, telegramID)
	if err != nil {
		// No session exists, start new one
		return os.startNewSession(ctx, userID, telegramID, text)
	}

	// Process message based on current state
	switch session.State {
	case StateAwaitingCapital:
		return os.handleCapitalInput(ctx, session, text)

	case StateAwaitingRiskProfile:
		return os.handleRiskProfileInput(ctx, session, text)

	case StateAwaitingExchange:
		return os.handleExchangeSelection(ctx, session, text)

	case StateProcessing:
		os.bot.SendMessage(telegramID, "⏳ Your portfolio is being created... Please wait.")
		return nil

	case StateComplete:
		os.bot.SendMessage(telegramID, "✅ Your onboarding is already complete! Use /status to check your portfolio.")
		return os.deleteSession(ctx, telegramID)

	default:
		return os.deleteSession(ctx, telegramID)
	}
}

// startNewSession creates a new onboarding session
func (os *OnboardingService) startNewSession(ctx context.Context, userID uuid.UUID, telegramID int64, initialText string) error {
	session := &OnboardingSession{
		TelegramID:        telegramID,
		UserID:            userID,
		State:             StateAwaitingCapital,
		StartedAt:         time.Now(),
		LastInteractionAt: time.Now(),
	}

	// Try to parse initial text as capital amount
	if capital, err := os.parseCapitalAmount(initialText); err == nil && capital >= 100 {
		session.Capital = capital
		session.State = StateAwaitingRiskProfile

		if err := os.saveSession(ctx, session); err != nil {
			return err
		}

		return os.askRiskProfile(telegramID)
	}

	// Save session and ask for capital
	if err := os.saveSession(ctx, session); err != nil {
		return err
	}

	return os.askCapitalAmount(telegramID)
}

// handleCapitalInput processes capital amount input
func (os *OnboardingService) handleCapitalInput(ctx context.Context, session *OnboardingSession, text string) error {
	capital, err := os.parseCapitalAmount(text)
	if err != nil || capital < 100 {
		msg := "❌ Invalid amount. Please enter a number ≥ $100.\n\nExample: 1000"
		return os.bot.SendMessage(session.TelegramID, msg)
	}

	session.Capital = capital
	session.State = StateAwaitingRiskProfile
	session.LastInteractionAt = time.Now()

	if err := os.saveSession(ctx, session); err != nil {
		return err
	}

	return os.askRiskProfile(session.TelegramID)
}

// handleRiskProfileInput processes risk profile selection
func (os *OnboardingService) handleRiskProfileInput(ctx context.Context, session *OnboardingSession, text string) error {
	text = strings.ToLower(strings.TrimSpace(text))

	var riskProfile string
	switch text {
	case "1", "conservative", "cons":
		riskProfile = "conservative"
	case "2", "moderate", "mod":
		riskProfile = "moderate"
	case "3", "aggressive", "agg":
		riskProfile = "aggressive"
	default:
		msg, err := os.templates.Render("telegram/onboarding_risk_invalid", nil)
		if err != nil {
			return errors.Wrap(err, "failed to render onboarding_risk_invalid template")
		}
		return os.bot.SendMessage(session.TelegramID, msg)
	}

	session.RiskProfile = riskProfile
	session.State = StateAwaitingExchange
	session.LastInteractionAt = time.Now()

	if err := os.saveSession(ctx, session); err != nil {
		return err
	}

	return os.askExchangeAccount(ctx, session)
}

// handleExchangeSelection processes exchange account selection
func (os *OnboardingService) handleExchangeSelection(ctx context.Context, session *OnboardingSession, text string) error {
	text = strings.TrimSpace(text)

	// Check if user wants to use demo mode
	if text == "0" || strings.ToLower(text) == "demo" {
		return os.startOnboardingWithDemo(ctx, session)
	}

	// Parse exchange account selection
	// User can type account number or UUID
	accounts, err := os.exchAcctRepo.GetByUser(ctx, session.UserID)
	if err != nil {
		return errors.Wrap(err, "failed to get exchange accounts")
	}

	if len(accounts) == 0 {
		// No accounts available, suggest adding one
		msg := "❌ You don't have any exchange accounts connected.\n\nUse /add_exchange to connect your exchange first, or type *demo* to try demo mode."
		return os.bot.SendMessage(session.TelegramID, msg)
	}

	// Try to parse as index
	if idx, err := strconv.Atoi(text); err == nil && idx > 0 && idx <= len(accounts) {
		session.ExchangeAccountID = &accounts[idx-1].ID
	} else if accountID, err := uuid.Parse(text); err == nil {
		// Verify user owns this account
		for _, acc := range accounts {
			if acc.ID == accountID {
				session.ExchangeAccountID = &accountID
				break
			}
		}
	}

	if session.ExchangeAccountID == nil {
		msg := "❌ Invalid selection. Please choose a number from the list above."
		return os.bot.SendMessage(session.TelegramID, msg)
	}

	// Move to processing
	session.State = StateProcessing
	session.LastInteractionAt = time.Now()

	if err := os.saveSession(ctx, session); err != nil {
		return err
	}

	// Notify user
	os.bot.SendMessage(session.TelegramID, "⏳ Creating your portfolio... This may take 1-2 minutes.")

	// Start portfolio initialization workflow
	if os.orchestrator != nil {
		if err := os.orchestrator.StartOnboarding(ctx, session); err != nil {
			os.log.Error("Onboarding workflow failed", "error", err)
			session.State = StateError
			os.saveSession(ctx, session)
			os.bot.SendMessage(session.TelegramID, fmt.Sprintf("❌ Portfolio creation failed: %v\n\nPlease try again with /invest", err))
			os.deleteSession(ctx, session.TelegramID)
			return err
		}
	} else {
		os.log.Warn("Orchestrator not configured, cannot start workflow")
		os.bot.SendMessage(session.TelegramID, "❌ Onboarding service not available")
		return os.deleteSession(ctx, session.TelegramID)
	}

	// Mark complete and cleanup
	os.bot.SendMessage(session.TelegramID, "✅ Your portfolio has been created! Use /status to view it.")
	return os.deleteSession(ctx, session.TelegramID)
}

// startOnboardingWithDemo starts onboarding in demo mode (no real exchange)
func (os *OnboardingService) startOnboardingWithDemo(ctx context.Context, session *OnboardingSession) error {
	// TODO: Implement demo mode
	msg := "🚧 Demo mode not yet implemented.\n\nPlease connect your exchange with /add_exchange"
	return os.bot.SendMessage(session.TelegramID, msg)
}

// Helper methods for prompting user

func (os *OnboardingService) askCapitalAmount(telegramID int64) error {
	msg, err := os.templates.Render("telegram/onboarding_capital", nil)
	if err != nil {
		return errors.Wrap(err, "failed to render onboarding_capital template")
	}

	return os.bot.SendMessage(telegramID, msg)
}

func (os *OnboardingService) askRiskProfile(telegramID int64) error {
	msg, err := os.templates.Render("telegram/onboarding_risk_profile", nil)
	if err != nil {
		return errors.Wrap(err, "failed to render onboarding_risk_profile template")
	}

	return os.bot.SendMessage(telegramID, msg)
}

func (os *OnboardingService) askExchangeAccount(ctx context.Context, session *OnboardingSession) error {
	// Get user's exchange accounts
	accounts, err := os.exchAcctRepo.GetByUser(ctx, session.UserID)
	if err != nil {
		return errors.Wrap(err, "failed to get exchange accounts")
	}

	if len(accounts) == 0 {
		msg, err := os.templates.Render("telegram/onboarding_exchange_empty", nil)
		if err != nil {
			return errors.Wrap(err, "failed to render onboarding_exchange_empty template")
		}
		return os.bot.SendMessage(session.TelegramID, msg)
	}

	// Render exchange selection template
	data := map[string]interface{}{
		"Accounts": accounts,
	}

	msg, err := os.templates.Render("telegram/onboarding_exchange_select", data)
	if err != nil {
		return errors.Wrap(err, "failed to render onboarding_exchange_select template")
	}

	return os.bot.SendMessage(session.TelegramID, msg)
}

// parseCapitalAmount parses capital amount from text
func (os *OnboardingService) parseCapitalAmount(text string) (float64, error) {
	// Remove common symbols
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "$", "")
	text = strings.ReplaceAll(text, ",", "")
	text = strings.ReplaceAll(text, " ", "")

	amount, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, errors.Wrap(err, "invalid number format")
	}

	return amount, nil
}

// Redis session management

func (os *OnboardingService) getSessionKey(telegramID int64) string {
	return fmt.Sprintf("onboarding:%d", telegramID)
}

func (os *OnboardingService) getSession(ctx context.Context, telegramID int64) (*OnboardingSession, error) {
	key := os.getSessionKey(telegramID)

	data, err := os.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, errors.Wrapf(errors.ErrNotFound, "no onboarding session found")
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get onboarding session")
	}

	var session OnboardingSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal session")
	}

	return &session, nil
}

func (os *OnboardingService) saveSession(ctx context.Context, session *OnboardingSession) error {
	key := os.getSessionKey(session.TelegramID)

	data, err := json.Marshal(session)
	if err != nil {
		return errors.Wrap(err, "failed to marshal session")
	}

	// Set with 30 minute TTL
	if err := os.redis.Set(ctx, key, data, 30*time.Minute).Err(); err != nil {
		return errors.Wrap(err, "failed to save session")
	}

	return nil
}

func (os *OnboardingService) deleteSession(ctx context.Context, telegramID int64) error {
	key := os.getSessionKey(telegramID)
	return os.redis.Del(ctx, key).Err()
}

// CleanupExpiredSessions removes expired sessions (called periodically)
func (os *OnboardingService) CleanupExpiredSessions(ctx context.Context) error {
	// Redis TTL handles this automatically
	return nil
}
