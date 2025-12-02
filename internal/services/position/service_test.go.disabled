package position

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/position"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/logger"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// MockPositionRepository is a mock for position.Repository
type MockPositionRepository struct {
	mock.Mock
}

func (m *MockPositionRepository) Create(ctx context.Context, pos *position.Position) error {
	args := m.Called(ctx, pos)
	return args.Error(0)
}

func (m *MockPositionRepository) Update(ctx context.Context, pos *position.Position) error {
	args := m.Called(ctx, pos)
	return args.Error(0)
}

func (m *MockPositionRepository) GetByID(ctx context.Context, id uuid.UUID) (*position.Position, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*position.Position), args.Error(1)
}

func (m *MockPositionRepository) GetOpenByUser(ctx context.Context, userID uuid.UUID) ([]*position.Position, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*position.Position), args.Error(1)
}

func (m *MockPositionRepository) GetAllByUser(ctx context.Context, userID uuid.UUID) ([]*position.Position, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*position.Position), args.Error(1)
}

func (m *MockPositionRepository) Close(ctx context.Context, id uuid.UUID, exitPrice, realizedPnL decimal.Decimal) error {
	args := m.Called(ctx, id, exitPrice, realizedPnL)
	return args.Error(0)
}

func (m *MockPositionRepository) UpdatePnL(ctx context.Context, id uuid.UUID, currentPrice, unrealizedPnL, unrealizedPnLPct decimal.Decimal) error {
	args := m.Called(ctx, id, currentPrice, unrealizedPnL, unrealizedPnLPct)
	return args.Error(0)
}

func (m *MockPositionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPositionRepository) GetClosedInRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]*position.Position, error) {
	args := m.Called(ctx, userID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*position.Position), args.Error(1)
}

func (m *MockPositionRepository) GetByTradingPair(ctx context.Context, tradingPairID uuid.UUID) ([]*position.Position, error) {
	args := m.Called(ctx, tradingPairID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*position.Position), args.Error(1)
}

func TestService_UpdateFromWebSocket_CreateNewPosition(t *testing.T) {
	// Setup
	mockRepo := new(MockPositionRepository)
	log := logger.Get()
	service := NewService(mockRepo, log)

	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()

	update := &PositionWebSocketUpdate{
		UserID:        userID,
		AccountID:     accountID,
		Symbol:        "BTCUSDT",
		Side:          "LONG",
		Amount:        decimal.NewFromFloat(0.5),
		EntryPrice:    decimal.NewFromFloat(50000),
		MarkPrice:     decimal.NewFromFloat(50100),
		UnrealizedPnL: decimal.NewFromFloat(50),
	}

	// Mock: no existing positions
	mockRepo.On("GetOpenByUser", ctx, userID).Return([]*position.Position{}, nil)
	mockRepo.On("Create", ctx, mock.AnythingOfType("*position.Position")).Return(nil)

	// Execute
	err := service.UpdateFromWebSocket(ctx, update)

	// Assert
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)

	// Verify Create was called with correct data
	calls := mockRepo.Calls
	assert.Len(t, calls, 2) // GetOpenByUser + Create
	createCall := calls[1]
	createdPos := createCall.Arguments.Get(1).(*position.Position)
	assert.Equal(t, userID, createdPos.UserID)
	assert.Equal(t, accountID, createdPos.ExchangeAccountID)
	assert.Equal(t, "BTCUSDT", createdPos.Symbol)
	assert.Equal(t, position.PositionLong, createdPos.Side)
	assert.Equal(t, update.Amount, createdPos.Size)
	assert.Equal(t, position.PositionOpen, createdPos.Status)
}

func TestService_UpdateFromWebSocket_UpdateExistingPosition(t *testing.T) {
	// Setup
	mockRepo := new(MockPositionRepository)
	log := logger.Get()
	service := NewService(mockRepo, log)

	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()
	positionID := uuid.New()

	existingPos := &position.Position{
		ID:                positionID,
		UserID:            userID,
		ExchangeAccountID: accountID,
		Symbol:            "BTCUSDT",
		Side:              position.PositionLong,
		Size:              decimal.NewFromFloat(0.3),
		EntryPrice:        decimal.NewFromFloat(50000),
		CurrentPrice:      decimal.NewFromFloat(50100),
		UnrealizedPnL:     decimal.NewFromFloat(30),
		Status:            position.PositionOpen,
	}

	update := &PositionWebSocketUpdate{
		UserID:        userID,
		AccountID:     accountID,
		Symbol:        "BTCUSDT",
		Side:          "LONG",
		Amount:        decimal.NewFromFloat(0.5), // increased
		EntryPrice:    decimal.NewFromFloat(50000),
		MarkPrice:     decimal.NewFromFloat(50200), // price moved up
		UnrealizedPnL: decimal.NewFromFloat(100),   // more profit
	}

	// Mock: return existing position
	mockRepo.On("GetOpenByUser", ctx, userID).Return([]*position.Position{existingPos}, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*position.Position")).Return(nil)

	// Execute
	err := service.UpdateFromWebSocket(ctx, update)

	// Assert
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)

	// Verify Update was called and position was updated
	assert.Equal(t, update.Amount, existingPos.Size)
	assert.Equal(t, update.MarkPrice, existingPos.CurrentPrice)
	assert.Equal(t, update.UnrealizedPnL, existingPos.UnrealizedPnL)
}

func TestService_UpdateFromWebSocket_ClosePosition(t *testing.T) {
	// Setup
	mockRepo := new(MockPositionRepository)
	log := logger.Get()
	service := NewService(mockRepo, log)

	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()
	positionID := uuid.New()

	existingPos := &position.Position{
		ID:                positionID,
		UserID:            userID,
		ExchangeAccountID: accountID,
		Symbol:            "BTCUSDT",
		Side:              position.PositionLong,
		Size:              decimal.NewFromFloat(0.5),
		Status:            position.PositionOpen,
	}

	update := &PositionWebSocketUpdate{
		UserID:    userID,
		AccountID: accountID,
		Symbol:    "BTCUSDT",
		Amount:    decimal.Zero, // position closed
	}

	// Mock: return existing position
	mockRepo.On("GetOpenByUser", ctx, userID).Return([]*position.Position{existingPos}, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*position.Position")).Return(nil)

	// Execute
	err := service.UpdateFromWebSocket(ctx, update)

	// Assert
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)

	// Verify position was closed
	assert.Equal(t, position.PositionClosed, existingPos.Status)
}

func TestService_ClosePosition_Success(t *testing.T) {
	// Setup
	mockRepo := new(MockPositionRepository)
	log := logger.Get()
	service := NewService(mockRepo, log)

	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()
	positionID := uuid.New()

	existingPos := &position.Position{
		ID:                positionID,
		UserID:            userID,
		ExchangeAccountID: accountID,
		Symbol:            "BTCUSDT",
		Status:            position.PositionOpen,
	}

	// Mock
	mockRepo.On("GetOpenByUser", ctx, userID).Return([]*position.Position{existingPos}, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*position.Position")).Return(nil)

	// Execute
	err := service.ClosePosition(ctx, userID, accountID, "BTCUSDT")

	// Assert
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	assert.Equal(t, position.PositionClosed, existingPos.Status)
}

func TestService_ClosePosition_NotFound(t *testing.T) {
	// Setup
	mockRepo := new(MockPositionRepository)
	log := logger.Get()
	service := NewService(mockRepo, log)

	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()

	// Mock: no positions
	mockRepo.On("GetOpenByUser", ctx, userID).Return([]*position.Position{}, nil)

	// Execute
	err := service.ClosePosition(ctx, userID, accountID, "BTCUSDT")

	// Assert - should not error when position not found
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestParsePositionUpdateFromEvent(t *testing.T) {
	tests := []struct {
		name    string
		event   *eventspb.UserDataPositionUpdateEvent
		wantErr bool
	}{
		{
			name: "valid event",
			event: &eventspb.UserDataPositionUpdateEvent{
				Base: &eventspb.BaseEvent{
					UserId: uuid.New().String(),
				},
				AccountId:     uuid.New().String(),
				Symbol:        "BTCUSDT",
				Side:          "LONG",
				Amount:        "0.5",
				EntryPrice:    "50000",
				MarkPrice:     "50100",
				UnrealizedPnl: "50",
				EventTime:     timestamppb.Now(),
			},
			wantErr: false,
		},
		{
			name: "invalid user_id",
			event: &eventspb.UserDataPositionUpdateEvent{
				Base: &eventspb.BaseEvent{
					UserId: "invalid-uuid",
				},
				AccountId: uuid.New().String(),
			},
			wantErr: true,
		},
		{
			name: "invalid account_id",
			event: &eventspb.UserDataPositionUpdateEvent{
				Base: &eventspb.BaseEvent{
					UserId: uuid.New().String(),
				},
				AccountId: "invalid-uuid",
			},
			wantErr: true,
		},
		{
			name: "invalid amount",
			event: &eventspb.UserDataPositionUpdateEvent{
				Base: &eventspb.BaseEvent{
					UserId: uuid.New().String(),
				},
				AccountId: uuid.New().String(),
				Amount:    "not-a-number",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePositionUpdateFromEvent(tt.event)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.event.Symbol, result.Symbol)
				assert.Equal(t, tt.event.Side, result.Side)
			}
		})
	}
}

func TestMapPositionSide(t *testing.T) {
	tests := []struct {
		input    string
		expected position.PositionSide
	}{
		{"LONG", position.PositionLong},
		{"SHORT", position.PositionShort},
		{"", position.PositionLong},        // default
		{"INVALID", position.PositionLong}, // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapPositionSide(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseDecimal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    decimal.Decimal
		wantErr bool
	}{
		{"valid decimal", "123.45", decimal.NewFromFloat(123.45), false},
		{"zero string", "0", decimal.Zero, false},
		{"empty string", "", decimal.Zero, false},
		{"integer", "100", decimal.NewFromInt(100), false},
		{"invalid", "not-a-number", decimal.Zero, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDecimal(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, tt.want.Equal(result), "expected %s, got %s", tt.want, result)
			}
		})
	}
}
