package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/limit_profile"
	"prometheus/internal/testsupport"
)

func TestLimitProfileRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewLimitProfileRepository(testDB.DB())
	ctx := context.Background()

	// Create test limit profile
	limits := limit_profile.FreeTierLimits()
	profile := &limit_profile.LimitProfile{
		ID:          uuid.New(),
		Name:        "test_free_tier",
		Description: "Free tier for testing",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := profile.SetLimits(&limits)
	require.NoError(t, err, "SetLimits should not fail")

	// Test Create
	err = repo.Create(ctx, profile)
	require.NoError(t, err, "Create should not return error")

	// Verify profile can be retrieved
	retrieved, err := repo.GetByID(ctx, profile.ID)
	require.NoError(t, err)
	assert.Equal(t, profile.Name, retrieved.Name)
	assert.Equal(t, profile.Description, retrieved.Description)
	assert.True(t, retrieved.IsActive)

	// Verify limits are correctly stored
	retrievedLimits, err := retrieved.ParseLimits()
	require.NoError(t, err)
	assert.Equal(t, limits.ExchangesCount, retrievedLimits.ExchangesCount)
	assert.Equal(t, limits.ActivePositions, retrievedLimits.ActivePositions)
	assert.Equal(t, limits.MonthlyAIRequests, retrievedLimits.MonthlyAIRequests)
}

func TestLimitProfileRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewLimitProfileRepository(testDB.DB())
	ctx := context.Background()

	// Create test profile using fixture
	fixtures := NewTestFixtures(t, testDB.DB())
	profileID := fixtures.CreateLimitProfile(
		WithLimitProfileName("test_get_by_id"),
		WithLimitProfileDescription("Test GetByID method"),
	)

	// Test GetByID
	retrieved, err := repo.GetByID(ctx, profileID)
	require.NoError(t, err)
	assert.Equal(t, profileID, retrieved.ID)
	assert.Equal(t, "test_get_by_id", retrieved.Name)
	assert.Equal(t, "Test GetByID method", retrieved.Description)

	// Test non-existent ID
	_, err = repo.GetByID(ctx, uuid.New())
	assert.Error(t, err, "Should return error for non-existent ID")
}

func TestLimitProfileRepository_GetByName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewLimitProfileRepository(testDB.DB())
	ctx := context.Background()

	// Test GetByName with default profiles from migration
	// The migration creates 'free', 'basic', 'premium' profiles
	freeProfile, err := repo.GetByName(ctx, "free")
	require.NoError(t, err)
	assert.Equal(t, "free", freeProfile.Name)
	assert.True(t, freeProfile.IsActive)

	// Parse and verify limits
	limits, err := freeProfile.ParseLimits()
	require.NoError(t, err)
	assert.Equal(t, 1, limits.ExchangesCount, "Free tier should allow 1 exchange")
	assert.Equal(t, 2, limits.ActivePositions, "Free tier should allow 2 positions")
	assert.False(t, limits.LiveTradingAllowed, "Free tier should not allow live trading")

	// Test basic profile
	basicProfile, err := repo.GetByName(ctx, "basic")
	require.NoError(t, err)
	assert.Equal(t, "basic", basicProfile.Name)

	basicLimits, err := basicProfile.ParseLimits()
	require.NoError(t, err)
	assert.Equal(t, 2, basicLimits.ExchangesCount, "Basic tier should allow 2 exchanges")
	assert.True(t, basicLimits.LiveTradingAllowed, "Basic tier should allow live trading")

	// Test premium profile
	premiumProfile, err := repo.GetByName(ctx, "premium")
	require.NoError(t, err)
	assert.Equal(t, "premium", premiumProfile.Name)

	premiumLimits, err := premiumProfile.ParseLimits()
	require.NoError(t, err)
	assert.Equal(t, 10, premiumLimits.ExchangesCount, "Premium tier should allow 10 exchanges")
	assert.Equal(t, -1, premiumLimits.DailyTradesCount, "Premium tier should have unlimited daily trades")
	assert.True(t, premiumLimits.CustomAgentsAllowed, "Premium tier should allow custom agents")

	// Test non-existent name
	_, err = repo.GetByName(ctx, "non_existent_tier")
	assert.Error(t, err, "Should return error for non-existent profile name")
}

func TestLimitProfileRepository_GetAll(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewLimitProfileRepository(testDB.DB())
	ctx := context.Background()

	// Get all profiles (should include at least the 3 default ones from migration)
	profiles, err := repo.GetAll(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(profiles), 3, "Should have at least 3 default profiles")

	// Verify we have the default profiles
	names := make(map[string]bool)
	for _, p := range profiles {
		names[p.Name] = true
	}

	assert.True(t, names["free"], "Should have free profile")
	assert.True(t, names["basic"], "Should have basic profile")
	assert.True(t, names["premium"], "Should have premium profile")
}

func TestLimitProfileRepository_GetAllActive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewLimitProfileRepository(testDB.DB())
	ctx := context.Background()
	fixtures := NewTestFixtures(t, testDB.DB())

	// Create active and inactive profiles
	_ = fixtures.CreateLimitProfile(
		WithLimitProfileName("test_active_1"),
		WithLimitProfileActive(true),
	)
	_ = fixtures.CreateLimitProfile(
		WithLimitProfileName("test_active_2"),
		WithLimitProfileActive(true),
	)
	inactiveID := fixtures.CreateLimitProfile(
		WithLimitProfileName("test_inactive"),
		WithLimitProfileActive(false),
	)

	// Get all active profiles
	activeProfiles, err := repo.GetAllActive(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(activeProfiles), 5, "Should have at least 5 active profiles (3 default + 2 created)")

	// Verify inactive profile is not in the list
	for _, p := range activeProfiles {
		assert.NotEqual(t, inactiveID, p.ID, "Inactive profile should not be returned")
	}

	// Verify all returned profiles are active
	for _, p := range activeProfiles {
		assert.True(t, p.IsActive, "All returned profiles should be active")
	}
}

func TestLimitProfileRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewLimitProfileRepository(testDB.DB())
	ctx := context.Background()

	// Create initial profile
	limits := limit_profile.FreeTierLimits()
	profile := &limit_profile.LimitProfile{
		ID:          uuid.New(),
		Name:        "test_update",
		Description: "Original description",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := profile.SetLimits(&limits)
	require.NoError(t, err)

	err = repo.Create(ctx, profile)
	require.NoError(t, err)

	// Update profile
	profile.Name = "test_update_modified"
	profile.Description = "Updated description"
	profile.IsActive = false

	// Modify limits
	updatedLimits := limit_profile.BasicTierLimits()
	err = profile.SetLimits(&updatedLimits)
	require.NoError(t, err)

	err = repo.Update(ctx, profile)
	require.NoError(t, err)

	// Verify updates
	retrieved, err := repo.GetByID(ctx, profile.ID)
	require.NoError(t, err)
	assert.Equal(t, "test_update_modified", retrieved.Name)
	assert.Equal(t, "Updated description", retrieved.Description)
	assert.False(t, retrieved.IsActive)

	// Verify limits were updated
	retrievedLimits, err := retrieved.ParseLimits()
	require.NoError(t, err)
	assert.Equal(t, updatedLimits.ExchangesCount, retrievedLimits.ExchangesCount)
	assert.Equal(t, updatedLimits.MonthlyAIRequests, retrievedLimits.MonthlyAIRequests)
}

func TestLimitProfileRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewLimitProfileRepository(testDB.DB())
	ctx := context.Background()
	fixtures := NewTestFixtures(t, testDB.DB())

	// Create profile to delete
	profileID := fixtures.CreateLimitProfile(
		WithLimitProfileName("test_delete"),
	)

	// Verify profile exists and is active
	profile, err := repo.GetByID(ctx, profileID)
	require.NoError(t, err)
	assert.True(t, profile.IsActive)

	// Delete profile (soft delete)
	err = repo.Delete(ctx, profileID)
	require.NoError(t, err)

	// Verify profile is still in database but inactive
	deleted, err := repo.GetByID(ctx, profileID)
	require.NoError(t, err, "Soft deleted profile should still be retrievable by ID")
	assert.False(t, deleted.IsActive, "Deleted profile should be inactive")

	// Verify it doesn't appear in GetByName (which filters by is_active)
	_, err = repo.GetByName(ctx, "test_delete")
	assert.Error(t, err, "Inactive profile should not be returned by GetByName")

	// Verify it doesn't appear in GetAllActive
	activeProfiles, err := repo.GetAllActive(ctx)
	require.NoError(t, err)
	for _, p := range activeProfiles {
		assert.NotEqual(t, profileID, p.ID, "Deleted profile should not be in active list")
	}
}

func TestLimitProfileRepository_LimitsJSONB(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewLimitProfileRepository(testDB.DB())
	ctx := context.Background()

	// Create profile with custom limits
	customLimits := limit_profile.Limits{
		ExchangesCount:       5,
		ActivePositions:      10,
		DailyTradesCount:     100,
		MonthlyTradesCount:   1000,
		TradingPairsCount:    50,
		MonthlyAIRequests:    5000,
		MaxAgentsCount:       10,
		AdvancedAgentsAccess: true,
		CustomAgentsAllowed:  true,
		MaxAgentMemoryMB:     500,
		MaxPromptTokens:      16000,
		AdvancedAIModels:     true,
		AgentHistoryDays:     90,
		ReasoningHistoryDays: 30,
		MaxMemoriesPerAgent:  500,
		DataRetentionDays:    180,
		AnalysisHistoryDays:  60,
		SessionHistoryDays:   60,
		BacktestingAllowed:   true,
		MaxBacktestMonths:    24,
		PaperTradingAllowed:  true,
		LiveTradingAllowed:   true,
		MaxLeverage:          5.0,
		MaxPositionSizeUSD:   10000,
		MaxTotalExposureUSD:  50000,
		AdvancedOrderTypes:   true,
		PriorityExecution:    true,
		WebhooksAllowed:      true,
		MaxWebhooks:          20,
		APIAccessAllowed:     true,
		APIRateLimit:         300,
		RealtimeDataAccess:   true,
		HistoricalDataYears:  5,
		AdvancedCharts:       true,
		CustomIndicators:     true,
		PortfolioAnalytics:   true,
		RiskManagementTools:  true,
		AlertsCount:          200,
		CustomReportsAllowed: true,
		PrioritySupport:      true,
		DedicatedSupport:     false,
	}

	profile := &limit_profile.LimitProfile{
		ID:          uuid.New(),
		Name:        "custom_tier",
		Description: "Custom tier with specific limits",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := profile.SetLimits(&customLimits)
	require.NoError(t, err)

	// Create and retrieve
	err = repo.Create(ctx, profile)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(ctx, profile.ID)
	require.NoError(t, err)

	// Parse and verify all JSONB fields
	parsedLimits, err := retrieved.ParseLimits()
	require.NoError(t, err)

	// Verify exchange & trading limits
	assert.Equal(t, 5, parsedLimits.ExchangesCount)
	assert.Equal(t, 10, parsedLimits.ActivePositions)
	assert.Equal(t, 100, parsedLimits.DailyTradesCount)
	assert.Equal(t, 1000, parsedLimits.MonthlyTradesCount)
	assert.Equal(t, 50, parsedLimits.TradingPairsCount)

	// Verify AI & agent limits
	assert.Equal(t, 5000, parsedLimits.MonthlyAIRequests)
	assert.Equal(t, 10, parsedLimits.MaxAgentsCount)
	assert.True(t, parsedLimits.AdvancedAgentsAccess)
	assert.True(t, parsedLimits.CustomAgentsAllowed)
	assert.Equal(t, 500, parsedLimits.MaxAgentMemoryMB)
	assert.Equal(t, 16000, parsedLimits.MaxPromptTokens)

	// Verify history & retention
	assert.Equal(t, 90, parsedLimits.AgentHistoryDays)
	assert.Equal(t, 30, parsedLimits.ReasoningHistoryDays)
	assert.Equal(t, 500, parsedLimits.MaxMemoriesPerAgent)
	assert.Equal(t, 180, parsedLimits.DataRetentionDays)

	// Verify trading features
	assert.True(t, parsedLimits.BacktestingAllowed)
	assert.Equal(t, 24, parsedLimits.MaxBacktestMonths)
	assert.True(t, parsedLimits.LiveTradingAllowed)
	assert.Equal(t, 5.0, parsedLimits.MaxLeverage)
	assert.Equal(t, float64(10000), parsedLimits.MaxPositionSizeUSD)
	assert.Equal(t, float64(50000), parsedLimits.MaxTotalExposureUSD)

	// Verify advanced features
	assert.True(t, parsedLimits.AdvancedOrderTypes)
	assert.True(t, parsedLimits.PriorityExecution)
	assert.True(t, parsedLimits.WebhooksAllowed)
	assert.Equal(t, 20, parsedLimits.MaxWebhooks)
	assert.True(t, parsedLimits.APIAccessAllowed)
	assert.Equal(t, 300, parsedLimits.APIRateLimit)

	// Verify data & analytics
	assert.True(t, parsedLimits.RealtimeDataAccess)
	assert.Equal(t, 5, parsedLimits.HistoricalDataYears)
	assert.True(t, parsedLimits.AdvancedCharts)
	assert.True(t, parsedLimits.CustomIndicators)
	assert.True(t, parsedLimits.PortfolioAnalytics)
	assert.True(t, parsedLimits.RiskManagementTools)

	// Verify support & alerts
	assert.Equal(t, 200, parsedLimits.AlertsCount)
	assert.True(t, parsedLimits.CustomReportsAllowed)
	assert.True(t, parsedLimits.PrioritySupport)
	assert.False(t, parsedLimits.DedicatedSupport)
}

func TestLimitProfileRepository_TierLimits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewLimitProfileRepository(testDB.DB())
	ctx := context.Background()

	tests := []struct {
		name                 string
		limitsFunc           func() limit_profile.Limits
		expectedExchanges    int
		expectedPositions    int
		expectedLiveTrading  bool
		expectedCustomAgents bool
	}{
		{
			name:                 "Free Tier",
			limitsFunc:           limit_profile.FreeTierLimits,
			expectedExchanges:    1,
			expectedPositions:    2,
			expectedLiveTrading:  false,
			expectedCustomAgents: false,
		},
		{
			name:                 "Basic Tier",
			limitsFunc:           limit_profile.BasicTierLimits,
			expectedExchanges:    2,
			expectedPositions:    5,
			expectedLiveTrading:  true,
			expectedCustomAgents: false,
		},
		{
			name:                 "Premium Tier",
			limitsFunc:           limit_profile.PremiumTierLimits,
			expectedExchanges:    10,
			expectedPositions:    20,
			expectedLiveTrading:  true,
			expectedCustomAgents: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits := tt.limitsFunc()

			profile := &limit_profile.LimitProfile{
				ID:          uuid.New(),
				Name:        "test_" + tt.name,
				Description: tt.name + " limits test",
				IsActive:    true,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}

			err := profile.SetLimits(&limits)
			require.NoError(t, err)

			err = repo.Create(ctx, profile)
			require.NoError(t, err)

			retrieved, err := repo.GetByID(ctx, profile.ID)
			require.NoError(t, err)

			parsedLimits, err := retrieved.ParseLimits()
			require.NoError(t, err)

			assert.Equal(t, tt.expectedExchanges, parsedLimits.ExchangesCount, "Exchanges count mismatch")
			assert.Equal(t, tt.expectedPositions, parsedLimits.ActivePositions, "Positions count mismatch")
			assert.Equal(t, tt.expectedLiveTrading, parsedLimits.LiveTradingAllowed, "Live trading flag mismatch")
			assert.Equal(t, tt.expectedCustomAgents, parsedLimits.CustomAgentsAllowed, "Custom agents flag mismatch")
		})
	}
}
