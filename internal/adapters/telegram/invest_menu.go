package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"

	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/menu_session"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// ExchangeService defines interface for exchange operations
type ExchangeService interface {
	GetUserAccounts(ctx context.Context, userID uuid.UUID) ([]*exchange_account.ExchangeAccount, error)
	GetAccount(ctx context.Context, accountID uuid.UUID) (*exchange_account.ExchangeAccount, error)
}

// InvestMenuService handles /invest flow using MenuNavigator
// Much cleaner with auto-parameter handling!
type InvestMenuService struct {
	menuNav         *MenuNavigator
	exchangeService ExchangeService
	userService     UserService
	orchestrator    OnboardingOrchestrator
	log             *logger.Logger
}

// NewInvestMenuService creates invest menu service (v2)
func NewInvestMenuService(
	menuNav *MenuNavigator,
	exchangeService ExchangeService,
	userService UserService,
	orchestrator OnboardingOrchestrator,
	log *logger.Logger,
) *InvestMenuService {
	return &InvestMenuService{
		menuNav:         menuNav,
		exchangeService: exchangeService,
		userService:     userService,
		orchestrator:    orchestrator,
		log:             log.With("component", "invest_menu"),
	}
}

// StartInvest starts invest flow with exchange selection
func (ims *InvestMenuService) StartInvest(ctx context.Context, userID uuid.UUID, telegramID int64) error {
	initialData := map[string]interface{}{
		"user_id": userID.String(),
	}

	screen := ims.buildExchangeSelectionScreen()
	return ims.menuNav.StartMenu(ctx, telegramID, screen, initialData)
}

// HandleCallback processes invest menu callbacks
func (ims *InvestMenuService) HandleCallback(ctx context.Context, userID uuid.UUID, telegramID int64, messageID int, data string) error {
	// Handle cancel button
	if data == "cancel" {
		_ = ims.menuNav.EndMenu(ctx, telegramID)
		return nil
	}

	screens := ims.getScreens()
	return ims.menuNav.HandleCallback(ctx, telegramID, messageID, data, screens)
}

// HandleMessage processes text messages (amount input)
func (ims *InvestMenuService) HandleMessage(ctx context.Context, userID uuid.UUID, telegramID int64, text string) error {
	session, err := ims.menuNav.GetSession(ctx, telegramID)
	if err != nil {
		return errors.Wrap(err, "no active invest session")
	}

	// Parse amount
	amount, err := parseAmount(text)
	if err != nil || amount < 100 {
		return ims.menuNav.bot.SendMessage(telegramID, "❌ Invalid amount. Please enter a number ≥ $100.\n\nExample: 1000")
	}

	// Get account_id from session (saved as "a")
	accountIDStr, ok := session.GetString("a")
	if !ok {
		return fmt.Errorf("account_id (key 'a') not found in session")
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return errors.Wrap(err, "invalid account_id")
	}

	// End menu
	_ = ims.menuNav.EndMenu(ctx, telegramID)

	// Notify user
	_ = ims.menuNav.bot.SendMessage(telegramID, "⏳ Creating your portfolio... This may take 1-2 minutes.")

	// Start onboarding
	if ims.orchestrator != nil {
		onboardingSession := &OnboardingSession{
			TelegramID:        telegramID,
			UserID:            userID,
			Capital:           amount,
			ExchangeAccountID: &accountID,
			RiskProfile:       "moderate",
		}

		if err := ims.orchestrator.StartOnboarding(ctx, onboardingSession); err != nil {
			ims.log.Errorw("Onboarding workflow failed", "error", err)
			return ims.menuNav.bot.SendMessage(telegramID, fmt.Sprintf("❌ Portfolio creation failed: %v\n\nPlease try again with /invest", err))
		}
	}

	return ims.menuNav.bot.SendMessage(telegramID, "✅ Your portfolio has been created! Use /status to view it.")
}

// IsInInvest checks if user is in invest flow
func (ims *InvestMenuService) IsInInvest(ctx context.Context, telegramID int64) (bool, error) {
	return ims.menuNav.IsInMenu(ctx, telegramID)
}

// MenuHandler interface implementation

// GetScreenIDs returns all screen IDs this handler owns (for MenuRegistry)
func (ims *InvestMenuService) GetScreenIDs() []string {
	return []string{"sel", "amt", "cancel"}
}

// IsInMenu checks if user has active session (alias for IsInInvest)
func (ims *InvestMenuService) IsInMenu(ctx context.Context, telegramID int64) (bool, error) {
	return ims.IsInInvest(ctx, telegramID)
}

// ClearSession clears invest session (for restart)
func (ims *InvestMenuService) ClearSession(ctx context.Context, telegramID int64) error {
	return ims.menuNav.EndMenu(ctx, telegramID)
}

// getScreens returns all invest screens
func (ims *InvestMenuService) getScreens() map[string]*Screen {
	return map[string]*Screen{
		"sel": ims.buildExchangeSelectionScreen(), // "sel" = selection screen
		"amt": ims.buildEnterAmountScreen(),       // "amt" = amount input screen
	}
}

// Screen builders

func (ims *InvestMenuService) buildExchangeSelectionScreen() *Screen {
	return &Screen{
		ID:       "sel", // Short ID: "sel" = selection
		Template: "telegram/invest/select_exchange",
		OnEnter:  nil, // No special action on enter
		Data: func(ctx context.Context, session *menu_session.Session) (map[string]interface{}, error) {
			userIDStr, ok := session.GetString("user_id")
			if !ok {
				return nil, fmt.Errorf("user_id not found in session")
			}

			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				return nil, errors.Wrap(err, "invalid user_id")
			}

			accounts, err := ims.exchangeService.GetUserAccounts(ctx, userID)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get exchange accounts")
			}

			// Filter active accounts
			type ExchangeInfo struct {
				StatusEmoji string
				Exchange    string
				Label       string
			}

			activeAccounts := make([]ExchangeInfo, 0)
			for _, account := range accounts {
				if account.IsActive {
					activeAccounts = append(activeAccounts, ExchangeInfo{
						StatusEmoji: "✅",
						Exchange:    strings.Title(string(account.Exchange)),
						Label:       account.Label,
					})
				}
			}

			return map[string]interface{}{
				"Exchanges": activeAccounts,
			}, nil
		},
		Keyboard: func(ctx context.Context, nav *MenuNavigator, session *menu_session.Session) (tgbotapi.InlineKeyboardMarkup, error) {
			userIDStr, ok := session.GetString("user_id")
			if !ok {
				return tgbotapi.InlineKeyboardMarkup{}, fmt.Errorf("user_id not found in session")
			}

			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				return tgbotapi.InlineKeyboardMarkup{}, errors.Wrap(err, "invalid user_id")
			}

			accounts, err := ims.exchangeService.GetUserAccounts(ctx, userID)
			if err != nil {
				return tgbotapi.InlineKeyboardMarkup{}, errors.Wrap(err, "failed to get exchange accounts")
			}

			var rows [][]tgbotapi.InlineKeyboardButton
			for _, account := range accounts {
				if !account.IsActive {
					continue
				}

				buttonText := fmt.Sprintf("📊 %s - %s", strings.Title(string(account.Exchange)), account.Label)
				// Use short screen name "amt" and short key "a" to save space (Telegram limit: 64 bytes)
				callbackData := nav.MakeCallback("amt", "a", account.ID.String())
				row := tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData),
				)
				rows = append(rows, row)
			}

			// Cancel button
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", "cancel"),
			))

			return tgbotapi.NewInlineKeyboardMarkup(rows...), nil
		},
	}
}

func (ims *InvestMenuService) buildEnterAmountScreen() *Screen {
	return &Screen{
		ID:       "amt", // Short ID
		Template: "telegram/invest/enter_amount",
		OnEnter: func(ctx context.Context, nav *MenuNavigator, params map[string]string) error {
			// Parameters already auto-saved to session by MenuNavigator!
			// Nothing to do here - just log for debugging
			ims.log.Debugw("Entered amount screen",
				"params", params,
			)
			return nil
		},
		Data: nil, // No dynamic data
		Keyboard: func(ctx context.Context, nav *MenuNavigator, session *menu_session.Session) (tgbotapi.InlineKeyboardMarkup, error) {
			// Back button will be added automatically by MenuNavigator
			return tgbotapi.InlineKeyboardMarkup{}, nil
		},
	}
}

// parseAmount parses capital amount from text
func parseAmount(text string) (float64, error) {
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
