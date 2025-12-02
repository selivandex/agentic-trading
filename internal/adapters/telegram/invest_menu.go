package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/user"
	strategyservice "prometheus/internal/services/strategy"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/telegram"
)

// ExchangeService defines interface for exchange operations
type ExchangeService interface {
	GetActiveUserAccounts(ctx context.Context, userID uuid.UUID) ([]*exchange_account.ExchangeAccount, error)
	GetAccount(ctx context.Context, accountID uuid.UUID) (*exchange_account.ExchangeAccount, error)
}

// InvestmentValidator validates investment operations against user limits
type InvestmentValidator interface {
	ValidateInvestment(ctx context.Context, usr *user.User, requestedCapital float64) (*strategyservice.ValidationResult, error)
}

// JobPublisher interface for publishing portfolio initialization jobs
type JobPublisher interface {
	PublishPortfolioInitializationJob(
		ctx context.Context,
		userID, strategyID string,
		telegramID int64,
		capital float64,
		exchangeAccountID, riskProfile, marketType string,
	) error
}

// InvestFlowKeys defines parameter keys for callback data
type InvestFlowKeys struct {
	Account     string // exchange account ID
	MarketType  string // market type (spot/futures)
	RiskProfile string // risk profile
}

var investKeys = InvestFlowKeys{
	Account:     "account_id",
	MarketType:  "market_type",
	RiskProfile: "risk_profile",
}

// RiskProfileOption represents a risk profile selection option
type RiskProfileOption struct {
	Emoji       string
	Label       string
	Value       string
	Description string
	Features    []string
}

// Implement telegram.MenuOption interface
func (r RiskProfileOption) GetValue() string { return r.Value }
func (r RiskProfileOption) GetLabel() string { return r.Label }
func (r RiskProfileOption) GetEmoji() string { return r.Emoji }

// MarketTypeOption represents a market type selection option
type MarketTypeOption struct {
	Emoji       string
	Label       string
	Value       string
	Description string
	Features    []string
}

// Implement telegram.MenuOption interface
func (m MarketTypeOption) GetValue() string { return m.Value }
func (m MarketTypeOption) GetLabel() string { return m.Label }
func (m MarketTypeOption) GetEmoji() string { return m.Emoji }

var (
	// riskProfileOptions defines available risk profiles
	riskProfileOptions = []RiskProfileOption{
		{
			Emoji:       "🛡️",
			Label:       "Conservative",
			Value:       "conservative",
			Description: "Lower risk, stable returns",
			Features: []string{
				"Focus on major coins (BTC, ETH)",
				"Smaller position sizes",
				"Higher cash reserves",
			},
		},
		{
			Emoji:       "⚖️",
			Label:       "Moderate",
			Value:       "moderate",
			Description: "Balanced approach",
			Features: []string{
				"Mix of major and mid-cap coins",
				"Moderate position sizes",
				"Balanced risk/reward",
			},
		},
		{
			Emoji:       "🚀",
			Label:       "Aggressive",
			Value:       "aggressive",
			Description: "Higher returns potential",
			Features: []string{
				"Includes smaller cap opportunities",
				"Larger position sizes",
				"Higher volatility tolerance",
			},
		},
	}

	// marketTypeOptions defines available market types
	marketTypeOptions = []MarketTypeOption{
		{
			Emoji:       "📊",
			Label:       "Spot Trading",
			Value:       "spot",
			Description: "Trade actual cryptocurrencies",
			Features: []string{
				"No leverage, lower risk",
				"Own the underlying asset",
			},
		},
		{
			Emoji:       "⚡",
			Label:       "Futures Trading",
			Value:       "futures",
			Description: "Trade contracts with leverage",
			Features: []string{
				"Higher risk, higher potential returns",
				"Long/short positions available",
			},
		},
	}
)

// InvestMenuService handles /invest flow using MenuNavigator framework
type InvestMenuService struct {
	menuNav             *telegram.MenuNavigator
	exchangeService     ExchangeService
	userService         UserService
	jobPublisher        JobPublisher
	investmentValidator InvestmentValidator
	log                 *logger.Logger
}

// NewInvestMenuService creates invest menu service
func NewInvestMenuService(
	menuNav *telegram.MenuNavigator,
	exchangeService ExchangeService,
	userService UserService,
	jobPublisher JobPublisher,
	investmentValidator InvestmentValidator,
	log *logger.Logger,
) *InvestMenuService {
	return &InvestMenuService{
		menuNav:             menuNav,
		exchangeService:     exchangeService,
		userService:         userService,
		jobPublisher:        jobPublisher,
		investmentValidator: investmentValidator,
		log:                 log.With("component", "invest_menu"),
	}
}

// StartInvest starts invest flow with exchange selection
func (ims *InvestMenuService) StartInvest(ctx context.Context, userID uuid.UUID, telegramID int64) error {
	// Get only active exchange accounts from DB
	accounts, err := ims.exchangeService.GetActiveUserAccounts(ctx, userID)
	if err != nil {
		ims.log.Errorw("Failed to get active user accounts", "error", err, "user_id", userID)
		return errors.Wrap(err, "failed to get active user accounts")
	}

	// If no active accounts, send helpful message using template
	if len(accounts) == 0 {
		ims.log.Infow("User has no active exchange accounts",
			"user_id", userID,
			"telegram_id", telegramID,
		)

		// Render template for "no active exchanges" message
		msg, err := ims.menuNav.RenderTemplate("invest/no_active_exchanges", nil)
		if err != nil {
			ims.log.Errorw("Failed to render no_active_exchanges template", "error", err)
			// Fallback to simple message
			msg = "❌ No exchange accounts found. Use /settings to add an exchange account."
		}

		return ims.menuNav.GetBot().SendMessage(telegramID, msg)
	}

	ims.log.Debugw("Starting invest flow",
		"user_id", userID,
		"telegram_id", telegramID,
		"active_accounts", len(accounts),
	)

	initialData := map[string]interface{}{
		"user_id":    userID.String(),
		"_menu_type": "invest", // Mark session as invest menu
	}

	screen := ims.buildExchangeSelectionScreen()
	return ims.menuNav.StartMenu(ctx, telegramID, screen, initialData)
}

// HandleCallback processes invest menu callbacks
func (ims *InvestMenuService) HandleCallback(ctx context.Context, userID interface{}, telegramID int64, messageID int, data string) error {
	// Note: "cancel" and "back" are handled by MenuRegistry before reaching here

	screens := ims.getScreens()
	return ims.menuNav.HandleCallback(ctx, telegramID, messageID, data, screens)
}

// HandleMessage processes text messages (amount input)
func (ims *InvestMenuService) HandleMessage(ctx context.Context, userID interface{}, telegramID int64, messageID int, text string) error {
	session, err := ims.menuNav.GetSession(ctx, telegramID)
	if err != nil {
		return errors.Wrap(err, "no active invest session")
	}

	// Parse amount
	amount, err := parseAmount(text)
	if err != nil {
		// Return error - framework will handle sending error message
		return errors.New("❌ Invalid amount. Please enter a valid number.\n\nExample: 1000")
	}

	// Get user to validate against their limits
	usr, err := ims.userService.GetByTelegramID(ctx, telegramID)
	if err != nil {
		ims.log.Errorw("Failed to get user for validation", "error", err, "telegram_id", telegramID)
		return errors.New("❌ Failed to process your request. Please try /cancel and start over.")
	}

	// Validate investment amount against user limits and profile
	if ims.investmentValidator != nil {
		validation, err := ims.investmentValidator.ValidateInvestment(ctx, usr, amount)
		if err != nil {
			ims.log.Errorw("Investment validation error", "error", err, "user_id", usr.ID, "amount", amount)
			return errors.New("❌ Failed to validate investment. Please try a different amount.")
		}

		if !validation.Allowed {
			ims.log.Infow("Investment rejected by validation",
				"user_id", usr.ID,
				"amount", amount,
				"reason", validation.Reason,
			)

			// Return error - user can try different amount without restarting
			errorMsg := fmt.Sprintf("❌ %s", validation.Reason)
			if validation.MaxAllowed > 0 {
				errorMsg += fmt.Sprintf("\n\n💡 Maximum you can invest: $%.2f", validation.MaxAllowed)
			}
			errorMsg += "\n\nPlease enter a different amount:"

			return errors.New(errorMsg)
		}

		ims.log.Debugw("Investment validation passed",
			"user_id", usr.ID,
			"amount", amount,
			"current_exposure", validation.CurrentExposure,
		)
	}

	// Save amount to session
	session.SetData("amount", amount)

	// Save session to storage before finalizing
	// Note: menuNav needs access to bot - will fix architecture
	return ims.finalizeInvestment(ctx, session)
}

// GetMenuType returns menu type identifier
func (ims *InvestMenuService) GetMenuType() string {
	return "invest"
}

// EndMenu ends the invest menu session
func (ims *InvestMenuService) EndMenu(ctx context.Context, telegramID int64) error {
	return ims.menuNav.EndMenu(ctx, telegramID)
}

// GetScreenIDs returns all screen IDs this handler owns (for MenuRegistry)
func (ims *InvestMenuService) GetScreenIDs() []string {
	return []string{"sel", "mkt", "risk", "amt"}
}

// getScreens returns all invest screens (using framework builders - DRY!)
func (ims *InvestMenuService) getScreens() map[string]*telegram.Screen {
	return map[string]*telegram.Screen{
		"sel":  ims.buildExchangeSelectionScreen(),
		"mkt":  ims.buildMarketTypeSelectionScreen(),
		"risk": ims.buildRiskSelectionScreen(),
		"amt":  ims.buildEnterAmountScreen(),
	}
}

// buildExchangeSelectionScreen builds exchange selection screen using framework
func (ims *InvestMenuService) buildExchangeSelectionScreen() *telegram.Screen {
	return ims.menuNav.BuildListScreen(telegram.ListScreenConfig{
		ID:           "sel",
		Template:     "invest/select_exchange",
		NextScreenID: "mkt",
		ParamKey:     investKeys.Account,
		ItemsKey:     "Exchanges",
		Items: func(ctx context.Context, session telegram.Session) ([]telegram.ListItem, error) {
			userIDStr, ok := session.GetString("user_id")
			if !ok {
				return nil, errors.New("user_id not found in session")
			}

			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				return nil, errors.Wrap(err, "invalid user_id")
			}

			accounts, err := ims.exchangeService.GetActiveUserAccounts(ctx, userID)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get active exchange accounts")
			}

			ims.log.Debugw("Fetched active user exchange accounts",
				"user_id", userID,
				"count", len(accounts),
			)

			// Template data structure
			type ExchangeInfo struct {
				StatusEmoji string
				Exchange    string
				Label       string
			}

			var items []telegram.ListItem
			for _, account := range accounts {
				items = append(items, telegram.ListItem{
					ID:         account.ID.String(),
					ButtonText: fmt.Sprintf("📊 %s - %s", strings.Title(string(account.Exchange)), account.Label),
					TemplateData: ExchangeInfo{
						StatusEmoji: "✅",
						Exchange:    strings.Title(string(account.Exchange)),
						Label:       account.Label,
					},
				})
			}

			return items, nil
		},
	})
}

// buildMarketTypeSelectionScreen builds market type selection using framework
func (ims *InvestMenuService) buildMarketTypeSelectionScreen() *telegram.Screen {
	return ims.menuNav.BuildOptionScreen(telegram.OptionScreenConfig{
		ID:           "mkt",
		Template:     "invest/select_market_type",
		NextScreenID: "risk",
		ParamKey:     investKeys.MarketType,
		Options: func(ctx context.Context, session telegram.Session) ([]telegram.MenuOption, error) {
			options := make([]telegram.MenuOption, len(marketTypeOptions))
			for i, opt := range marketTypeOptions {
				options[i] = opt
			}
			return options, nil
		},
		TemplateData: func(ctx context.Context, session telegram.Session) (map[string]interface{}, error) {
			return map[string]interface{}{
				"MarketTypes": marketTypeOptions,
			}, nil
		},
	})
}

// buildRiskSelectionScreen builds risk profile selection using framework
func (ims *InvestMenuService) buildRiskSelectionScreen() *telegram.Screen {
	return ims.menuNav.BuildOptionScreen(telegram.OptionScreenConfig{
		ID:           "risk",
		Template:     "invest/select_risk",
		NextScreenID: "amt",
		ParamKey:     investKeys.RiskProfile,
		Options: func(ctx context.Context, session telegram.Session) ([]telegram.MenuOption, error) {
			options := make([]telegram.MenuOption, len(riskProfileOptions))
			for i, opt := range riskProfileOptions {
				options[i] = opt
			}
			return options, nil
		},
		TemplateData: func(ctx context.Context, session telegram.Session) (map[string]interface{}, error) {
			return map[string]interface{}{
				"RiskProfiles": riskProfileOptions,
			}, nil
		},
	})
}

// buildEnterAmountScreen builds amount input screen using framework
func (ims *InvestMenuService) buildEnterAmountScreen() *telegram.Screen {
	return ims.menuNav.BuildTextInputScreen(
		"amt",
		"invest/enter_amount",
		func(ctx context.Context, nav *telegram.MenuNavigator, params map[string]string) error {
			ims.log.Debugw("Entered amount screen", "params", params)
			return nil
		},
	)
}

// finalizeInvestment creates strategy and publishes portfolio initialization job
func (ims *InvestMenuService) finalizeInvestment(ctx context.Context, session telegram.Session) error {
	// Get parameters from session
	userIDStr, _ := session.GetString("user_id")
	accountIDStr, _ := session.GetString(investKeys.Account)
	marketType, _ := session.GetString(investKeys.MarketType)
	riskProfile, _ := session.GetString(investKeys.RiskProfile)
	amountData, _ := session.GetData("amount")

	// Convert amount
	var amount float64
	switch v := amountData.(type) {
	case float64:
		amount = v
	case int:
		amount = float64(v)
	default:
		return errors.New("invalid amount type in session")
	}

	userID, _ := uuid.Parse(userIDStr)
	accountID, _ := uuid.Parse(accountIDStr)
	telegramID := session.GetTelegramID()

	// End menu
	_ = ims.menuNav.EndMenu(ctx, telegramID)

	// Publish job to Kafka
	if ims.jobPublisher != nil {
		accountIDForJob := ""
		if accountID != uuid.Nil {
			accountIDForJob = accountID.String()
		}

		strategyID := uuid.New()

		if err := ims.jobPublisher.PublishPortfolioInitializationJob(
			ctx,
			userID.String(),
			strategyID.String(),
			telegramID,
			amount,
			accountIDForJob,
			riskProfile,
			marketType,
		); err != nil {
			ims.log.Errorw("Failed to publish portfolio job", "error", err)
			return errors.New("failed to start portfolio creation")
		}

		marketTypeEmoji := "📊"
		if marketType == "futures" {
			marketTypeEmoji = "⚡"
		}

		successMsg := fmt.Sprintf(
			"⏳ Creating your %s %s portfolio in the background...\n\n"+
				"💰 Amount: $%.2f\n"+
				"🎯 Risk: %s\n\n"+
				"You'll be notified when ready (1-2 min)",
			marketTypeEmoji, marketType, amount, riskProfile,
		)

		// Need to store bot reference - will add to MenuNavigator
		ims.log.Infow("Portfolio job published", "strategy_id", strategyID, "amount", amount)
		_ = successMsg // TODO: send via bot
	}

	return nil
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

// Verify InvestMenuService implements telegram.MenuHandler interface at compile time
var _ telegram.MenuHandler = (*InvestMenuService)(nil)
