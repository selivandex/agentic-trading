package strategy_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/strategy"
)

// mockRepository is a mock implementation of strategy.Repository
type mockRepository struct {
	createFunc func(ctx context.Context, s *strategy.Strategy) error
	getFunc    func(ctx context.Context, id uuid.UUID) (*strategy.Strategy, error)
}

func (m *mockRepository) Create(ctx context.Context, s *strategy.Strategy) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, s)
	}
	return nil
}

func (m *mockRepository) GetByID(ctx context.Context, id uuid.UUID) (*strategy.Strategy, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*strategy.Strategy, error) {
	return nil, nil
}

func (m *mockRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*strategy.Strategy, error) {
	return nil, nil
}

func (m *mockRepository) Update(ctx context.Context, s *strategy.Strategy) error {
	return nil
}

func (m *mockRepository) UpdateEquity(ctx context.Context, id uuid.UUID, currentEquity, cashReserve, totalPnL, totalPnLPercent decimal.Decimal) error {
	return nil
}

func (m *mockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockRepository) GetTotalAllocatedCapital(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

func (m *mockRepository) GetTotalCurrentEquity(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

// TestCreateStrategy_InitializesJSONFields tests that nil JSON fields are initialized correctly
func TestCreateStrategy_InitializesJSONFields(t *testing.T) {
	ctx := context.Background()

	var capturedStrategy *strategy.Strategy
	mockRepo := &mockRepository{
		createFunc: func(ctx context.Context, s *strategy.Strategy) error {
			capturedStrategy = s
			return nil
		},
	}

	service := strategy.NewService(mockRepo)

	testCases := []struct {
		name               string
		strategy           *strategy.Strategy
		expectedReasoning  []byte
		expectedAllocation []byte
	}{
		{
			name: "nil JSON fields",
			strategy: &strategy.Strategy{
				UserID:            uuid.New(),
				Name:              "Test Strategy",
				AllocatedCapital:  decimal.NewFromInt(1000),
				RiskTolerance:     strategy.RiskModerate,
				ReasoningLog:      nil, // Should be initialized to []
				TargetAllocations: nil, // Should be initialized to {}
			},
			expectedReasoning:  []byte("[]"),
			expectedAllocation: []byte("{}"),
		},
		{
			name: "existing JSON fields should not be overwritten",
			strategy: &strategy.Strategy{
				UserID:            uuid.New(),
				Name:              "Test Strategy",
				AllocatedCapital:  decimal.NewFromInt(1000),
				RiskTolerance:     strategy.RiskModerate,
				ReasoningLog:      []byte(`[{"step": 1}]`),
				TargetAllocations: []byte(`{"BTC": 0.5}`),
			},
			expectedReasoning:  []byte(`[{"step": 1}]`),
			expectedAllocation: []byte(`{"BTC": 0.5}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			capturedStrategy = nil

			err := service.Create(ctx, tc.strategy)
			require.NoError(t, err)
			require.NotNil(t, capturedStrategy)

			// Check JSON fields are initialized
			assert.Equal(t, string(tc.expectedReasoning), string(capturedStrategy.ReasoningLog),
				"ReasoningLog should be initialized to empty JSON array")
			assert.Equal(t, string(tc.expectedAllocation), string(capturedStrategy.TargetAllocations),
				"TargetAllocations should be initialized to empty JSON object")

			// Check other fields
			assert.NotEqual(t, uuid.Nil, capturedStrategy.ID, "ID should be generated")
			assert.Equal(t, strategy.StrategyActive, capturedStrategy.Status, "Status should be Active")
			assert.False(t, capturedStrategy.CreatedAt.IsZero(), "CreatedAt should be set")
			assert.False(t, capturedStrategy.UpdatedAt.IsZero(), "UpdatedAt should be set")

			// Check financial fields
			assert.Equal(t, tc.strategy.AllocatedCapital, capturedStrategy.CurrentEquity,
				"CurrentEquity should equal AllocatedCapital initially")
			assert.Equal(t, tc.strategy.AllocatedCapital, capturedStrategy.CashReserve,
				"CashReserve should equal AllocatedCapital initially")
			assert.True(t, capturedStrategy.TotalPnL.IsZero(), "TotalPnL should be zero")
			assert.True(t, capturedStrategy.TotalPnLPercent.IsZero(), "TotalPnLPercent should be zero")
		})
	}
}

// TestCreateStrategy_Validation tests input validation
func TestCreateStrategy_Validation(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockRepository{}
	service := strategy.NewService(mockRepo)

	testCases := []struct {
		name        string
		strategy    *strategy.Strategy
		expectError bool
	}{
		{
			name:        "nil strategy",
			strategy:    nil,
			expectError: true,
		},
		{
			name: "missing user_id",
			strategy: &strategy.Strategy{
				Name:             "Test",
				AllocatedCapital: decimal.NewFromInt(1000),
				RiskTolerance:    strategy.RiskModerate,
			},
			expectError: true,
		},
		{
			name: "zero allocated capital",
			strategy: &strategy.Strategy{
				UserID:           uuid.New(),
				Name:             "Test",
				AllocatedCapital: decimal.Zero,
				RiskTolerance:    strategy.RiskModerate,
			},
			expectError: true,
		},
		{
			name: "negative allocated capital",
			strategy: &strategy.Strategy{
				UserID:           uuid.New(),
				Name:             "Test",
				AllocatedCapital: decimal.NewFromInt(-100),
				RiskTolerance:    strategy.RiskModerate,
			},
			expectError: true,
		},
		{
			name: "invalid risk tolerance",
			strategy: &strategy.Strategy{
				UserID:           uuid.New(),
				Name:             "Test",
				AllocatedCapital: decimal.NewFromInt(1000),
				RiskTolerance:    strategy.RiskTolerance("invalid"),
			},
			expectError: true,
		},
		{
			name: "valid strategy",
			strategy: &strategy.Strategy{
				UserID:           uuid.New(),
				Name:             "Test",
				AllocatedCapital: decimal.NewFromInt(1000),
				RiskTolerance:    strategy.RiskModerate,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.Create(ctx, tc.strategy)
			if tc.expectError {
				assert.Error(t, err, "Expected error for invalid input")
			} else {
				assert.NoError(t, err, "Expected no error for valid input")
			}
		})
	}
}

// TestGetByID tests the GetByID method
func TestGetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("valid ID", func(t *testing.T) {
		expectedStrategy := &strategy.Strategy{
			ID:     uuid.New(),
			UserID: uuid.New(),
			Name:   "Test",
		}

		mockRepo := &mockRepository{
			getFunc: func(ctx context.Context, id uuid.UUID) (*strategy.Strategy, error) {
				return expectedStrategy, nil
			},
		}

		service := strategy.NewService(mockRepo)
		result, err := service.GetByID(ctx, expectedStrategy.ID)

		require.NoError(t, err)
		assert.Equal(t, expectedStrategy, result)
	})

	t.Run("nil ID", func(t *testing.T) {
		mockRepo := &mockRepository{}
		service := strategy.NewService(mockRepo)

		result, err := service.GetByID(ctx, uuid.Nil)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// TestUpdateEquity tests the UpdateEquity method
func TestUpdateEquity(t *testing.T) {
	ctx := context.Background()

	testStrategy := &strategy.Strategy{
		ID:               uuid.New(),
		UserID:           uuid.New(),
		AllocatedCapital: decimal.NewFromInt(1000),
		CashReserve:      decimal.NewFromInt(500),
	}

	mockRepo := &mockRepository{
		getFunc: func(ctx context.Context, id uuid.UUID) (*strategy.Strategy, error) {
			return testStrategy, nil
		},
	}

	service := strategy.NewService(mockRepo)

	// Update equity with positions value
	positionsValue := decimal.NewFromInt(600)
	err := service.UpdateEquity(ctx, testStrategy.ID, positionsValue)

	require.NoError(t, err)

	// Verify calculations
	expectedEquity := decimal.NewFromInt(500).Add(decimal.NewFromInt(600)) // cash + positions = 1100
	assert.Equal(t, expectedEquity, testStrategy.CurrentEquity)

	expectedPnL := decimal.NewFromInt(100) // 1100 - 1000 = 100
	assert.Equal(t, expectedPnL, testStrategy.TotalPnL)

	expectedPnLPercent := decimal.NewFromInt(10) // (1100/1000 - 1) * 100 = 10%
	assert.Equal(t, expectedPnLPercent, testStrategy.TotalPnLPercent)
}
