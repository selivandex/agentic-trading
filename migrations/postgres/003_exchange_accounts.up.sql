-- Exchange Accounts table

CREATE TABLE exchange_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Exchange details
    exchange exchange_type NOT NULL,
    label VARCHAR(255),

    -- Encrypted credentials
    api_key_encrypted BYTEA NOT NULL,
    secret_encrypted BYTEA NOT NULL,
    passphrase BYTEA,

    -- Configuration
    is_testnet BOOLEAN DEFAULT false,
    permissions TEXT[] DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    last_sync_at TIMESTAMPTZ,

    -- User Data WebSocket fields (for real-time order/position updates)
    listen_key_encrypted BYTEA,
    listen_key_expires_at TIMESTAMPTZ,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT unique_user_exchange_label UNIQUE(user_id, exchange, label)
);

-- Indexes
CREATE INDEX idx_exchange_accounts_user ON exchange_accounts(user_id);
CREATE INDEX idx_exchange_accounts_active ON exchange_accounts(is_active) WHERE is_active = true;
CREATE INDEX idx_exchange_accounts_listen_key_expires ON exchange_accounts(listen_key_expires_at)
    WHERE listen_key_expires_at IS NOT NULL AND is_active = true;

-- Trigger for updated_at
CREATE TRIGGER exchange_accounts_updated_at BEFORE UPDATE ON exchange_accounts
FOR EACH ROW EXECUTE FUNCTION update_updated_at();
