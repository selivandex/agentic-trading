package resolvers_test

import (
	"context"
	"fmt"
	"testing"

	"prometheus/internal/api/graphql/generated"
	"prometheus/internal/domain/user"
	"prometheus/internal/testsupport/seeds"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateUserMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create limit profile for testing
	seeder := seeds.New(setup.pg.DB())
	limitProfile := seeder.LimitProfile().WithFreeTier().MustInsert()

	tests := []struct {
		name        string
		input       generated.CreateUserInput
		wantErr     bool
		errContains string
		validate    func(t *testing.T, result *user.User)
	}{
		{
			name: "create user with email",
			input: generated.CreateUserInput{
				Email:          strPtr(fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8])),
				FirstName:      "John",
				LastName:       "Doe",
				LanguageCode:   "en",
				LimitProfileID: &limitProfile.ID,
			},
			wantErr: false,
			validate: func(t *testing.T, result *user.User) {
				assert.NotNil(t, result.Email)
				assert.Equal(t, "John", result.FirstName)
				assert.Equal(t, "Doe", result.LastName)
				assert.True(t, result.IsActive)
				assert.False(t, result.IsPremium)
			},
		},
		{
			name: "create user with telegram ID",
			input: generated.CreateUserInput{
				TelegramID:       strPtr("123456789"),
				TelegramUsername: strPtr("johndoe"),
				FirstName:        "Jane",
				LastName:         "Smith",
				LanguageCode:     "en",
			},
			wantErr: false,
			validate: func(t *testing.T, result *user.User) {
				assert.NotNil(t, result.TelegramID)
				assert.Equal(t, int64(123456789), *result.TelegramID)
				assert.Equal(t, "johndoe", result.TelegramUsername)
			},
		},
		{
			name: "create premium user",
			input: generated.CreateUserInput{
				Email:        strPtr(fmt.Sprintf("premium-%s@example.com", uuid.New().String()[:8])),
				FirstName:    "Premium",
				LastName:     "User",
				LanguageCode: "en",
				IsPremium:    boolPtr(true),
			},
			wantErr: false,
			validate: func(t *testing.T, result *user.User) {
				assert.True(t, result.IsPremium)
			},
		},
		{
			name: "create inactive user",
			input: generated.CreateUserInput{
				Email:        strPtr(fmt.Sprintf("inactive-%s@example.com", uuid.New().String()[:8])),
				FirstName:    "Inactive",
				LastName:     "User",
				LanguageCode: "en",
				IsActive:     boolPtr(false),
			},
			wantErr: false,
			validate: func(t *testing.T, result *user.User) {
				assert.False(t, result.IsActive)
			},
		},
		{
			name: "create user with custom settings",
			input: generated.CreateUserInput{
				Email:        strPtr(fmt.Sprintf("custom-%s@example.com", uuid.New().String()[:8])),
				FirstName:    "Custom",
				LastName:     "Settings",
				LanguageCode: "en",
				Settings: &generated.SettingsInput{
					RiskLevel:    strPtr("aggressive"),
					MaxPositions: intPtr(5),
				},
			},
			wantErr: false,
			validate: func(t *testing.T, result *user.User) {
				assert.Equal(t, "aggressive", result.Settings.RiskLevel)
				assert.Equal(t, 5, result.Settings.MaxPositions)
			},
		},
		{
			name: "create user without email or telegram",
			input: generated.CreateUserInput{
				FirstName:    "Invalid",
				LastName:     "User",
				LanguageCode: "en",
			},
			wantErr:     true,
			errContains: "user must have either telegram_id or email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().CreateUser(ctx, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.NotEqual(t, uuid.Nil, result.ID)
			assert.Equal(t, tt.input.FirstName, result.FirstName)
			assert.Equal(t, tt.input.LastName, result.LastName)
			assert.Equal(t, tt.input.LanguageCode, result.LanguageCode)

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestUpdateUserMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create test user
	seeder := seeds.New(setup.pg.DB())
	testUser := seeder.User().
		WithEmail(fmt.Sprintf("update-test-%s@example.com", uuid.New().String()[:8])).
		WithFirstName("Original").
		WithLastName("Name").
		MustInsert()

	limitProfile := seeder.LimitProfile().WithFreeTier().MustInsert()

	tests := []struct {
		name        string
		userID      uuid.UUID
		input       generated.UpdateUserInput
		wantErr     bool
		errContains string
		validate    func(t *testing.T, result *user.User)
	}{
		{
			name:   "update first name only",
			userID: testUser.ID,
			input: generated.UpdateUserInput{
				FirstName: strPtr("Updated"),
			},
			wantErr: false,
			validate: func(t *testing.T, result *user.User) {
				assert.Equal(t, "Updated", result.FirstName)
				assert.Equal(t, "Name", result.LastName) // unchanged
			},
		},
		{
			name:   "update multiple fields",
			userID: testUser.ID,
			input: generated.UpdateUserInput{
				FirstName:    strPtr("John"),
				LastName:     strPtr("Smith"),
				LanguageCode: strPtr("ru"),
			},
			wantErr: false,
			validate: func(t *testing.T, result *user.User) {
				assert.Equal(t, "John", result.FirstName)
				assert.Equal(t, "Smith", result.LastName)
				assert.Equal(t, "ru", result.LanguageCode)
			},
		},
		{
			name:   "update premium status",
			userID: testUser.ID,
			input: generated.UpdateUserInput{
				IsPremium: boolPtr(true),
			},
			wantErr: false,
			validate: func(t *testing.T, result *user.User) {
				assert.True(t, result.IsPremium)
			},
		},
		{
			name:   "update limit profile",
			userID: testUser.ID,
			input: generated.UpdateUserInput{
				LimitProfileID: &limitProfile.ID,
			},
			wantErr: false,
			validate: func(t *testing.T, result *user.User) {
				assert.NotNil(t, result.LimitProfileID)
				assert.Equal(t, limitProfile.ID, *result.LimitProfileID)
			},
		},
		{
			name:   "update telegram info",
			userID: testUser.ID,
			input: generated.UpdateUserInput{
				TelegramID:       strPtr("987654321"),
				TelegramUsername: strPtr("newusername"),
			},
			wantErr: false,
			validate: func(t *testing.T, result *user.User) {
				assert.NotNil(t, result.TelegramID)
				assert.Equal(t, int64(987654321), *result.TelegramID)
				assert.Equal(t, "newusername", result.TelegramUsername)
			},
		},
		{
			name:        "non-existent user",
			userID:      uuid.New(),
			input:       generated.UpdateUserInput{FirstName: strPtr("Test")},
			wantErr:     true,
			errContains: "failed to get user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().UpdateUser(ctx, tt.userID, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.userID, result.ID)

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestDeleteUserMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create user to delete
	seeder := seeds.New(setup.pg.DB())
	userToDelete := seeder.User().
		WithEmail(fmt.Sprintf("delete-%s@example.com", uuid.New().String()[:8])).
		MustInsert()

	tests := []struct {
		name        string
		userID      uuid.UUID
		wantErr     bool
		errContains string
	}{
		{
			name:    "delete existing user",
			userID:  userToDelete.ID,
			wantErr: false,
		},
		{
			name:        "delete non-existent user",
			userID:      uuid.New(),
			wantErr:     true,
			errContains: "failed to get user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().DeleteUser(ctx, tt.userID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.True(t, result)

			// Verify user is actually deleted
			_, err = setup.userService.GetByID(ctx, tt.userID)
			assert.Error(t, err) // Should not find deleted user
		})
	}
}

func TestUpdateUserSettingsMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create test user
	seeder := seeds.New(setup.pg.DB())
	testUser := seeder.User().
		WithEmail(fmt.Sprintf("settings-%s@example.com", uuid.New().String()[:8])).
		MustInsert()

	tests := []struct {
		name        string
		userID      uuid.UUID
		input       generated.UpdateUserSettingsInput
		wantErr     bool
		errContains string
		validate    func(t *testing.T, result *user.User)
	}{
		{
			name:   "update risk level",
			userID: testUser.ID,
			input: generated.UpdateUserSettingsInput{
				RiskLevel: strPtr("aggressive"),
			},
			wantErr: false,
			validate: func(t *testing.T, result *user.User) {
				assert.Equal(t, "aggressive", result.Settings.RiskLevel)
			},
		},
		{
			name:   "update max positions",
			userID: testUser.ID,
			input: generated.UpdateUserSettingsInput{
				MaxPositions: intPtr(10),
			},
			wantErr: false,
			validate: func(t *testing.T, result *user.User) {
				assert.Equal(t, 10, result.Settings.MaxPositions)
			},
		},
		{
			name:   "update notifications",
			userID: testUser.ID,
			input: generated.UpdateUserSettingsInput{
				NotificationsOn: boolPtr(false),
			},
			wantErr: false,
			validate: func(t *testing.T, result *user.User) {
				assert.False(t, result.Settings.NotificationsOn)
			},
		},
		{
			name:   "update circuit breaker",
			userID: testUser.ID,
			input: generated.UpdateUserSettingsInput{
				CircuitBreakerOn: boolPtr(false),
			},
			wantErr: false,
			validate: func(t *testing.T, result *user.User) {
				assert.False(t, result.Settings.CircuitBreakerOn)
			},
		},
		{
			name:   "update position limits",
			userID: testUser.ID,
			input: generated.UpdateUserSettingsInput{
				MaxPositionSizeUsd:  float64Ptr(5000.0),
				MaxTotalExposureUsd: float64Ptr(20000.0),
				MinPositionSizeUsd:  float64Ptr(50.0),
			},
			wantErr: false,
			validate: func(t *testing.T, result *user.User) {
				assert.Equal(t, 5000.0, result.Settings.MaxPositionSizeUSD)
				assert.Equal(t, 20000.0, result.Settings.MaxTotalExposureUSD)
				assert.Equal(t, 50.0, result.Settings.MinPositionSizeUSD)
			},
		},
		{
			name:   "update AI settings",
			userID: testUser.ID,
			input: generated.UpdateUserSettingsInput{
				DefaultAIProvider: strPtr("openai"),
				DefaultAIModel:    strPtr("gpt-4"),
			},
			wantErr: false,
			validate: func(t *testing.T, result *user.User) {
				assert.Equal(t, "openai", result.Settings.DefaultAIProvider)
				assert.Equal(t, "gpt-4", result.Settings.DefaultAIModel)
			},
		},
		{
			name:   "update allowed exchanges",
			userID: testUser.ID,
			input: generated.UpdateUserSettingsInput{
				AllowedExchanges: []string{"binance", "bybit", "okx"},
			},
			wantErr: false,
			validate: func(t *testing.T, result *user.User) {
				assert.ElementsMatch(t, []string{"binance", "bybit", "okx"}, result.Settings.AllowedExchanges)
			},
		},
		{
			name:        "non-existent user",
			userID:      uuid.New(),
			input:       generated.UpdateUserSettingsInput{RiskLevel: strPtr("moderate")},
			wantErr:     true,
			errContains: "failed to get user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().UpdateUserSettings(ctx, tt.userID, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.userID, result.ID)

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestSetUserActiveMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create test user
	seeder := seeds.New(setup.pg.DB())
	testUser := seeder.User().
		WithEmail(fmt.Sprintf("active-%s@example.com", uuid.New().String()[:8])).
		MustInsert()

	tests := []struct {
		name        string
		userID      uuid.UUID
		isActive    bool
		wantErr     bool
		errContains string
	}{
		{
			name:     "deactivate user",
			userID:   testUser.ID,
			isActive: false,
			wantErr:  false,
		},
		{
			name:     "activate user",
			userID:   testUser.ID,
			isActive: true,
			wantErr:  false,
		},
		{
			name:        "non-existent user",
			userID:      uuid.New(),
			isActive:    true,
			wantErr:     true,
			errContains: "failed to get user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().SetUserActive(ctx, tt.userID, tt.isActive)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.userID, result.ID)
			assert.Equal(t, tt.isActive, result.IsActive)
		})
	}
}

func TestSetUserPremiumMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create test user
	seeder := seeds.New(setup.pg.DB())
	testUser := seeder.User().
		WithEmail(fmt.Sprintf("premium-%s@example.com", uuid.New().String()[:8])).
		MustInsert()

	tests := []struct {
		name        string
		userID      uuid.UUID
		isPremium   bool
		wantErr     bool
		errContains string
	}{
		{
			name:      "set premium to true",
			userID:    testUser.ID,
			isPremium: true,
			wantErr:   false,
		},
		{
			name:      "set premium to false",
			userID:    testUser.ID,
			isPremium: false,
			wantErr:   false,
		},
		{
			name:        "non-existent user",
			userID:      uuid.New(),
			isPremium:   true,
			wantErr:     true,
			errContains: "failed to get user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().SetUserPremium(ctx, tt.userID, tt.isPremium)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.userID, result.ID)
			assert.Equal(t, tt.isPremium, result.IsPremium)
		})
	}
}

func TestUserMutationsWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Full lifecycle test: create → update → update settings → set premium → deactivate → delete
	t.Run("complete user lifecycle", func(t *testing.T) {
		// 1. Create user
		createInput := generated.CreateUserInput{
			Email:        strPtr(fmt.Sprintf("lifecycle-%s@example.com", uuid.New().String()[:8])),
			FirstName:    "Lifecycle",
			LastName:     "Test",
			LanguageCode: "en",
		}

		created, err := setup.resolver.Mutation().CreateUser(ctx, createInput)
		require.NoError(t, err)
		assert.True(t, created.IsActive)
		assert.False(t, created.IsPremium)

		// 2. Update user info
		updateInput := generated.UpdateUserInput{
			FirstName: strPtr("Updated"),
			LastName:  strPtr("User"),
		}

		updated, err := setup.resolver.Mutation().UpdateUser(ctx, created.ID, updateInput)
		require.NoError(t, err)
		assert.Equal(t, "Updated", updated.FirstName)
		assert.Equal(t, "User", updated.LastName)

		// 3. Update settings
		settingsInput := generated.UpdateUserSettingsInput{
			RiskLevel:    strPtr("aggressive"),
			MaxPositions: intPtr(10),
		}

		withSettings, err := setup.resolver.Mutation().UpdateUserSettings(ctx, created.ID, settingsInput)
		require.NoError(t, err)
		assert.Equal(t, "aggressive", withSettings.Settings.RiskLevel)
		assert.Equal(t, 10, withSettings.Settings.MaxPositions)

		// 4. Set premium
		premium, err := setup.resolver.Mutation().SetUserPremium(ctx, created.ID, true)
		require.NoError(t, err)
		assert.True(t, premium.IsPremium)

		// 5. Deactivate user
		deactivated, err := setup.resolver.Mutation().SetUserActive(ctx, created.ID, false)
		require.NoError(t, err)
		assert.False(t, deactivated.IsActive)

		// 6. Delete user
		deleted, err := setup.resolver.Mutation().DeleteUser(ctx, created.ID)
		require.NoError(t, err)
		assert.True(t, deleted)
	})
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

func float64Ptr(f float64) *float64 {
	return &f
}
