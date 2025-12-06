package fundwatchlist

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"prometheus/internal/domain/fundwatchlist"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

func testLogger() *logger.Logger {
	zapLogger, _ := zap.NewDevelopment()
	return &logger.Logger{SugaredLogger: zapLogger.Sugar()}
}

// MockDomainService is a mock for fundwatchlist domain service
type MockDomainService struct {
	createFunc      func(context.Context, *fundwatchlist.Watchlist) error
	getByIDFunc     func(context.Context, uuid.UUID) (*fundwatchlist.Watchlist, error)
	getBySymbolFunc func(context.Context, string, string) (*fundwatchlist.Watchlist, error)
	getActiveFunc   func(context.Context) ([]*fundwatchlist.Watchlist, error)
	getAllFunc      func(context.Context) ([]*fundwatchlist.Watchlist, error)
	updateFunc      func(context.Context, *fundwatchlist.Watchlist) error
	deleteFunc      func(context.Context, uuid.UUID) error
	isActiveFunc    func(context.Context, string, string) (bool, error)
	pauseFunc       func(context.Context, string, string, string) error
	resumeFunc      func(context.Context, string, string) error
}

func (m *MockDomainService) Create(ctx context.Context, w *fundwatchlist.Watchlist) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, w)
	}
	return nil
}

func (m *MockDomainService) GetByID(ctx context.Context, id uuid.UUID) (*fundwatchlist.Watchlist, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, errors.ErrNotFound
}

func (m *MockDomainService) GetBySymbol(ctx context.Context, symbol, marketType string) (*fundwatchlist.Watchlist, error) {
	if m.getBySymbolFunc != nil {
		return m.getBySymbolFunc(ctx, symbol, marketType)
	}
	return nil, errors.ErrNotFound
}

func (m *MockDomainService) GetActive(ctx context.Context) ([]*fundwatchlist.Watchlist, error) {
	if m.getActiveFunc != nil {
		return m.getActiveFunc(ctx)
	}
	return []*fundwatchlist.Watchlist{}, nil
}

func (m *MockDomainService) GetAll(ctx context.Context) ([]*fundwatchlist.Watchlist, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx)
	}
	return []*fundwatchlist.Watchlist{}, nil
}

func (m *MockDomainService) Update(ctx context.Context, w *fundwatchlist.Watchlist) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, w)
	}
	return nil
}

func (m *MockDomainService) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *MockDomainService) IsActive(ctx context.Context, symbol, marketType string) (bool, error) {
	if m.isActiveFunc != nil {
		return m.isActiveFunc(ctx, symbol, marketType)
	}
	return false, nil
}

func (m *MockDomainService) Pause(ctx context.Context, symbol, marketType, reason string) error {
	if m.pauseFunc != nil {
		return m.pauseFunc(ctx, symbol, marketType, reason)
	}
	return nil
}

func (m *MockDomainService) Resume(ctx context.Context, symbol, marketType string) error {
	if m.resumeFunc != nil {
		return m.resumeFunc(ctx, symbol, marketType)
	}
	return nil
}

// MockRepository is a mock for fundwatchlist repository
type MockRepository struct {
	getWithFiltersFunc func(context.Context, fundwatchlist.FilterOptions) ([]*fundwatchlist.Watchlist, error)
	countByScopeFunc   func(context.Context) (map[string]int, error)
}

func (m *MockRepository) Create(ctx context.Context, entry *fundwatchlist.Watchlist) error {
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*fundwatchlist.Watchlist, error) {
	return nil, errors.ErrNotFound
}

func (m *MockRepository) GetBySymbol(ctx context.Context, symbol, marketType string) (*fundwatchlist.Watchlist, error) {
	return nil, errors.ErrNotFound
}

func (m *MockRepository) GetActive(ctx context.Context) ([]*fundwatchlist.Watchlist, error) {
	return []*fundwatchlist.Watchlist{}, nil
}

func (m *MockRepository) GetAll(ctx context.Context) ([]*fundwatchlist.Watchlist, error) {
	return []*fundwatchlist.Watchlist{}, nil
}

func (m *MockRepository) GetWithFilters(ctx context.Context, filter fundwatchlist.FilterOptions) ([]*fundwatchlist.Watchlist, error) {
	if m.getWithFiltersFunc != nil {
		return m.getWithFiltersFunc(ctx, filter)
	}
	return []*fundwatchlist.Watchlist{}, nil
}

func (m *MockRepository) CountByScope(ctx context.Context) (map[string]int, error) {
	if m.countByScopeFunc != nil {
		return m.countByScopeFunc(ctx)
	}
	return map[string]int{}, nil
}

func (m *MockRepository) Update(ctx context.Context, entry *fundwatchlist.Watchlist) error {
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockRepository) IsActive(ctx context.Context, symbol, marketType string) (bool, error) {
	return false, nil
}

func TestService_GetByID(t *testing.T) {
	id := uuid.New()
	expected := &fundwatchlist.Watchlist{
		ID:         id,
		Symbol:     "BTC/USDT",
		MarketType: "spot",
		Category:   "major",
		Tier:       1,
		IsActive:   true,
	}

	mockDomainSvc := &MockDomainService{
		getByIDFunc: func(ctx context.Context, reqID uuid.UUID) (*fundwatchlist.Watchlist, error) {
			assert.Equal(t, id, reqID)
			return expected, nil
		},
	}

	log := testLogger()
	mockRepo := &MockRepository{}
	svc := NewService(mockDomainSvc, mockRepo, log)

	result, err := svc.GetByID(context.Background(), id)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestService_GetBySymbol(t *testing.T) {
	symbol := "ETH/USDT"
	marketType := "spot"
	expected := &fundwatchlist.Watchlist{
		ID:         uuid.New(),
		Symbol:     symbol,
		MarketType: marketType,
		Category:   "major",
		Tier:       1,
		IsActive:   true,
	}

	mockDomainSvc := &MockDomainService{
		getBySymbolFunc: func(ctx context.Context, sym, mkt string) (*fundwatchlist.Watchlist, error) {
			assert.Equal(t, symbol, sym)
			assert.Equal(t, marketType, mkt)
			return expected, nil
		},
	}

	log := testLogger()
	mockRepo := &MockRepository{}
	svc := NewService(mockDomainSvc, mockRepo, log)

	result, err := svc.GetBySymbol(context.Background(), symbol, marketType)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestService_GetMonitored(t *testing.T) {
	tests := []struct {
		name       string
		entries    []*fundwatchlist.Watchlist
		marketType *string
		expected   int
	}{
		{
			name: "returns only monitored entries",
			entries: []*fundwatchlist.Watchlist{
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
			entries: []*fundwatchlist.Watchlist{
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
			mockDomainSvc := &MockDomainService{
				getActiveFunc: func(ctx context.Context) ([]*fundwatchlist.Watchlist, error) {
					return tt.entries, nil
				},
			}

			log := testLogger()
			mockRepo := &MockRepository{}
			svc := NewService(mockDomainSvc, mockRepo, log)

			result, err := svc.GetMonitored(context.Background(), tt.marketType)

			assert.NoError(t, err)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestService_CreateWatchlist(t *testing.T) {
	mockDomainSvc := &MockDomainService{
		createFunc: func(ctx context.Context, w *fundwatchlist.Watchlist) error {
			assert.Equal(t, "BTC/USDT", w.Symbol)
			assert.Equal(t, "spot", w.MarketType)
			return nil
		},
	}

	log := testLogger()
	mockRepo := &MockRepository{}
	svc := NewService(mockDomainSvc, mockRepo, log)

	params := CreateWatchlistParams{
		Symbol:     "BTC/USDT",
		MarketType: "spot",
		Category:   "major",
		Tier:       1,
	}

	result, err := svc.CreateWatchlist(context.Background(), params)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "BTC/USDT", result.Symbol)
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
			id := uuid.New()
			entry := &fundwatchlist.Watchlist{
				ID:         id,
				Symbol:     "BTC/USDT",
				MarketType: "spot",
				IsActive:   true,
				IsPaused:   false,
			}

			mockDomainSvc := &MockDomainService{
				getByIDFunc: func(ctx context.Context, reqID uuid.UUID) (*fundwatchlist.Watchlist, error) {
					assert.Equal(t, id, reqID)
					return entry, nil
				},
				updateFunc: func(ctx context.Context, w *fundwatchlist.Watchlist) error {
					assert.Equal(t, tt.isPaused, w.IsPaused)
					if tt.reason != nil {
						require.NotNil(t, w.PausedReason)
						assert.Equal(t, *tt.reason, *w.PausedReason)
					} else {
						assert.Nil(t, w.PausedReason)
					}
					return nil
				},
			}

			log := testLogger()
			mockRepo := &MockRepository{}
			svc := NewService(mockDomainSvc, mockRepo, log)

			result, err := svc.TogglePause(context.Background(), id, tt.isPaused, tt.reason)

			assert.NoError(t, err)
			assert.Equal(t, tt.isPaused, result.IsPaused)
			if tt.isPaused && tt.reason != nil {
				assert.NotNil(t, result.PausedReason)
				assert.Equal(t, *tt.reason, *result.PausedReason)
			}
		})
	}
}

func TestService_DeleteWatchlist(t *testing.T) {
	id := uuid.New()

	mockDomainSvc := &MockDomainService{
		deleteFunc: func(ctx context.Context, reqID uuid.UUID) error {
			assert.Equal(t, id, reqID)
			return nil
		},
	}

	log := testLogger()
	mockRepo := &MockRepository{}
	svc := NewService(mockDomainSvc, mockRepo, log)

	err := svc.DeleteWatchlist(context.Background(), id)

	assert.NoError(t, err)
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
