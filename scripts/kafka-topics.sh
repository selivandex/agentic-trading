#!/usr/bin/env bash

# Kafka topic management script for domain-level architecture
# See internal/events/helpers.go for topic definitions

set -e

# Configuration
KAFKA_CONTAINER="${KAFKA_CONTAINER:-flowly-kafka}"
KAFKA_BOOTSTRAP="${KAFKA_BOOTSTRAP:-localhost:9092}"
KAFKA_PARTITIONS="${KAFKA_PARTITIONS:-3}"
KAFKA_REPLICATION="${KAFKA_REPLICATION:-1}"

# Domain-level topics (keep in sync with internal/events/helpers.go)
TOPICS=(
    "ai.events"
    "market.events"
    "trading.events"
    "risk.events"
    "position.events"
    "agent.events"
    "system.events"
    "notifications"
    "websocket.events"
    "analytics"
    "user-data-all-topics"
    "user-data-order-updates"
    "user-data-position-updates"
    "user-data-balance-updates"
    "user-data-margin-calls"
    "user-data-account-config"
)

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Functions
create_topics() {
    echo "Creating Kafka topics (domain-level architecture)..."
    local created=0
    local existed=0
    
    for topic in "${TOPICS[@]}"; do
        if docker exec -t "$KAFKA_CONTAINER" kafka-topics --create \
            --topic "$topic" \
            --bootstrap-server "$KAFKA_BOOTSTRAP" \
            --partitions "$KAFKA_PARTITIONS" \
            --replication-factor "$KAFKA_REPLICATION" \
            2>/dev/null; then
            echo -e "  ${GREEN}✓${NC} $topic (created)"
            ((created++))
        else
            echo -e "  ${YELLOW}✓${NC} $topic (already exists)"
            ((existed++))
        fi
    done
    
    echo ""
    echo -e "${GREEN}✓${NC} Kafka topics ready: $created created, $existed existed (${#TOPICS[@]} total)"
}

list_topics() {
    echo "Listing Kafka topics..."
    docker exec -t "$KAFKA_CONTAINER" kafka-topics --list --bootstrap-server "$KAFKA_BOOTSTRAP"
}

delete_topics() {
    echo -e "${RED}⚠️  WARNING: This will DELETE all application Kafka topics!${NC}"
    read -p "Are you sure? Type 'yes' to confirm: " confirm
    
    if [ "$confirm" = "yes" ]; then
        local deleted=0
        local notfound=0
        
        for topic in "${TOPICS[@]}"; do
            if docker exec -t "$KAFKA_CONTAINER" kafka-topics --delete \
                --topic "$topic" \
                --bootstrap-server "$KAFKA_BOOTSTRAP" \
                2>/dev/null; then
                echo -e "  ${RED}✗${NC} $topic (deleted)"
                ((deleted++))
            else
                echo -e "  ${YELLOW}-${NC} $topic (not found)"
                ((notfound++))
            fi
        done
        
        echo ""
        echo -e "${GREEN}✓${NC} Topic deletion complete: $deleted deleted, $notfound not found"
    else
        echo "Cancelled."
    fi
}

describe_topic() {
    local topic="$1"
    if [ -z "$topic" ]; then
        echo "Usage: $0 describe <topic-name>"
        exit 1
    fi
    
    echo "Describing topic: $topic"
    docker exec -t "$KAFKA_CONTAINER" kafka-topics --describe \
        --topic "$topic" \
        --bootstrap-server "$KAFKA_BOOTSTRAP"
}

show_help() {
    cat << EOF
Kafka Topic Management Script (Domain-Level Architecture)

Usage: $0 <command> [options]

Commands:
    create      Create all domain-level Kafka topics
    list        List all Kafka topics
    delete      Delete all application topics (with confirmation)
    describe    Describe a specific topic
    help        Show this help message

Domain-Level Topics (${#TOPICS[@]} total):
EOF
    for topic in "${TOPICS[@]}"; do
        echo "    - $topic"
    done
    
    cat << EOF

Environment Variables:
    KAFKA_CONTAINER     Docker container name (default: flowly-kafka)
    KAFKA_BOOTSTRAP     Kafka bootstrap server (default: localhost:9092)
    KAFKA_PARTITIONS    Number of partitions (default: 3)
    KAFKA_REPLICATION   Replication factor (default: 1)

Examples:
    $0 create
    $0 list
    $0 describe websocket.events
    KAFKA_PARTITIONS=5 $0 create

See internal/events/helpers.go for topic definitions.
EOF
}

# Main
case "${1:-}" in
    create)
        create_topics
        ;;
    list)
        list_topics
        ;;
    delete)
        delete_topics
        ;;
    describe)
        describe_topic "$2"
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        echo "Error: Unknown command '${1:-}'"
        echo ""
        show_help
        exit 1
        ;;
esac

