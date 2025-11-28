package postgres

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// randomTelegramID generates unique telegram IDs for tests
func randomTelegramID() int64 {
	return 100000000 + rand.Int63n(900000000)
}

// TestFixtures provides factory methods for creating test data
type TestFixtures struct {
	db *sqlx.DB
	t  *testing.T
}

// NewTestFixtures creates a new test fixtures factory
func NewTestFixtures(t *testing.T, db *sqlx.DB) *TestFixtures {
	t.Helper()
	return &TestFixtures{
		db: db,
		t:  t,
	}
}

// CreateUser creates a test user in the database
func (f *TestFixtures) CreateUser(opts ...func(*UserFixture)) uuid.UUID {
	f.t.Helper()

	fixture := &UserFixture{
		TelegramID: 100000 + rand.Int63n(900000),
		Username:   fmt.Sprintf("test_user_%d", rand.Intn(999999)),
		IsActive:   true,
		Settings:   "{}",
	}

	for _, opt := range opts {
		opt(fixture)
	}

	id := uuid.New()
	query := `INSERT INTO users (id, telegram_id, telegram_username, first_name, last_name, is_active, is_premium, settings, created_at, updated_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())`

	_, err := f.db.Exec(query, id, fixture.TelegramID, fixture.Username, fixture.FirstName, fixture.LastName,
		fixture.IsActive, fixture.IsPremium, fixture.Settings)
	require.NoError(f.t, err, "Failed to create test user")

	return id
}

// CreateExchangeAccount creates a test exchange account in the database
func (f *TestFixtures) CreateExchangeAccount(userID uuid.UUID, opts ...func(*ExchangeAccountFixture)) uuid.UUID {
	f.t.Helper()

	fixture := &ExchangeAccountFixture{
		Exchange:  "binance",
		Label:     "Test Account",
		IsTestnet: true,
		IsActive:  true,
	}

	for _, opt := range opts {
		opt(fixture)
	}

	id := uuid.New()
	query := `INSERT INTO exchange_accounts (id, user_id, exchange, label, api_key_encrypted, secret_encrypted, is_testnet, is_active, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())`

	_, err := f.db.Exec(query, id, userID, fixture.Exchange, fixture.Label, "encrypted_key", "encrypted_secret",
		fixture.IsTestnet, fixture.IsActive)
	require.NoError(f.t, err, "Failed to create test exchange account")

	return id
}

// CreateTradingPair creates a test trading pair in the database
func (f *TestFixtures) CreateTradingPair(userID, exchangeAccountID uuid.UUID, opts ...func(*TradingPairFixture)) uuid.UUID {
	f.t.Helper()

	fixture := &TradingPairFixture{
		Symbol:     "BTC/USDT",
		MarketType: "spot",
		Budget:     decimal.NewFromFloat(1000.0),
		IsActive:   true,
		IsPaused:   false,
	}

	for _, opt := range opts {
		opt(fixture)
	}

	id := uuid.New()
	query := `INSERT INTO trading_pairs (id, user_id, exchange_account_id, symbol, market_type, budget, 
			  max_position_size, max_leverage, stop_loss_percent, take_profit_percent, ai_provider, strategy_mode, 
			  timeframes, is_active, is_paused, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW(), NOW())`

	_, err := f.db.Exec(query, id, userID, exchangeAccountID, fixture.Symbol, fixture.MarketType, fixture.Budget,
		decimal.NewFromFloat(500.0), 1, decimal.NewFromFloat(2.0), decimal.NewFromFloat(5.0),
		"claude", "auto", "{1h}", fixture.IsActive, fixture.IsPaused)
	require.NoError(f.t, err, "Failed to create test trading pair")

	return id
}

// CreateOrder creates a test order in the database
func (f *TestFixtures) CreateOrder(userID, tradingPairID, exchangeAccountID uuid.UUID, opts ...func(*OrderFixture)) uuid.UUID {
	f.t.Helper()

	fixture := &OrderFixture{
		Symbol:          "BTC/USDT",
		MarketType:      "spot",
		Side:            "buy",
		Type:            "limit",
		Status:          "open",
		Price:           decimal.NewFromFloat(40000.0),
		Amount:          decimal.NewFromFloat(0.1),
		ExchangeOrderID: fmt.Sprintf("ORDER_%d", rand.Intn(999999)),
	}

	for _, opt := range opts {
		opt(fixture)
	}

	id := uuid.New()
	query := `INSERT INTO orders (id, user_id, trading_pair_id, exchange_account_id, exchange_order_id, symbol, 
			  market_type, side, type, status, price, amount, filled_amount, avg_fill_price, agent_id, fee_currency, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, NOW(), NOW())`

	_, err := f.db.Exec(query, id, userID, tradingPairID, exchangeAccountID, fixture.ExchangeOrderID,
		fixture.Symbol, fixture.MarketType, fixture.Side, fixture.Type, fixture.Status, fixture.Price, fixture.Amount,
		decimal.Zero, decimal.Zero, "test_agent", "USDT")
	require.NoError(f.t, err, "Failed to create test order")

	return id
}

// CreatePosition creates a test position in the database
func (f *TestFixtures) CreatePosition(userID, tradingPairID, exchangeAccountID uuid.UUID, opts ...func(*PositionFixture)) uuid.UUID {
	f.t.Helper()

	fixture := &PositionFixture{
		Symbol:     "BTC/USDT",
		MarketType: "spot",
		Side:       "long",
		Size:       decimal.NewFromFloat(1.0),
		EntryPrice: decimal.NewFromFloat(40000.0),
		Status:     "open",
	}

	for _, opt := range opts {
		opt(fixture)
	}

	id := uuid.New()
	query := `INSERT INTO positions (id, user_id, trading_pair_id, exchange_account_id, symbol, market_type, side, 
			  size, entry_price, current_price, leverage, margin_mode, unrealized_pnl, unrealized_pnl_pct, realized_pnl, 
			  status, opened_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, NOW(), NOW())`

	_, err := f.db.Exec(query, id, userID, tradingPairID, exchangeAccountID, fixture.Symbol, fixture.MarketType,
		fixture.Side, fixture.Size, fixture.EntryPrice, fixture.EntryPrice, 1, "cross",
		decimal.Zero, decimal.Zero, decimal.Zero, fixture.Status)
	require.NoError(f.t, err, "Failed to create test position")

	return id
}

// WithFullStack creates user, exchange account, and trading pair - returns all IDs
func (f *TestFixtures) WithFullStack(opts ...func(*StackFixture)) (userID, exchangeAccountID, tradingPairID uuid.UUID) {
	f.t.Helper()

	fixture := &StackFixture{}
	for _, opt := range opts {
		opt(fixture)
	}

	userID = f.CreateUser(fixture.UserOpts...)
	exchangeAccountID = f.CreateExchangeAccount(userID, fixture.ExchangeAccountOpts...)
	tradingPairID = f.CreateTradingPair(userID, exchangeAccountID, fixture.TradingPairOpts...)

	return
}

// Fixture option types
type UserFixture struct {
	TelegramID int64
	Username   string
	FirstName  string
	LastName   string
	IsActive   bool
	IsPremium  bool
	Settings   string
}

type ExchangeAccountFixture struct {
	Exchange  string
	Label     string
	IsTestnet bool
	IsActive  bool
}

type TradingPairFixture struct {
	Symbol     string
	MarketType string
	Budget     decimal.Decimal
	IsActive   bool
	IsPaused   bool
}

type OrderFixture struct {
	Symbol          string
	MarketType      string
	Side            string
	Type            string
	Status          string
	Price           decimal.Decimal
	Amount          decimal.Decimal
	ExchangeOrderID string
}

type PositionFixture struct {
	Symbol     string
	MarketType string
	Side       string
	Size       decimal.Decimal
	EntryPrice decimal.Decimal
	Status     string
}

type StackFixture struct {
	UserOpts            []func(*UserFixture)
	ExchangeAccountOpts []func(*ExchangeAccountFixture)
	TradingPairOpts     []func(*TradingPairFixture)
}

// Option builders for common customizations
func WithTelegramID(id int64) func(*UserFixture) {
	return func(f *UserFixture) {
		f.TelegramID = id
	}
}

func WithUsername(username string) func(*UserFixture) {
	return func(f *UserFixture) {
		f.Username = username
	}
}

func WithPremium(isPremium bool) func(*UserFixture) {
	return func(f *UserFixture) {
		f.IsPremium = isPremium
	}
}

func WithExchange(exchange string) func(*ExchangeAccountFixture) {
	return func(f *ExchangeAccountFixture) {
		f.Exchange = exchange
	}
}

func WithSymbol(symbol string) func(*TradingPairFixture) {
	return func(f *TradingPairFixture) {
		f.Symbol = symbol
	}
}

func WithBudget(budget decimal.Decimal) func(*TradingPairFixture) {
	return func(f *TradingPairFixture) {
		f.Budget = budget
	}
}

func WithPaused(paused bool) func(*TradingPairFixture) {
	return func(f *TradingPairFixture) {
		f.IsPaused = paused
	}
}

func WithOrderSide(side string) func(*OrderFixture) {
	return func(f *OrderFixture) {
		f.Side = side
	}
}

func WithOrderStatus(status string) func(*OrderFixture) {
	return func(f *OrderFixture) {
		f.Status = status
	}
}

func WithPositionSide(side string) func(*PositionFixture) {
	return func(f *PositionFixture) {
		f.Side = side
	}
}

func WithPositionStatus(status string) func(*PositionFixture) {
	return func(f *PositionFixture) {
		f.Status = status
	}
}

// CreateMemory creates a test memory in the database
func (f *TestFixtures) CreateMemory(userID uuid.UUID, opts ...func(*MemoryFixture)) uuid.UUID {
	f.t.Helper()

	fixture := &MemoryFixture{
		Scope:     "user",
		AgentID:   "test_agent",
		Type:      "observation",
		Content:   "Test memory content",
		Symbol:    "BTC/USDT",
		Timeframe: "1h",
	}

	for _, opt := range opts {
		opt(fixture)
	}

	id := uuid.New()
	query := `INSERT INTO memories (
		id, scope, user_id, agent_id, session_id, type, content,
		symbol, timeframe, importance, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())`

	_, err := f.db.Exec(query, id, fixture.Scope, userID, fixture.AgentID, fixture.SessionID,
		fixture.Type, fixture.Content, fixture.Symbol, fixture.Timeframe, 0.5)
	require.NoError(f.t, err, "Failed to create test memory")

	return id
}

// MemoryFixture holds memory configuration
type MemoryFixture struct {
	Scope     string
	AgentID   string
	SessionID string
	Type      string
	Content   string
	Symbol    string
	Timeframe string
}

func WithMemoryScope(scope string) func(*MemoryFixture) {
	return func(f *MemoryFixture) {
		f.Scope = scope
	}
}

func WithMemoryAgent(agentID string) func(*MemoryFixture) {
	return func(f *MemoryFixture) {
		f.AgentID = agentID
	}
}

func WithMemoryType(memType string) func(*MemoryFixture) {
	return func(f *MemoryFixture) {
		f.Type = memType
	}
}
