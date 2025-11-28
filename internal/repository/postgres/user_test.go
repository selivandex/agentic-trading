package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/user"
	"prometheus/internal/testsupport"
)

func TestUserRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewUserRepository(testDB.DB())
	ctx := context.Background()

	// Create test user with default settings
	u := &user.User{
		ID:               uuid.New(),
		TelegramID:       randomTelegramID(),
		TelegramUsername: "testuser_" + uuid.New().String()[:8],
		FirstName:        "Test",
		LastName:         "User",
		LanguageCode:     "en",
		IsActive:         true,
		IsPremium:        false,
		Settings:         user.DefaultSettings(),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Test Create
	err := repo.Create(ctx, u)
	require.NoError(t, err, "Create should not return error")

	// Verify user can be retrieved
	retrieved, err := repo.GetByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, u.TelegramID, retrieved.TelegramID)
	assert.Equal(t, u.TelegramUsername, retrieved.TelegramUsername)
	assert.Equal(t, u.FirstName, retrieved.FirstName)
	assert.Equal(t, u.Settings.RiskLevel, retrieved.Settings.RiskLevel)
}

func TestUserRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewUserRepository(testDB.DB())
	ctx := context.Background()

	// Create test user
	u := &user.User{
		ID:               uuid.New(),
		TelegramID:       randomTelegramID(),
		TelegramUsername: "getbyidtest_" + uuid.New().String()[:8],
		FirstName:        "GetByID",
		LastName:         "Test",
		LanguageCode:     "en",
		IsActive:         true,
		IsPremium:        true,
		Settings:         user.DefaultSettings(),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := repo.Create(ctx, u)
	require.NoError(t, err)

	// Test GetByID
	retrieved, err := repo.GetByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, u.ID, retrieved.ID)
	assert.Equal(t, u.TelegramID, retrieved.TelegramID)
	assert.Equal(t, u.IsPremium, retrieved.IsPremium)

	// Test non-existent ID
	_, err = repo.GetByID(ctx, uuid.New())
	assert.Error(t, err, "Should return error for non-existent ID")
}

func TestUserRepository_GetByTelegramID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewUserRepository(testDB.DB())
	ctx := context.Background()

	telegramID := randomTelegramID()

	// Create test user
	u := &user.User{
		ID:               uuid.New(),
		TelegramID:       telegramID,
		TelegramUsername: "telegramtest_" + uuid.New().String()[:8],
		FirstName:        "Telegram",
		LastName:         "Test",
		LanguageCode:     "ru",
		IsActive:         true,
		IsPremium:        false,
		Settings:         user.DefaultSettings(),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := repo.Create(ctx, u)
	require.NoError(t, err)

	// Test GetByTelegramID - critical for Telegram bot
	retrieved, err := repo.GetByTelegramID(ctx, telegramID)
	require.NoError(t, err)
	assert.Equal(t, u.ID, retrieved.ID)
	assert.Equal(t, telegramID, retrieved.TelegramID)
	assert.Equal(t, "ru", retrieved.LanguageCode)

	// Test non-existent Telegram ID
	_, err = repo.GetByTelegramID(ctx, 999999999)
	assert.Error(t, err, "Should return error for non-existent Telegram ID")
}

func TestUserRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewUserRepository(testDB.DB())
	ctx := context.Background()

	// Create initial user
	u := &user.User{
		ID:               uuid.New(),
		TelegramID:       randomTelegramID(),
		TelegramUsername: "updatetest_" + uuid.New().String()[:8],
		FirstName:        "Update",
		LastName:         "Test",
		LanguageCode:     "en",
		IsActive:         true,
		IsPremium:        false,
		Settings:         user.DefaultSettings(),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := repo.Create(ctx, u)
	require.NoError(t, err)

	// Update user fields
	u.FirstName = "Updated"
	u.LastName = "Name"
	u.TelegramUsername = "updated_username"
	u.IsPremium = true
	u.Settings.RiskLevel = "aggressive"
	u.Settings.MaxPositions = 5

	err = repo.Update(ctx, u)
	require.NoError(t, err)

	// Verify updates
	retrieved, err := repo.GetByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", retrieved.FirstName)
	assert.Equal(t, "Name", retrieved.LastName)
	assert.Equal(t, "updated_username", retrieved.TelegramUsername)
	assert.True(t, retrieved.IsPremium)
	assert.Equal(t, "aggressive", retrieved.Settings.RiskLevel)
	assert.Equal(t, 5, retrieved.Settings.MaxPositions)
}

func TestUserRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewUserRepository(testDB.DB())
	ctx := context.Background()

	// Create multiple users
	for i := 0; i < 5; i++ {
		u := &user.User{
			ID:               uuid.New(),
			TelegramID:       randomTelegramID(),
			TelegramUsername: "listtest_" + uuid.New().String()[:8],
			FirstName:        "User",
			LastName:         string(rune(i + '0')),
			LanguageCode:     "en",
			IsActive:         true,
			IsPremium:        i%2 == 0,
			Settings:         user.DefaultSettings(),
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		err := repo.Create(ctx, u)
		require.NoError(t, err)
	}

	// Test List with pagination
	users, err := repo.List(ctx, 3, 0)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(users), 3, "Should respect limit")

	// Test offset
	users2, err := repo.List(ctx, 2, 2)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(users2), 2, "Should respect limit with offset")
}

func TestUserRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewUserRepository(testDB.DB())
	ctx := context.Background()

	// Create user to delete
	u := &user.User{
		ID:               uuid.New(),
		TelegramID:       randomTelegramID(),
		TelegramUsername: "deletetest_" + uuid.New().String()[:8],
		FirstName:        "Delete",
		LastName:         "Test",
		LanguageCode:     "en",
		IsActive:         true,
		IsPremium:        false,
		Settings:         user.DefaultSettings(),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := repo.Create(ctx, u)
	require.NoError(t, err)

	// Verify user exists
	_, err = repo.GetByID(ctx, u.ID)
	require.NoError(t, err)

	// Delete user
	err = repo.Delete(ctx, u.ID)
	require.NoError(t, err)

	// Verify user is deleted
	_, err = repo.GetByID(ctx, u.ID)
	assert.Error(t, err, "Should return error after deletion")
}

func TestUserRepository_SettingsJSONB(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewUserRepository(testDB.DB())
	ctx := context.Background()

	// Create user with custom settings
	customSettings := user.Settings{
		DefaultAIProvider:   "openai",
		DefaultAIModel:      "gpt-4",
		RiskLevel:           "aggressive",
		MaxPositions:        10,
		MaxPortfolioRisk:    20.0,
		MaxDailyDrawdown:    10.0,
		MaxConsecutiveLoss:  5,
		NotificationsOn:     false,
		DailyReportTime:     "18:00",
		Timezone:            "America/New_York",
		CircuitBreakerOn:    false,
		MaxPositionSizeUSD:  5000.0,
		MaxTotalExposureUSD: 25000.0,
		MinPositionSizeUSD:  50.0,
		MaxLeverageMultiple: 3.0,
		AllowedExchanges:    []string{"binance", "bybit", "okx"},
	}

	u := &user.User{
		ID:               uuid.New(),
		TelegramID:       randomTelegramID(),
		TelegramUsername: "jsonbtest_" + uuid.New().String()[:8],
		FirstName:        "JSONB",
		LastName:         "Test",
		LanguageCode:     "en",
		IsActive:         true,
		IsPremium:        true,
		Settings:         customSettings,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Create and retrieve
	err := repo.Create(ctx, u)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(ctx, u.ID)
	require.NoError(t, err)

	// Verify JSONB integrity
	assert.Equal(t, "openai", retrieved.Settings.DefaultAIProvider)
	assert.Equal(t, "gpt-4", retrieved.Settings.DefaultAIModel)
	assert.Equal(t, "aggressive", retrieved.Settings.RiskLevel)
	assert.Equal(t, 10, retrieved.Settings.MaxPositions)
	assert.Equal(t, 20.0, retrieved.Settings.MaxPortfolioRisk)
	assert.Equal(t, 3.0, retrieved.Settings.MaxLeverageMultiple)
	assert.Len(t, retrieved.Settings.AllowedExchanges, 3)
	assert.Contains(t, retrieved.Settings.AllowedExchanges, "okx")
}
