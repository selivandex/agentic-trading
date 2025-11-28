package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/journal"
	"prometheus/internal/testsupport"
)

func TestJournalRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewJournalRepository(testDB.DB())
	ctx := context.Background()

	tradeID := uuid.New()

	entry := &journal.JournalEntry{
		ID:              uuid.New(),
		UserID:          userID,
		TradeID:         tradeID,
		Symbol:          "BTC/USDT",
		Side:            "long",
		EntryPrice:      decimal.NewFromFloat(40000.0),
		ExitPrice:       decimal.NewFromFloat(42000.0),
		Size:            decimal.NewFromFloat(1.0),
		PnL:             decimal.NewFromFloat(2000.0),
		PnLPercent:      decimal.NewFromFloat(5.0),
		StrategyUsed:    "momentum_breakout",
		Timeframe:       "4h",
		SetupType:       "breakout",
		MarketRegime:    "trend",
		EntryReasoning:  "Strong volume breakout above resistance",
		ExitReasoning:   "Target reached at 5% profit",
		ConfidenceScore: 0.85,
		RSIAtEntry:      65.5,
		ATRAtEntry:      1200.0,
		VolumeAtEntry:   1500000.0,
		WasCorrectEntry: true,
		WasCorrectExit:  true,
		MaxDrawdown:     decimal.NewFromFloat(-500.0),
		MaxProfit:       decimal.NewFromFloat(2100.0),
		HoldDuration:    14400, // 4 hours in seconds
		LessonsLearned:  "Breakout with volume confirmation worked well",
		ImprovementTips: "Consider wider profit targets in strong trends",
		CreatedAt:       time.Now(),
	}

	// Test Create
	err := repo.Create(ctx, entry)
	require.NoError(t, err, "Create should not return error")

	// Verify entry can be retrieved
	retrieved, err := repo.GetByID(ctx, entry.ID)
	require.NoError(t, err)
	assert.Equal(t, entry.Symbol, retrieved.Symbol)
	assert.Equal(t, entry.StrategyUsed, retrieved.StrategyUsed)
	assert.True(t, entry.PnL.Equal(retrieved.PnL))
}

func TestJournalRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewJournalRepository(testDB.DB())
	ctx := context.Background()

	entry := &journal.JournalEntry{
		ID:              uuid.New(),
		UserID:          userID,
		TradeID:         uuid.New(),
		Symbol:          "ETH/USDT",
		Side:            "short",
		EntryPrice:      decimal.NewFromFloat(2500.0),
		ExitPrice:       decimal.NewFromFloat(2400.0),
		Size:            decimal.NewFromFloat(5.0),
		PnL:             decimal.NewFromFloat(500.0),
		PnLPercent:      decimal.NewFromFloat(4.0),
		StrategyUsed:    "mean_reversion",
		Timeframe:       "1h",
		SetupType:       "reversal",
		MarketRegime:    "range",
		EntryReasoning:  "RSI overbought, resistance rejection",
		ExitReasoning:   "Support level reached",
		ConfidenceScore: 0.75,
		RSIAtEntry:      72.0,
		WasCorrectEntry: true,
		WasCorrectExit:  true,
		HoldDuration:    3600,
		LessonsLearned:  "Mean reversion in range works",
		CreatedAt:       time.Now(),
	}

	err := repo.Create(ctx, entry)
	require.NoError(t, err)

	// Test GetByID
	retrieved, err := repo.GetByID(ctx, entry.ID)
	require.NoError(t, err)
	assert.Equal(t, entry.ID, retrieved.ID)
	assert.Equal(t, entry.Symbol, retrieved.Symbol)
	assert.Equal(t, "mean_reversion", retrieved.StrategyUsed)
	assert.Equal(t, 72.0, retrieved.RSIAtEntry)

	// Test non-existent ID
	_, err = repo.GetByID(ctx, uuid.New())
	assert.Error(t, err, "Should return error for non-existent ID")
}

func TestJournalRepository_GetEntriesSince(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewJournalRepository(testDB.DB())
	ctx := context.Background()
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)

	// Create entries at different times
	times := []time.Time{twoDaysAgo, yesterday, now}
	for i, createdAt := range times {
		entry := &journal.JournalEntry{
			ID:              uuid.New(),
			UserID:          userID,
			TradeID:         uuid.New(),
			Symbol:          "BTC/USDT",
			Side:            "long",
			EntryPrice:      decimal.NewFromFloat(40000.0 + float64(i*1000)),
			ExitPrice:       decimal.NewFromFloat(41000.0 + float64(i*1000)),
			Size:            decimal.NewFromFloat(1.0),
			PnL:             decimal.NewFromFloat(1000.0),
			PnLPercent:      decimal.NewFromFloat(2.5),
			StrategyUsed:    "test_strategy",
			Timeframe:       "1h",
			SetupType:       "breakout",
			MarketRegime:    "trend",
			WasCorrectEntry: true,
			WasCorrectExit:  true,
			HoldDuration:    3600,
			CreatedAt:       createdAt,
		}
		err := repo.Create(ctx, entry)
		require.NoError(t, err)
	}

	// Test GetEntriesSince - last 24 hours
	since := yesterday.Add(-1 * time.Hour)
	entries, err := repo.GetEntriesSince(ctx, userID, since)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 2, "Should find entries from last 24 hours")

	// Verify all entries are after 'since'
	for _, entry := range entries {
		assert.True(t, entry.CreatedAt.After(since) || entry.CreatedAt.Equal(since))
	}
}

func TestJournalRepository_GetStrategyStats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewJournalRepository(testDB.DB())
	ctx := context.Background()

	since := time.Now().Add(-30 * 24 * time.Hour) // Last 30 days

	// Create trades for "momentum_breakout" strategy - 3 wins, 2 losses
	momentumTrades := []struct {
		pnl        float64
		pnlPercent float64
	}{
		{2000.0, 5.0},   // win
		{1500.0, 3.75},  // win
		{-800.0, -2.0},  // loss
		{2500.0, 6.25},  // win
		{-1000.0, -2.5}, // loss
	}

	for i, trade := range momentumTrades {
		entry := &journal.JournalEntry{
			ID:              uuid.New(),
			UserID:          userID,
			TradeID:         uuid.New(),
			Symbol:          "BTC/USDT",
			Side:            "long",
			EntryPrice:      decimal.NewFromFloat(40000.0),
			ExitPrice:       decimal.NewFromFloat(40000.0 + trade.pnl),
			Size:            decimal.NewFromFloat(1.0),
			PnL:             decimal.NewFromFloat(trade.pnl),
			PnLPercent:      decimal.NewFromFloat(trade.pnlPercent),
			StrategyUsed:    "momentum_breakout",
			Timeframe:       "4h",
			SetupType:       "breakout",
			MarketRegime:    "trend",
			WasCorrectEntry: trade.pnl > 0,
			WasCorrectExit:  true,
			HoldDuration:    14400,
			CreatedAt:       time.Now().Add(-time.Duration(i) * time.Hour),
		}
		err := repo.Create(ctx, entry)
		require.NoError(t, err)
	}

	// Create trades for "mean_reversion" strategy - 2 wins, 1 loss
	meanReversionTrades := []struct {
		pnl        float64
		pnlPercent float64
	}{
		{800.0, 2.0},    // win
		{-500.0, -1.25}, // loss
		{1200.0, 3.0},   // win
	}

	for i, trade := range meanReversionTrades {
		entry := &journal.JournalEntry{
			ID:              uuid.New(),
			UserID:          userID,
			TradeID:         uuid.New(),
			Symbol:          "ETH/USDT",
			Side:            "short",
			EntryPrice:      decimal.NewFromFloat(2500.0),
			ExitPrice:       decimal.NewFromFloat(2500.0 - trade.pnl/5.0),
			Size:            decimal.NewFromFloat(5.0),
			PnL:             decimal.NewFromFloat(trade.pnl),
			PnLPercent:      decimal.NewFromFloat(trade.pnlPercent),
			StrategyUsed:    "mean_reversion",
			Timeframe:       "1h",
			SetupType:       "reversal",
			MarketRegime:    "range",
			WasCorrectEntry: trade.pnl > 0,
			WasCorrectExit:  true,
			HoldDuration:    3600,
			CreatedAt:       time.Now().Add(-time.Duration(i*2) * time.Hour),
		}
		err := repo.Create(ctx, entry)
		require.NoError(t, err)
	}

	// Test GetStrategyStats
	stats, err := repo.GetStrategyStats(ctx, userID, since)
	require.NoError(t, err)
	assert.Len(t, stats, 2, "Should return stats for 2 strategies")

	// Verify momentum_breakout stats
	var momentumStats *journal.StrategyStats
	for i := range stats {
		if stats[i].StrategyName == "momentum_breakout" {
			momentumStats = &stats[i]
			break
		}
	}
	require.NotNil(t, momentumStats, "Should find momentum_breakout stats")
	assert.Equal(t, 5, momentumStats.TotalTrades)
	assert.Equal(t, 3, momentumStats.WinningTrades)
	assert.Equal(t, 2, momentumStats.LosingTrades)
	assert.Equal(t, 60.0, momentumStats.WinRate) // 3/5 = 60%

	// Verify mean_reversion stats
	var meanRevStats *journal.StrategyStats
	for i := range stats {
		if stats[i].StrategyName == "mean_reversion" {
			meanRevStats = &stats[i]
			break
		}
	}
	require.NotNil(t, meanRevStats, "Should find mean_reversion stats")
	assert.Equal(t, 3, meanRevStats.TotalTrades)
	assert.Equal(t, 2, meanRevStats.WinningTrades)
	assert.Equal(t, 1, meanRevStats.LosingTrades)
	assert.InDelta(t, 66.67, meanRevStats.WinRate, 0.1) // 2/3 ≈ 66.67%
}

func TestJournalRepository_GetByStrategy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewJournalRepository(testDB.DB())
	ctx := context.Background()
	strategy := "trend_following"

	// Create entries for specific strategy
	for i := 0; i < 5; i++ {
		entry := &journal.JournalEntry{
			ID:              uuid.New(),
			UserID:          userID,
			TradeID:         uuid.New(),
			Symbol:          "BTC/USDT",
			Side:            "long",
			EntryPrice:      decimal.NewFromFloat(40000.0),
			ExitPrice:       decimal.NewFromFloat(41000.0),
			Size:            decimal.NewFromFloat(1.0),
			PnL:             decimal.NewFromFloat(1000.0),
			PnLPercent:      decimal.NewFromFloat(2.5),
			StrategyUsed:    strategy,
			Timeframe:       "1d",
			SetupType:       "trend_follow",
			MarketRegime:    "trend",
			WasCorrectEntry: true,
			WasCorrectExit:  true,
			HoldDuration:    86400, // 1 day
			CreatedAt:       time.Now().Add(-time.Duration(i) * time.Hour),
		}
		err := repo.Create(ctx, entry)
		require.NoError(t, err)
	}

	// Test GetByStrategy with limit
	entries, err := repo.GetByStrategy(ctx, userID, strategy, 3)
	require.NoError(t, err)
	assert.Len(t, entries, 3, "Should respect limit")

	// Verify all entries are for correct strategy
	for _, entry := range entries {
		assert.Equal(t, strategy, entry.StrategyUsed)
	}
}

func TestJournalRepository_ExistsForTrade(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewJournalRepository(testDB.DB())
	ctx := context.Background()

	tradeID := uuid.New()

	// Test non-existent trade
	exists, err := repo.ExistsForTrade(ctx, tradeID)
	require.NoError(t, err)
	assert.False(t, exists, "Should return false for non-existent trade")

	// Create journal entry for trade
	entry := &journal.JournalEntry{
		ID:              uuid.New(),
		UserID:          userID,
		TradeID:         tradeID,
		Symbol:          "BTC/USDT",
		Side:            "long",
		EntryPrice:      decimal.NewFromFloat(40000.0),
		ExitPrice:       decimal.NewFromFloat(41000.0),
		Size:            decimal.NewFromFloat(1.0),
		PnL:             decimal.NewFromFloat(1000.0),
		PnLPercent:      decimal.NewFromFloat(2.5),
		StrategyUsed:    "test_strategy",
		Timeframe:       "1h",
		SetupType:       "test",
		MarketRegime:    "trend",
		WasCorrectEntry: true,
		WasCorrectExit:  true,
		HoldDuration:    3600,
		CreatedAt:       time.Now(),
	}

	err = repo.Create(ctx, entry)
	require.NoError(t, err)

	// Test existing trade
	exists, err = repo.ExistsForTrade(ctx, tradeID)
	require.NoError(t, err)
	assert.True(t, exists, "Should return true for existing trade")
}

func TestJournalRepository_GetByUserAndDateRange(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewJournalRepository(testDB.DB())
	ctx := context.Background()
	now := time.Now()
	start := now.Add(-7 * 24 * time.Hour)  // 7 days ago
	end := now.Add(-3 * 24 * time.Hour)    // 3 days ago
	outside := now.Add(-10 * 24 * time.Hour) // 10 days ago (outside range)

	// Create entry outside range
	entryOutside := &journal.JournalEntry{
		ID:              uuid.New(),
		UserID:          userID,
		TradeID:         uuid.New(),
		Symbol:          "BTC/USDT",
		Side:            "long",
		EntryPrice:      decimal.NewFromFloat(40000.0),
		ExitPrice:       decimal.NewFromFloat(41000.0),
		Size:            decimal.NewFromFloat(1.0),
		PnL:             decimal.NewFromFloat(1000.0),
		PnLPercent:      decimal.NewFromFloat(2.5),
		StrategyUsed:    "test_strategy",
		Timeframe:       "1h",
		SetupType:       "test",
		MarketRegime:    "trend",
		WasCorrectEntry: true,
		WasCorrectExit:  true,
		HoldDuration:    3600,
		CreatedAt:       outside,
	}
	err := repo.Create(ctx, entryOutside)
	require.NoError(t, err)

	// Create entries inside range
	for i := 0; i < 3; i++ {
		entry := &journal.JournalEntry{
			ID:              uuid.New(),
			UserID:          userID,
			TradeID:         uuid.New(),
			Symbol:          "BTC/USDT",
			Side:            "long",
			EntryPrice:      decimal.NewFromFloat(40000.0),
			ExitPrice:       decimal.NewFromFloat(41000.0),
			Size:            decimal.NewFromFloat(1.0),
			PnL:             decimal.NewFromFloat(1000.0),
			PnLPercent:      decimal.NewFromFloat(2.5),
			StrategyUsed:    "test_strategy",
			Timeframe:       "1h",
			SetupType:       "test",
			MarketRegime:    "trend",
			WasCorrectEntry: true,
			WasCorrectExit:  true,
			HoldDuration:    3600,
			CreatedAt:       start.Add(time.Duration(i) * 24 * time.Hour),
		}
		err = repo.Create(ctx, entry)
		require.NoError(t, err)
	}

	// Test GetByUserAndDateRange
	entries, err := repo.GetByUserAndDateRange(ctx, userID, start, end)
	require.NoError(t, err)
	assert.Len(t, entries, 3, "Should return only entries in date range")

	// Verify all entries are within range
	for _, entry := range entries {
		assert.True(t, entry.CreatedAt.After(start) || entry.CreatedAt.Equal(start))
		assert.True(t, entry.CreatedAt.Before(end))
	}
}

func TestJournalRepository_LessonsAndImprovement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewJournalRepository(testDB.DB())
	ctx := context.Background()

	// Create entry with detailed lessons
	entry := &journal.JournalEntry{
		ID:              uuid.New(),
		UserID:          userID,
		TradeID:         uuid.New(),
		Symbol:          "BTC/USDT",
		Side:            "long",
		EntryPrice:      decimal.NewFromFloat(40000.0),
		ExitPrice:       decimal.NewFromFloat(38000.0),
		Size:            decimal.NewFromFloat(1.0),
		PnL:             decimal.NewFromFloat(-2000.0),
		PnLPercent:      decimal.NewFromFloat(-5.0),
		StrategyUsed:    "breakout_failed",
		Timeframe:       "4h",
		SetupType:       "breakout",
		MarketRegime:    "range",
		EntryReasoning:  "False breakout above resistance",
		ExitReasoning:   "Stop loss triggered",
		ConfidenceScore: 0.7,
		RSIAtEntry:      75.0,
		WasCorrectEntry: false,
		WasCorrectExit:  true,
		MaxDrawdown:     decimal.NewFromFloat(-2200.0),
		MaxProfit:       decimal.NewFromFloat(300.0),
		HoldDuration:    7200,
		LessonsLearned:  "Breakout failed in ranging market. RSI was overbought. Volume confirmation was missing.",
		ImprovementTips: "Wait for volume confirmation on breakouts. Avoid trading breakouts in range-bound markets. Use RSI filter to avoid overbought entries.",
		CreatedAt:       time.Now(),
	}

	err := repo.Create(ctx, entry)
	require.NoError(t, err)

	// Retrieve and verify lessons
	retrieved, err := repo.GetByID(ctx, entry.ID)
	require.NoError(t, err)
	assert.Contains(t, retrieved.LessonsLearned, "Breakout failed")
	assert.Contains(t, retrieved.LessonsLearned, "RSI was overbought")
	assert.Contains(t, retrieved.ImprovementTips, "Wait for volume confirmation")
	assert.Contains(t, retrieved.ImprovementTips, "Use RSI filter")
	assert.False(t, retrieved.WasCorrectEntry)
	assert.True(t, retrieved.WasCorrectExit)
}

