package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/trading_pair"
	"prometheus/internal/testsupport"
)

func TestTradingPairRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewTradingPairRepository(testDB.DB())
	ctx := context.Background()

	// Use fixtures factory
	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()
	exchangeAccountID := fixtures.CreateExchangeAccount(userID)

	pair := &trading_pair.TradingPair{
		ID:                uuid.New(),
		UserID:            userID,
		ExchangeAccountID: exchangeAccountID,
		Symbol:            "BTC/USDT",
		MarketType:        trading_pair.MarketSpot,
		Budget:            decimal.NewFromFloat(1000.0),
		MaxPositionSize:   decimal.NewFromFloat(500.0),
		MaxLeverage:       1,
		StopLossPercent:   decimal.NewFromFloat(2.0),
		TakeProfitPercent: decimal.NewFromFloat(5.0),
		AIProvider:        "claude",
		StrategyMode:      trading_pair.StrategyAuto,
		Timeframes:        []string{"1h", "4h", "1d"},
		IsActive:          true,
		IsPaused:          false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Test Create
	err := repo.Create(ctx, pair)
	require.NoError(t, err, "Create should not return error")

	// Verify pair can be retrieved
	retrieved, err := repo.GetByID(ctx, pair.ID)
	require.NoError(t, err)
	assert.Equal(t, pair.Symbol, retrieved.Symbol)
	assert.Equal(t, pair.MarketType, retrieved.MarketType)
	assert.True(t, pair.Budget.Equal(retrieved.Budget))
}

func TestTradingPairRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewTradingPairRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()
	exchangeAccountID := fixtures.CreateExchangeAccount(userID)

	pair := &trading_pair.TradingPair{
		ID:                uuid.New(),
		UserID:            userID,
		ExchangeAccountID: exchangeAccountID,
		Symbol:            "ETH/USDT",
		MarketType:        trading_pair.MarketFutures,
		Budget:            decimal.NewFromFloat(2000.0),
		MaxPositionSize:   decimal.NewFromFloat(1000.0),
		MaxLeverage:       3,
		StopLossPercent:   decimal.NewFromFloat(3.0),
		TakeProfitPercent: decimal.NewFromFloat(8.0),
		AIProvider:        "openai",
		StrategyMode:      trading_pair.StrategySemiAuto,
		Timeframes:        []string{"15m", "1h"},
		IsActive:          true,
		IsPaused:          false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := repo.Create(ctx, pair)
	require.NoError(t, err)

	// Test GetByID
	retrieved, err := repo.GetByID(ctx, pair.ID)
	require.NoError(t, err)
	assert.Equal(t, pair.ID, retrieved.ID)
	assert.Equal(t, trading_pair.MarketFutures, retrieved.MarketType)
	assert.Equal(t, 3, retrieved.MaxLeverage)
	assert.Equal(t, trading_pair.StrategySemiAuto, retrieved.StrategyMode)
	assert.Len(t, retrieved.Timeframes, 2)

	// Test non-existent ID
	_, err = repo.GetByID(ctx, uuid.New())
	assert.Error(t, err, "Should return error for non-existent ID")
}

func TestTradingPairRepository_GetByUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewTradingPairRepository(testDB.DB())
	ctx := context.Background()

	// Use fixtures factory
	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()
	exchangeAccountID := fixtures.CreateExchangeAccount(userID)

	// Create multiple trading pairs for same user
	symbols := []string{"BTC/USDT", "ETH/USDT", "SOL/USDT"}
	for _, symbol := range symbols {
		pair := &trading_pair.TradingPair{
			ID:                uuid.New(),
			UserID:            userID,
			ExchangeAccountID: exchangeAccountID,
			Symbol:            symbol,
			MarketType:        trading_pair.MarketSpot,
			Budget:            decimal.NewFromFloat(1000.0),
			MaxPositionSize:   decimal.NewFromFloat(500.0),
			MaxLeverage:       1,
			StopLossPercent:   decimal.NewFromFloat(2.0),
			TakeProfitPercent: decimal.NewFromFloat(5.0),
			AIProvider:        "claude",
			StrategyMode:      trading_pair.StrategyAuto,
			Timeframes:        []string{"1h"},
			IsActive:          true,
			IsPaused:          false,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}
		err := repo.Create(ctx, pair)
		require.NoError(t, err)
	}

	// Test GetByUser
	pairs, err := repo.GetByUser(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, pairs, 3, "Should return all 3 pairs")

	// Verify all belong to same user
	for _, p := range pairs {
		assert.Equal(t, userID, p.UserID)
	}
}

func TestTradingPairRepository_GetActiveByUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewTradingPairRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()
	exchangeAccountID := fixtures.CreateExchangeAccount(userID)

	// Create active pair
	activePair := &trading_pair.TradingPair{
		ID:                uuid.New(),
		UserID:            userID,
		ExchangeAccountID: exchangeAccountID,
		Symbol:            "BTC/USDT",
		MarketType:        trading_pair.MarketSpot,
		Budget:            decimal.NewFromFloat(1000.0),
		MaxPositionSize:   decimal.NewFromFloat(500.0),
		AIProvider:        "claude",
		StrategyMode:      trading_pair.StrategyAuto,
		Timeframes:        []string{"1h"},
		IsActive:          true,
		IsPaused:          false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	err := repo.Create(ctx, activePair)
	require.NoError(t, err)

	// Create inactive pair
	inactivePair := &trading_pair.TradingPair{
		ID:                uuid.New(),
		UserID:            userID,
		ExchangeAccountID: exchangeAccountID,
		Symbol:            "ETH/USDT",
		MarketType:        trading_pair.MarketSpot,
		Budget:            decimal.NewFromFloat(1000.0),
		MaxPositionSize:   decimal.NewFromFloat(500.0),
		AIProvider:        "claude",
		StrategyMode:      trading_pair.StrategyAuto,
		Timeframes:        []string{"1h"},
		IsActive:          false,
		IsPaused:          false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	err = repo.Create(ctx, inactivePair)
	require.NoError(t, err)

	// Create paused pair (should still be counted as active)
	pausedPair := &trading_pair.TradingPair{
		ID:                uuid.New(),
		UserID:            userID,
		ExchangeAccountID: exchangeAccountID,
		Symbol:            "SOL/USDT",
		MarketType:        trading_pair.MarketSpot,
		Budget:            decimal.NewFromFloat(1000.0),
		MaxPositionSize:   decimal.NewFromFloat(500.0),
		AIProvider:        "claude",
		StrategyMode:      trading_pair.StrategyAuto,
		Timeframes:        []string{"1h"},
		IsActive:          true,
		IsPaused:          true,
		PausedReason:      func() *string { s := "User requested pause"; return &s }(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	err = repo.Create(ctx, pausedPair)
	require.NoError(t, err)

	// Test GetActiveByUser - should return only is_active = true
	activePairs, err := repo.GetActiveByUser(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, activePairs, 2, "Should return 2 active pairs (including paused)")

	// Verify all returned pairs are active
	for _, p := range activePairs {
		assert.True(t, p.IsActive)
	}
}

func TestTradingPairRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewTradingPairRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()
	exchangeAccountID := fixtures.CreateExchangeAccount(userID)

	pair := &trading_pair.TradingPair{
		ID:                uuid.New(),
		UserID:            userID,
		ExchangeAccountID: exchangeAccountID,
		Symbol:            "BTC/USDT",
		MarketType:        trading_pair.MarketSpot,
		Budget:            decimal.NewFromFloat(1000.0),
		MaxPositionSize:   decimal.NewFromFloat(500.0),
		MaxLeverage:       1,
		StopLossPercent:   decimal.NewFromFloat(2.0),
		TakeProfitPercent: decimal.NewFromFloat(5.0),
		AIProvider:        "claude",
		StrategyMode:      trading_pair.StrategyAuto,
		Timeframes:        []string{"1h"},
		IsActive:          true,
		IsPaused:          false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := repo.Create(ctx, pair)
	require.NoError(t, err)

	// Update pair settings
	pair.Budget = decimal.NewFromFloat(2000.0)
	pair.MaxPositionSize = decimal.NewFromFloat(1000.0)
	pair.StopLossPercent = decimal.NewFromFloat(3.0)
	pair.TakeProfitPercent = decimal.NewFromFloat(8.0)
	pair.StrategyMode = trading_pair.StrategySemiAuto
	pair.Timeframes = []string{"15m", "1h", "4h"}

	err = repo.Update(ctx, pair)
	require.NoError(t, err)

	// Verify updates
	retrieved, err := repo.GetByID(ctx, pair.ID)
	require.NoError(t, err)
	assert.True(t, decimal.NewFromFloat(2000.0).Equal(retrieved.Budget))
	assert.True(t, decimal.NewFromFloat(1000.0).Equal(retrieved.MaxPositionSize))
	assert.True(t, decimal.NewFromFloat(3.0).Equal(retrieved.StopLossPercent))
	assert.True(t, decimal.NewFromFloat(8.0).Equal(retrieved.TakeProfitPercent))
	assert.Equal(t, trading_pair.StrategySemiAuto, retrieved.StrategyMode)
	assert.Len(t, retrieved.Timeframes, 3)
}

func TestTradingPairRepository_Pause(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewTradingPairRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()
	exchangeAccountID := fixtures.CreateExchangeAccount(userID)

	pair := &trading_pair.TradingPair{
		ID:                uuid.New(),
		UserID:            userID,
		ExchangeAccountID: exchangeAccountID,
		Symbol:            "BTC/USDT",
		MarketType:        trading_pair.MarketSpot,
		Budget:            decimal.NewFromFloat(1000.0),
		MaxPositionSize:   decimal.NewFromFloat(500.0),
		AIProvider:        "claude",
		StrategyMode:      trading_pair.StrategyAuto,
		Timeframes:        []string{"1h"},
		IsActive:          true,
		IsPaused:          false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := repo.Create(ctx, pair)
	require.NoError(t, err)

	// Test Pause
	pauseReason := "Market volatility too high"
	err = repo.Pause(ctx, pair.ID, pauseReason)
	require.NoError(t, err)

	// Verify paused
	retrieved, err := repo.GetByID(ctx, pair.ID)
	require.NoError(t, err)
	assert.True(t, retrieved.IsPaused)
	require.NotNil(t, retrieved.PausedReason)
	assert.Equal(t, pauseReason, *retrieved.PausedReason)
	assert.True(t, retrieved.IsActive, "Should still be active")
}

func TestTradingPairRepository_Resume(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewTradingPairRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()
	exchangeAccountID := fixtures.CreateExchangeAccount(userID)

	// Create paused pair
	pair := &trading_pair.TradingPair{
		ID:                uuid.New(),
		UserID:            userID,
		ExchangeAccountID: exchangeAccountID,
		Symbol:            "ETH/USDT",
		MarketType:        trading_pair.MarketSpot,
		Budget:            decimal.NewFromFloat(1000.0),
		MaxPositionSize:   decimal.NewFromFloat(500.0),
		AIProvider:        "claude",
		StrategyMode:      trading_pair.StrategyAuto,
		Timeframes:        []string{"1h"},
		IsActive:          true,
		IsPaused:          true,
		PausedReason:      func() *string { s := "Temporary pause"; return &s }(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := repo.Create(ctx, pair)
	require.NoError(t, err)

	// Test Resume
	err = repo.Resume(ctx, pair.ID)
	require.NoError(t, err)

	// Verify resumed
	retrieved, err := repo.GetByID(ctx, pair.ID)
	require.NoError(t, err)
	assert.False(t, retrieved.IsPaused)
	assert.Empty(t, retrieved.PausedReason)
	assert.True(t, retrieved.IsActive)
}

func TestTradingPairRepository_Disable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewTradingPairRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()
	exchangeAccountID := fixtures.CreateExchangeAccount(userID)

	pair := &trading_pair.TradingPair{
		ID:                uuid.New(),
		UserID:            userID,
		ExchangeAccountID: exchangeAccountID,
		Symbol:            "BTC/USDT",
		MarketType:        trading_pair.MarketSpot,
		Budget:            decimal.NewFromFloat(1000.0),
		MaxPositionSize:   decimal.NewFromFloat(500.0),
		AIProvider:        "claude",
		StrategyMode:      trading_pair.StrategyAuto,
		Timeframes:        []string{"1h"},
		IsActive:          true,
		IsPaused:          false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := repo.Create(ctx, pair)
	require.NoError(t, err)

	// Test Disable
	err = repo.Disable(ctx, pair.ID)
	require.NoError(t, err)

	// Verify disabled
	retrieved, err := repo.GetByID(ctx, pair.ID)
	require.NoError(t, err)
	assert.False(t, retrieved.IsActive)
}

func TestTradingPairRepository_GetActiveBySymbol(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewTradingPairRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	symbol := "TEST" + uuid.New().String()[:8] + "/USDT" // Unique symbol for this test

	// Create multiple active pairs for same symbol (different users)
	for i := 0; i < 3; i++ {
		userID := fixtures.CreateUser()
		exchangeAccountID := fixtures.CreateExchangeAccount(userID)

		pair := &trading_pair.TradingPair{
			ID:                uuid.New(),
			UserID:            userID,
			ExchangeAccountID: exchangeAccountID,
			Symbol:            symbol,
			MarketType:        trading_pair.MarketSpot,
			Budget:            decimal.NewFromFloat(1000.0),
			MaxPositionSize:   decimal.NewFromFloat(500.0),
			AIProvider:        "claude",
			StrategyMode:      trading_pair.StrategyAuto,
			Timeframes:        []string{"1h"},
			IsActive:          true,
			IsPaused:          false,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}
		err := repo.Create(ctx, pair)
		require.NoError(t, err)
	}

	// Create inactive pair for same symbol
	inactiveUserID := fixtures.CreateUser()
	inactiveExchangeAccountID := fixtures.CreateExchangeAccount(inactiveUserID)

	inactivePair := &trading_pair.TradingPair{
		ID:                uuid.New(),
		UserID:            inactiveUserID,
		ExchangeAccountID: inactiveExchangeAccountID,
		Symbol:            symbol,
		MarketType:        trading_pair.MarketSpot,
		Budget:            decimal.NewFromFloat(1000.0),
		MaxPositionSize:   decimal.NewFromFloat(500.0),
		AIProvider:        "claude",
		StrategyMode:      trading_pair.StrategyAuto,
		Timeframes:        []string{"1h"},
		IsActive:          false,
		IsPaused:          false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	err := repo.Create(ctx, inactivePair)
	require.NoError(t, err)

	// Test GetActiveBySymbol
	pairs, err := repo.GetActiveBySymbol(ctx, symbol)
	require.NoError(t, err)
	assert.Len(t, pairs, 3, "Should return only active pairs for symbol")

	// Verify all are active and for correct symbol
	for _, p := range pairs {
		assert.Equal(t, symbol, p.Symbol)
		assert.True(t, p.IsActive)
	}
}

func TestTradingPairRepository_StrategyModes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewTradingPairRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()
	exchangeAccountID := fixtures.CreateExchangeAccount(userID)

	// Test all strategy modes with different symbols to avoid unique constraint
	modes := []struct {
		mode   trading_pair.StrategyMode
		symbol string
	}{
		{trading_pair.StrategyAuto, "BTC/USDT"},
		{trading_pair.StrategySemiAuto, "ETH/USDT"},
		{trading_pair.StrategySignals, "SOL/USDT"},
	}

	for _, test := range modes {
		pair := &trading_pair.TradingPair{
			ID:                uuid.New(),
			UserID:            userID,
			ExchangeAccountID: exchangeAccountID,
			Symbol:            test.symbol,
			MarketType:        trading_pair.MarketSpot,
			Budget:            decimal.NewFromFloat(1000.0),
			MaxPositionSize:   decimal.NewFromFloat(500.0),
			AIProvider:        "claude",
			StrategyMode:      test.mode,
			Timeframes:        []string{"1h"},
			IsActive:          true,
			IsPaused:          false,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := repo.Create(ctx, pair)
		require.NoError(t, err, "Should create pair with mode: "+test.mode.String())

		// Verify mode
		retrieved, err := repo.GetByID(ctx, pair.ID)
		require.NoError(t, err)
		assert.Equal(t, test.mode, retrieved.StrategyMode)
	}
}

func TestTradingPairRepository_FuturesWithLeverage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewTradingPairRepository(testDB.DB())
	ctx := context.Background()

	// Setup test data using fixtures
	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()
	exchangeAccountID := fixtures.CreateExchangeAccount(userID)

	// Create futures pair with leverage
	pair := &trading_pair.TradingPair{
		ID:                uuid.New(),
		UserID:            userID,
		ExchangeAccountID: exchangeAccountID,
		Symbol:            "BTC/USDT",
		MarketType:        trading_pair.MarketFutures,
		Budget:            decimal.NewFromFloat(5000.0),
		MaxPositionSize:   decimal.NewFromFloat(2500.0),
		MaxLeverage:       10,
		StopLossPercent:   decimal.NewFromFloat(1.5),
		TakeProfitPercent: decimal.NewFromFloat(3.0),
		AIProvider:        "claude",
		StrategyMode:      trading_pair.StrategyAuto,
		Timeframes:        []string{"5m", "15m", "1h"},
		IsActive:          true,
		IsPaused:          false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := repo.Create(ctx, pair)
	require.NoError(t, err)

	// Verify futures-specific settings
	retrieved, err := repo.GetByID(ctx, pair.ID)
	require.NoError(t, err)
	assert.Equal(t, trading_pair.MarketFutures, retrieved.MarketType)
	assert.Equal(t, 10, retrieved.MaxLeverage)
	assert.True(t, decimal.NewFromFloat(1.5).Equal(retrieved.StopLossPercent))
}
