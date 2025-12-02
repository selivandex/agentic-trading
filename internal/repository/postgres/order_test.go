package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/order"
	"prometheus/internal/testsupport"
)

// Helper to generate unique telegram IDs

func TestOrderRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewOrderRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID, exchangeAccountID, _ := fixtures.WithFullStack()
	strategyID := uuid.New() // Mock strategy ID

	o := &order.Order{
		ID:                uuid.New(),
		UserID:            userID,
		StrategyID:        &strategyID, //      tradingPairID,
		ExchangeAccountID: exchangeAccountID,
		ExchangeOrderID:   "BINANCE_12345",
		Symbol:            "BTC/USDT",
		MarketType:        "spot",
		Side:              order.OrderSideBuy,
		Type:              order.OrderTypeLimit,
		Status:            order.OrderStatusOpen,
		Price:             decimal.NewFromFloat(42000.0),
		Amount:            decimal.NewFromFloat(0.1),
		FilledAmount:      decimal.NewFromFloat(0.0),
		AvgFillPrice:      decimal.NewFromFloat(0.0),
		StopPrice:         decimal.NewFromFloat(0.0),
		ReduceOnly:        false,
		AgentID:           "opportunity_synthesizer",
		Reasoning:         "Strong bullish momentum, RSI oversold recovery",
		Fee:               decimal.NewFromFloat(0.0),
		FeeCurrency:       "USDT",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Test Create
	err := repo.Create(ctx, o)
	require.NoError(t, err, "Create should not return error")

	// Verify order can be retrieved
	retrieved, err := repo.GetByID(ctx, o.ID)
	require.NoError(t, err)
	assert.Equal(t, o.Symbol, retrieved.Symbol)
	assert.Equal(t, o.Side, retrieved.Side)
	assert.True(t, o.Price.Equal(retrieved.Price))
}

func TestOrderRepository_CreateBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewOrderRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID, exchangeAccountID, _ := fixtures.WithFullStack()
	strategyID := uuid.New() // Mock strategy ID

	// Create batch of orders
	orders := []*order.Order{
		{
			ID:                uuid.New(),
			UserID:            userID,
			StrategyID:        &strategyID, //      tradingPairID,
			ExchangeAccountID: exchangeAccountID,
			ExchangeOrderID:   "BATCH_1",
			Symbol:            "BTC/USDT",
			MarketType:        "spot",
			Side:              order.OrderSideBuy,
			Type:              order.OrderTypeLimit,
			Status:            order.OrderStatusOpen,
			Price:             decimal.NewFromFloat(41000.0),
			Amount:            decimal.NewFromFloat(0.1),
			FilledAmount:      decimal.Zero,
			AvgFillPrice:      decimal.Zero,
			AgentID:           "test_agent",
			Reasoning:         "Batch order 1",
			FeeCurrency:       "USDT",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                uuid.New(),
			UserID:            userID,
			StrategyID:        &strategyID, //      tradingPairID,
			ExchangeAccountID: exchangeAccountID,
			ExchangeOrderID:   "BATCH_2",
			Symbol:            "ETH/USDT",
			MarketType:        "spot",
			Side:              order.OrderSideSell,
			Type:              order.OrderTypeMarket,
			Status:            order.OrderStatusPending,
			Price:             decimal.NewFromFloat(2500.0),
			Amount:            decimal.NewFromFloat(1.0),
			FilledAmount:      decimal.Zero,
			AvgFillPrice:      decimal.Zero,
			AgentID:           "test_agent",
			Reasoning:         "Batch order 2",
			FeeCurrency:       "USDT",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                uuid.New(),
			UserID:            userID,
			StrategyID:        &strategyID, //      tradingPairID,
			ExchangeAccountID: exchangeAccountID,
			ExchangeOrderID:   "BATCH_3",
			Symbol:            "SOL/USDT",
			MarketType:        "spot",
			Side:              order.OrderSideBuy,
			Type:              order.OrderTypeLimit,
			Status:            order.OrderStatusOpen,
			Price:             decimal.NewFromFloat(100.0),
			Amount:            decimal.NewFromFloat(10.0),
			FilledAmount:      decimal.Zero,
			AvgFillPrice:      decimal.Zero,
			AgentID:           "test_agent",
			Reasoning:         "Batch order 3",
			FeeCurrency:       "USDT",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
	}

	// Test CreateBatch
	err := repo.CreateBatch(ctx, orders)
	require.NoError(t, err, "CreateBatch should not return error")

	// Verify all orders can be retrieved
	for _, o := range orders {
		retrieved, err := repo.GetByID(ctx, o.ID)
		require.NoError(t, err)
		assert.Equal(t, o.Symbol, retrieved.Symbol)
	}
}

func TestOrderRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewOrderRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID, exchangeAccountID, _ := fixtures.WithFullStack()
	strategyID := uuid.New() // Mock strategy ID

	o := &order.Order{
		ID:                uuid.New(),
		UserID:            userID,
		StrategyID:        &strategyID, //      tradingPairID,
		ExchangeAccountID: exchangeAccountID,
		ExchangeOrderID:   "TEST_ORDER",
		Symbol:            "BTC/USDT",
		MarketType:        "futures",
		Side:              order.OrderSideSell,
		Type:              order.OrderTypeStopMarket,
		Status:            order.OrderStatusOpen,
		Price:             decimal.NewFromFloat(40000.0),
		Amount:            decimal.NewFromFloat(0.5),
		FilledAmount:      decimal.Zero,
		AvgFillPrice:      decimal.Zero,
		StopPrice:         decimal.NewFromFloat(39500.0),
		ReduceOnly:        true,
		AgentID:           "position_manager",
		Reasoning:         "Stop loss protection",
		FeeCurrency:       "USDT",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := repo.Create(ctx, o)
	require.NoError(t, err)

	// Test GetByID
	retrieved, err := repo.GetByID(ctx, o.ID)
	require.NoError(t, err)
	assert.Equal(t, o.ID, retrieved.ID)
	assert.Equal(t, o.Symbol, retrieved.Symbol)
	assert.Equal(t, order.OrderTypeStopMarket, retrieved.Type)
	assert.True(t, retrieved.ReduceOnly)
	assert.True(t, o.StopPrice.Equal(retrieved.StopPrice))

	// Test non-existent ID
	_, err = repo.GetByID(ctx, uuid.New())
	assert.Error(t, err, "Should return error for non-existent ID")
}

func TestOrderRepository_GetByExchangeOrderID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewOrderRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID, exchangeAccountID, _ := fixtures.WithFullStack()
	strategyID := uuid.New() // Mock strategy ID

	exchangeOrderID := "EXCHANGE_ORDER_" + uuid.New().String()[:8]

	o := &order.Order{
		ID:                uuid.New(),
		UserID:            userID,
		StrategyID:        &strategyID, //      tradingPairID,
		ExchangeAccountID: exchangeAccountID,
		ExchangeOrderID:   exchangeOrderID,
		Symbol:            "ETH/USDT",
		MarketType:        "spot",
		Side:              order.OrderSideBuy,
		Type:              order.OrderTypeMarket,
		Status:            order.OrderStatusFilled,
		Price:             decimal.NewFromFloat(2500.0),
		Amount:            decimal.NewFromFloat(2.0),
		FilledAmount:      decimal.NewFromFloat(2.0),
		AvgFillPrice:      decimal.NewFromFloat(2498.5),
		AgentID:           "test_agent",
		FeeCurrency:       "USDT",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := repo.Create(ctx, o)
	require.NoError(t, err)

	// Test GetByExchangeOrderID
	retrieved, err := repo.GetByExchangeOrderID(ctx, exchangeOrderID)
	require.NoError(t, err)
	assert.Equal(t, o.ID, retrieved.ID)
	assert.Equal(t, exchangeOrderID, retrieved.ExchangeOrderID)
}

func TestOrderRepository_GetOpenByUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewOrderRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID, exchangeAccountID, _ := fixtures.WithFullStack()
	strategyID := uuid.New() // Mock strategy ID

	// Create mix of open and filled orders
	statuses := []order.OrderStatus{
		order.OrderStatusOpen,
		order.OrderStatusOpen,
		order.OrderStatusFilled,
		order.OrderStatusPending,
		order.OrderStatusCanceled,
	}

	for i, status := range statuses {
		o := &order.Order{
			ID:                uuid.New(),
			UserID:            userID,
			StrategyID:        &strategyID, //      tradingPairID,
			ExchangeAccountID: exchangeAccountID,
			ExchangeOrderID:   "ORDER_" + string(rune(i+'0')),
			Symbol:            "BTC/USDT",
			MarketType:        "spot",
			Side:              order.OrderSideBuy,
			Type:              order.OrderTypeLimit,
			Status:            status,
			Price:             decimal.NewFromFloat(40000.0 + float64(i*100)),
			Amount:            decimal.NewFromFloat(0.1),
			FilledAmount:      decimal.Zero,
			AvgFillPrice:      decimal.Zero,
			AgentID:           "test_agent",
			FeeCurrency:       "USDT",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}
		err := repo.Create(ctx, o)
		require.NoError(t, err)
	}

	// Test GetOpenByUser - should return only open/pending orders
	openOrders, err := repo.GetOpenByUser(ctx, userID)
	require.NoError(t, err)

	// Should have 3 open/pending orders
	assert.GreaterOrEqual(t, len(openOrders), 3, "Should return open and pending orders")

	// Verify all returned orders are open or pending
	for _, o := range openOrders {
		assert.True(t,
			o.Status == order.OrderStatusOpen || o.Status == order.OrderStatusPending,
			"All returned orders should be open or pending")
	}
}

func TestOrderRepository_UpdateStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewOrderRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID, exchangeAccountID, _ := fixtures.WithFullStack()
	strategyID := uuid.New() // Mock strategy ID

	o := &order.Order{
		ID:                uuid.New(),
		UserID:            userID,
		StrategyID:        &strategyID, //      tradingPairID,
		ExchangeAccountID: exchangeAccountID,
		ExchangeOrderID:   "UPDATE_TEST",
		Symbol:            "BTC/USDT",
		MarketType:        "spot",
		Side:              order.OrderSideBuy,
		Type:              order.OrderTypeLimit,
		Status:            order.OrderStatusOpen,
		Price:             decimal.NewFromFloat(42000.0),
		Amount:            decimal.NewFromFloat(1.0),
		FilledAmount:      decimal.Zero,
		AvgFillPrice:      decimal.Zero,
		AgentID:           "test_agent",
		FeeCurrency:       "USDT",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := repo.Create(ctx, o)
	require.NoError(t, err)

	// Test UpdateStatus - partial fill
	filledAmount := decimal.NewFromFloat(0.5)
	avgPrice := decimal.NewFromFloat(41950.0)
	err = repo.UpdateStatus(ctx, o.ID, order.OrderStatusPartial, filledAmount, avgPrice)
	require.NoError(t, err)

	// Verify updates
	retrieved, err := repo.GetByID(ctx, o.ID)
	require.NoError(t, err)
	assert.Equal(t, order.OrderStatusPartial, retrieved.Status)
	assert.True(t, filledAmount.Equal(retrieved.FilledAmount))
	assert.True(t, avgPrice.Equal(retrieved.AvgFillPrice))

	// Update to filled
	filledAmount = decimal.NewFromFloat(1.0)
	avgPrice = decimal.NewFromFloat(41975.0)
	err = repo.UpdateStatus(ctx, o.ID, order.OrderStatusFilled, filledAmount, avgPrice)
	require.NoError(t, err)

	retrieved, err = repo.GetByID(ctx, o.ID)
	require.NoError(t, err)
	assert.Equal(t, order.OrderStatusFilled, retrieved.Status)
	assert.True(t, filledAmount.Equal(retrieved.FilledAmount))
}

func TestOrderRepository_UpdateStatusBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewOrderRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID, exchangeAccountID, _ := fixtures.WithFullStack()
	strategyID := uuid.New() // Mock strategy ID

	// Create multiple open orders
	orderIDs := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		o := &order.Order{
			ID:                uuid.New(),
			UserID:            userID,
			StrategyID:        &strategyID, //      tradingPairID,
			ExchangeAccountID: exchangeAccountID,
			ExchangeOrderID:   "BATCH_UPDATE_" + string(rune(i+'0')),
			Symbol:            "BTC/USDT",
			MarketType:        "spot",
			Side:              order.OrderSideBuy,
			Type:              order.OrderTypeLimit,
			Status:            order.OrderStatusOpen,
			Price:             decimal.NewFromFloat(40000.0),
			Amount:            decimal.NewFromFloat(0.1),
			FilledAmount:      decimal.Zero,
			AvgFillPrice:      decimal.Zero,
			AgentID:           "test_agent",
			FeeCurrency:       "USDT",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}
		err := repo.Create(ctx, o)
		require.NoError(t, err)
		orderIDs[i] = o.ID
	}

	// Test UpdateStatusBatch
	updates := []OrderStatusUpdate{
		{
			OrderID:      orderIDs[0],
			Status:       order.OrderStatusFilled,
			FilledAmount: decimal.NewFromFloat(0.1),
			AvgFillPrice: decimal.NewFromFloat(39950.0),
		},
		{
			OrderID:      orderIDs[1],
			Status:       order.OrderStatusPartial,
			FilledAmount: decimal.NewFromFloat(0.05),
			AvgFillPrice: decimal.NewFromFloat(40000.0),
		},
		{
			OrderID:      orderIDs[2],
			Status:       order.OrderStatusCanceled,
			FilledAmount: decimal.Zero,
			AvgFillPrice: decimal.Zero,
		},
	}

	err := repo.UpdateStatusBatch(ctx, updates)
	require.NoError(t, err)

	// Verify all updates
	for i, update := range updates {
		retrieved, err := repo.GetByID(ctx, orderIDs[i])
		require.NoError(t, err)
		assert.Equal(t, update.Status, retrieved.Status)
		assert.True(t, update.FilledAmount.Equal(retrieved.FilledAmount))
	}
}

func TestOrderRepository_Cancel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewOrderRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID, exchangeAccountID, _ := fixtures.WithFullStack()
	strategyID := uuid.New() // Mock strategy ID

	o := &order.Order{
		ID:                uuid.New(),
		UserID:            userID,
		StrategyID:        &strategyID, //      tradingPairID,
		ExchangeAccountID: exchangeAccountID,
		ExchangeOrderID:   "CANCEL_TEST",
		Symbol:            "BTC/USDT",
		MarketType:        "spot",
		Side:              order.OrderSideBuy,
		Type:              order.OrderTypeLimit,
		Status:            order.OrderStatusOpen,
		Price:             decimal.NewFromFloat(41000.0),
		Amount:            decimal.NewFromFloat(0.1),
		FilledAmount:      decimal.Zero,
		AvgFillPrice:      decimal.Zero,
		AgentID:           "test_agent",
		FeeCurrency:       "USDT",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := repo.Create(ctx, o)
	require.NoError(t, err)

	// Test Cancel
	err = repo.Cancel(ctx, o.ID)
	require.NoError(t, err)

	// Verify status is canceled
	retrieved, err := repo.GetByID(ctx, o.ID)
	require.NoError(t, err)
	assert.Equal(t, order.OrderStatusCanceled, retrieved.Status)
}

func TestOrderRepository_ParentChildOrders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewOrderRepository(testDB.DB())
	ctx := context.Background()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID, exchangeAccountID, _ := fixtures.WithFullStack()
	strategyID := uuid.New() // Mock strategy ID

	// Create parent order (main position)
	parentOrder := &order.Order{
		ID:                uuid.New(),
		UserID:            userID,
		StrategyID:        &strategyID, //      tradingPairID,
		ExchangeAccountID: exchangeAccountID,
		ExchangeOrderID:   "PARENT_ORDER",
		Symbol:            "BTC/USDT",
		MarketType:        "futures",
		Side:              order.OrderSideBuy,
		Type:              order.OrderTypeMarket,
		Status:            order.OrderStatusFilled,
		Price:             decimal.NewFromFloat(42000.0),
		Amount:            decimal.NewFromFloat(1.0),
		FilledAmount:      decimal.NewFromFloat(1.0),
		AvgFillPrice:      decimal.NewFromFloat(41995.0),
		AgentID:           "opportunity_synthesizer",
		Reasoning:         "Entry order",
		FeeCurrency:       "USDT",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := repo.Create(ctx, parentOrder)
	require.NoError(t, err)

	// Create child order (stop loss)
	childOrder := &order.Order{
		ID:                uuid.New(),
		UserID:            userID,
		StrategyID:        &strategyID, //      tradingPairID,
		ExchangeAccountID: exchangeAccountID,
		ExchangeOrderID:   "CHILD_SL_ORDER",
		Symbol:            "BTC/USDT",
		MarketType:        "futures",
		Side:              order.OrderSideSell,
		Type:              order.OrderTypeStopMarket,
		Status:            order.OrderStatusOpen,
		Price:             decimal.Zero,
		Amount:            decimal.NewFromFloat(1.0),
		FilledAmount:      decimal.Zero,
		AvgFillPrice:      decimal.Zero,
		StopPrice:         decimal.NewFromFloat(40000.0),
		ReduceOnly:        true,
		ParentOrderID:     &parentOrder.ID, // Link to parent
		AgentID:           "position_manager",
		Reasoning:         "Stop loss protection",
		FeeCurrency:       "USDT",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err = repo.Create(ctx, childOrder)
	require.NoError(t, err)

	// Verify parent-child relationship
	retrieved, err := repo.GetByID(ctx, childOrder.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved.ParentOrderID)
	assert.Equal(t, parentOrder.ID, *retrieved.ParentOrderID)
}
