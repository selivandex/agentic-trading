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

	fixtures := NewTestFixtures(t, testDB.Tx())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.Tx())
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

	fixtures := NewTestFixtures(t, testDB.Tx())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.Tx())
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

	fixtures := NewTestFixtures(t, testDB.Tx())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.Tx())
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

	fixtures := NewTestFixtures(t, testDB.Tx())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.Tx())
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

	fixtures := NewTestFixtures(t, testDB.Tx())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.Tx())
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

	fixtures := NewTestFixtures(t, testDB.Tx())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.Tx())
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

	fixtures := NewTestFixtures(t, testDB.Tx())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.Tx())
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

	fixtures := NewTestFixtures(t, testDB.Tx())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.Tx())
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

	fixtures := NewTestFixtures(t, testDB.Tx())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.Tx())
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

	fixtures := NewTestFixtures(t, testDB.Tx())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.Tx())
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

func TestExchangeAccountRepository_UpdateListenKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.Tx())
	userID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.Tx())
	ctx := context.Background()

	// Create account without listenKey
	account := &exchange_account.ExchangeAccount{
		ID:              uuid.New(),
		UserID:          userID,
		Exchange:        exchange_account.ExchangeBinance,
		Label:           "Binance Account",
		APIKeyEncrypted: []byte("encrypted_api_key"),
		SecretEncrypted: []byte("encrypted_secret"),
		IsTestnet:       true,
		Permissions:     []string{"futures", "trade"},
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err := repo.Create(ctx, account)
	require.NoError(t, err)

	// Verify listenKey is empty initially
	retrieved, err := repo.GetByID(ctx, account.ID)
	require.NoError(t, err)
	// PostgreSQL stores NULL bytea as empty slice, not nil
	assert.Empty(t, retrieved.ListenKeyEncrypted, "ListenKeyEncrypted should be empty initially")
	assert.Nil(t, retrieved.ListenKeyExpiresAt, "ListenKeyExpiresAt should be nil initially")

	// Update listenKey
	expiresAt := time.Now().Add(60 * time.Minute)
	account.ListenKeyEncrypted = []byte("encrypted_listen_key_abc123")
	account.ListenKeyExpiresAt = &expiresAt

	err = repo.Update(ctx, account)
	require.NoError(t, err, "Update should save listenKey to database")

	// Retrieve again and verify listenKey was persisted
	retrieved, err = repo.GetByID(ctx, account.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved.ListenKeyEncrypted, "ListenKeyEncrypted should be persisted")
	assert.NotNil(t, retrieved.ListenKeyExpiresAt, "ListenKeyExpiresAt should be persisted")
	assert.Equal(t, []byte("encrypted_listen_key_abc123"), retrieved.ListenKeyEncrypted)
	assert.WithinDuration(t, expiresAt, *retrieved.ListenKeyExpiresAt, time.Second)

	// Update listenKey to new value (simulate renewal)
	newExpiresAt := time.Now().Add(120 * time.Minute)
	account.ListenKeyEncrypted = []byte("encrypted_listen_key_new_xyz789")
	account.ListenKeyExpiresAt = &newExpiresAt

	err = repo.Update(ctx, account)
	require.NoError(t, err)

	// Verify new listenKey
	retrieved, err = repo.GetByID(ctx, account.ID)
	require.NoError(t, err)
	assert.Equal(t, []byte("encrypted_listen_key_new_xyz789"), retrieved.ListenKeyEncrypted)
	assert.WithinDuration(t, newExpiresAt, *retrieved.ListenKeyExpiresAt, time.Second)

	// Clear listenKey (set to nil)
	account.ListenKeyEncrypted = nil
	account.ListenKeyExpiresAt = nil

	err = repo.Update(ctx, account)
	require.NoError(t, err)

	// Verify listenKey is cleared
	retrieved, err = repo.GetByID(ctx, account.ID)
	require.NoError(t, err)
	// PostgreSQL stores empty bytea as []byte{} not nil
	assert.True(t, len(retrieved.ListenKeyEncrypted) == 0, "ListenKeyEncrypted should be empty after clearing")
	assert.Nil(t, retrieved.ListenKeyExpiresAt, "ListenKeyExpiresAt should be nil after clearing")
}

func TestExchangeAccountRepository_GetAllActiveWithListenKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.Tx())
	user1ID := fixtures.CreateUser()
	user2ID := fixtures.CreateUser()

	repo := NewExchangeAccountRepository(testDB.Tx())
	ctx := context.Background()

	expiresAt := time.Now().Add(60 * time.Minute)

	// Create active account with listenKey for user1
	account1 := &exchange_account.ExchangeAccount{
		ID:                 uuid.New(),
		UserID:             user1ID,
		Exchange:           exchange_account.ExchangeBinance,
		Label:              "User1 Binance",
		APIKeyEncrypted:    []byte("encrypted_key_1"),
		SecretEncrypted:    []byte("encrypted_secret_1"),
		IsTestnet:          true,
		Permissions:        []string{"futures"},
		IsActive:           true,
		ListenKeyEncrypted: []byte("encrypted_listen_key_1"),
		ListenKeyExpiresAt: &expiresAt,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	err := repo.Create(ctx, account1)
	require.NoError(t, err)

	// Create active account without listenKey for user2
	account2 := &exchange_account.ExchangeAccount{
		ID:              uuid.New(),
		UserID:          user2ID,
		Exchange:        exchange_account.ExchangeBybit,
		Label:           "User2 Bybit",
		APIKeyEncrypted: []byte("encrypted_key_2"),
		SecretEncrypted: []byte("encrypted_secret_2"),
		IsTestnet:       false,
		Permissions:     []string{"spot"},
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	err = repo.Create(ctx, account2)
	require.NoError(t, err)

	// Create inactive account (should not be returned)
	account3 := &exchange_account.ExchangeAccount{
		ID:              uuid.New(),
		UserID:          user1ID,
		Exchange:        exchange_account.ExchangeOKX,
		Label:           "Inactive OKX",
		APIKeyEncrypted: []byte("encrypted_key_3"),
		SecretEncrypted: []byte("encrypted_secret_3"),
		IsTestnet:       false,
		Permissions:     []string{"futures"},
		IsActive:        false,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	err = repo.Create(ctx, account3)
	require.NoError(t, err)

	// Get all active accounts
	accounts, err := repo.GetAllActive(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(accounts), 2, "Should return at least 2 active accounts")

	// Find account1 and verify listenKey
	var foundAccount1 *exchange_account.ExchangeAccount
	for _, acc := range accounts {
		if acc.ID == account1.ID {
			foundAccount1 = acc
			break
		}
	}
	require.NotNil(t, foundAccount1, "Should find account1 in results")

	// Re-fetch account1 directly to ensure we have the latest data
	account1Fresh, err := repo.GetByID(ctx, account1.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, account1Fresh.ListenKeyEncrypted, "account1 should have ListenKeyEncrypted")
	assert.NotNil(t, account1Fresh.ListenKeyExpiresAt, "account1 should have ListenKeyExpiresAt")
	assert.Equal(t, []byte("encrypted_listen_key_1"), account1Fresh.ListenKeyEncrypted)

	// Verify account2 has no listenKey
	var foundAccount2 *exchange_account.ExchangeAccount
	for _, acc := range accounts {
		if acc.ID == account2.ID {
			foundAccount2 = acc
			break
		}
	}
	require.NotNil(t, foundAccount2, "Should find account2 in results")
	account2Fresh, err := repo.GetByID(ctx, account2.ID)
	require.NoError(t, err)
	assert.Empty(t, account2Fresh.ListenKeyEncrypted, "account2 should not have ListenKeyEncrypted")
	assert.Nil(t, account2Fresh.ListenKeyExpiresAt, "account2 should not have ListenKeyExpiresAt")
}
