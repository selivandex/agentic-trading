package telegram

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/menu_session"
	"prometheus/internal/domain/user"
	strategyservice "prometheus/internal/services/strategy"
	"prometheus/pkg/logger"
)

// MockExchangeService is a mock for ExchangeService
type MockExchangeService struct {
	mock.Mock
}

func (m *MockExchangeService) GetUserAccounts(ctx context.Context, userID uuid.UUID) ([]*exchange_account.ExchangeAccount, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*exchange_account.ExchangeAccount), args.Error(1)
}

func (m *MockExchangeService) GetAccount(ctx context.Context, accountID uuid.UUID) (*exchange_account.ExchangeAccount, error) {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*exchange_account.ExchangeAccount), args.Error(1)
}

// MockUserService is a mock for UserService
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetByTelegramID(ctx context.Context, telegramID int64) (*user.User, error) {
	args := m.Called(ctx, telegramID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

// MockJobPublisher is a mock for JobPublisher
type MockJobPublisher struct {
	mock.Mock
}

func (m *MockJobPublisher) PublishPortfolioInitializationJob(
	ctx context.Context,
	userID, strategyID string,
	telegramID int64,
	capital float64,
	exchangeAccountID, riskProfile, marketType string,
) error {
	args := m.Called(ctx, userID, strategyID, telegramID, capital, exchangeAccountID, riskProfile, marketType)
	return args.Error(0)
}

// MockInvestmentValidator is a mock for InvestmentValidator
type MockInvestmentValidator struct {
	mock.Mock
}

func (m *MockInvestmentValidator) ValidateInvestment(ctx context.Context, usr *user.User, requestedCapital float64) (*strategyservice.ValidationResult, error) {
	args := m.Called(ctx, usr, requestedCapital)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*strategyservice.ValidationResult), args.Error(1)
}

func TestRiskProfileOptions(t *testing.T) {
	// Test that risk profile options are properly defined
	assert.Len(t, riskProfileOptions, 3, "Should have 3 risk profiles")

	// Check conservative
	conservative := riskProfileOptions[0]
	assert.Equal(t, "🛡️", conservative.Emoji)
	assert.Equal(t, "Conservative", conservative.Label)
	assert.Equal(t, "conservative", conservative.Value)
	assert.NotEmpty(t, conservative.Description)
	assert.NotEmpty(t, conservative.Features)

	// Check moderate
	moderate := riskProfileOptions[1]
	assert.Equal(t, "⚖️", moderate.Emoji)
	assert.Equal(t, "Moderate", moderate.Label)
	assert.Equal(t, "moderate", moderate.Value)
	assert.NotEmpty(t, moderate.Description)
	assert.NotEmpty(t, moderate.Features)

	// Check aggressive
	aggressive := riskProfileOptions[2]
	assert.Equal(t, "🚀", aggressive.Emoji)
	assert.Equal(t, "Aggressive", aggressive.Label)
	assert.Equal(t, "aggressive", aggressive.Value)
	assert.NotEmpty(t, aggressive.Description)
	assert.NotEmpty(t, aggressive.Features)
}

func TestMarketTypeOptions(t *testing.T) {
	// Test that market type options are properly defined
	assert.Len(t, marketTypeOptions, 2, "Should have 2 market types")

	// Check spot
	spot := marketTypeOptions[0]
	assert.Equal(t, "📊", spot.Emoji)
	assert.Equal(t, "Spot Trading", spot.Label)
	assert.Equal(t, "spot", spot.Value)
	assert.NotEmpty(t, spot.Description)
	assert.NotEmpty(t, spot.Features)

	// Check futures
	futures := marketTypeOptions[1]
	assert.Equal(t, "⚡", futures.Emoji)
	assert.Equal(t, "Futures Trading", futures.Label)
	assert.Equal(t, "futures", futures.Value)
	assert.NotEmpty(t, futures.Description)
	assert.NotEmpty(t, futures.Features)
}

func TestInvestMenuService_GetScreenIDs(t *testing.T) {
	log := logger.Get()
	service := &InvestMenuService{
		log: log,
	}

	screenIDs := service.GetScreenIDs()

	// Should return all screen IDs in correct order
	expected := []string{"sel", "mkt", "risk", "amt", "cancel"}
	assert.Equal(t, expected, screenIDs)
}

func TestInvestMenuService_BuildRiskSelectionScreen(t *testing.T) {
	log := logger.Get()
	service := &InvestMenuService{
		log: log,
	}

	screen := service.buildRiskSelectionScreen()

	assert.Equal(t, "risk", screen.ID)
	assert.Equal(t, "invest/select_risk", screen.Template)
	assert.NotNil(t, screen.Data, "Data function should not be nil")
	assert.NotNil(t, screen.Keyboard, "Keyboard function should not be nil")

	// Test Data function
	ctx := context.Background()
	session := menu_session.NewSession(123456, "risk", nil)

	data, err := screen.Data(ctx, session)
	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.Contains(t, data, "RiskProfiles")

	riskProfiles, ok := data["RiskProfiles"].([]RiskProfileOption)
	assert.True(t, ok, "RiskProfiles should be []RiskProfileOption")
	assert.Len(t, riskProfiles, 3)
}

func TestInvestMenuService_BuildMarketTypeSelectionScreen(t *testing.T) {
	log := logger.Get()
	service := &InvestMenuService{
		log: log,
	}

	screen := service.buildMarketTypeSelectionScreen()

	assert.Equal(t, "mkt", screen.ID)
	assert.Equal(t, "invest/select_market_type", screen.Template)
	assert.NotNil(t, screen.Data, "Data function should not be nil")
	assert.NotNil(t, screen.Keyboard, "Keyboard function should not be nil")

	// Test Data function
	ctx := context.Background()
	session := menu_session.NewSession(123456, "mkt", nil)

	data, err := screen.Data(ctx, session)
	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.Contains(t, data, "MarketTypes")

	marketTypes, ok := data["MarketTypes"].([]MarketTypeOption)
	assert.True(t, ok, "MarketTypes should be []MarketTypeOption")
	assert.Len(t, marketTypes, 2)
}

func TestParseAmount(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{
			name:    "simple number",
			input:   "1000",
			want:    1000.0,
			wantErr: false,
		},
		{
			name:    "number with dollar sign",
			input:   "$1000",
			want:    1000.0,
			wantErr: false,
		},
		{
			name:    "number with commas",
			input:   "1,000",
			want:    1000.0,
			wantErr: false,
		},
		{
			name:    "number with spaces",
			input:   "1 000",
			want:    1000.0,
			wantErr: false,
		},
		{
			name:    "decimal number",
			input:   "1000.50",
			want:    1000.50,
			wantErr: false,
		},
		{
			name:    "complex format",
			input:   "$1,000.50",
			want:    1000.50,
			wantErr: false,
		},
		{
			name:    "invalid text",
			input:   "abc",
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAmount(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
