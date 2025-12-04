-- migrations/clickhouse/009_add_websocket_columns.down.sql
-- Rollback WebSocket columns

ALTER TABLE ohlcv
    DROP COLUMN IF EXISTS market_type,
    DROP COLUMN IF EXISTS close_time,
    DROP COLUMN IF EXISTS taker_buy_base_volume,
    DROP COLUMN IF EXISTS taker_buy_quote_volume,
    DROP COLUMN IF EXISTS is_closed,
    DROP COLUMN IF EXISTS event_time;



