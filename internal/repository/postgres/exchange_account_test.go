package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/testsupport"
)

func TestExchangeAccountRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.DB())
	ctx := context.Background()

	// Create exchange account with dummy encrypted credentials
	account := &exchange_account.ExchangeAccount{
		ID:              uuid.New(),
		UserID:          userID,
		Exchange:        exchange_account.ExchangeBinance,
		Label:           "Main Binance Account",
		APIKeyEncrypted: []byte("dummy_encrypted_api_key"),
		SecretEncrypted: []byte("dummy_encrypted_secret"),
		Passphrase:      nil,
		IsTestnet:       false,
		Permissions:     []string{"spot", "trade", "read"},
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Test Create
	err := repo.Create(ctx, account)
	require.NoError(t, err, "Create should not return error")

	// Verify account can be retrieved
	retrieved, err := repo.GetByID(ctx, account.ID)
	require.NoError(t, err)
	assert.Equal(t, account.Exchange, retrieved.Exchange)
	assert.Equal(t, account.Label, retrieved.Label)
	assert.Equal(t, account.Permissions, retrieved.Permissions)
}

func TestExchangeAccountRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.DB())
	ctx := context.Background()

	account := &exchange_account.ExchangeAccount{
		ID:              uuid.New(),
		UserID:          userID,
		Exchange:        exchange_account.ExchangeBybit,
		Label:           "Bybit Testnet",
		APIKeyEncrypted: []byte("encrypted_key"),
		SecretEncrypted: []byte("encrypted_secret"),
		IsTestnet:       true,
		Permissions:     []string{"futures", "read"},
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err := repo.Create(ctx, account)
	require.NoError(t, err)

	// Test GetByID
	retrieved, err := repo.GetByID(ctx, account.ID)
	require.NoError(t, err)
	assert.Equal(t, account.ID, retrieved.ID)
	assert.Equal(t, account.UserID, retrieved.UserID)
	assert.Equal(t, exchange_account.ExchangeBybit, retrieved.Exchange)
	assert.True(t, retrieved.IsTestnet)

	// Test non-existent ID
	_, err = repo.GetByID(ctx, uuid.New())
	assert.Error(t, err, "Should return error for non-existent ID")
}

func TestExchangeAccountRepository_GetByUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.DB())
	ctx := context.Background()

	// Create multiple accounts for same user
	exchanges := []exchange_account.ExchangeType{
		exchange_account.ExchangeBinance,
		exchange_account.ExchangeBybit,
		exchange_account.ExchangeOKX,
	}

	for i, exch := range exchanges {
		account := &exchange_account.ExchangeAccount{
			ID:              uuid.New(),
			UserID:          userID,
			Exchange:        exch,
			Label:           exch.String() + " Account",
			APIKeyEncrypted: []byte("encrypted_key_" + string(rune(i+'0'))),
			SecretEncrypted: []byte("encrypted_secret_" + string(rune(i+'0'))),
			IsTestnet:       false,
			Permissions:     []string{"spot", "trade"},
			IsActive:        i != 2, // Make OKX inactive
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		err := repo.Create(ctx, account)
		require.NoError(t, err)
	}

	// Test GetByUser - should return all accounts
	accounts, err := repo.GetByUser(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, accounts, 3, "Should return all 3 accounts")

	// Verify all belong to same user
	for _, acc := range accounts {
		assert.Equal(t, userID, acc.UserID)
	}
}

func TestExchangeAccountRepository_GetActiveByUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.DB())
	ctx := context.Background()

	// Create active account
	activeAccount := &exchange_account.ExchangeAccount{
		ID:              uuid.New(),
		UserID:          userID,
		Exchange:        exchange_account.ExchangeBinance,
		Label:           "Active Binance",
		APIKeyEncrypted: []byte("encrypted_key"),
		SecretEncrypted: []byte("encrypted_secret"),
		IsTestnet:       false,
		Permissions:     []string{"spot", "trade"},
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	err := repo.Create(ctx, activeAccount)
	require.NoError(t, err)

	// Create inactive account
	inactiveAccount := &exchange_account.ExchangeAccount{
		ID:              uuid.New(),
		UserID:          userID,
		Exchange:        exchange_account.ExchangeBybit,
		Label:           "Inactive Bybit",
		APIKeyEncrypted: []byte("encrypted_key_2"),
		SecretEncrypted: []byte("encrypted_secret_2"),
		IsTestnet:       false,
		Permissions:     []string{"futures"},
		IsActive:        false,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	err = repo.Create(ctx, inactiveAccount)
	require.NoError(t, err)

	// Test GetActiveByUser - should return only active
	accounts, err := repo.GetActiveByUser(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, accounts, 1, "Should return only active account")
	assert.Equal(t, activeAccount.ID, accounts[0].ID)
	assert.True(t, accounts[0].IsActive)
}

func TestExchangeAccountRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.DB())
	ctx := context.Background()

	// Create account
	account := &exchange_account.ExchangeAccount{
		ID:              uuid.New(),
		UserID:          userID,
		Exchange:        exchange_account.ExchangeBinance,
		Label:           "Original Label",
		APIKeyEncrypted: []byte("encrypted_key"),
		SecretEncrypted: []byte("encrypted_secret"),
		IsTestnet:       false,
		Permissions:     []string{"spot"},
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err := repo.Create(ctx, account)
	require.NoError(t, err)

	// Update account
	account.Label = "Updated Label"
	account.Permissions = []string{"spot", "futures", "trade"}
	account.IsActive = false

	err = repo.Update(ctx, account)
	require.NoError(t, err)

	// Verify updates
	retrieved, err := repo.GetByID(ctx, account.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Label", retrieved.Label)
	assert.Equal(t, []string{"spot", "futures", "trade"}, retrieved.Permissions)
	assert.False(t, retrieved.IsActive)
}

func TestExchangeAccountRepository_UpdateLastSync(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.DB())
	ctx := context.Background()

	account := &exchange_account.ExchangeAccount{
		ID:              uuid.New(),
		UserID:          userID,
		Exchange:        exchange_account.ExchangeBinance,
		Label:           "Test Account",
		APIKeyEncrypted: []byte("encrypted_key"),
		SecretEncrypted: []byte("encrypted_secret"),
		IsTestnet:       false,
		Permissions:     []string{"spot"},
		IsActive:        true,
		LastSyncAt:      nil, // No sync yet
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err := repo.Create(ctx, account)
	require.NoError(t, err)

	// Verify LastSyncAt is nil
	retrieved, err := repo.GetByID(ctx, account.ID)
	require.NoError(t, err)
	assert.Nil(t, retrieved.LastSyncAt)

	// Update last sync
	err = repo.UpdateLastSync(ctx, account.ID)
	require.NoError(t, err)

	// Verify LastSyncAt is now set
	retrieved, err = repo.GetByID(ctx, account.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved.LastSyncAt)
	assert.WithinDuration(t, time.Now(), *retrieved.LastSyncAt, 5*time.Second)
}

func TestExchangeAccountRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.DB())
	ctx := context.Background()

	account := &exchange_account.ExchangeAccount{
		ID:              uuid.New(),
		UserID:          userID,
		Exchange:        exchange_account.ExchangeBybit,
		Label:           "To Delete",
		APIKeyEncrypted: []byte("encrypted_key"),
		SecretEncrypted: []byte("encrypted_secret"),
		IsTestnet:       true,
		Permissions:     []string{"read"},
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err := repo.Create(ctx, account)
	require.NoError(t, err)

	// Verify account exists
	_, err = repo.GetByID(ctx, account.ID)
	require.NoError(t, err)

	// Delete account
	err = repo.Delete(ctx, account.ID)
	require.NoError(t, err)

	// Verify account is deleted
	_, err = repo.GetByID(ctx, account.ID)
	assert.Error(t, err, "Should return error after deletion")
}

func TestExchangeAccountRepository_OKXWithPassphrase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.DB())
	ctx := context.Background()

	// OKX requires passphrase
	account := &exchange_account.ExchangeAccount{
		ID:              uuid.New(),
		UserID:          userID,
		Exchange:        exchange_account.ExchangeOKX,
		Label:           "OKX Account",
		APIKeyEncrypted: []byte("encrypted_api_key"),
		SecretEncrypted: []byte("encrypted_secret"),
		Passphrase:      []byte("encrypted_passphrase"), // OKX-specific
		IsTestnet:       false,
		Permissions:     []string{"spot", "futures", "trade"},
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err := repo.Create(ctx, account)
	require.NoError(t, err)

	// Retrieve and verify passphrase is stored
	retrieved, err := repo.GetByID(ctx, account.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved.Passphrase)
	assert.Equal(t, []byte("encrypted_passphrase"), retrieved.Passphrase)
}

func TestExchangeAccountRepository_PermissionsArray(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.DB())
	ctx := context.Background()

	// Test with various permission combinations
	account := &exchange_account.ExchangeAccount{
		ID:              uuid.New(),
		UserID:          userID,
		Exchange:        exchange_account.ExchangeBinance,
		Label:           "Permissions Test",
		APIKeyEncrypted: []byte("encrypted_key"),
		SecretEncrypted: []byte("encrypted_secret"),
		IsTestnet:       false,
		Permissions:     []string{"spot", "futures", "margin", "trade", "read", "withdraw"},
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err := repo.Create(ctx, account)
	require.NoError(t, err)

	// Retrieve and verify permissions array
	retrieved, err := repo.GetByID(ctx, account.ID)
	require.NoError(t, err)
	assert.Len(t, retrieved.Permissions, 6)
	assert.Contains(t, retrieved.Permissions, "spot")
	assert.Contains(t, retrieved.Permissions, "trade")
	assert.Contains(t, retrieved.Permissions, "withdraw")
}

func TestExchangeAccountRepository_MultipleExchangeTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.DB())
	ctx := context.Background()

	// Test all exchange types
	exchanges := []exchange_account.ExchangeType{
		exchange_account.ExchangeBinance,
		exchange_account.ExchangeBybit,
		exchange_account.ExchangeOKX,
		exchange_account.ExchangeKucoin,
		exchange_account.ExchangeGate,
	}

	for _, exch := range exchanges {
		account := &exchange_account.ExchangeAccount{
			ID:              uuid.New(),
			UserID:          userID,
			Exchange:        exch,
			Label:           exch.String() + " Test",
			APIKeyEncrypted: []byte("encrypted_key"),
			SecretEncrypted: []byte("encrypted_secret"),
			IsTestnet:       false,
			Permissions:     []string{"read"},
			IsActive:        true,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		err := repo.Create(ctx, account)
		require.NoError(t, err, "Should create account for "+exch.String())
	}

	// Retrieve all and verify
	accounts, err := repo.GetByUser(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, accounts, 5, "Should have all 5 exchange types")

	// Verify each exchange type is present
	exchangeMap := make(map[exchange_account.ExchangeType]bool)
	for _, acc := range accounts {
		exchangeMap[acc.Exchange] = true
	}

	for _, exch := range exchanges {
		assert.True(t, exchangeMap[exch], exch.String()+" should be present")
	}
}

