package fund_watchlist

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"prometheus/internal/domain/fund_watchlist"
	"prometheus/pkg/logger"
)

func testLogger() *logger.Logger {
	zapLogger, _ := zap.NewDevelopment()
	return &logger.Logger{SugaredLogger: zapLogger.Sugar()}
}

// MockRepository is a mock for fund_watchlist.Repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, entry *fund_watchlist.FundWatchlist) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*fund_watchlist.FundWatchlist, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*fund_watchlist.FundWatchlist), args.Error(1)
}

func (m *MockRepository) GetBySymbol(ctx context.Context, symbol, marketType string) (*fund_watchlist.FundWatchlist, error) {
	args := m.Called(ctx, symbol, marketType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*fund_watchlist.FundWatchlist), args.Error(1)
}

func (m *MockRepository) GetActive(ctx context.Context) ([]*fund_watchlist.FundWatchlist, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*fund_watchlist.FundWatchlist), args.Error(1)
}

func (m *MockRepository) GetAll(ctx context.Context) ([]*fund_watchlist.FundWatchlist, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*fund_watchlist.FundWatchlist), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, entry *fund_watchlist.FundWatchlist) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) IsActive(ctx context.Context, symbol, marketType string) (bool, error) {
	args := m.Called(ctx, symbol, marketType)
	return args.Bool(0), args.Error(1)
}

func TestService_GetByID(t *testing.T) {
	mockRepo := new(MockRepository)
	log := testLogger()
	svc := NewService(mockRepo, log)

	id := uuid.New()
	expected := &fund_watchlist.FundWatchlist{
		ID:         id,
		Symbol:     "BTC/USDT",
		MarketType: "spot",
		Category:   "major",
		Tier:       1,
		IsActive:   true,
	}

	mockRepo.On("GetByID", mock.Anything, id).Return(expected, nil)

	result, err := svc.GetByID(context.Background(), id)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	mockRepo.AssertExpectations(t)
}

func TestService_GetBySymbol(t *testing.T) {
	mockRepo := new(MockRepository)
	log := testLogger()
	svc := NewService(mockRepo, log)

	symbol := "ETH/USDT"
	marketType := "spot"
	expected := &fund_watchlist.FundWatchlist{
		ID:         uuid.New(),
		Symbol:     symbol,
		MarketType: marketType,
		Category:   "major",
		Tier:       1,
		IsActive:   true,
	}

	mockRepo.On("GetBySymbol", mock.Anything, symbol, marketType).Return(expected, nil)

	result, err := svc.GetBySymbol(context.Background(), symbol, marketType)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	mockRepo.AssertExpectations(t)
}

func TestService_GetMonitored(t *testing.T) {
	tests := []struct {
		name       string
		entries    []*fund_watchlist.FundWatchlist
		marketType *string
		expected   int
	}{
		{
			name: "returns only monitored entries",
			entries: []*fund_watchlist.FundWatchlist{
				{
					ID:         uuid.New(),
					Symbol:     "BTC/USDT",
					MarketType: "spot",
					IsActive:   true,
					IsPaused:   false, // monitored
				},
				{
					ID:         uuid.New(),
					Symbol:     "ETH/USDT",
					MarketType: "spot",
					IsActive:   true,
					IsPaused:   true, // not monitored (paused)
				},
				{
					ID:         uuid.New(),
					Symbol:     "SOL/USDT",
					MarketType: "futures",
					IsActive:   true,
					IsPaused:   false, // monitored
				},
			},
			marketType: nil,
			expected:   2, // BTC and SOL
		},
		{
			name: "filters by market type",
			entries: []*fund_watchlist.FundWatchlist{
				{
					ID:         uuid.New(),
					Symbol:     "BTC/USDT",
					MarketType: "spot",
					IsActive:   true,
					IsPaused:   false,
				},
				{
					ID:         uuid.New(),
					Symbol:     "SOL/USDT",
					MarketType: "futures",
					IsActive:   true,
					IsPaused:   false,
				},
			},
			marketType: stringPtr("spot"),
			expected:   1, // only BTC
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			log := testLogger()
			svc := NewService(mockRepo, log)

			mockRepo.On("GetActive", mock.Anything).Return(tt.entries, nil)

			result, err := svc.GetMonitored(context.Background(), tt.marketType)

			assert.NoError(t, err)
			assert.Len(t, result, tt.expected)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_Create(t *testing.T) {
	mockRepo := new(MockRepository)
	log := testLogger()
	svc := NewService(mockRepo, log)

	entry := &fund_watchlist.FundWatchlist{
		ID:         uuid.New(),
		Symbol:     "BTC/USDT",
		MarketType: "spot",
		Category:   "major",
		Tier:       1,
		IsActive:   true,
	}

	mockRepo.On("Create", mock.Anything, entry).Return(nil)

	err := svc.Create(context.Background(), entry)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_TogglePause(t *testing.T) {
	tests := []struct {
		name     string
		isPaused bool
		reason   *string
	}{
		{
			name:     "pause with reason",
			isPaused: true,
			reason:   stringPtr("Low liquidity detected"),
		},
		{
			name:     "unpause clears reason",
			isPaused: false,
			reason:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			log := testLogger()
			svc := NewService(mockRepo, log)

			id := uuid.New()
			entry := &fund_watchlist.FundWatchlist{
				ID:         id,
				Symbol:     "BTC/USDT",
				MarketType: "spot",
				IsActive:   true,
				IsPaused:   false,
			}

			mockRepo.On("GetByID", mock.Anything, id).Return(entry, nil)
			mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(e *fund_watchlist.FundWatchlist) bool {
				return e.IsPaused == tt.isPaused &&
					((tt.reason == nil && e.PausedReason == nil) ||
						(tt.reason != nil && e.PausedReason != nil && *e.PausedReason == *tt.reason))
			})).Return(nil)

			result, err := svc.TogglePause(context.Background(), id, tt.isPaused, tt.reason)

			assert.NoError(t, err)
			assert.Equal(t, tt.isPaused, result.IsPaused)
			if tt.isPaused && tt.reason != nil {
				assert.NotNil(t, result.PausedReason)
				assert.Equal(t, *tt.reason, *result.PausedReason)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_Delete(t *testing.T) {
	mockRepo := new(MockRepository)
	log := testLogger()
	svc := NewService(mockRepo, log)

	id := uuid.New()

	mockRepo.On("Delete", mock.Anything, id).Return(nil)

	err := svc.Delete(context.Background(), id)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
