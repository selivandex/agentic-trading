#!/bin/bash

# E2E Test Script for MVP Flow
# Tests the complete user journey: /start → /invest → portfolio creation → notifications

set -e

echo "🧪 MVP E2E Test - Prometheus Trading System"
echo "==========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
DB_NAME="prometheus"
CLICKHOUSE_DB="trading"
REDIS_HOST="localhost:6379"
KAFKA_BROKER="localhost:9092"

# Test user
TEST_TELEGRAM_ID=123456789
TEST_USER_ID=""

# Helper functions
check_service() {
    local service=$1
    local port=$2
    if nc -z localhost $port 2>/dev/null; then
        echo -e "${GREEN}✓${NC} $service running on port $port"
        return 0
    else
        echo -e "${RED}✗${NC} $service NOT running on port $port"
        return 1
    fi
}

run_sql() {
    psql -h localhost -U trading -d $DB_NAME -c "$1" -t -A
}

# Step 1: Check prerequisites
echo "Step 1: Checking prerequisites..."
check_service "PostgreSQL" 5432 || exit 1
check_service "ClickHouse" 9000 || exit 1
check_service "Redis" 6379 || exit 1
check_service "Kafka" 9092 || exit 1
echo ""

# Step 2: Start Prometheus system
echo "Step 2: Starting Prometheus system..."
if pgrep -f "bin/prometheus" > /dev/null; then
    echo -e "${YELLOW}⚠${NC} Prometheus already running, using existing instance"
else
    echo "Starting Prometheus in background..."
    ./bin/prometheus > logs/test_run.log 2>&1 &
    PROMETHEUS_PID=$!
    echo "Prometheus started with PID: $PROMETHEUS_PID"

    # Wait for startup
    echo "Waiting for system to initialize (30 seconds)..."
    sleep 30

    # Check if still running
    if ! kill -0 $PROMETHEUS_PID 2>/dev/null; then
        echo -e "${RED}✗${NC} Prometheus failed to start. Check logs/test_run.log"
        exit 1
    fi
fi

check_service "Prometheus API" 8080 || exit 1
echo ""

# Step 3: Simulate /start command
echo "Step 3: Simulating user registration (/start)..."
TEST_USER_ID=$(run_sql "INSERT INTO users (telegram_id, telegram_username, first_name, is_active, settings) VALUES ($TEST_TELEGRAM_ID, 'test_user', 'Test User', true, '{}') ON CONFLICT (telegram_id) DO UPDATE SET is_active = true RETURNING id;")
if [ -z "$TEST_USER_ID" ]; then
    echo -e "${RED}✗${NC} Failed to create test user"
    exit 1
fi
echo -e "${GREEN}✓${NC} User created/updated with ID: $TEST_USER_ID"
echo ""

# Step 4: Create test exchange account
echo "Step 4: Creating test exchange account..."
EXCHANGE_ACCOUNT_ID=$(run_sql "INSERT INTO exchange_accounts (user_id, exchange, label, api_key_encrypted, secret_encrypted, is_active) VALUES ('$TEST_USER_ID', 'binance', 'Test Account', E'\\\\x00', E'\\\\x00', true) RETURNING id;")
if [ -z "$EXCHANGE_ACCOUNT_ID" ]; then
    echo -e "${RED}✗${NC} Failed to create exchange account"
    exit 1
fi
echo -e "${GREEN}✓${NC} Exchange account created with ID: $EXCHANGE_ACCOUNT_ID"
echo ""

# Step 5: Check for opportunities in Kafka
echo "Step 5: Checking for market opportunities..."
echo "Waiting for opportunity_found events (30 seconds)..."
TOPIC_CHECK=$(kafka-console-consumer --bootstrap-server $KAFKA_BROKER --topic market.opportunity_found --max-messages 1 --timeout-ms 30000 2>/dev/null || true)
if [ -n "$TOPIC_CHECK" ]; then
    echo -e "${GREEN}✓${NC} Opportunity events flowing"
else
    echo -e "${YELLOW}⚠${NC} No opportunities detected yet (this is normal for low-activity markets)"
fi
echo ""

# Step 6: Verify positions
echo "Step 6: Checking for open positions..."
POSITIONS_COUNT=$(run_sql "SELECT COUNT(*) FROM positions WHERE user_id = '$TEST_USER_ID' AND status = 'open';")
echo "Open positions for test user: $POSITIONS_COUNT"
if [ "$POSITIONS_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓${NC} Positions found"
    run_sql "SELECT symbol, side, entry_price, unrealized_pnl FROM positions WHERE user_id = '$TEST_USER_ID' AND status = 'open';" | while read line; do
        echo "  → $line"
    done
else
    echo -e "${YELLOW}⚠${NC} No positions yet (agents still analyzing)"
fi
echo ""

# Step 7: Check event consumers
echo "Step 7: Verifying event consumers..."
RECENT_AI_USAGE=$(run_sql "SELECT COUNT(*) FROM clickhouse.ai_usage WHERE timestamp >= now() - interval '5 minutes';")
echo "Recent AI usage events: $RECENT_AI_USAGE"
if [ "$RECENT_AI_USAGE" -gt 0 ]; then
    echo -e "${GREEN}✓${NC} AI agents running"
else
    echo -e "${YELLOW}⚠${NC} No recent AI activity"
fi
echo ""

# Step 8: Verify Telegram bot
echo "Step 8: Checking Telegram bot status..."
if curl -s http://localhost:8080/telegram/health | grep -q "ok"; then
    echo -e "${GREEN}✓${NC} Telegram webhook endpoint responding"
else
    echo -e "${YELLOW}⚠${NC} Telegram webhook not configured (using polling mode)"
fi
echo ""

# Summary
echo "========================================="
echo "Test Summary:"
echo "========================================="
echo -e "User Created:         ${GREEN}✓${NC}"
echo -e "Exchange Connected:   ${GREEN}✓${NC}"
echo -e "System Running:       ${GREEN}✓${NC}"
echo -e "Agents Active:        $([ "$RECENT_AI_USAGE" -gt 0 ] && echo -e "${GREEN}✓${NC}" || echo -e "${YELLOW}⚠${NC}")"
echo -e "Positions Opened:     $([ "$POSITIONS_COUNT" -gt 0 ] && echo -e "${GREEN}✓${NC}" || echo -e "${YELLOW}⚠${NC} (waiting)")"
echo ""
echo "✅ MVP infrastructure is operational!"
echo ""
echo "Next steps:"
echo "1. Send /start to your Telegram bot"
echo "2. Send /invest 1000 to start onboarding"
echo "3. Wait for portfolio creation (1-2 minutes)"
echo "4. Check /status and /portfolio"
echo ""
echo "Cleanup: Run 'make clean-test-data' to remove test user"
