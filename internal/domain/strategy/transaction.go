package strategy

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Transaction represents a single cash movement in a strategy
// Every change to strategy.CashReserve must create a transaction record
type Transaction struct {
	ID         uuid.UUID `db:"id"`
	StrategyID uuid.UUID `db:"strategy_id"`
	UserID     uuid.UUID `db:"user_id"`

	// Transaction details
	Type          TransactionType `db:"type"`
	Amount        decimal.Decimal `db:"amount"`         // Positive = credit, Negative = debit
	BalanceBefore decimal.Decimal `db:"balance_before"` // cash_reserve before this transaction
	BalanceAfter  decimal.Decimal `db:"balance_after"`  // cash_reserve after (balance_before + amount)

	// Related entities (optional)
	PositionID *uuid.UUID `db:"position_id"`
	OrderID    *uuid.UUID `db:"order_id"`

	// Metadata
	Description string          `db:"description"`
	Metadata    json.RawMessage `db:"metadata"`

	// Timestamp
	CreatedAt time.Time `db:"created_at"`
}

// TransactionType represents the type of transaction
type TransactionType string

const (
	// Inflows (positive amounts)
	TransactionDeposit       TransactionType = "deposit"        // Initial or additional funding
	TransactionPositionClose TransactionType = "position_close" // Position closed (amount = realized PnL)
	TransactionDividend      TransactionType = "dividend"       // Dividend received
	TransactionInterest      TransactionType = "interest"       // Interest earned

	// Outflows (negative amounts)
	TransactionWithdrawal   TransactionType = "withdrawal"    // Withdraw funds
	TransactionPositionOpen TransactionType = "position_open" // Cash allocated to position
	TransactionFee          TransactionType = "fee"           // Trading fee
	TransactionFundingRate  TransactionType = "funding_rate"  // Funding rate payment (futures)

	// Special
	TransactionAdjustment TransactionType = "adjustment" // Manual adjustment (admin)
)

// Valid checks if transaction type is valid
func (t TransactionType) Valid() bool {
	switch t {
	case TransactionDeposit, TransactionWithdrawal,
		TransactionPositionOpen, TransactionPositionClose,
		TransactionFee, TransactionFundingRate,
		TransactionDividend, TransactionInterest,
		TransactionAdjustment:
		return true
	default:
		return false
	}
}

// IsCredit returns true if this transaction type typically adds money
func (t TransactionType) IsCredit() bool {
	switch t {
	case TransactionDeposit, TransactionPositionClose, TransactionDividend, TransactionInterest:
		return true
	default:
		return false
	}
}

// IsDebit returns true if this transaction type typically removes money
func (t TransactionType) IsDebit() bool {
	switch t {
	case TransactionWithdrawal, TransactionPositionOpen, TransactionFee, TransactionFundingRate:
		return true
	default:
		return false
	}
}

// TransactionMetadata contains additional transaction-specific data
type TransactionMetadata struct {
	// For position_open/close
	Symbol     string  `json:"symbol,omitempty"`
	Side       string  `json:"side,omitempty"`
	EntryPrice float64 `json:"entry_price,omitempty"`
	ExitPrice  float64 `json:"exit_price,omitempty"`
	Size       float64 `json:"size,omitempty"`
	PnL        float64 `json:"pnl,omitempty"`

	// For fees
	FeeType string  `json:"fee_type,omitempty"` // maker, taker, withdrawal
	FeeRate float64 `json:"fee_rate,omitempty"` // percentage

	// For adjustments
	Reason      string `json:"reason,omitempty"`
	AdminUserID string `json:"admin_user_id,omitempty"`

	// Generic
	Notes string `json:"notes,omitempty"`
}

// ParseMetadata parses transaction metadata
func (t *Transaction) ParseMetadata() (*TransactionMetadata, error) {
	if len(t.Metadata) == 0 {
		return &TransactionMetadata{}, nil
	}

	var metadata TransactionMetadata
	if err := json.Unmarshal(t.Metadata, &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}

// SetMetadata sets transaction metadata
func (t *Transaction) SetMetadata(metadata *TransactionMetadata) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	t.Metadata = data
	return nil
}

// NewTransaction creates a new transaction (helper)
func NewTransaction(
	strategyID uuid.UUID,
	userID uuid.UUID,
	txType TransactionType,
	amount decimal.Decimal,
	balanceBefore decimal.Decimal,
	description string,
) *Transaction {
	balanceAfter := balanceBefore.Add(amount)

	return &Transaction{
		ID:            uuid.New(),
		StrategyID:    strategyID,
		UserID:        userID,
		Type:          txType,
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		Description:   description,
		Metadata:      []byte("{}"), // Initialize with empty JSON object (PostgreSQL requirement)
		CreatedAt:     time.Now(),
	}
}

// Validation helpers

// Validate checks if transaction is valid
func (t *Transaction) Validate() error {
	if t.StrategyID == uuid.Nil {
		return ErrInvalidStrategyID
	}
	if t.UserID == uuid.Nil {
		return ErrInvalidUserID
	}
	if !t.Type.Valid() {
		return ErrInvalidTransactionType
	}
	if t.Amount.IsZero() {
		return ErrZeroAmount
	}
	if t.BalanceBefore.IsNegative() {
		return ErrNegativeBalance
	}
	if t.BalanceAfter.IsNegative() {
		return ErrNegativeBalance
	}
	// Verify math integrity: balance_before + amount = balance_after
	expectedBalance := t.BalanceBefore.Add(t.Amount)
	if !expectedBalance.Equal(t.BalanceAfter) {
		return ErrBalanceMismatch
	}
	return nil
}

// Errors
var (
	ErrInvalidStrategyID      = &TransactionError{Code: "invalid_strategy_id", Message: "strategy_id is required"}
	ErrInvalidUserID          = &TransactionError{Code: "invalid_user_id", Message: "user_id is required"}
	ErrInvalidTransactionType = &TransactionError{Code: "invalid_type", Message: "invalid transaction type"}
	ErrZeroAmount             = &TransactionError{Code: "zero_amount", Message: "amount cannot be zero"}
	ErrNegativeBalance        = &TransactionError{Code: "negative_balance", Message: "balance cannot be negative"}
	ErrBalanceMismatch        = &TransactionError{Code: "balance_mismatch", Message: "balance_before + amount must equal balance_after"}
)

// TransactionError represents a transaction-specific error
type TransactionError struct {
	Code    string
	Message string
}

func (e *TransactionError) Error() string {
	return e.Message
}
