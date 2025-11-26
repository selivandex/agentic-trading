package state

import (
	"iter"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/adk/session"
)

func TestStateHelpers_AppLevel(t *testing.T) {
	state := newTestState()

	// Test maintenance mode
	err := SetAppMaintenanceMode(state, true)
	require.NoError(t, err)

	inMaintenance, err := GetAppMaintenanceMode(state)
	require.NoError(t, err)
	assert.True(t, inMaintenance)

	// Test trading enabled
	err = SetAppTradingEnabled(state, false)
	require.NoError(t, err)

	tradingEnabled, err := GetAppTradingEnabled(state)
	require.NoError(t, err)
	assert.False(t, tradingEnabled)
}

func TestStateHelpers_UserLevel(t *testing.T) {
	state := newTestState()

	// Test risk tolerance
	err := SetUserRiskTolerance(state, "aggressive")
	require.NoError(t, err)

	riskLevel, err := GetUserRiskTolerance(state)
	require.NoError(t, err)
	assert.Equal(t, "aggressive", riskLevel)

	// Test PnL
	err = SetUserTotalPnL(state, 1250.50)
	require.NoError(t, err)

	pnl, err := GetUserTotalPnL(state)
	require.NoError(t, err)
	assert.Equal(t, 1250.50, pnl)

	// Test last activity
	now := time.Now()
	err = SetUserLastActivity(state, now)
	require.NoError(t, err)

	lastActivity, err := GetUserLastActivity(state)
	require.NoError(t, err)
	assert.WithinDuration(t, now, lastActivity, time.Second)
}

func TestStateHelpers_SessionLevel(t *testing.T) {
	state := newTestState()

	// Test current symbol
	err := SetCurrentSymbol(state, "BTC/USDT")
	require.NoError(t, err)

	symbol, err := GetCurrentSymbol(state)
	require.NoError(t, err)
	assert.Equal(t, "BTC/USDT", symbol)

	// Test market type
	err = SetCurrentMarketType(state, "futures")
	require.NoError(t, err)

	marketType, err := GetCurrentMarketType(state)
	require.NoError(t, err)
	assert.Equal(t, "futures", marketType)
}

func TestStateHelpers_TemporaryState(t *testing.T) {
	state := newTestState()

	// Test tool call counter
	err := IncrementToolCallCount(state)
	require.NoError(t, err)

	count := GetToolCallCount(state)
	assert.Equal(t, 1, count)

	// Increment again
	err = IncrementToolCallCount(state)
	require.NoError(t, err)

	count = GetToolCallCount(state)
	assert.Equal(t, 2, count)

	// Test temp timestamps
	now := time.Now()
	err = SetTempStartTime(state, now)
	require.NoError(t, err)

	startTime, err := GetTempStartTime(state)
	require.NoError(t, err)
	assert.WithinDuration(t, now, startTime, time.Second)

	// Test token counts
	SetTempPromptTokens(state, 100)
	SetTempCompletionTokens(state, 200)

	promptTokens, completionTokens := GetTempTokens(state)
	assert.Equal(t, 100, promptTokens)
	assert.Equal(t, 200, completionTokens)
}

// newTestState creates a test state implementation
func newTestState() session.State {
	return &testState{data: make(map[string]any)}
}

type testState struct {
	data map[string]any
}

func (s *testState) Get(key string) (any, error) {
	if val, ok := s.data[key]; ok {
		return val, nil
	}
	return nil, session.ErrStateKeyNotExist
}

func (s *testState) Set(key string, val any) error {
	s.data[key] = val
	return nil
}

func (s *testState) All() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for k, v := range s.data {
			if !yield(k, v) {
				return
			}
		}
	}
}
