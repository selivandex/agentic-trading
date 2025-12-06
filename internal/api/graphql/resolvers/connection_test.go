package resolvers_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"prometheus/internal/api/graphql/resolvers"
	"prometheus/internal/domain/fundwatchlist"
	"prometheus/internal/domain/strategy"
	"prometheus/internal/domain/user"
	"prometheus/internal/repository/postgres"
	authService "prometheus/internal/services/auth"
	fundService "prometheus/internal/services/fundwatchlist"
	strategyService "prometheus/internal/services/strategy"
	userService "prometheus/internal/services/user"
	"prometheus/internal/testsupport"
	"prometheus/internal/testsupport/seeds"
	"prometheus/pkg/auth"
	"prometheus/pkg/logger"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testLogger() *logger.Logger {
	zapLogger, _ := zap.NewDevelopment()
	return &logger.Logger{SugaredLogger: zapLogger.Sugar()}
}

// limitProfileAdapter adapts postgres LimitProfileRepository to domain interface
type limitProfileAdapter struct {
	repo *postgres.LimitProfileRepository
}

func (a *limitProfileAdapter) GetByName(ctx context.Context, name string) (user.LimitProfileInfo, error) {
	profile, err := a.repo.GetByName(ctx, name)
	if err != nil {
		return user.LimitProfileInfo{}, err
	}
	return user.LimitProfileInfo{ID: profile.ID}, nil
}

// testSetup contains all dependencies for resolver tests
type testSetup struct {
	pg                   *testsupport.PostgresTestHelper
	resolver             *resolvers.Resolver
	userService          *userService.Service
	strategyService      *strategyService.Service
	fundWatchlistService *fundService.Service
	log                  *logger.Logger
	testUser             *user.User
}

// setupResolverTest creates all necessary dependencies for resolver tests
func setupResolverTest(t *testing.T) *testSetup {
	t.Helper()

	// Setup test database with transaction (auto-rollback)
	pg := testsupport.NewTestPostgres(t)

	// Seed required data
	seeder := seeds.New(pg.DB())
	limitProfile := seeder.LimitProfile().
		WithFreeTier().
		MustInsert()

	// Create test user with unique email
	testUser := seeder.User().
		WithEmail(fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8])).
		WithLimitProfile(limitProfile.ID).
		MustInsert()

	// Create repositories
	userRepo := postgres.NewUserRepository(pg.DB())
	strategyRepo := postgres.NewStrategyRepository(pg.DB())
	fundWatchlistRepo := postgres.NewFundWatchlistRepository(pg.DB())
	limitProfileRepo := postgres.NewLimitProfileRepository(pg.DB())

	// Create services
	log := testLogger()
	jwtService := auth.NewJWTService("test-secret-key-min-32-characters-long", "test-issuer", 3600)
	authSvc := authService.NewService(userRepo, jwtService, log)

	limitProfileAdapter := &limitProfileAdapter{repo: limitProfileRepo}
	domainUserSvc := user.NewService(userRepo, limitProfileAdapter)
	userSvc := userService.NewService(domainUserSvc, log)

	// Strategy domain service (for queries)
	domainStrategySvc := strategy.NewService(strategyRepo)
	strategySvc := strategyService.NewService(domainStrategySvc, strategyRepo, nil, nil, log)

	// FundWatchlist domain service
	domainFundWatchlistSvc := fundwatchlist.NewService(fundWatchlistRepo)
	fundSvc := fundService.NewService(domainFundWatchlistSvc, fundWatchlistRepo, log)

	// Create resolver manually (no constructor)
	resolver := &resolvers.Resolver{
		AuthService:          authSvc,
		UserService:          userSvc,
		StrategyService:      strategySvc,
		FundWatchlistService: fundSvc,
		Log:                  log,
	}

	return &testSetup{
		pg:                   pg,
		resolver:             resolver,
		userService:          userSvc,
		strategyService:      strategySvc,
		fundWatchlistService: fundSvc,
		log:                  log,
		testUser:             testUser,
	}
}

func TestUserStrategiesConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create test strategies using seeds
	seeder := seeds.New(setup.pg.DB())
	for i := 0; i < 15; i++ {
		seeder.Strategy().
			WithUserID(setup.testUser.ID).
			WithName("Test Strategy " + string(rune(i+'A'))).
			MustInsert()
	}

	t.Run("forward pagination - first page", func(t *testing.T) {
		first := 5
		conn, err := setup.resolver.Query().UserStrategies(
			ctx,
			setup.testUser.ID,
			nil, // scope
			nil, // status filter
			nil, // search
			nil, // filters
			&first,
			nil, // after
			nil, // last
			nil, // before
		)

		require.NoError(t, err)
		assert.NotNil(t, conn)
		assert.Len(t, conn.Edges, 5)
		assert.Equal(t, 15, conn.TotalCount)
		assert.False(t, conn.PageInfo.HasPreviousPage)
		assert.True(t, conn.PageInfo.HasNextPage)
		assert.NotNil(t, conn.PageInfo.StartCursor)
		assert.NotNil(t, conn.PageInfo.EndCursor)
	})

	t.Run("forward pagination - with after cursor", func(t *testing.T) {
		// Get first page
		first := 5
		firstPage, err := setup.resolver.Query().UserStrategies(
			ctx,
			setup.testUser.ID,
			nil, // scope
			nil, // status
			nil, // search
			nil, // filters
			&first,
			nil,
			nil,
			nil,
		)
		require.NoError(t, err)

		// Get second page using endCursor
		secondPage, err := setup.resolver.Query().UserStrategies(
			ctx,
			setup.testUser.ID,
			nil, // scope
			nil, // status
			nil, // search
			nil, // filters
			&first,
			firstPage.PageInfo.EndCursor,
			nil,
			nil,
		)

		require.NoError(t, err)
		assert.Len(t, secondPage.Edges, 5)
		assert.True(t, secondPage.PageInfo.HasPreviousPage)
		assert.True(t, secondPage.PageInfo.HasNextPage)

		// Ensure different strategies
		assert.NotEqual(t, firstPage.Edges[0].Node.ID, secondPage.Edges[0].Node.ID)
	})

	t.Run("forward pagination - last page", func(t *testing.T) {
		// Get first two pages
		first := 5
		firstPage, _ := setup.resolver.Query().UserStrategies(
			ctx, setup.testUser.ID, nil, nil, nil, nil, &first, nil, nil, nil)
		secondPage, _ := setup.resolver.Query().UserStrategies(
			ctx, setup.testUser.ID, nil, nil, nil, nil, &first, firstPage.PageInfo.EndCursor, nil, nil)

		// Get third (last) page
		thirdPage, err := setup.resolver.Query().UserStrategies(
			ctx,
			setup.testUser.ID,
			nil, // scope
			nil, // status
			nil, // search
			nil, // filters
			&first,
			secondPage.PageInfo.EndCursor,
			nil,
			nil,
		)

		require.NoError(t, err)
		assert.Len(t, thirdPage.Edges, 5) // 15 total, 5 per page
		assert.True(t, thirdPage.PageInfo.HasPreviousPage)
		assert.False(t, thirdPage.PageInfo.HasNextPage)
	})

	t.Run("filter by status", func(t *testing.T) {
		activeStatus := strategy.StrategyActive
		first := 20

		conn, err := setup.resolver.Query().UserStrategies(
			ctx,
			setup.testUser.ID,
			nil,           // scope
			&activeStatus, // status
			nil,           // search
			nil,           // filters
			&first,
			nil,
			nil,
			nil,
		)

		require.NoError(t, err)
		assert.Equal(t, 15, conn.TotalCount) // All created strategies are active
		for _, edge := range conn.Edges {
			assert.Equal(t, strategy.StrategyActive, edge.Node.Status)
		}
	})

	t.Run("search by name", func(t *testing.T) {
		// Search for specific strategy by name
		search := "Test Strategy E"
		first := 20

		conn, err := setup.resolver.Query().UserStrategies(
			ctx,
			setup.testUser.ID,
			nil,     // scope
			nil,     // status
			&search, // search
			nil,     // filters
			&first,
			nil,
			nil,
			nil,
		)

		require.NoError(t, err)
		assert.Equal(t, 1, conn.TotalCount)
		assert.Len(t, conn.Edges, 1)
		assert.Contains(t, conn.Edges[0].Node.Name, "Test Strategy E")
	})

	t.Run("search by partial name - case insensitive", func(t *testing.T) {
		// Search for strategies containing "test" (should match all)
		search := "test"
		first := 20

		conn, err := setup.resolver.Query().UserStrategies(
			ctx,
			setup.testUser.ID,
			nil,     // scope
			nil,     // status
			&search, // search
			nil,     // filters
			&first,
			nil,
			nil,
			nil,
		)

		require.NoError(t, err)
		assert.Equal(t, 15, conn.TotalCount) // All 15 strategies contain "test" in name
		for _, edge := range conn.Edges {
			assert.Contains(t, strings.ToLower(edge.Node.Name), "test")
		}
	})

	t.Run("search with no results", func(t *testing.T) {
		search := "NonExistentStrategy"
		first := 20

		conn, err := setup.resolver.Query().UserStrategies(
			ctx,
			setup.testUser.ID,
			nil,     // scope
			nil,     // status
			&search, // search
			nil,     // filters
			&first,
			nil,
			nil,
			nil,
		)

		require.NoError(t, err)
		assert.Equal(t, 0, conn.TotalCount)
		assert.Empty(t, conn.Edges)
	})

	t.Run("empty result", func(t *testing.T) {
		nonExistentUser := uuid.New()
		first := 10

		conn, err := setup.resolver.Query().UserStrategies(
			ctx,
			nonExistentUser,
			nil, // scope
			nil, // status
			nil, // search
			nil, // filters
			&first,
			nil,
			nil,
			nil,
		)

		require.NoError(t, err)
		assert.Empty(t, conn.Edges)
		assert.Equal(t, 0, conn.TotalCount)
		assert.False(t, conn.PageInfo.HasNextPage)
		assert.False(t, conn.PageInfo.HasPreviousPage)
	})

	t.Run("invalid pagination params - both first and last", func(t *testing.T) {
		first := 5
		last := 5

		_, err := setup.resolver.Query().UserStrategies(
			ctx,
			setup.testUser.ID,
			nil, // scope
			nil, // status
			nil, // search
			nil, // filters
			&first,
			nil,
			&last,
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot provide both")
	})
}

func TestFundWatchlistsConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create test watchlist items with unique symbols
	baseSymbols := []string{"BTC", "ETH", "SOL", "AVAX", "DOT"}
	for i, base := range baseSymbols {
		symbol := fmt.Sprintf("%s-%s/USDT", base, uuid.New().String()[:6])
		_, err := setup.fundWatchlistService.CreateWatchlist(ctx, fundService.CreateWatchlistParams{
			Symbol:     symbol,
			MarketType: "spot",
			Category:   "major",
			Tier:       i%3 + 1, // Tiers 1, 2, 3
		})
		require.NoError(t, err)

		// Pause every other entry
		if i%2 == 0 {
			// Get created entry to pause it
			created, err := setup.fundWatchlistService.GetBySymbol(ctx, symbol, "spot")
			require.NoError(t, err)
			_, err = setup.fundWatchlistService.TogglePause(ctx, created.ID, true, nil)
			require.NoError(t, err)
		}
	}

	t.Run("forward pagination - first page", func(t *testing.T) {
		first := 3
		conn, err := setup.resolver.Query().FundWatchlistsConnection(
			ctx,
			nil, // scope
			nil, // isActive
			nil, // category
			nil, // tier
			nil, // search
			nil, // filters
			&first,
			nil, // after
			nil, // last
			nil, // before
		)

		require.NoError(t, err)
		assert.NotNil(t, conn)
		assert.Len(t, conn.Edges, 3)         // Should return exactly 'first' items
		assert.True(t, conn.TotalCount >= 5) // At least our 5 test items
		assert.False(t, conn.PageInfo.HasPreviousPage)
		// May or may not have next page depending on other test data
	})

	t.Run("filter by tier", func(t *testing.T) {
		tier := 1
		first := 10

		conn, err := setup.resolver.Query().FundWatchlistsConnection(
			ctx,
			nil,   // scope
			nil,   // isActive
			nil,   // category
			&tier, // tier
			nil,   // search
			nil,   // filters
			&first,
			nil,
			nil,
			nil,
		)

		require.NoError(t, err)
		for _, edge := range conn.Edges {
			assert.Equal(t, 1, edge.Node.Tier)
		}
	})

	t.Run("filter by category", func(t *testing.T) {
		category := "major"
		first := 10

		conn, err := setup.resolver.Query().FundWatchlistsConnection(
			ctx,
			nil,       // scope
			nil,       // isActive
			&category, // category
			nil,       // tier
			nil,       // search
			nil,       // filters
			&first,
			nil,
			nil,
			nil,
		)

		require.NoError(t, err)
		assert.True(t, conn.TotalCount >= 5) // At least our 5 test items are major
		// Verify all returned items match filter
		for _, edge := range conn.Edges {
			assert.Equal(t, "major", edge.Node.Category)
		}
	})
}

func TestMonitoredSymbolsConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create test watchlist items - monitored (active and not paused) with unique symbols
	for i := 0; i < 8; i++ {
		symbol := fmt.Sprintf("COIN%s-%d/USDT", uuid.New().String()[:6], i)
		created, err := setup.fundWatchlistService.CreateWatchlist(ctx, fundService.CreateWatchlistParams{
			Symbol:     symbol,
			MarketType: "spot",
			Category:   "major",
			Tier:       1,
		})
		require.NoError(t, err)

		// Deactivate or pause based on index
		if i >= 6 {
			// Last 2 entries: pause them
			_, err = setup.fundWatchlistService.TogglePause(ctx, created.ID, true, nil)
			require.NoError(t, err)
		} else if i == 5 {
			// One entry: deactivate it
			_, err = setup.fundWatchlistService.UpdateWatchlist(ctx, created.ID, fundService.UpdateWatchlistParams{
				IsActive: boolPtr(false),
			})
			require.NoError(t, err)
		}
	}

	t.Run("forward pagination", func(t *testing.T) {
		first := 3
		marketType := "spot"

		conn, err := setup.resolver.Query().MonitoredSymbolsConnection(
			ctx,
			nil,         // scope
			&marketType, // marketType
			&first,
			nil,
			nil,
			nil,
		)

		require.NoError(t, err)
		assert.NotNil(t, conn)
		assert.Len(t, conn.Edges, 3)
		// Should only return active and not paused items
		assert.True(t, conn.TotalCount >= 6)

		// Verify all returned items are monitored
		for _, edge := range conn.Edges {
			assert.True(t, edge.Node.IsActive)
			assert.False(t, edge.Node.IsPaused)
		}
	})

	t.Run("pagination through all monitored symbols", func(t *testing.T) {
		first := 2
		marketType := "spot"

		// First page
		firstPage, err := setup.resolver.Query().MonitoredSymbolsConnection(
			ctx,
			nil,         // scope
			&marketType, // marketType
			&first,
			nil,
			nil,
			nil,
		)
		require.NoError(t, err)
		assert.Len(t, firstPage.Edges, 2)
		assert.True(t, firstPage.PageInfo.HasNextPage)

		// Second page
		secondPage, err := setup.resolver.Query().MonitoredSymbolsConnection(
			ctx,
			nil,         // scope
			&marketType, // marketType
			&first,
			firstPage.PageInfo.EndCursor,
			nil,
			nil,
		)
		require.NoError(t, err)
		assert.Len(t, secondPage.Edges, 2)
		assert.True(t, secondPage.PageInfo.HasPreviousPage)
	})
}

func TestUsersConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create additional test users with unique emails
	seeder := seeds.New(setup.pg.DB())
	for i := 0; i < 10; i++ {
		seeder.User().
			WithEmail(fmt.Sprintf("user-%d-%s@example.com", i, uuid.New().String()[:6])).
			MustInsert()
	}

	t.Run("forward pagination - first page", func(t *testing.T) {
		first := 5
		conn, err := setup.resolver.Query().UsersConnection(
			ctx,
			&first,
			nil, // after
			nil, // last
			nil, // before
		)

		require.NoError(t, err)
		assert.NotNil(t, conn)
		assert.Len(t, conn.Edges, 5)
		assert.True(t, conn.TotalCount >= 11) // Initial user + 10 created
		assert.False(t, conn.PageInfo.HasPreviousPage)
		assert.True(t, conn.PageInfo.HasNextPage)
	})

	t.Run("forward pagination - second page", func(t *testing.T) {
		first := 5

		// Get first page
		firstPage, err := setup.resolver.Query().UsersConnection(
			ctx,
			&first,
			nil,
			nil,
			nil,
		)
		require.NoError(t, err)

		// Get second page
		secondPage, err := setup.resolver.Query().UsersConnection(
			ctx,
			&first,
			firstPage.PageInfo.EndCursor,
			nil,
			nil,
		)

		require.NoError(t, err)
		assert.Len(t, secondPage.Edges, 5)
		assert.True(t, secondPage.PageInfo.HasPreviousPage)

		// Ensure different users
		assert.NotEqual(t, firstPage.Edges[0].Node.ID, secondPage.Edges[0].Node.ID)
	})

	t.Run("get all users with large first", func(t *testing.T) {
		first := 100

		conn, err := setup.resolver.Query().UsersConnection(
			ctx,
			&first,
			nil,
			nil,
			nil,
		)

		require.NoError(t, err)
		assert.True(t, len(conn.Edges) >= 11) // At least our test users
		assert.True(t, conn.TotalCount >= 11)
		// If we got less edges than first, it means we've reached the end
		if len(conn.Edges) < first {
			assert.False(t, conn.PageInfo.HasNextPage)
		}
	})
}

func TestConnectionEdgeCursors(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create strategies using seeds
	seeder := seeds.New(setup.pg.DB())
	for i := 0; i < 5; i++ {
		seeder.Strategy().
			WithUserID(setup.testUser.ID).
			WithName("Strategy " + string(rune(i+'A'))).
			MustInsert()
	}

	t.Run("cursor consistency", func(t *testing.T) {
		first := 2

		// Get first page
		firstPage, err := setup.resolver.Query().UserStrategies(
			ctx,
			setup.testUser.ID,
			nil, // scope
			nil, // status
			nil, // search
			nil, // filters
			&first,
			nil,
			nil,
			nil,
		)
		require.NoError(t, err)

		// EndCursor of first page should equal StartCursor of second page item
		secondPage, err := setup.resolver.Query().UserStrategies(
			ctx,
			setup.testUser.ID,
			nil, // scope
			nil, // status
			nil, // search
			nil, // filters
			&first,
			firstPage.PageInfo.EndCursor,
			nil,
			nil,
		)
		require.NoError(t, err)

		// Each page should have unique cursors
		for i, edge := range firstPage.Edges {
			assert.NotEmpty(t, edge.Cursor)
			if i > 0 {
				assert.NotEqual(t, firstPage.Edges[i-1].Cursor, edge.Cursor)
			}
		}

		for _, edge := range secondPage.Edges {
			assert.NotEmpty(t, edge.Cursor)
		}
	})
}
