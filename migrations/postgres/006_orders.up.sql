-- Orders table
CREATE TABLE
  orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    user_id UUID NOT NULL REFERENCES users (id),
    strategy_id UUID REFERENCES user_strategies (id) ON DELETE SET NULL,
    exchange_account_id UUID NOT NULL REFERENCES exchange_accounts (id),
    exchange_order_id VARCHAR(255),
    symbol VARCHAR(50) NOT NULL,
    market_type market_type NOT NULL,
    side order_side NOT NULL,
    type order_type NOT NULL,
    status order_status NOT NULL DEFAULT 'pending',
    price DECIMAL(20, 8),
    amount DECIMAL(20, 8) NOT NULL,
    filled_amount DECIMAL(20, 8) DEFAULT 0,
    avg_fill_price DECIMAL(20, 8),
    stop_price DECIMAL(20, 8),
    reduce_only BOOLEAN DEFAULT false,
    agent_id VARCHAR(100),
    reasoning TEXT,
    parent_order_id UUID REFERENCES orders (id),
    fee DECIMAL(20, 8) DEFAULT 0,
    fee_currency VARCHAR(20),
    created_at TIMESTAMPTZ DEFAULT NOW (),
    updated_at TIMESTAMPTZ DEFAULT NOW (),
    filled_at TIMESTAMPTZ
  );

-- Indexes
CREATE INDEX idx_orders_user ON orders (user_id);

CREATE INDEX idx_orders_status ON orders (status);

CREATE INDEX idx_orders_exchange ON orders (exchange_order_id);

CREATE INDEX idx_orders_symbol ON orders (symbol);

CREATE INDEX idx_orders_created ON orders (created_at DESC);

-- Trigger for updated_at
CREATE TRIGGER orders_updated_at BEFORE
UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION update_updated_at ();
