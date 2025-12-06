-- Positions table
CREATE TABLE
  positions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    user_id UUID NOT NULL REFERENCES users (id),
    strategy_id UUID REFERENCES user_strategies (id) ON DELETE SET NULL,
    exchange_account_id UUID NOT NULL REFERENCES exchange_accounts (id),
    symbol VARCHAR(50) NOT NULL,
    market_type market_type NOT NULL,
    side position_side NOT NULL,
    size DECIMAL(20, 8) NOT NULL,
    entry_price DECIMAL(20, 8) NOT NULL,
    current_price DECIMAL(20, 8),
    liquidation_price DECIMAL(20, 8),
    leverage INTEGER DEFAULT 1,
    margin_mode VARCHAR(20) DEFAULT 'cross',
    unrealized_pnl DECIMAL(20, 8) DEFAULT 0,
    unrealized_pnl_pct DECIMAL(10, 4) DEFAULT 0,
    realized_pnl DECIMAL(20, 8) DEFAULT 0,
    stop_loss_price DECIMAL(20, 8),
    take_profit_price DECIMAL(20, 8),
    trailing_stop_pct DECIMAL(5, 2),
    stop_loss_order_id UUID REFERENCES orders (id),
    take_profit_order_id UUID REFERENCES orders (id),
    open_reasoning TEXT,
    status position_status NOT NULL DEFAULT 'open',
    opened_at TIMESTAMPTZ DEFAULT NOW (),
    closed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW ()
  );

-- Indexes
CREATE INDEX idx_positions_user ON positions (user_id);

CREATE INDEX idx_positions_status ON positions (status);

CREATE INDEX idx_positions_symbol ON positions (symbol);

CREATE INDEX idx_positions_strategy ON positions (strategy_id);

-- Comment
COMMENT ON COLUMN positions.strategy_id IS 'Link to user_strategies table for capital allocation and PnL tracking';

-- Trigger for updated_at
CREATE TRIGGER positions_updated_at BEFORE
UPDATE ON positions FOR EACH ROW EXECUTE FUNCTION update_updated_at ();
