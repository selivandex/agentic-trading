-- Fund Watchlist table (global symbols monitored by the fund)
CREATE TABLE
  fund_watchlist (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    symbol VARCHAR(50) NOT NULL UNIQUE,
    market_type market_type NOT NULL,
    -- Metadata
    category VARCHAR(50), -- "major", "defi", "layer1", etc
    tier INTEGER DEFAULT 3, -- 1=BTC/ETH, 2=top20, 3=top100
    -- State
    is_active BOOLEAN DEFAULT true,
    is_paused BOOLEAN DEFAULT false,
    paused_reason TEXT,
    -- Analytics
    last_analyzed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW (),
    updated_at TIMESTAMPTZ DEFAULT NOW (),
    CONSTRAINT unique_symbol_market UNIQUE (symbol, market_type)
  );

-- Indexes
CREATE INDEX idx_fund_watchlist_active ON fund_watchlist (is_active, is_paused);

CREATE INDEX idx_fund_watchlist_symbol ON fund_watchlist (symbol);

-- Insert default watchlist
INSERT INTO
  fund_watchlist (symbol, market_type, category, tier)
VALUES
  ('BTC/USDT', 'spot', 'major', 1),
  ('ETH/USDT', 'spot', 'major', 1),
  ('BNB/USDT', 'spot', 'exchange', 2),
  ('SOL/USDT', 'spot', 'layer1', 2),
  ('XRP/USDT', 'spot', 'major', 2),
  ('ADA/USDT', 'spot', 'layer1', 2)
