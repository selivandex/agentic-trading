package resolvers_test

import (
	"context"
	"testing"

	"prometheus/internal/api/graphql/generated"
	"prometheus/internal/domain/strategy"
	"prometheus/internal/testsupport/seeds"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateStrategyMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	tests := []struct {
		name        string
		userID      uuid.UUID
		input       generated.CreateStrategyInput
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful creation",
			userID: setup.testUser.ID,
			input: generated.CreateStrategyInput{
				Name:               "Test Strategy",
				Description:        "Test Description",
				AllocatedCapital:   decimal.NewFromInt(10000),
				MarketType:         strategy.MarketSpot,
				RiskTolerance:      strategy.RiskModerate,
				RebalanceFrequency: strategy.RebalanceWeekly,
			},
			wantErr: false,
		},
		{
			name:   "with target allocations",
			userID: setup.testUser.ID,
			input: generated.CreateStrategyInput{
				Name:               "Diversified Strategy",
				Description:        "Multi-asset strategy",
				AllocatedCapital:   decimal.NewFromInt(50000),
				MarketType:         strategy.MarketSpot,
				RiskTolerance:      strategy.RiskAggressive,
				RebalanceFrequency: strategy.RebalanceDaily,
				TargetAllocations: map[string]interface{}{
					"BTC/USDT": 0.5,
					"ETH/USDT": 0.3,
					"SOL/USDT": 0.2,
				},
			},
			wantErr: false,
		},
		{
			name:   "zero allocated capital",
			userID: setup.testUser.ID,
			input: generated.CreateStrategyInput{
				Name:               "Invalid Strategy",
				Description:        "Should fail",
				AllocatedCapital:   decimal.Zero,
				MarketType:         strategy.MarketSpot,
				RiskTolerance:      strategy.RiskModerate,
				RebalanceFrequency: strategy.RebalanceWeekly,
			},
			wantErr:     true,
			errContains: "allocated_capital must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().CreateStrategy(ctx, tt.userID, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.NotEqual(t, uuid.Nil, result.ID)
			assert.Equal(t, tt.userID, result.UserID)
			assert.Equal(t, tt.input.Name, result.Name)
			assert.Equal(t, tt.input.Description, result.Description)
			assert.True(t, tt.input.AllocatedCapital.Equal(result.AllocatedCapital))
			assert.Equal(t, tt.input.MarketType, result.MarketType)
			assert.Equal(t, tt.input.RiskTolerance, result.RiskTolerance)
			assert.Equal(t, tt.input.RebalanceFrequency, result.RebalanceFrequency)
			assert.Equal(t, strategy.StrategyActive, result.Status)
			assert.True(t, result.CurrentEquity.Equal(tt.input.AllocatedCapital))
			assert.True(t, result.CashReserve.Equal(tt.input.AllocatedCapital))
		})
	}
}

func TestUpdateStrategyMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create test strategy
	seeder := seeds.New(setup.pg.DB())
	testStrategy := seeder.Strategy().
		WithUserID(setup.testUser.ID).
		WithName("Original Strategy").
		WithDescription("Original Description").
		WithRiskTolerance(strategy.RiskConservative).
		MustInsert()

	tests := []struct {
		name        string
		strategyID  uuid.UUID
		input       generated.UpdateStrategyInput
		wantErr     bool
		errContains string
		validate    func(t *testing.T, result *strategy.Strategy)
	}{
		{
			name:       "update name only",
			strategyID: testStrategy.ID,
			input: generated.UpdateStrategyInput{
				Name: strPtr("Updated Strategy Name"),
			},
			wantErr: false,
			validate: func(t *testing.T, result *strategy.Strategy) {
				assert.Equal(t, "Updated Strategy Name", result.Name)
				assert.Equal(t, "Original Description", result.Description) // unchanged
			},
		},
		{
			name:       "update risk tolerance",
			strategyID: testStrategy.ID,
			input: generated.UpdateStrategyInput{
				RiskTolerance: (*strategy.RiskTolerance)(strPtr(string(strategy.RiskAggressive))),
			},
			wantErr: false,
			validate: func(t *testing.T, result *strategy.Strategy) {
				assert.Equal(t, strategy.RiskAggressive, result.RiskTolerance)
			},
		},
		{
			name:       "update multiple fields",
			strategyID: testStrategy.ID,
			input: generated.UpdateStrategyInput{
				Name:               strPtr("Multi Update"),
				Description:        strPtr("Updated Description"),
				RebalanceFrequency: (*strategy.RebalanceFrequency)(strPtr(string(strategy.RebalanceMonthly))),
			},
			wantErr: false,
			validate: func(t *testing.T, result *strategy.Strategy) {
				assert.Equal(t, "Multi Update", result.Name)
				assert.Equal(t, "Updated Description", result.Description)
				assert.Equal(t, strategy.RebalanceMonthly, result.RebalanceFrequency)
			},
		},
		{
			name:       "update target allocations",
			strategyID: testStrategy.ID,
			input: generated.UpdateStrategyInput{
				TargetAllocations: map[string]interface{}{
					"BTC/USDT": 0.6,
					"ETH/USDT": 0.4,
				},
			},
			wantErr: false,
			validate: func(t *testing.T, result *strategy.Strategy) {
				assert.NotNil(t, result.TargetAllocations)
			},
		},
		{
			name:        "non-existent strategy",
			strategyID:  uuid.New(),
			input:       generated.UpdateStrategyInput{Name: strPtr("Test")},
			wantErr:     true,
			errContains: "failed to get strategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().UpdateStrategy(ctx, tt.strategyID, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.strategyID, result.ID)

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestPauseStrategyMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create active strategy
	seeder := seeds.New(setup.pg.DB())
	activeStrategy := seeder.Strategy().
		WithUserID(setup.testUser.ID).
		WithStatus(strategy.StrategyActive).
		MustInsert()

	// Create already paused strategy
	pausedStrategy := seeder.Strategy().
		WithUserID(setup.testUser.ID).
		WithStatus(strategy.StrategyPaused).
		MustInsert()

	tests := []struct {
		name        string
		strategyID  uuid.UUID
		wantErr     bool
		errContains string
	}{
		{
			name:       "pause active strategy",
			strategyID: activeStrategy.ID,
			wantErr:    false,
		},
		{
			name:        "pause already paused strategy",
			strategyID:  pausedStrategy.ID,
			wantErr:     true,
			errContains: "can only pause active strategies",
		},
		{
			name:        "non-existent strategy",
			strategyID:  uuid.New(),
			wantErr:     true,
			errContains: "failed to get strategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().PauseStrategy(ctx, tt.strategyID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, strategy.StrategyPaused, result.Status)
			assert.Equal(t, tt.strategyID, result.ID)
		})
	}
}

func TestResumeStrategyMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create paused strategy
	seeder := seeds.New(setup.pg.DB())
	pausedStrategy := seeder.Strategy().
		WithUserID(setup.testUser.ID).
		WithStatus(strategy.StrategyPaused).
		MustInsert()

	// Create active strategy
	activeStrategy := seeder.Strategy().
		WithUserID(setup.testUser.ID).
		WithStatus(strategy.StrategyActive).
		MustInsert()

	tests := []struct {
		name        string
		strategyID  uuid.UUID
		wantErr     bool
		errContains string
	}{
		{
			name:       "resume paused strategy",
			strategyID: pausedStrategy.ID,
			wantErr:    false,
		},
		{
			name:        "resume active strategy",
			strategyID:  activeStrategy.ID,
			wantErr:     true,
			errContains: "can only resume paused strategies",
		},
		{
			name:        "non-existent strategy",
			strategyID:  uuid.New(),
			wantErr:     true,
			errContains: "failed to get strategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().ResumeStrategy(ctx, tt.strategyID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, strategy.StrategyActive, result.Status)
			assert.Equal(t, tt.strategyID, result.ID)
		})
	}
}

func TestCloseStrategyMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create active strategy
	seeder := seeds.New(setup.pg.DB())
	activeStrategy := seeder.Strategy().
		WithUserID(setup.testUser.ID).
		WithStatus(strategy.StrategyActive).
		MustInsert()

	// Create closed strategy
	closedStrategy := seeder.Strategy().
		WithUserID(setup.testUser.ID).
		WithStatus(strategy.StrategyClosed).
		MustInsert()

	tests := []struct {
		name        string
		strategyID  uuid.UUID
		wantErr     bool
		errContains string
	}{
		{
			name:       "close active strategy",
			strategyID: activeStrategy.ID,
			wantErr:    false,
		},
		{
			name:        "close already closed strategy",
			strategyID:  closedStrategy.ID,
			wantErr:     true,
			errContains: "strategy is already closed",
		},
		{
			name:        "non-existent strategy",
			strategyID:  uuid.New(),
			wantErr:     true,
			errContains: "failed to get strategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().CloseStrategy(ctx, tt.strategyID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, strategy.StrategyClosed, result.Status)
			assert.NotNil(t, result.ClosedAt)
			assert.Equal(t, tt.strategyID, result.ID)
		})
	}
}

func TestDeleteStrategyMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create closed strategy (can be deleted)
	seeder := seeds.New(setup.pg.DB())
	closedStrategy := seeder.Strategy().
		WithUserID(setup.testUser.ID).
		WithStatus(strategy.StrategyClosed).
		MustInsert()

	// Create active strategy (cannot be deleted)
	activeStrategy := seeder.Strategy().
		WithUserID(setup.testUser.ID).
		WithStatus(strategy.StrategyActive).
		MustInsert()

	tests := []struct {
		name        string
		strategyID  uuid.UUID
		wantErr     bool
		errContains string
	}{
		{
			name:       "delete closed strategy",
			strategyID: closedStrategy.ID,
			wantErr:    false,
		},
		{
			name:        "delete active strategy",
			strategyID:  activeStrategy.ID,
			wantErr:     true,
			errContains: "can only delete closed strategies",
		},
		{
			name:        "non-existent strategy",
			strategyID:  uuid.New(),
			wantErr:     true,
			errContains: "failed to get strategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().DeleteStrategy(ctx, tt.strategyID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.True(t, result)

			// Verify strategy is actually deleted
			_, err = setup.strategyService.GetStrategyByID(ctx, tt.strategyID)
			assert.Error(t, err) // Should not find deleted strategy
		})
	}
}

func TestStrategyMutationsWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Full lifecycle test: create → update → pause → resume → close → delete
	t.Run("complete strategy lifecycle", func(t *testing.T) {
		// 1. Create strategy
		createInput := generated.CreateStrategyInput{
			Name:               "Lifecycle Test Strategy",
			Description:        "Testing full lifecycle",
			AllocatedCapital:   decimal.NewFromInt(25000),
			MarketType:         strategy.MarketSpot,
			RiskTolerance:      strategy.RiskModerate,
			RebalanceFrequency: strategy.RebalanceWeekly,
		}

		created, err := setup.resolver.Mutation().CreateStrategy(ctx, setup.testUser.ID, createInput)
		require.NoError(t, err)
		assert.Equal(t, strategy.StrategyActive, created.Status)

		// 2. Update strategy
		updateInput := generated.UpdateStrategyInput{
			Name:          strPtr("Updated Lifecycle Strategy"),
			RiskTolerance: (*strategy.RiskTolerance)(strPtr(string(strategy.RiskAggressive))),
		}

		updated, err := setup.resolver.Mutation().UpdateStrategy(ctx, created.ID, updateInput)
		require.NoError(t, err)
		assert.Equal(t, "Updated Lifecycle Strategy", updated.Name)
		assert.Equal(t, strategy.RiskAggressive, updated.RiskTolerance)

		// 3. Pause strategy
		paused, err := setup.resolver.Mutation().PauseStrategy(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, strategy.StrategyPaused, paused.Status)

		// 4. Resume strategy
		resumed, err := setup.resolver.Mutation().ResumeStrategy(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, strategy.StrategyActive, resumed.Status)

		// 5. Close strategy
		closed, err := setup.resolver.Mutation().CloseStrategy(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, strategy.StrategyClosed, closed.Status)
		assert.NotNil(t, closed.ClosedAt)

		// 6. Delete strategy
		deleted, err := setup.resolver.Mutation().DeleteStrategy(ctx, created.ID)
		require.NoError(t, err)
		assert.True(t, deleted)
	})
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}
