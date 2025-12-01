package strategy_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"prometheus/internal/domain/strategy"
)

func TestNewTransaction_InitializesMetadata(t *testing.T) {
	strategyID := uuid.New()
	userID := uuid.New()

	tx := strategy.NewTransaction(
		strategyID,
		userID,
		strategy.TransactionDeposit,
		decimal.NewFromInt(1000),
		decimal.Zero,
		"Test deposit",
	)

	// Check that Metadata is initialized (not nil)
	assert.NotNil(t, tx.Metadata, "Metadata should not be nil")
	assert.Equal(t, "{}", string(tx.Metadata), "Metadata should be empty JSON object")

	// Check other fields
	assert.Equal(t, strategyID, tx.StrategyID)
	assert.Equal(t, userID, tx.UserID)
	assert.Equal(t, strategy.TransactionDeposit, tx.Type)
	assert.Equal(t, decimal.NewFromInt(1000), tx.Amount)
	assert.Equal(t, decimal.Zero, tx.BalanceBefore)
	assert.Equal(t, decimal.NewFromInt(1000), tx.BalanceAfter)
	assert.Equal(t, "Test deposit", tx.Description)
	assert.False(t, tx.CreatedAt.IsZero())
}

func TestTransaction_Validate(t *testing.T) {
	validTx := strategy.NewTransaction(
		uuid.New(),
		uuid.New(),
		strategy.TransactionDeposit,
		decimal.NewFromInt(1000),
		decimal.Zero,
		"Test",
	)

	err := validTx.Validate()
	assert.NoError(t, err, "Valid transaction should pass validation")
}

func TestTransaction_ParseMetadata_Empty(t *testing.T) {
	tx := strategy.NewTransaction(
		uuid.New(),
		uuid.New(),
		strategy.TransactionDeposit,
		decimal.NewFromInt(1000),
		decimal.Zero,
		"Test",
	)

	metadata, err := tx.ParseMetadata()
	assert.NoError(t, err)
	assert.NotNil(t, metadata)
}
