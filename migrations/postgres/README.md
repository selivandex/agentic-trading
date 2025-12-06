# PostgreSQL Migrations

This directory contains database migrations in the correct dependency order.

## Migration Order & Dependencies

### 001 - Extensions and ENUMs
- **Dependencies:** None
- **Creates:** PostgreSQL extensions (uuid-ossp, vector) and all ENUM types
- **Must run first:** All other migrations depend on ENUMs

### 002 - Users
- **Dependencies:** 001 (ENUMs)
- **Creates:** `users` table with both Telegram and email/password authentication
- **Base table:** Almost all other tables reference this table
- **Includes:** `update_updated_at()` function used by multiple tables

### 003 - Exchange Accounts
- **Dependencies:** 002 (users)
- **Creates:** `exchange_accounts` table for storing encrypted API credentials
- **References:** users.id

### 004 - User Strategies
- **Dependencies:** 002 (users)
- **Creates:** `user_strategies` table for portfolio management
- **References:** users.id
- **Used by:** orders, positions, portfolio_decisions, strategy_transactions

### 005 - Fund Watchlist
- **Dependencies:** 001 (ENUMs)
- **Creates:** `fund_watchlist` table with default crypto pairs
- **Independent:** No foreign keys to other tables

### 006 - Orders
- **Dependencies:** 002 (users), 003 (exchange_accounts), 004 (user_strategies)
- **Creates:** `orders` table
- **Self-reference:** parent_order_id references orders.id
- **Referenced by:** positions (for stop-loss/take-profit orders)

### 007 - Positions
- **Dependencies:** 002 (users), 003 (exchange_accounts), 004 (user_strategies), 006 (orders)
- **Creates:** `positions` table
- **Includes:** strategy_id column (no separate migration needed)
- **Referenced by:** strategy_transactions

### 008 - Journal and Risk
- **Dependencies:** 002 (users)
- **Creates:**
  - `journal_entries` - trading journal
  - `circuit_breaker_states` - risk management state
  - `risk_events` - risk event log

### 009 - Agent Memories
- **Dependencies:** 002 (users)
- **Creates:** `memories` table with vector embeddings
- **Includes:** memory_scope ENUM (user/collective/working)

### 010 - Performance Indexes
- **Dependencies:** 006 (orders), 007 (positions), 008 (risk tables)
- **Creates:** Performance indexes for frequently queried tables
- **No tables:** Only creates indexes

### 011 - ADK Sessions
- **Dependencies:** None (stores ADK session state)
- **Creates:**
  - `adk_sessions` - Google ADK session state
  - `adk_events` - ADK event log

### 012 - Limit Profiles
- **Dependencies:** 002 (users)
- **Creates:** `limit_profiles` table
- **Adds:** `limit_profile_id` column to users table (with FK)

### 013 - Agents
- **Dependencies:** 012 (limit_profiles)
- **Creates:** `agents` table with AI model configuration
- **Referenced by:** portfolio_decisions

### 014 - Portfolio Decisions
- **Dependencies:** 002 (users), 004 (user_strategies), 006 (orders), 013 (agents)
- **Creates:** `portfolio_decisions` table
- **Purpose:** Stores both PortfolioManager decisions AND expert agent analyses
- **Self-reference:** parent_decision_id for expert analyses

### 015 - Strategy Transactions
- **Dependencies:** 002 (users), 004 (user_strategies), 006 (orders), 007 (positions)
- **Creates:** `strategy_transactions` table
- **Purpose:** Ledger for strategy cash flows (deposits, fees, PnL, etc)
- **Includes:** strategy_transaction_type ENUM

## Running Migrations

Migrations are applied in numerical order by the migration tool (e.g., golang-migrate).

```bash
# Apply all migrations
make migrate-up

# Rollback last migration
make migrate-down

# Rollback all migrations
make migrate-down-all
```

## Key Design Decisions

1. **All ENUMs in one migration:** Easier to maintain and reference
2. **Email/password in users:** No separate migration needed
3. **strategy_id in positions:** Included in table creation, not separate migration
4. **No trading_pairs table:** Deprecated in favor of fund_watchlist
5. **Limit profiles separate:** Allows flexible user tier management

## Foreign Key Dependency Graph

```
extensions_and_enums (001)
  └── users (002)
        ├── exchange_accounts (003)
        │     ├── orders (006) ────┐
        │     └── positions (007)  │
        ├── user_strategies (004)  │
        │     ├── orders (006) ────┤
        │     ├── positions (007)  │
        │     ├── portfolio_decisions (014)
        │     └── strategy_transactions (015)
        ├── journal_entries (008)
        ├── circuit_breaker_states (008)
        ├── risk_events (008)
        ├── memories (009)
        └── limit_profiles (012)
              └── agents (013)
                    └── portfolio_decisions (014)

fund_watchlist (005) - independent
adk_sessions (011) - independent
performance_indexes (010) - depends on multiple tables
```

## Rollback Order

Migrations should be rolled back in reverse order (015 → 001) to respect foreign key constraints.

## Notes

- All timestamps use `TIMESTAMPTZ` for timezone awareness
- All UUIDs use `uuid_generate_v4()` or `gen_random_uuid()`
- All monetary values use `DECIMAL(20,8)` for precision
- Indexes are created inline with tables where possible
- Comments are added for explainability
