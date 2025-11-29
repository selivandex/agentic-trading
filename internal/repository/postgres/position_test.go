package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/position"
	"prometheus/internal/testsupport"
)

// Helper to generate unique telegram IDs

func TestPositionRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewPositionRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID, exchangeAccountID, tradingPairID := fixtures.WithFullStack()

	pos := &position.Position{
		ID:                uuid.New(),
		UserID:            userID,
		TradingPairID:     tradingPairID,
		ExchangeAccountID: exchangeAccountID,
		Symbol:            "BTC/USDT",
		MarketType:        "spot",
		Side:              position.PositionLong,
		Size:              decimal.NewFromFloat(0.5),
		EntryPrice:        decimal.NewFromFloat(42000.0),
		CurrentPrice:      decimal.NewFromFloat(42000.0),
		LiquidationPrice:  decimal.Zero,
		Leverage:          1,
		MarginMode:        "cross",
		UnrealizedPnL:     decimal.Zero,
		UnrealizedPnLPct:  decimal.Zero,
		RealizedPnL:       decimal.Zero,
		StopLossPrice:     decimal.NewFromFloat(40000.0),
		TakeProfitPrice:   decimal.NewFromFloat(45000.0),
		OpenReasoning:     "Strong bullish momentum, breakout confirmed",
		Status:            position.PositionOpen,
		OpenedAt:          time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Test Create
	err := repo.Create(ctx, pos)
	require.NoError(t, err, "Create should not return error")

	// Verify position can be retrieved
	retrieved, err := repo.GetByID(ctx, pos.ID)
	require.NoError(t, err)
	assert.Equal(t, pos.Symbol, retrieved.Symbol)
	assert.Equal(t, pos.Side, retrieved.Side)
	assert.True(t, pos.EntryPrice.Equal(retrieved.EntryPrice))
}

func TestPositionRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewPositionRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID, exchangeAccountID, tradingPairID := fixtures.WithFullStack()

	pos := &position.Position{
		ID:                uuid.New(),
		UserID:            userID,
		TradingPairID:     tradingPairID,
		ExchangeAccountID: exchangeAccountID,
		Symbol:            "ETH/USDT",
		MarketType:        "futures",
		Side:              position.PositionShort,
		Size:              decimal.NewFromFloat(5.0),
		EntryPrice:        decimal.NewFromFloat(2500.0),
		CurrentPrice:      decimal.NewFromFloat(2500.0),
		LiquidationPrice:  decimal.NewFromFloat(2700.0),
		Leverage:          3,
		MarginMode:        "isolated",
		UnrealizedPnL:     decimal.Zero,
		UnrealizedPnLPct:  decimal.Zero,
		RealizedPnL:       decimal.Zero,
		OpenReasoning:     "Bearish divergence detected",
		Status:            position.PositionOpen,
		OpenedAt:          time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := repo.Create(ctx, pos)
	require.NoError(t, err)

	// Test GetByID
	retrieved, err := repo.GetByID(ctx, pos.ID)
	require.NoError(t, err)
	assert.Equal(t, pos.ID, retrieved.ID)
	assert.Equal(t, position.PositionShort, retrieved.Side)
	assert.Equal(t, 3, retrieved.Leverage)
	assert.Equal(t, "isolated", retrieved.MarginMode)

	// Test non-existent ID
	_, err = repo.GetByID(ctx, uuid.New())
	assert.Error(t, err, "Should return error for non-existent ID")
}

func TestPositionRepository_GetOpenByUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewPositionRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID, exchangeAccountID, tradingPairID := fixtures.WithFullStack()

	// Create mix of open and closed positions
	symbols := []string{"BTC/USDT", "ETH/USDT", "SOL/USDT"}
	statuses := []position.PositionStatus{
		position.PositionOpen,
		position.PositionOpen,
		position.PositionClosed,
	}

	for i, symbol := range symbols {
		pos := &position.Position{
			ID:                uuid.New(),
			UserID:            userID,
			TradingPairID:     tradingPairID,
			ExchangeAccountID: exchangeAccountID,
			Symbol:            symbol,
			MarketType:        "spot",
			Side:              position.PositionLong,
			Size:              decimal.NewFromFloat(1.0),
			EntryPrice:        decimal.NewFromFloat(1000.0),
			CurrentPrice:      decimal.NewFromFloat(1000.0),
			Status:            statuses[i],
			OpenedAt:          time.Now(),
			UpdatedAt:         time.Now(),
		}

		if statuses[i] == position.PositionClosed {
			now := time.Now()
			pos.ClosedAt = &now
		}

		err := repo.Create(ctx, pos)
		require.NoError(t, err)
	}

	// Test GetOpenByUser - should return only open positions
	openPositions, err := repo.GetOpenByUser(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, openPositions, 2, "Should return only open positions")

	// Verify all returned positions are open
	for _, pos := range openPositions {
		assert.Equal(t, position.PositionOpen, pos.Status)
		assert.Nil(t, pos.ClosedAt)
	}
}

func TestPositionRepository_UpdatePnL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewPositionRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID, exchangeAccountID, tradingPairID := fixtures.WithFullStack()

	pos := &position.Position{
		ID:                uuid.New(),
		UserID:            userID,
		TradingPairID:     tradingPairID,
		ExchangeAccountID: exchangeAccountID,
		Symbol:            "BTC/USDT",
		MarketType:        "spot",
		Side:              position.PositionLong,
		Size:              decimal.NewFromFloat(1.0),
		EntryPrice:        decimal.NewFromFloat(40000.0),
		CurrentPrice:      decimal.NewFromFloat(40000.0),
		UnrealizedPnL:     decimal.Zero,
		UnrealizedPnLPct:  decimal.Zero,
		Status:            position.PositionOpen,
		OpenedAt:          time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := repo.Create(ctx, pos)
	require.NoError(t, err)

	// Test UpdatePnL - price increased
	newPrice := decimal.NewFromFloat(42000.0)
	unrealizedPnL := decimal.NewFromFloat(2000.0) // (42000 - 40000) * 1.0
	unrealizedPnLPct := decimal.NewFromFloat(5.0) // 5% gain

	err = repo.UpdatePnL(ctx, pos.ID, newPrice, unrealizedPnL, unrealizedPnLPct)
	require.NoError(t, err)

	// Verify updates
	retrieved, err := repo.GetByID(ctx, pos.ID)
	require.NoError(t, err)
	assert.True(t, newPrice.Equal(retrieved.CurrentPrice))
	assert.True(t, unrealizedPnL.Equal(retrieved.UnrealizedPnL))
	assert.True(t, unrealizedPnLPct.Equal(retrieved.UnrealizedPnLPct))

	// Test UpdatePnL - price decreased (loss)
	newPrice = decimal.NewFromFloat(38000.0)
	unrealizedPnL = decimal.NewFromFloat(-2000.0) // (38000 - 40000) * 1.0
	unrealizedPnLPct = decimal.NewFromFloat(-5.0) // -5% loss

	err = repo.UpdatePnL(ctx, pos.ID, newPrice, unrealizedPnL, unrealizedPnLPct)
	require.NoError(t, err)

	retrieved, err = repo.GetByID(ctx, pos.ID)
	require.NoError(t, err)
	assert.True(t, newPrice.Equal(retrieved.CurrentPrice))
	assert.True(t, unrealizedPnL.Equal(retrieved.UnrealizedPnL))
	assert.True(t, unrealizedPnLPct.Equal(retrieved.UnrealizedPnLPct))
}

func TestPositionRepository_Close(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewPositionRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID, exchangeAccountID, tradingPairID := fixtures.WithFullStack()

	pos := &position.Position{
		ID:                uuid.New(),
		UserID:            userID,
		TradingPairID:     tradingPairID,
		ExchangeAccountID: exchangeAccountID,
		Symbol:            "ETH/USDT",
		MarketType:        "spot",
		Side:              position.PositionLong,
		Size:              decimal.NewFromFloat(10.0),
		EntryPrice:        decimal.NewFromFloat(2000.0),
		CurrentPrice:      decimal.NewFromFloat(2000.0),
		UnrealizedPnL:     decimal.Zero,
		Status:            position.PositionOpen,
		OpenedAt:          time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := repo.Create(ctx, pos)
	require.NoError(t, err)

	// Test Close with profit
	exitPrice := decimal.NewFromFloat(2200.0)
	realizedPnL := decimal.NewFromFloat(2000.0) // (2200 - 2000) * 10

	err = repo.Close(ctx, pos.ID, exitPrice, realizedPnL)
	require.NoError(t, err)

	// Verify position is closed
	retrieved, err := repo.GetByID(ctx, pos.ID)
	require.NoError(t, err)
	assert.Equal(t, position.PositionClosed, retrieved.Status)
	assert.True(t, exitPrice.Equal(retrieved.CurrentPrice))
	assert.True(t, realizedPnL.Equal(retrieved.RealizedPnL))
	assert.NotNil(t, retrieved.ClosedAt)
}

func TestPositionRepository_UpdatePnLBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewPositionRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID, exchangeAccountID, tradingPairID := fixtures.WithFullStack()

	// Create multiple open positions
	positionIDs := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		pos := &position.Position{
			ID:                uuid.New(),
			UserID:            userID,
			TradingPairID:     tradingPairID,
			ExchangeAccountID: exchangeAccountID,
			Symbol:            "BTC/USDT",
			MarketType:        "spot",
			Side:              position.PositionLong,
			Size:              decimal.NewFromFloat(1.0),
			EntryPrice:        decimal.NewFromFloat(40000.0),
			CurrentPrice:      decimal.NewFromFloat(40000.0),
			UnrealizedPnL:     decimal.Zero,
			UnrealizedPnLPct:  decimal.Zero,
			Status:            position.PositionOpen,
			OpenedAt:          time.Now(),
			UpdatedAt:         time.Now(),
		}
		err := repo.Create(ctx, pos)
		require.NoError(t, err)
		positionIDs[i] = pos.ID
	}

	// Test UpdatePnLBatch
	updates := []PositionPnLUpdate{
		{
			PositionID:       positionIDs[0],
			CurrentPrice:     decimal.NewFromFloat(42000.0),
			UnrealizedPnL:    decimal.NewFromFloat(2000.0),
			UnrealizedPnLPct: decimal.NewFromFloat(5.0),
		},
		{
			PositionID:       positionIDs[1],
			CurrentPrice:     decimal.NewFromFloat(41000.0),
			UnrealizedPnL:    decimal.NewFromFloat(1000.0),
			UnrealizedPnLPct: decimal.NewFromFloat(2.5),
		},
		{
			PositionID:       positionIDs[2],
			CurrentPrice:     decimal.NewFromFloat(38000.0),
			UnrealizedPnL:    decimal.NewFromFloat(-2000.0),
			UnrealizedPnLPct: decimal.NewFromFloat(-5.0),
		},
	}

	err := repo.UpdatePnLBatch(ctx, updates)
	require.NoError(t, err)

	// Verify all updates
	for i, update := range updates {
		retrieved, err := repo.GetByID(ctx, positionIDs[i])
		require.NoError(t, err)
		assert.True(t, update.CurrentPrice.Equal(retrieved.CurrentPrice))
		assert.True(t, update.UnrealizedPnL.Equal(retrieved.UnrealizedPnL))
		assert.True(t, update.UnrealizedPnLPct.Equal(retrieved.UnrealizedPnLPct))
	}
}

func TestPositionRepository_GetByTradingPair(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewPositionRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID, exchangeAccountID, tradingPairID := fixtures.WithFullStack()

	// Create positions for same trading pair
	for i := 0; i < 3; i++ {
		pos := &position.Position{
			ID:                uuid.New(),
			UserID:            userID,
			TradingPairID:     tradingPairID,
			ExchangeAccountID: exchangeAccountID,
			Symbol:            "BTC/USDT",
			MarketType:        "spot",
			Side:              position.PositionLong,
			Size:              decimal.NewFromFloat(1.0),
			EntryPrice:        decimal.NewFromFloat(40000.0),
			CurrentPrice:      decimal.NewFromFloat(40000.0),
			Status:            position.PositionOpen,
			OpenedAt:          time.Now(),
			UpdatedAt:         time.Now(),
		}
		err := repo.Create(ctx, pos)
		require.NoError(t, err)
	}

	// Test GetByTradingPair
	positions, err := repo.GetByTradingPair(ctx, tradingPairID)
	require.NoError(t, err)
	assert.Len(t, positions, 3, "Should return all positions for trading pair")

	// Verify all belong to same trading pair
	for _, pos := range positions {
		assert.Equal(t, tradingPairID, pos.TradingPairID)
	}
}

func TestPositionRepository_ShortPosition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewPositionRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID, exchangeAccountID, tradingPairID := fixtures.WithFullStack()

	// Create short position with leverage
	pos := &position.Position{
		ID:                uuid.New(),
		UserID:            userID,
		TradingPairID:     tradingPairID,
		ExchangeAccountID: exchangeAccountID,
		Symbol:            "BTC/USDT",
		MarketType:        "futures",
		Side:              position.PositionShort,
		Size:              decimal.NewFromFloat(1.0),
		EntryPrice:        decimal.NewFromFloat(45000.0),
		CurrentPrice:      decimal.NewFromFloat(45000.0),
		LiquidationPrice:  decimal.NewFromFloat(48000.0),
		Leverage:          5,
		MarginMode:        "cross",
		UnrealizedPnL:     decimal.Zero,
		UnrealizedPnLPct:  decimal.Zero,
		StopLossPrice:     decimal.NewFromFloat(46000.0),
		TakeProfitPrice:   decimal.NewFromFloat(43000.0),
		OpenReasoning:     "Short from resistance, bearish momentum",
		Status:            position.PositionOpen,
		OpenedAt:          time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := repo.Create(ctx, pos)
	require.NoError(t, err)

	// Verify short position details
	retrieved, err := repo.GetByID(ctx, pos.ID)
	require.NoError(t, err)
	assert.Equal(t, position.PositionShort, retrieved.Side)
	assert.Equal(t, 5, retrieved.Leverage)
	assert.True(t, decimal.NewFromFloat(48000.0).Equal(retrieved.LiquidationPrice))

	// Update PnL - price goes down (profit for short)
	newPrice := decimal.NewFromFloat(43000.0)
	unrealizedPnL := decimal.NewFromFloat(2000.0) // (45000 - 43000) * 1.0 * 5 leverage effect
	unrealizedPnLPct := decimal.NewFromFloat(22.22)

	err = repo.UpdatePnL(ctx, pos.ID, newPrice, unrealizedPnL, unrealizedPnLPct)
	require.NoError(t, err)

	retrieved, err = repo.GetByID(ctx, pos.ID)
	require.NoError(t, err)
	assert.True(t, newPrice.Equal(retrieved.CurrentPrice))
	assert.True(t, unrealizedPnL.Equal(retrieved.UnrealizedPnL))
}

func TestPositionRepository_StopLossTakeProfit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewPositionRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID, exchangeAccountID, tradingPairID := fixtures.WithFullStack()

	// Create SL/TP orders first
	stopLossOrderID := fixtures.CreateOrder(userID, tradingPairID, exchangeAccountID,
		WithOrderSide("sell"), WithOrderStatus("open"))
	takeProfitOrderID := fixtures.CreateOrder(userID, tradingPairID, exchangeAccountID,
		WithOrderSide("sell"), WithOrderStatus("open"))

	pos := &position.Position{
		ID:                uuid.New(),
		UserID:            userID,
		TradingPairID:     tradingPairID,
		ExchangeAccountID: exchangeAccountID,
		Symbol:            "ETH/USDT",
		MarketType:        "futures",
		Side:              position.PositionLong,
		Size:              decimal.NewFromFloat(10.0),
		EntryPrice:        decimal.NewFromFloat(2500.0),
		CurrentPrice:      decimal.NewFromFloat(2500.0),
		Leverage:          2,
		MarginMode:        "isolated",
		StopLossPrice:     decimal.NewFromFloat(2400.0),
		TakeProfitPrice:   decimal.NewFromFloat(2700.0),
		TrailingStopPct:   decimal.NewFromFloat(2.0),
		StopLossOrderID:   &stopLossOrderID,
		TakeProfitOrderID: &takeProfitOrderID,
		Status:            position.PositionOpen,
		OpenedAt:          time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := repo.Create(ctx, pos)
	require.NoError(t, err)

	// Verify SL/TP settings
	retrieved, err := repo.GetByID(ctx, pos.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved.StopLossOrderID)
	assert.NotNil(t, retrieved.TakeProfitOrderID)
	assert.Equal(t, stopLossOrderID, *retrieved.StopLossOrderID)
	assert.Equal(t, takeProfitOrderID, *retrieved.TakeProfitOrderID)
	assert.True(t, decimal.NewFromFloat(2400.0).Equal(retrieved.StopLossPrice))
	assert.True(t, decimal.NewFromFloat(2700.0).Equal(retrieved.TakeProfitPrice))
	assert.True(t, decimal.NewFromFloat(2.0).Equal(retrieved.TrailingStopPct))
}

func TestPositionRepository_GetClosedInRange(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewPositionRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID, exchangeAccountID, tradingPairID := fixtures.WithFullStack()

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)

	// Create closed positions at different times
	closeTimes := []time.Time{yesterday, yesterday, twoDaysAgo}

	for _, closeTime := range closeTimes {
		pos := &position.Position{
			ID:                uuid.New(),
			UserID:            userID,
			TradingPairID:     tradingPairID,
			ExchangeAccountID: exchangeAccountID,
			Symbol:            "BTC/USDT",
			MarketType:        "spot",
			Side:              position.PositionLong,
			Size:              decimal.NewFromFloat(1.0),
			EntryPrice:        decimal.NewFromFloat(40000.0),
			CurrentPrice:      decimal.NewFromFloat(42000.0),
			RealizedPnL:       decimal.NewFromFloat(2000.0),
			Status:            position.PositionClosed,
			OpenedAt:          closeTime.Add(-1 * time.Hour),
			ClosedAt:          &closeTime,
			UpdatedAt:         closeTime,
		}
		err := repo.Create(ctx, pos)
		require.NoError(t, err)
	}

	// Test GetClosedInRange - last 24 hours
	start := yesterday.Add(-1 * time.Hour)
	end := now.Add(1 * time.Hour)

	positions, err := repo.GetClosedInRange(ctx, userID, start, end)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(positions), 2, "Should find positions closed in last 24 hours")

	// Verify all are closed
	for _, pos := range positions {
		assert.Equal(t, position.PositionClosed, pos.Status)
		assert.NotNil(t, pos.ClosedAt)
	}
}
