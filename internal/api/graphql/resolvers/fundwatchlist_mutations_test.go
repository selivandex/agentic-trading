package resolvers_test

import (
	"context"
	"testing"

	"prometheus/internal/api/graphql/generated"
	"prometheus/internal/domain/fundwatchlist"
	"prometheus/internal/testsupport/seeds"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateFundWatchlistMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	tests := []struct {
		name        string
		input       generated.CreateFundWatchlistInput
		wantErr     bool
		errContains string
		validate    func(t *testing.T, result *fundwatchlist.Watchlist)
	}{
		{
			name: "successful creation - tier 1",
			input: generated.CreateFundWatchlistInput{
				Symbol:     "BTC/USDT-" + randomString(6),
				MarketType: "spot",
				Category:   "major",
				Tier:       1,
			},
			wantErr: false,
			validate: func(t *testing.T, result *fundwatchlist.Watchlist) {
				assert.NotEqual(t, uuid.Nil, result.ID)
				assert.Equal(t, "spot", result.MarketType)
				assert.Equal(t, "major", result.Category)
				assert.Equal(t, 1, result.Tier)
				assert.True(t, result.IsActive)
				assert.False(t, result.IsPaused)
			},
		},
		{
			name: "create futures symbol",
			input: generated.CreateFundWatchlistInput{
				Symbol:     "ETH/USDT-" + randomString(6),
				MarketType: "futures",
				Category:   "defi",
				Tier:       2,
			},
			wantErr: false,
			validate: func(t *testing.T, result *fundwatchlist.Watchlist) {
				assert.Equal(t, "futures", result.MarketType)
				assert.Equal(t, "defi", result.Category)
				assert.Equal(t, 2, result.Tier)
			},
		},
		{
			name: "create tier 3 symbol",
			input: generated.CreateFundWatchlistInput{
				Symbol:     "DOGE/USDT-" + randomString(6),
				MarketType: "spot",
				Category:   "meme",
				Tier:       3,
			},
			wantErr: false,
			validate: func(t *testing.T, result *fundwatchlist.Watchlist) {
				assert.Equal(t, "meme", result.Category)
				assert.Equal(t, 3, result.Tier)
			},
		},
		{
			name: "empty symbol",
			input: generated.CreateFundWatchlistInput{
				Symbol:     "",
				MarketType: "spot",
				Category:   "major",
				Tier:       1,
			},
			wantErr:     true,
			errContains: "symbol is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().CreateFundWatchlist(ctx, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestUpdateFundWatchlistMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create test watchlist entry
	seeder := seeds.New(setup.pg.DB())
	testEntry := seeder.FundWatchlist().
		WithSymbol("BTC/USDT-" + randomString(6)).
		WithMarketType("spot").
		WithCategory("major").
		WithTier(1).
		MustInsert()

	tests := []struct {
		name        string
		entryID     uuid.UUID
		input       generated.UpdateFundWatchlistInput
		wantErr     bool
		errContains string
		validate    func(t *testing.T, result *fundwatchlist.Watchlist)
	}{
		{
			name:    "update category",
			entryID: testEntry.ID,
			input: generated.UpdateFundWatchlistInput{
				Category: strPtr("layer1"),
			},
			wantErr: false,
			validate: func(t *testing.T, result *fundwatchlist.Watchlist) {
				assert.Equal(t, "layer1", result.Category)
				assert.Equal(t, 1, result.Tier) // unchanged
			},
		},
		{
			name:    "update tier",
			entryID: testEntry.ID,
			input: generated.UpdateFundWatchlistInput{
				Tier: intPtr(2),
			},
			wantErr: false,
			validate: func(t *testing.T, result *fundwatchlist.Watchlist) {
				assert.Equal(t, 2, result.Tier)
			},
		},
		{
			name:    "deactivate entry",
			entryID: testEntry.ID,
			input: generated.UpdateFundWatchlistInput{
				IsActive: boolPtr(false),
			},
			wantErr: false,
			validate: func(t *testing.T, result *fundwatchlist.Watchlist) {
				assert.False(t, result.IsActive)
			},
		},
		{
			name:    "pause entry",
			entryID: testEntry.ID,
			input: generated.UpdateFundWatchlistInput{
				IsPaused:     boolPtr(true),
				PausedReason: strPtr("Testing pause"),
			},
			wantErr: false,
			validate: func(t *testing.T, result *fundwatchlist.Watchlist) {
				assert.True(t, result.IsPaused)
				assert.NotNil(t, result.PausedReason)
				assert.Equal(t, "Testing pause", *result.PausedReason)
			},
		},
		{
			name:    "update multiple fields",
			entryID: testEntry.ID,
			input: generated.UpdateFundWatchlistInput{
				Category: strPtr("defi"),
				Tier:     intPtr(3),
				IsPaused: boolPtr(false),
			},
			wantErr: false,
			validate: func(t *testing.T, result *fundwatchlist.Watchlist) {
				assert.Equal(t, "defi", result.Category)
				assert.Equal(t, 3, result.Tier)
				assert.False(t, result.IsPaused)
			},
		},
		{
			name:        "non-existent entry",
			entryID:     uuid.New(),
			input:       generated.UpdateFundWatchlistInput{Category: strPtr("test")},
			wantErr:     true,
			errContains: "failed to get fund watchlist entry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().UpdateFundWatchlist(ctx, tt.entryID, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.entryID, result.ID)

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestDeleteFundWatchlistMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create entry to delete
	seeder := seeds.New(setup.pg.DB())
	entryToDelete := seeder.FundWatchlist().
		WithSymbol("DELETE/USDT-" + randomString(6)).
		MustInsert()

	tests := []struct {
		name        string
		entryID     uuid.UUID
		wantErr     bool
		errContains string
	}{
		{
			name:    "delete existing entry",
			entryID: entryToDelete.ID,
			wantErr: false,
		},
		{
			name:        "delete non-existent entry",
			entryID:     uuid.New(),
			wantErr:     true,
			errContains: "failed to delete fund watchlist entry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().DeleteFundWatchlist(ctx, tt.entryID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.True(t, result)

			// Verify entry is actually deleted
			_, err = setup.resolver.Query().FundWatchlist(ctx, tt.entryID)
			assert.Error(t, err) // Should not find deleted entry
		})
	}
}

func TestToggleFundWatchlistPauseMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create test entry
	seeder := seeds.New(setup.pg.DB())
	testEntry := seeder.FundWatchlist().
		WithSymbol("TOGGLE/USDT-" + randomString(6)).
		WithIsPaused(false).
		MustInsert()

	tests := []struct {
		name        string
		entryID     uuid.UUID
		isPaused    bool
		reason      *string
		wantErr     bool
		errContains string
		validate    func(t *testing.T, result *fundwatchlist.Watchlist)
	}{
		{
			name:     "pause entry with reason",
			entryID:  testEntry.ID,
			isPaused: true,
			reason:   strPtr("Temporary market volatility"),
			wantErr:  false,
			validate: func(t *testing.T, result *fundwatchlist.Watchlist) {
				assert.True(t, result.IsPaused)
				assert.NotNil(t, result.PausedReason)
				assert.Equal(t, "Temporary market volatility", *result.PausedReason)
				assert.False(t, result.IsMonitored()) // paused = not monitored
			},
		},
		{
			name:     "unpause entry",
			entryID:  testEntry.ID,
			isPaused: false,
			reason:   nil,
			wantErr:  false,
			validate: func(t *testing.T, result *fundwatchlist.Watchlist) {
				assert.False(t, result.IsPaused)
				assert.Nil(t, result.PausedReason)
				assert.True(t, result.IsMonitored()) // active and not paused
			},
		},
		{
			name:     "pause without reason",
			entryID:  testEntry.ID,
			isPaused: true,
			reason:   nil,
			wantErr:  false,
			validate: func(t *testing.T, result *fundwatchlist.Watchlist) {
				assert.True(t, result.IsPaused)
				assert.Nil(t, result.PausedReason)
			},
		},
		{
			name:        "non-existent entry",
			entryID:     uuid.New(),
			isPaused:    true,
			reason:      nil,
			wantErr:     true,
			errContains: "failed to get fund watchlist entry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().ToggleFundWatchlistPause(ctx, tt.entryID, tt.isPaused, tt.reason)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.entryID, result.ID)

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestFundWatchlistMutationsWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Full lifecycle test: create → update → pause → unpause → delete
	t.Run("complete fundwatchlist lifecycle", func(t *testing.T) {
		// 1. Create watchlist entry
		createInput := generated.CreateFundWatchlistInput{
			Symbol:     "LIFECYCLE/USDT-" + randomString(6),
			MarketType: "spot",
			Category:   "major",
			Tier:       1,
		}

		created, err := setup.resolver.Mutation().CreateFundWatchlist(ctx, createInput)
		require.NoError(t, err)
		assert.True(t, created.IsActive)
		assert.False(t, created.IsPaused)
		assert.Equal(t, "major", created.Category)

		// 2. Update category and tier
		updateInput := generated.UpdateFundWatchlistInput{
			Category: strPtr("layer1"),
			Tier:     intPtr(2),
		}

		updated, err := setup.resolver.Mutation().UpdateFundWatchlist(ctx, created.ID, updateInput)
		require.NoError(t, err)
		assert.Equal(t, "layer1", updated.Category)
		assert.Equal(t, 2, updated.Tier)

		// 3. Pause entry
		paused, err := setup.resolver.Mutation().ToggleFundWatchlistPause(
			ctx,
			created.ID,
			true,
			strPtr("Testing pause functionality"),
		)
		require.NoError(t, err)
		assert.True(t, paused.IsPaused)
		assert.NotNil(t, paused.PausedReason)

		// 4. Unpause entry
		unpaused, err := setup.resolver.Mutation().ToggleFundWatchlistPause(ctx, created.ID, false, nil)
		require.NoError(t, err)
		assert.False(t, unpaused.IsPaused)
		assert.Nil(t, unpaused.PausedReason)

		// 5. Deactivate entry
		deactivateInput := generated.UpdateFundWatchlistInput{
			IsActive: boolPtr(false),
		}
		deactivated, err := setup.resolver.Mutation().UpdateFundWatchlist(ctx, created.ID, deactivateInput)
		require.NoError(t, err)
		assert.False(t, deactivated.IsActive)

		// 6. Delete entry
		deleted, err := setup.resolver.Mutation().DeleteFundWatchlist(ctx, created.ID)
		require.NoError(t, err)
		assert.True(t, deleted)

		// Verify deletion
		_, err = setup.resolver.Query().FundWatchlist(ctx, created.ID)
		assert.Error(t, err)
	})
}

func TestFundWatchlistQueriesWithMutations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create multiple watchlist entries
	seeder := seeds.New(setup.pg.DB())

	// Create active entries
	active1 := seeder.FundWatchlist().
		WithSymbol("ACTIVE1/USDT-" + randomString(6)).
		WithCategory("major").
		WithTier(1).
		WithIsActive(true).
		WithIsPaused(false).
		MustInsert()

	_ = seeder.FundWatchlist().
		WithSymbol("ACTIVE2/USDT-" + randomString(6)).
		WithCategory("defi").
		WithTier(2).
		WithIsActive(true).
		WithIsPaused(false).
		MustInsert()

	// Create paused entry
	paused := seeder.FundWatchlist().
		WithSymbol("PAUSED/USDT-" + randomString(6)).
		WithCategory("major").
		WithTier(1).
		WithIsActive(true).
		WithIsPaused(true).
		MustInsert()

	t.Run("query by ID", func(t *testing.T) {
		result, err := setup.resolver.Query().FundWatchlist(ctx, active1.ID)
		require.NoError(t, err)
		assert.Equal(t, active1.ID, result.ID)
		assert.Equal(t, active1.Symbol, result.Symbol)
	})

	t.Run("query by symbol", func(t *testing.T) {
		result, err := setup.resolver.Query().FundWatchlistBySymbol(ctx, active1.Symbol, active1.MarketType)
		require.NoError(t, err)
		assert.Equal(t, active1.ID, result.ID)
		assert.Equal(t, active1.Symbol, result.Symbol)
	})

	t.Run("filter active entries in connection", func(t *testing.T) {
		isActive := true
		first := 10

		conn, err := setup.resolver.Query().FundWatchlistsConnection(
			ctx,
			nil,       // scope
			&isActive, // isActive
			nil,       // category
			nil,       // tier
			nil,       // search
			nil,       // filters
			&first,    // first
			nil,       // after
			nil,       // last
			nil,       // before
		)

		require.NoError(t, err)
		assert.NotEmpty(t, conn.Edges)
		// All returned entries should be active
		for _, edge := range conn.Edges {
			assert.True(t, edge.Node.IsActive)
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
			&first,    // first
			nil,       // after
			nil,       // last
			nil,       // before
		)

		require.NoError(t, err)
		// All returned entries should be major category
		for _, edge := range conn.Edges {
			assert.Equal(t, "major", edge.Node.Category)
		}
	})

	t.Run("monitored symbols excludes paused", func(t *testing.T) {
		marketType := "spot"
		monitored, err := setup.resolver.Query().MonitoredSymbols(ctx, &marketType)

		require.NoError(t, err)
		// Paused entry should not be in monitored list
		for _, item := range monitored {
			assert.True(t, item.IsMonitored())
			assert.False(t, item.IsPaused)
			if item.ID == paused.ID {
				t.Error("Paused entry should not be in monitored list")
			}
		}
	})
}

func TestFundWatchlistEdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	t.Run("create duplicate symbol same market", func(t *testing.T) {
		symbol := "DUP/USDT-" + randomString(6)
		input := generated.CreateFundWatchlistInput{
			Symbol:     symbol,
			MarketType: "spot",
			Category:   "major",
			Tier:       1,
		}

		// First creation should succeed
		first, err := setup.resolver.Mutation().CreateFundWatchlist(ctx, input)
		require.NoError(t, err)
		assert.NotNil(t, first)

		// Second creation with same symbol should fail (unique constraint)
		_, err = setup.resolver.Mutation().CreateFundWatchlist(ctx, input)
		assert.Error(t, err)
	})

	t.Run("update paused reason without pausing", func(t *testing.T) {
		seeder := seeds.New(setup.pg.DB())
		entry := seeder.FundWatchlist().
			WithSymbol("REASON/USDT-" + randomString(6)).
			WithIsPaused(false).
			MustInsert()

		// Set reason without pausing - should work but reason ignored when not paused
		updated, err := setup.resolver.Mutation().UpdateFundWatchlist(
			ctx,
			entry.ID,
			generated.UpdateFundWatchlistInput{
				PausedReason: strPtr("This reason should be ignored"),
			},
		)

		require.NoError(t, err)
		// Entry is not paused, so it should still be monitored
		assert.True(t, updated.IsMonitored())
	})

	t.Run("tier boundaries", func(t *testing.T) {
		seeder := seeds.New(setup.pg.DB())
		entry := seeder.FundWatchlist().
			WithSymbol("TIER/USDT-" + randomString(6)).
			WithTier(1).
			MustInsert()

		// Update to tier 3
		updated, err := setup.resolver.Mutation().UpdateFundWatchlist(
			ctx,
			entry.ID,
			generated.UpdateFundWatchlistInput{
				Tier: intPtr(3),
			},
		)

		require.NoError(t, err)
		assert.Equal(t, 3, updated.Tier)
	})
}
