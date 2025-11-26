package risk

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/position"
	"prometheus/internal/domain/risk"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Mock repositories for testing

type mockRiskRepo struct {
	states map[uuid.UUID]*risk.CircuitBreakerState
	events []*risk.RiskEvent
}

func newMockRiskRepo() *mockRiskRepo {
	return &mockRiskRepo{
		states: make(map[uuid.UUID]*risk.CircuitBreakerState),
		events: make([]*risk.RiskEvent, 0),
	}
}

func (m *mockRiskRepo) GetState(ctx context.Context, userID uuid.UUID) (*risk.CircuitBreakerState, error) {
	state, ok := m.states[userID]
	if !ok {
		return nil, fmt.Errorf("state not found for user %s", userID)
	}
	return state, nil
}

func (m *mockRiskRepo) SaveState(ctx context.Context, state *risk.CircuitBreakerState) error {
	m.states[state.UserID] = state
	return nil
}

func (m *mockRiskRepo) ResetDaily(ctx context.Context) error {
	for userID := range m.states {
		state := m.states[userID]
		state.DailyPnL = decimal.Zero
		state.DailyPnLPercent = decimal.Zero
		state.DailyTradeCount = 0
		state.DailyWins = 0
		state.DailyLosses = 0
		state.ConsecutiveLosses = 0
		state.IsTriggered = false
		state.TriggeredAt = nil
		state.TriggerReason = ""
	}
	return nil
}

func (m *mockRiskRepo) CreateEvent(ctx context.Context, event *risk.RiskEvent) error {
	m.events = append(m.events, event)
	return nil
}

func (m *mockRiskRepo) GetEvents(ctx context.Context, userID uuid.UUID, limit int) ([]*risk.RiskEvent, error) {
	return m.events, nil
}

func (m *mockRiskRepo) AcknowledgeEvent(ctx context.Context, eventID uuid.UUID) error {
	return nil
}

type mockPosRepo struct {
	positions []*position.Position
}

func newMockPosRepo() *mockPosRepo {
	return &mockPosRepo{
		positions: make([]*position.Position, 0),
	}
}

func (m *mockPosRepo) Create(ctx context.Context, pos *position.Position) error {
	m.positions = append(m.positions, pos)
	return nil
}

func (m *mockPosRepo) GetByID(ctx context.Context, id uuid.UUID) (*position.Position, error) {
	return nil, nil
}

func (m *mockPosRepo) GetOpenByUser(ctx context.Context, userID uuid.UUID) ([]*position.Position, error) {
	result := make([]*position.Position, 0)
	for _, pos := range m.positions {
		if pos.UserID == userID && pos.Status == position.PositionOpen {
			result = append(result, pos)
		}
	}
	return result, nil
}

func (m *mockPosRepo) GetByTradingPair(ctx context.Context, tradingPairID uuid.UUID) ([]*position.Position, error) {
	return nil, nil
}

func (m *mockPosRepo) Update(ctx context.Context, pos *position.Position) error {
	return nil
}

func (m *mockPosRepo) UpdatePnL(ctx context.Context, id uuid.UUID, currentPrice, unrealizedPnL, unrealizedPnLPct decimal.Decimal) error {
	return nil
}

func (m *mockPosRepo) Close(ctx context.Context, id uuid.UUID, exitPrice, realizedPnL decimal.Decimal) error {
	return nil
}

func (m *mockPosRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

type mockRedis struct {
	data map[string]interface{}
}

func newMockRedis() *mockRedis {
	return &mockRedis{
		data: make(map[string]interface{}),
	}
}

func (m *mockRedis) Get(ctx context.Context, key string, dest interface{}) error {
	// Return error if not found (Redis behavior)
	_, exists := m.data[key]
	if !exists {
		return fmt.Errorf("redis: key not found")
	}
	return nil
}

func (m *mockRedis) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *mockRedis) Delete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		delete(m.data, key)
	}
	return nil
}

func (m *mockRedis) Exists(ctx context.Context, key string) (bool, error) {
	_, exists := m.data[key]
	return exists, nil
}

// Tests

func TestCanTrade_WithinLimits(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	riskRepo := newMockRiskRepo()
	posRepo := newMockPosRepo()
	log := logger.Get()

	// Create engine (redis mock not needed for this test)
	mockRedisCli := newMockRedis()
	engine := &RiskEngine{
		riskRepo: riskRepo,
		posRepo:  posRepo,
		redis:    mockRedisCli,
		log:      log,
	}

	// Should allow trading for new user
	canTrade, err := engine.CanTrade(ctx, userID)
	require.NoError(t, err)
	assert.True(t, canTrade)
}

func TestCanTrade_ExceedDrawdown(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	riskRepo := newMockRiskRepo()
	posRepo := newMockPosRepo()
	log := logger.Get()

	mockRedisCli := newMockRedis()
	engine := &RiskEngine{
		riskRepo: riskRepo,
		posRepo:  posRepo,
		redis:    mockRedisCli,
		log:      log,
	}

	// Create state with exceeded drawdown
	now := time.Now()
	state := &risk.CircuitBreakerState{
		UserID:             userID,
		IsTriggered:        false,
		DailyPnL:           decimal.NewFromFloat(-11.0), // -11 USDT
		DailyPnLPercent:    decimal.NewFromFloat(-11.0),
		MaxDailyDrawdown:   decimal.NewFromFloat(10.0), // Max drawdown is 10 USDT
		MaxConsecutiveLoss: 3,
		UpdatedAt:          now,
		ResetAt:            now.Add(24 * time.Hour),
	}
	riskRepo.SaveState(ctx, state)

	// Read state before calling CanTrade
	beforeState, _ := riskRepo.GetState(ctx, userID)
	t.Logf("Before CanTrade: triggered=%v, pnl=%s, maxDrawdown=%s",
		beforeState.IsTriggered, beforeState.DailyPnL, beforeState.MaxDailyDrawdown)

	// Should NOT allow trading and return domain error
	canTrade, err := engine.CanTrade(ctx, userID)
	require.Error(t, err, "Should return error when drawdown exceeded")
	assert.ErrorIs(t, err, errors.ErrDrawdownExceeded, "Should return ErrDrawdownExceeded")
	assert.False(t, canTrade, "Should NOT allow trading when drawdown exceeded")

	// Verify circuit breaker was tripped
	updatedState, err := riskRepo.GetState(ctx, userID)
	require.NoError(t, err)
	t.Logf("After CanTrade: triggered=%v, reason='%s'",
		updatedState.IsTriggered, updatedState.TriggerReason)

	assert.True(t, updatedState.IsTriggered, "Circuit breaker should be triggered")
	assert.Contains(t, updatedState.TriggerReason, "drawdown")
}

func TestCanTrade_ConsecutiveLosses(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	riskRepo := newMockRiskRepo()
	posRepo := newMockPosRepo()
	log := logger.Get()

	mockRedisCli := newMockRedis()
	engine := &RiskEngine{
		riskRepo: riskRepo,
		posRepo:  posRepo,
		redis:    mockRedisCli,
		log:      log,
	}

	// Create state with 3 consecutive losses (limit is 3)
	now := time.Now()
	state := &risk.CircuitBreakerState{
		UserID:             userID,
		IsTriggered:        false,
		DailyPnL:           decimal.Zero,
		DailyPnLPercent:    decimal.Zero,
		ConsecutiveLosses:  3,
		MaxDailyDrawdown:   decimal.NewFromFloat(100.0),
		MaxConsecutiveLoss: 3,
		UpdatedAt:          now,
		ResetAt:            now.Add(24 * time.Hour),
	}
	riskRepo.SaveState(ctx, state)

	// Should NOT allow trading and return domain error
	canTrade, err := engine.CanTrade(ctx, userID)
	require.Error(t, err, "Should return error with consecutive losses")
	assert.ErrorIs(t, err, errors.ErrConsecutiveLosses, "Should return ErrConsecutiveLosses")
	assert.False(t, canTrade, "Should NOT allow trading with max consecutive losses")

	// Verify circuit breaker was tripped
	updatedState, err := riskRepo.GetState(ctx, userID)
	require.NoError(t, err)
	assert.True(t, updatedState.IsTriggered, "Circuit breaker should be triggered")
	assert.Contains(t, updatedState.TriggerReason, "Consecutive losses")
}

func TestRecordTrade_UpdatesState(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	riskRepo := newMockRiskRepo()
	posRepo := newMockPosRepo()
	log := logger.Get()

	mockRedisCli := newMockRedis()
	engine := &RiskEngine{
		riskRepo: riskRepo,
		posRepo:  posRepo,
		redis:    mockRedisCli,
		log:      log,
	}

	// Record a winning trade
	err := engine.RecordTrade(ctx, userID, decimal.NewFromFloat(100.0))
	require.NoError(t, err)

	state, err := riskRepo.GetState(ctx, userID)
	require.NoError(t, err)
	assert.True(t, state.DailyPnL.Equal(decimal.NewFromFloat(100.0)), "Daily PnL should be 100")
	assert.Equal(t, 1, state.DailyTradeCount)
	assert.Equal(t, 1, state.DailyWins)
	assert.Equal(t, 0, state.ConsecutiveLosses)

	// Record a losing trade
	err = engine.RecordTrade(ctx, userID, decimal.NewFromFloat(-50.0))
	require.NoError(t, err)

	state, err = riskRepo.GetState(ctx, userID)
	require.NoError(t, err)
	assert.True(t, state.DailyPnL.Equal(decimal.NewFromFloat(50.0)), "Daily PnL should be 50")
	assert.Equal(t, 2, state.DailyTradeCount)
	assert.Equal(t, 1, state.DailyWins)
	assert.Equal(t, 1, state.DailyLosses)
	assert.Equal(t, 1, state.ConsecutiveLosses)
}

func TestRecordTrade_TripsCircuitBreaker(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	riskRepo := newMockRiskRepo()
	posRepo := newMockPosRepo()
	log := logger.Get()

	mockRedisCli := newMockRedis()
	engine := &RiskEngine{
		riskRepo: riskRepo,
		posRepo:  posRepo,
		redis:    mockRedisCli,
		log:      log,
	}

	// Initialize state
	now := time.Now()
	state := &risk.CircuitBreakerState{
		UserID:             userID,
		DailyPnL:           decimal.NewFromFloat(-90.0), // Already at -90
		MaxDailyDrawdown:   decimal.NewFromFloat(100.0), // Limit is -100
		MaxConsecutiveLoss: 3,
		UpdatedAt:          now,
		ResetAt:            now.Add(24 * time.Hour),
	}
	riskRepo.SaveState(ctx, state)

	// Record another losing trade that pushes over the limit
	err := engine.RecordTrade(ctx, userID, decimal.NewFromFloat(-20.0))
	require.NoError(t, err)

	// Verify circuit breaker tripped
	state, err = riskRepo.GetState(ctx, userID)
	require.NoError(t, err)
	assert.True(t, state.IsTriggered)
	assert.NotNil(t, state.TriggeredAt)
}

func TestResetDaily(t *testing.T) {
	ctx := context.Background()
	userID1 := uuid.New()
	userID2 := uuid.New()

	riskRepo := newMockRiskRepo()
	posRepo := newMockPosRepo()
	log := logger.Get()

	mockRedisCli := newMockRedis()
	engine := &RiskEngine{
		riskRepo: riskRepo,
		posRepo:  posRepo,
		redis:    mockRedisCli,
		log:      log,
	}

	// Create states for two users
	now := time.Now()
	state1 := &risk.CircuitBreakerState{
		UserID:            userID1,
		DailyPnL:          decimal.NewFromFloat(-50.0),
		DailyTradeCount:   5,
		ConsecutiveLosses: 2,
		UpdatedAt:         now,
	}
	state2 := &risk.CircuitBreakerState{
		UserID:            userID2,
		DailyPnL:          decimal.NewFromFloat(100.0),
		DailyTradeCount:   3,
		ConsecutiveLosses: 0,
		UpdatedAt:         now,
	}
	riskRepo.SaveState(ctx, state1)
	riskRepo.SaveState(ctx, state2)

	// Reset daily
	err := engine.ResetDaily(ctx)
	require.NoError(t, err)

	// Verify both users were reset
	state1, _ = riskRepo.GetState(ctx, userID1)
	assert.True(t, state1.DailyPnL.IsZero())
	assert.Equal(t, 0, state1.DailyTradeCount)
	assert.Equal(t, 0, state1.ConsecutiveLosses)

	state2, _ = riskRepo.GetState(ctx, userID2)
	assert.True(t, state2.DailyPnL.IsZero())
	assert.Equal(t, 0, state2.DailyTradeCount)
}

func TestCircuitBreaker_ManualTrip(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	riskRepo := newMockRiskRepo()
	posRepo := newMockPosRepo()
	log := logger.Get()

	mockRedisCli := newMockRedis()
	engine := &RiskEngine{
		riskRepo: riskRepo,
		posRepo:  posRepo,
		redis:    mockRedisCli,
		log:      log,
	}

	// Manually trip circuit breaker
	err := engine.TripCircuitBreaker(ctx, userID, "Manual kill switch activated")
	require.NoError(t, err)

	// Verify it's tripped - should return domain error
	canTrade, err := engine.CanTrade(ctx, userID)
	require.Error(t, err, "Should return error when circuit breaker is tripped")
	assert.ErrorIs(t, err, errors.ErrCircuitBreakerTripped)
	assert.False(t, canTrade)

	// Reset it
	err = engine.ResetCircuitBreaker(ctx, userID)
	require.NoError(t, err)

	// Verify it's reset
	canTrade, err = engine.CanTrade(ctx, userID)
	require.NoError(t, err)
	assert.True(t, canTrade)
}

func TestGetUserState(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	riskRepo := newMockRiskRepo()
	posRepo := newMockPosRepo()
	log := logger.Get()

	mockRedisCli := newMockRedis()
	engine := &RiskEngine{
		riskRepo: riskRepo,
		posRepo:  posRepo,
		redis:    mockRedisCli,
		log:      log,
	}

	// Get state for new user
	userState, err := engine.GetUserState(ctx, userID)
	require.NoError(t, err)
	assert.NotNil(t, userState)
	assert.Equal(t, userID, userState.UserID)
	assert.True(t, userState.CanTrade)
	assert.False(t, userState.IsCircuitTripped)
}
