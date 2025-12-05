package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/order"
	"prometheus/internal/testsupport"
)

// TestOrderRepository_GetPending tests fetching pending orders
func TestOrderRepository_GetPending(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.Tx())
	userID, accountID, strategyID := fixtures.WithFullStack()

	repo := NewOrderRepository(testDB.Tx())
	ctx := context.Background()

	// Create 3 pending orders
	fixtures.CreateOrder(userID, strategyID, accountID, WithOrderStatus("pending"), func(f *OrderFixture) {
		f.Symbol = "BTC/USDT"
	})
	fixtures.CreateOrder(userID, strategyID, accountID, WithOrderStatus("pending"), func(f *OrderFixture) {
		f.Symbol = "ETH/USDT"
	})
	fixtures.CreateOrder(userID, strategyID, accountID, WithOrderStatus("pending"), func(f *OrderFixture) {
		f.Symbol = "SOL/USDT"
	})

	// Create 2 filled orders (should not be returned)
	fixtures.CreateOrder(userID, strategyID, accountID, WithOrderStatus("filled"), func(f *OrderFixture) {
		f.Symbol = "BNB/USDT"
	})
	fixtures.CreateOrder(userID, strategyID, accountID, WithOrderStatus("filled"), func(f *OrderFixture) {
		f.Symbol = "AVAX/USDT"
	})

	// Create 1 rejected order (should not be returned)
	fixtures.CreateOrder(userID, strategyID, accountID, WithOrderStatus("rejected"), func(f *OrderFixture) {
		f.Symbol = "DOT/USDT"
	})

	// Test GetPending
	t.Run("GetPending returns only pending orders", func(t *testing.T) {
		pendingOrders, err := repo.GetPending(ctx, 10)
		require.NoError(t, err)

		// Should return exactly 3 pending orders
		assert.Len(t, pendingOrders, 3)

		// All should have pending status
		for _, ord := range pendingOrders {
			assert.Equal(t, order.OrderStatusPending, ord.Status)
		}
	})

	t.Run("GetPending respects limit", func(t *testing.T) {
		pendingOrders, err := repo.GetPending(ctx, 2)
		require.NoError(t, err)

		// Should return only 2 orders (limit applied)
		assert.Len(t, pendingOrders, 2)
	})
}

// TestOrderRepository_GetPendingByUser tests fetching pending orders for specific user
func TestOrderRepository_GetPendingByUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.Tx())

	// Create two users with their own exchange accounts and strategies
	user1ID, account1ID, strategy1ID := fixtures.WithFullStack()
	user2ID, account2ID, strategy2ID := fixtures.WithFullStack()

	repo := NewOrderRepository(testDB.Tx())
	ctx := context.Background()

	// Create pending orders for user1
	fixtures.CreateOrder(user1ID, strategy1ID, account1ID, WithOrderStatus("pending"), func(f *OrderFixture) {
		f.Symbol = "BTC/USDT"
	})
	fixtures.CreateOrder(user1ID, strategy1ID, account1ID, WithOrderStatus("pending"), func(f *OrderFixture) {
		f.Symbol = "ETH/USDT"
	})

	// Create pending order for user2
	fixtures.CreateOrder(user2ID, strategy2ID, account2ID, WithOrderStatus("pending"), func(f *OrderFixture) {
		f.Symbol = "SOL/USDT"
	})

	// Create filled order for user1 (should not be returned)
	fixtures.CreateOrder(user1ID, strategy1ID, account1ID, WithOrderStatus("filled"), func(f *OrderFixture) {
		f.Symbol = "BNB/USDT"
	})

	// Test GetPendingByUser for user1
	t.Run("GetPendingByUser returns only user's pending orders", func(t *testing.T) {
		user1Pending, err := repo.GetPendingByUser(ctx, user1ID)
		require.NoError(t, err)

		assert.Len(t, user1Pending, 2)
		for _, ord := range user1Pending {
			assert.Equal(t, user1ID, ord.UserID)
			assert.Equal(t, order.OrderStatusPending, ord.Status)
		}
	})

	// Test GetPendingByUser for user2
	t.Run("GetPendingByUser for different user", func(t *testing.T) {
		user2Pending, err := repo.GetPendingByUser(ctx, user2ID)
		require.NoError(t, err)

		assert.Len(t, user2Pending, 1)
		assert.Equal(t, user2ID, user2Pending[0].UserID)
	})

	// Test with non-existent user
	t.Run("GetPendingByUser returns empty for non-existent user", func(t *testing.T) {
		nonExistentUser := uuid.New()
		orders, err := repo.GetPendingByUser(ctx, nonExistentUser)
		require.NoError(t, err)
		assert.Empty(t, orders)
	})
}
