package trading

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/order"
)

// Mock repositories and services

type mockOrderRepository struct {
	mock.Mock
}

func (m *mockOrderRepository) Create(ctx context.Context, o *order.Order) error {
	args := m.Called(ctx, o)
	return args.Error(0)
}

func (m *mockOrderRepository) CreateBatch(ctx context.Context, orders []*order.Order) error {
	args := m.Called(ctx, orders)
	return args.Error(0)
}

func (m *mockOrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*order.Order, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*order.Order), args.Error(1)
}

func (m *mockOrderRepository) GetByExchangeOrderID(ctx context.Context, exchangeOrderID string) (*order.Order, error) {
	args := m.Called(ctx, exchangeOrderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*order.Order), args.Error(1)
}

func (m *mockOrderRepository) GetOpenByUser(ctx context.Context, userID uuid.UUID) ([]*order.Order, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*order.Order), args.Error(1)
}

func (m *mockOrderRepository) GetByStrategy(ctx context.Context, strategyID uuid.UUID) ([]*order.Order, error) {
	args := m.Called(ctx, strategyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*order.Order), args.Error(1)
}

func (m *mockOrderRepository) GetPending(ctx context.Context, limit int) ([]*order.Order, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*order.Order), args.Error(1)
}

func (m *mockOrderRepository) GetPendingByUser(ctx context.Context, userID uuid.UUID) ([]*order.Order, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*order.Order), args.Error(1)
}

func (m *mockOrderRepository) Update(ctx context.Context, o *order.Order) error {
	args := m.Called(ctx, o)
	return args.Error(0)
}

func (m *mockOrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status order.OrderStatus, filledAmount, avgPrice decimal.Decimal) error {
	args := m.Called(ctx, id, status, filledAmount, avgPrice)
	return args.Error(0)
}

func (m *mockOrderRepository) Cancel(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockOrderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type mockExchangeAccountRepository struct {
	mock.Mock
}

func (m *mockExchangeAccountRepository) Create(ctx context.Context, account *exchange_account.ExchangeAccount) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *mockExchangeAccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*exchange_account.ExchangeAccount, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*exchange_account.ExchangeAccount), args.Error(1)
}

func (m *mockExchangeAccountRepository) GetByUser(ctx context.Context, userID uuid.UUID) ([]*exchange_account.ExchangeAccount, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*exchange_account.ExchangeAccount), args.Error(1)
}

func (m *mockExchangeAccountRepository) GetActiveByUser(ctx context.Context, userID uuid.UUID) ([]*exchange_account.ExchangeAccount, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*exchange_account.ExchangeAccount), args.Error(1)
}

func (m *mockExchangeAccountRepository) GetAllActive(ctx context.Context) ([]*exchange_account.ExchangeAccount, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*exchange_account.ExchangeAccount), args.Error(1)
}

func (m *mockExchangeAccountRepository) Update(ctx context.Context, account *exchange_account.ExchangeAccount) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *mockExchangeAccountRepository) UpdateLastSync(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockExchangeAccountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type mockExchange struct {
	mock.Mock
}

func (m *mockExchange) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockExchange) GetTicker(ctx context.Context, symbol string) (*exchanges.Ticker, error) {
	args := m.Called(ctx, symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*exchanges.Ticker), args.Error(1)
}

func (m *mockExchange) GetOrderBook(ctx context.Context, symbol string, depth int) (*exchanges.OrderBook, error) {
	args := m.Called(ctx, symbol, depth)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*exchanges.OrderBook), args.Error(1)
}

func (m *mockExchange) GetOHLCV(ctx context.Context, symbol string, timeframe string, limit int) ([]exchanges.OHLCV, error) {
	args := m.Called(ctx, symbol, timeframe, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]exchanges.OHLCV), args.Error(1)
}

func (m *mockExchange) GetTrades(ctx context.Context, symbol string, limit int) ([]exchanges.Trade, error) {
	args := m.Called(ctx, symbol, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]exchanges.Trade), args.Error(1)
}

func (m *mockExchange) GetFundingRate(ctx context.Context, symbol string) (*exchanges.FundingRate, error) {
	args := m.Called(ctx, symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*exchanges.FundingRate), args.Error(1)
}

func (m *mockExchange) GetOpenInterest(ctx context.Context, symbol string) (*exchanges.OpenInterest, error) {
	args := m.Called(ctx, symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*exchanges.OpenInterest), args.Error(1)
}

func (m *mockExchange) GetBalance(ctx context.Context) (*exchanges.Balance, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*exchanges.Balance), args.Error(1)
}

func (m *mockExchange) GetPositions(ctx context.Context) ([]exchanges.Position, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]exchanges.Position), args.Error(1)
}

func (m *mockExchange) GetOpenOrders(ctx context.Context, symbol string) ([]exchanges.Order, error) {
	args := m.Called(ctx, symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]exchanges.Order), args.Error(1)
}

func (m *mockExchange) PlaceOrder(ctx context.Context, req *exchanges.OrderRequest) (*exchanges.Order, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*exchanges.Order), args.Error(1)
}

func (m *mockExchange) CancelOrder(ctx context.Context, symbol, orderID string) error {
	args := m.Called(ctx, symbol, orderID)
	return args.Error(0)
}

func (m *mockExchange) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	args := m.Called(ctx, symbol, leverage)
	return args.Error(0)
}

func (m *mockExchange) SetMarginMode(ctx context.Context, symbol string, mode exchanges.MarginMode) error {
	args := m.Called(ctx, symbol, mode)
	return args.Error(0)
}

type mockUserExchangeFactory struct {
	mock.Mock
}

func (m *mockUserExchangeFactory) GetClient(ctx context.Context, account *exchange_account.ExchangeAccount) (exchanges.Exchange, error) {
	args := m.Called(ctx, account)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(exchanges.Exchange), args.Error(1)
}

// Tests

func TestOrderExecutor_Run_Success(t *testing.T) {
	mockOrderRepo := new(mockOrderRepository)
	mockAccountRepo := new(mockExchangeAccountRepository)
	mockFactory := new(mockUserExchangeFactory)
	mockExch := new(mockExchange)

	executor := NewOrderExecutor(
		mockOrderRepo,
		mockAccountRepo,
		mockFactory,
		5*time.Second,
		true,
	)

	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()
	orderID := uuid.New()

	// Create test pending order
	pendingOrder := &order.Order{
		ID:                orderID,
		UserID:            userID,
		ExchangeAccountID: accountID,
		Symbol:            "BTC/USDT",
		MarketType:        "spot",
		Side:              order.OrderSideBuy,
		Type:              order.OrderTypeMarket,
		Status:            order.OrderStatusPending,
		Amount:            decimal.NewFromInt(100),
		Price:             decimal.Zero,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Create test exchange account
	testAccount := &exchange_account.ExchangeAccount{
		ID:       accountID,
		UserID:   userID,
		Exchange: exchange_account.ExchangeBinance,
		IsActive: true,
	}

	// Expected exchange order response
	exchangeOrder := &exchanges.Order{
		ID:       "BINANCE_12345",
		Symbol:   "BTC/USDT",
		Status:   exchanges.OrderStatusFilled,
		Quantity: decimal.NewFromInt(100),
	}

	// Setup mocks
	mockOrderRepo.On("GetPending", ctx, 50).Return([]*order.Order{pendingOrder}, nil)
	mockAccountRepo.On("GetByID", ctx, accountID).Return(testAccount, nil)
	mockFactory.On("GetClient", ctx, testAccount).Return(mockExch, nil)
	mockExch.On("PlaceOrder", ctx, mock.AnythingOfType("*exchanges.OrderRequest")).Return(exchangeOrder, nil)
	mockOrderRepo.On("Update", ctx, mock.AnythingOfType("*order.Order")).Return(nil)

	// Execute
	err := executor.Run(ctx)

	// Assertions
	require.NoError(t, err)
	mockOrderRepo.AssertExpectations(t)
	mockAccountRepo.AssertExpectations(t)
	mockFactory.AssertExpectations(t)
	mockExch.AssertExpectations(t)

	// Verify Update was called with correct status
	mockOrderRepo.AssertCalled(t, "Update", ctx, mock.MatchedBy(func(o *order.Order) bool {
		return o.ID == orderID &&
			o.ExchangeOrderID == "BINANCE_12345" &&
			o.Status == order.OrderStatusFilled
	}))
}

func TestOrderExecutor_Run_NoPendingOrders(t *testing.T) {
	mockOrderRepo := new(mockOrderRepository)
	mockAccountRepo := new(mockExchangeAccountRepository)
	mockFactory := new(mockUserExchangeFactory)

	executor := NewOrderExecutor(
		mockOrderRepo,
		mockAccountRepo,
		mockFactory,
		5*time.Second,
		true,
	)

	ctx := context.Background()

	// Setup: no pending orders
	mockOrderRepo.On("GetPending", ctx, 50).Return([]*order.Order{}, nil)

	// Execute
	err := executor.Run(ctx)

	// Assertions
	require.NoError(t, err)
	mockOrderRepo.AssertExpectations(t)

	// Should not call other methods if no pending orders
	mockAccountRepo.AssertNotCalled(t, "GetByID")
	mockFactory.AssertNotCalled(t, "GetClient")
}

func TestOrderExecutor_Run_InactiveAccount(t *testing.T) {
	mockOrderRepo := new(mockOrderRepository)
	mockAccountRepo := new(mockExchangeAccountRepository)
	mockFactory := new(mockUserExchangeFactory)

	executor := NewOrderExecutor(
		mockOrderRepo,
		mockAccountRepo,
		mockFactory,
		5*time.Second,
		true,
	)

	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()
	orderID := uuid.New()

	pendingOrder := &order.Order{
		ID:                orderID,
		UserID:            userID,
		ExchangeAccountID: accountID,
		Symbol:            "BTC/USDT",
		MarketType:        "spot",
		Side:              order.OrderSideBuy,
		Type:              order.OrderTypeMarket,
		Status:            order.OrderStatusPending,
		Amount:            decimal.NewFromInt(100),
		Reasoning:         "Original reasoning",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	inactiveAccount := &exchange_account.ExchangeAccount{
		ID:       accountID,
		UserID:   userID,
		Exchange: exchange_account.ExchangeBinance,
		IsActive: false, // Account is inactive
	}

	// Setup mocks
	mockOrderRepo.On("GetPending", ctx, 50).Return([]*order.Order{pendingOrder}, nil)
	mockAccountRepo.On("GetByID", ctx, accountID).Return(inactiveAccount, nil)
	mockOrderRepo.On("Update", ctx, mock.MatchedBy(func(o *order.Order) bool {
		return o.Status == order.OrderStatusRejected
	})).Return(nil)

	// Execute
	err := executor.Run(ctx)

	// Should complete iteration (error logged but not returned)
	require.NoError(t, err)

	// Verify order was rejected
	mockOrderRepo.AssertCalled(t, "Update", ctx, mock.MatchedBy(func(o *order.Order) bool {
		return o.ID == orderID &&
			o.Status == order.OrderStatusRejected &&
			o.Reasoning != "Original reasoning" // Should contain rejection reason
	}))

	// Should not call exchange
	mockFactory.AssertNotCalled(t, "GetClient")
}

func TestOrderExecutor_Run_RetryableError(t *testing.T) {
	mockOrderRepo := new(mockOrderRepository)
	mockAccountRepo := new(mockExchangeAccountRepository)
	mockFactory := new(mockUserExchangeFactory)
	mockExch := new(mockExchange)

	executor := NewOrderExecutor(
		mockOrderRepo,
		mockAccountRepo,
		mockFactory,
		5*time.Second,
		true,
	)

	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()
	orderID := uuid.New()

	pendingOrder := &order.Order{
		ID:                orderID,
		UserID:            userID,
		ExchangeAccountID: accountID,
		Symbol:            "BTC/USDT",
		MarketType:        "spot",
		Side:              order.OrderSideBuy,
		Type:              order.OrderTypeMarket,
		Status:            order.OrderStatusPending,
		Amount:            decimal.NewFromInt(100),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	testAccount := &exchange_account.ExchangeAccount{
		ID:       accountID,
		UserID:   userID,
		Exchange: exchange_account.ExchangeBinance,
		IsActive: true,
	}

	// Setup mocks
	mockOrderRepo.On("GetPending", ctx, 50).Return([]*order.Order{pendingOrder}, nil)
	mockAccountRepo.On("GetByID", ctx, accountID).Return(testAccount, nil)
	mockFactory.On("GetClient", ctx, testAccount).Return(mockExch, nil)

	// First 2 calls fail with retryable error, 3rd succeeds
	mockExch.On("PlaceOrder", ctx, mock.AnythingOfType("*exchanges.OrderRequest")).
		Return((*exchanges.Order)(nil), errors.New("timeout")).
		Once()
	mockExch.On("PlaceOrder", ctx, mock.AnythingOfType("*exchanges.OrderRequest")).
		Return((*exchanges.Order)(nil), errors.New("network error")).
		Once()
	mockExch.On("PlaceOrder", ctx, mock.AnythingOfType("*exchanges.OrderRequest")).
		Return(&exchanges.Order{
			ID:     "BINANCE_12345",
			Status: exchanges.OrderStatusFilled,
		}, nil).
		Once()

	mockOrderRepo.On("Update", ctx, mock.AnythingOfType("*order.Order")).Return(nil)

	// Execute
	err := executor.Run(ctx)

	// Should succeed after retries
	require.NoError(t, err)

	// PlaceOrder should be called 3 times (2 failures + 1 success)
	mockExch.AssertNumberOfCalls(t, "PlaceOrder", 3)

	// Final update should have exchange order ID
	mockOrderRepo.AssertCalled(t, "Update", ctx, mock.MatchedBy(func(o *order.Order) bool {
		return o.ExchangeOrderID == "BINANCE_12345"
	}))
}

func TestOrderExecutor_Run_NonRetryableError(t *testing.T) {
	mockOrderRepo := new(mockOrderRepository)
	mockAccountRepo := new(mockExchangeAccountRepository)
	mockFactory := new(mockUserExchangeFactory)
	mockExch := new(mockExchange)

	executor := NewOrderExecutor(
		mockOrderRepo,
		mockAccountRepo,
		mockFactory,
		5*time.Second,
		true,
	)

	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()
	orderID := uuid.New()

	pendingOrder := &order.Order{
		ID:                orderID,
		UserID:            userID,
		ExchangeAccountID: accountID,
		Symbol:            "BTC/USDT",
		MarketType:        "spot",
		Side:              order.OrderSideBuy,
		Type:              order.OrderTypeMarket,
		Status:            order.OrderStatusPending,
		Amount:            decimal.NewFromInt(100),
		Reasoning:         "Original",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	testAccount := &exchange_account.ExchangeAccount{
		ID:       accountID,
		UserID:   userID,
		Exchange: exchange_account.ExchangeBinance,
		IsActive: true,
	}

	// Setup mocks
	mockOrderRepo.On("GetPending", ctx, 50).Return([]*order.Order{pendingOrder}, nil)
	mockAccountRepo.On("GetByID", ctx, accountID).Return(testAccount, nil)
	mockFactory.On("GetClient", ctx, testAccount).Return(mockExch, nil)

	// Non-retryable error (insufficient balance)
	mockExch.On("PlaceOrder", ctx, mock.AnythingOfType("*exchanges.OrderRequest")).
		Return((*exchanges.Order)(nil), errors.New("insufficient balance")).
		Once()

	mockOrderRepo.On("Update", ctx, mock.AnythingOfType("*order.Order")).Return(nil)

	// Execute
	err := executor.Run(ctx)

	// Should complete iteration without error
	require.NoError(t, err)

	// PlaceOrder should be called only once (no retry)
	mockExch.AssertNumberOfCalls(t, "PlaceOrder", 1)

	// Order should be rejected
	mockOrderRepo.AssertCalled(t, "Update", ctx, mock.MatchedBy(func(o *order.Order) bool {
		return o.Status == order.OrderStatusRejected
	}))
}

func TestOrderExecutor_Run_ContextCancellation(t *testing.T) {
	mockOrderRepo := new(mockOrderRepository)
	mockAccountRepo := new(mockExchangeAccountRepository)
	mockFactory := new(mockUserExchangeFactory)

	executor := NewOrderExecutor(
		mockOrderRepo,
		mockAccountRepo,
		mockFactory,
		5*time.Second,
		true,
	)

	// Create cancelable context
	ctx, cancel := context.WithCancel(context.Background())

	userID := uuid.New()
	accountID := uuid.New()

	// Create multiple pending orders
	pendingOrders := []*order.Order{
		{
			ID:                uuid.New(),
			UserID:            userID,
			ExchangeAccountID: accountID,
			Symbol:            "BTC/USDT",
			MarketType:        "spot",
			Side:              order.OrderSideBuy,
			Type:              order.OrderTypeMarket,
			Status:            order.OrderStatusPending,
			Amount:            decimal.NewFromInt(100),
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                uuid.New(),
			UserID:            userID,
			ExchangeAccountID: accountID,
			Symbol:            "ETH/USDT",
			MarketType:        "spot",
			Side:              order.OrderSideBuy,
			Type:              order.OrderTypeMarket,
			Status:            order.OrderStatusPending,
			Amount:            decimal.NewFromInt(200),
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
	}

	mockOrderRepo.On("GetPending", ctx, 50).Return(pendingOrders, nil)

	// Cancel context immediately
	cancel()

	// Execute
	err := executor.Run(ctx)

	// Should return context.Canceled error
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)

	// Should not call exchange operations
	mockAccountRepo.AssertNotCalled(t, "GetByID")
	mockFactory.AssertNotCalled(t, "GetClient")
}
