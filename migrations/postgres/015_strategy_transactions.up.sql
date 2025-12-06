-- Create strategy transactions table
-- Note: strategy_transaction_type ENUM is created in 001_extensions_and_enums.up.sql
CREATE TABLE
  IF NOT EXISTS strategy_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    strategy_id UUID NOT NULL REFERENCES user_strategies (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- Transaction details
    type strategy_transaction_type NOT NULL,
    amount DECIMAL(20, 8) NOT NULL, -- Can be negative (withdrawals, fees)
    balance_before DECIMAL(20, 8) NOT NULL, -- cash_reserve balance before this transaction
    balance_after DECIMAL(20, 8) NOT NULL, -- cash_reserve balance after this transaction
    -- Related entity (optional)
    position_id UUID REFERENCES positions (id) ON DELETE SET NULL,
    order_id UUID REFERENCES orders (id) ON DELETE SET NULL,
    -- Metadata
    description TEXT,
    metadata JSONB, -- Additional transaction-specific data
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- Constraints
    CONSTRAINT valid_amount CHECK (amount != 0), -- Zero transactions don't make sense
    CONSTRAINT valid_balance_math CHECK (balance_before + amount = balance_after) -- Ensure math integrity
  );

-- Create indexes for efficient queries
CREATE INDEX idx_strategy_transactions_strategy_id ON strategy_transactions (strategy_id);

CREATE INDEX idx_strategy_transactions_user_id ON strategy_transactions (user_id);

CREATE INDEX idx_strategy_transactions_type ON strategy_transactions (type);

CREATE INDEX idx_strategy_transactions_created_at ON strategy_transactions (created_at DESC);

CREATE INDEX idx_strategy_transactions_position_id ON strategy_transactions (position_id)
WHERE
  position_id IS NOT NULL;

-- Create composite index for common query pattern (user's transactions in date range)
CREATE INDEX idx_strategy_transactions_user_created ON strategy_transactions (user_id, created_at DESC);

-- Add comments for documentation
COMMENT ON TABLE strategy_transactions IS 'Audit trail of all cash movements in user strategies';

COMMENT ON COLUMN strategy_transactions.amount IS 'Transaction amount. Positive = credit (deposit, profit), Negative = debit (withdrawal, fee, loss)';

COMMENT ON COLUMN strategy_transactions.balance_before IS 'Strategy cash_reserve balance before this transaction';

COMMENT ON COLUMN strategy_transactions.balance_after IS 'Strategy cash_reserve balance after this transaction (balance_before + amount = balance_after)';

COMMENT ON CONSTRAINT valid_balance_math ON strategy_transactions IS 'Ensures mathematical integrity: balance_before + amount must equal balance_after';
