.PHONY: help docker-up docker-down docker-logs clean deps proto-gen
.PHONY: db-create db-drop db-reset migrate-up migrate-down migrate-status migrate-force migrate-create
.PHONY: db-pg-create db-pg-drop migrate-pg-up migrate-pg-down migrate-pg-status
.PHONY: db-ch-create db-ch-drop migrate-ch-up migrate-ch-down migrate-ch-status
.PHONY: kafka-topics-create kafka-topics-list kafka-topics-delete
.PHONY: test-db-create test-db-drop test-db-reset test-migrate-up
.PHONY: gen-encryption-key lint fmt test test-short test-integration test-coverage test-repo
.PHONY: build build-linux build-prod run run-dev setup
.PHONY: docker-build docker-run
.PHONY: start stop restart status

# Load environment variables from .env file (if exists)
-include .env
export

# Database configuration (can be overridden for test environment)
DB_PG_HOST ?= $(POSTGRES_HOST)
DB_PG_PORT ?= $(POSTGRES_PORT)
DB_PG_USER ?= $(POSTGRES_USER)
DB_PG_PASSWORD ?= $(POSTGRES_PASSWORD)
DB_PG_NAME ?= $(POSTGRES_DB)
DB_PG_SSL_MODE ?= $(POSTGRES_SSL_MODE)

DB_CH_HOST ?= $(CLICKHOUSE_HOST)
DB_CH_PORT ?= $(CLICKHOUSE_PORT)
DB_CH_USER ?= $(CLICKHOUSE_USER)
DB_CH_PASSWORD ?= $(CLICKHOUSE_PASSWORD)
DB_CH_NAME ?= $(CLICKHOUSE_DB)

# Default target
help:
	@echo "Available commands:"
	@echo ""
	@echo "🐳 Docker:"
	@echo "  make docker-up          - Start all Docker services"
	@echo "  make docker-down        - Stop all Docker services"
	@echo ""
	@echo "🗄️  Database (Combined):"
	@echo "  make db-create          - Create all databases (PostgreSQL + ClickHouse)"
	@echo "  make db-drop            - Drop all databases (with confirmation)"
	@echo "  make db-reset           - Drop, create, and migrate all databases"
	@echo "  make migrate-up         - Apply all migrations (PostgreSQL + ClickHouse)"
	@echo "  make migrate-down       - Rollback 1 migration (PostgreSQL + ClickHouse)"
	@echo "  make migrate-status     - Show migration status for all databases"
	@echo "  make migrate-force      - Force migration version (fix dirty state)"
	@echo "  make migrate-create     - Create new migration file"
	@echo ""
	@echo "🗄️  PostgreSQL Only:"
	@echo "  make db-pg-create       - Create PostgreSQL database"
	@echo "  make db-pg-drop         - Drop PostgreSQL database"
	@echo "  make migrate-pg-up      - Apply PostgreSQL migrations"
	@echo "  make migrate-pg-down    - Rollback PostgreSQL migrations"
	@echo "  make migrate-pg-status  - Show PostgreSQL migration status"
	@echo ""
	@echo "🗄️  ClickHouse Only:"
	@echo "  make db-ch-create       - Create ClickHouse database"
	@echo "  make db-ch-drop         - Drop ClickHouse database"
	@echo "  make migrate-ch-up      - Apply ClickHouse migrations"
	@echo "  make migrate-ch-down    - Rollback ClickHouse migrations"
	@echo "  make migrate-ch-status  - Show ClickHouse migration status"
	@echo ""
	@echo "📨 Kafka:"
	@echo "  make kafka-topics-create - Create all required Kafka topics"
	@echo "  make kafka-topics-list   - List all Kafka topics"
	@echo "  make kafka-topics-delete - Delete all application topics"
	@echo ""
	@echo "🧪 Test Databases:"
	@echo "  make test-db-create     - Create test databases (PostgreSQL + ClickHouse, uses .env.test)"
	@echo "  make test-db-drop       - Drop test databases"
	@echo "  make test-db-reset      - Reset test databases (drop + create + migrate)"
	@echo "  make test-migrate-up    - Apply migrations to test databases"
	@echo ""
	@echo "🔧 Development:"
	@echo "  make gen-encryption-key - Generate encryption key"
	@echo "  make lint               - Run linter"
	@echo "  make fmt                - Format code"
	@echo "  make test               - Run all tests with coverage"
	@echo "  make test-short         - Run tests without integration tests"
	@echo "  make test-integration   - Run only integration tests"
	@echo "  make build              - Build application"
	@echo "  make run                - Run application"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make deps               - Install dependencies"
	@echo "  make proto-gen          - Generate protobuf code"
	@echo "  make setup              - Full setup (deps + docker + db + migrations)"
	@echo ""
	@echo "🚀 Server Control:"
	@echo "  make start              - Start server in background (saves PID to /tmp/prometheus.pid)"
	@echo "  make stop               - Stop server using saved PID"
	@echo "  make restart            - Restart server (stop + start)"
	@echo "  make status             - Check server status"

# Docker commands
docker-up:
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 5

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# ============================================================================
# PostgreSQL Commands
# ============================================================================

db-pg-create:
	@echo "Creating PostgreSQL database: $(DB_PG_NAME)..."
	@PGPASSWORD=$(DB_PG_PASSWORD) psql -h $(DB_PG_HOST) -p $(DB_PG_PORT) -U $(DB_PG_USER) -d postgres -c "CREATE DATABASE $(DB_PG_NAME);" 2>/dev/null || echo "PostgreSQL database already exists"
	@echo "✓ PostgreSQL database ready: $(DB_PG_NAME)"

db-pg-drop:
	@echo "Dropping PostgreSQL database: $(DB_PG_NAME)..."
	@PGPASSWORD=$(DB_PG_PASSWORD) psql -h $(DB_PG_HOST) -p $(DB_PG_PORT) -U $(DB_PG_USER) -d postgres -c "DROP DATABASE IF EXISTS $(DB_PG_NAME);"
	@echo "✓ PostgreSQL database dropped"

migrate-pg-up:
	@echo "Running PostgreSQL migrations on $(DB_PG_NAME)..."
	@migrate -database "postgres://$(DB_PG_USER):$(DB_PG_PASSWORD)@$(DB_PG_HOST):$(DB_PG_PORT)/$(DB_PG_NAME)?sslmode=$(DB_PG_SSL_MODE)" -path migrations/postgres up
	@echo "✓ PostgreSQL migrations completed"

migrate-pg-down:
	@echo "Rolling back PostgreSQL migrations (1 step) on $(DB_PG_NAME)..."
	@migrate -database "postgres://$(DB_PG_USER):$(DB_PG_PASSWORD)@$(DB_PG_HOST):$(DB_PG_PORT)/$(DB_PG_NAME)?sslmode=$(DB_PG_SSL_MODE)" -path migrations/postgres down 1
	@echo "✓ PostgreSQL rollback completed"

migrate-pg-status:
	@echo "PostgreSQL migration status for $(DB_PG_NAME):"
	@migrate -database "postgres://$(DB_PG_USER):$(DB_PG_PASSWORD)@$(DB_PG_HOST):$(DB_PG_PORT)/$(DB_PG_NAME)?sslmode=$(DB_PG_SSL_MODE)" -path migrations/postgres version

# ============================================================================
# ClickHouse Commands
# ============================================================================

db-ch-create:
	@echo "Creating ClickHouse database: $(DB_CH_NAME)..."
	@docker exec flowly-clickhouse clickhouse-client --query "CREATE DATABASE IF NOT EXISTS $(DB_CH_NAME);" 2>/dev/null || echo "ClickHouse database already exists"
	@echo "✓ ClickHouse database ready: $(DB_CH_NAME)"

db-ch-drop:
	@echo "Dropping ClickHouse database: $(DB_CH_NAME)..."
	@docker exec flowly-clickhouse clickhouse-client --query "DROP DATABASE IF EXISTS $(DB_CH_NAME);"
	@echo "✓ ClickHouse database dropped"

migrate-ch-up:
	@echo "Running ClickHouse migrations on $(DB_CH_NAME)..."
	@migrate -database "clickhouse://$(DB_CH_HOST):$(DB_CH_PORT)?database=$(DB_CH_NAME)&username=$(DB_CH_USER)&password=$(DB_CH_PASSWORD)&x-multi-statement=true" -path migrations/clickhouse up
	@echo "✓ ClickHouse migrations completed"

migrate-ch-down:
	@echo "Rolling back ClickHouse migrations (1 step) on $(DB_CH_NAME)..."
	@migrate -database "clickhouse://$(DB_CH_HOST):$(DB_CH_PORT)?database=$(DB_CH_NAME)&username=$(DB_CH_USER)&password=$(DB_CH_PASSWORD)&x-multi-statement=true" -path migrations/clickhouse down 1
	@echo "✓ ClickHouse rollback completed"

migrate-ch-status:
	@echo "ClickHouse migration status for $(DB_CH_NAME):"
	@migrate -database "clickhouse://$(DB_CH_HOST):$(DB_CH_PORT)?database=$(DB_CH_NAME)&username=$(DB_CH_USER)&password=$(DB_CH_PASSWORD)&x-multi-statement=true" -path migrations/clickhouse version

# ============================================================================
# Kafka Commands
# ============================================================================

KAFKA_CONTAINER ?= flowly-kafka
KAFKA_PARTITIONS ?= 3
KAFKA_REPLICATION ?= 1

kafka-topics-create:
	@KAFKA_CONTAINER=$(KAFKA_CONTAINER) KAFKA_PARTITIONS=$(KAFKA_PARTITIONS) KAFKA_REPLICATION=$(KAFKA_REPLICATION) \
		./scripts/kafka-topics.sh create

kafka-topics-list:
	@KAFKA_CONTAINER=$(KAFKA_CONTAINER) ./scripts/kafka-topics.sh list

kafka-topics-delete:
	@KAFKA_CONTAINER=$(KAFKA_CONTAINER) ./scripts/kafka-topics.sh delete

# ============================================================================
# Combined Database Commands (PostgreSQL + ClickHouse)
# ============================================================================

db-create: db-pg-create db-ch-create
	@echo "✓ All databases created"

db-drop:
	@echo "⚠️  WARNING: Dropping all databases..."
	@$(MAKE) db-pg-drop
	@$(MAKE) db-ch-drop
	@echo "✓ All databases dropped"

db-reset: db-drop db-create migrate-up
	@echo "✓ Databases reset complete"

# ============================================================================
# Test Database Commands (uses same functions with .env.test config)
# ============================================================================

test-db-create:
	@if [ ! -f .env.test ]; then \
		echo "❌ .env.test file not found"; \
		exit 1; \
	fi
	@set -a; . ./.env.test; set +a; \
	$(MAKE) db-pg-create \
		DB_PG_HOST=$$POSTGRES_HOST \
		DB_PG_PORT=$$POSTGRES_PORT \
		DB_PG_USER=$$POSTGRES_USER \
		DB_PG_PASSWORD=$$POSTGRES_PASSWORD \
		DB_PG_NAME=$$POSTGRES_DB \
		DB_PG_SSL_MODE=$$POSTGRES_SSL_MODE; \
	$(MAKE) db-ch-create \
		DB_CH_HOST=$${CLICKHOUSE_HOST:-$(CLICKHOUSE_HOST)} \
		DB_CH_PORT=$${CLICKHOUSE_PORT:-$(CLICKHOUSE_PORT)} \
		DB_CH_USER=$${CLICKHOUSE_USER:-$(CLICKHOUSE_USER)} \
		DB_CH_PASSWORD=$${CLICKHOUSE_PASSWORD:-$(CLICKHOUSE_PASSWORD)} \
		DB_CH_NAME=$${CLICKHOUSE_DB:-test_$(CLICKHOUSE_DB)}
	@echo "✓ All test databases created"

test-db-drop:
	@if [ ! -f .env.test ]; then \
		echo "❌ .env.test file not found"; \
		exit 1; \
	fi
	@echo "⚠️  WARNING: Dropping all test databases..."
	@set -a; . ./.env.test; set +a; \
	$(MAKE) db-pg-drop \
		DB_PG_HOST=$$POSTGRES_HOST \
		DB_PG_PORT=$$POSTGRES_PORT \
		DB_PG_USER=$$POSTGRES_USER \
		DB_PG_PASSWORD=$$POSTGRES_PASSWORD \
		DB_PG_NAME=$$POSTGRES_DB; \
	$(MAKE) db-ch-drop \
		DB_CH_HOST=$${CLICKHOUSE_HOST:-$(CLICKHOUSE_HOST)} \
		DB_CH_PORT=$${CLICKHOUSE_PORT:-$(CLICKHOUSE_PORT)} \
		DB_CH_USER=$${CLICKHOUSE_USER:-$(CLICKHOUSE_USER)} \
		DB_CH_PASSWORD=$${CLICKHOUSE_PASSWORD:-$(CLICKHOUSE_PASSWORD)} \
		DB_CH_NAME=$${CLICKHOUSE_DB:-test_$(CLICKHOUSE_DB)}
	@echo "✓ All test databases dropped"

test-db-reset:
	@if [ ! -f .env.test ]; then \
		echo "❌ .env.test file not found"; \
		exit 1; \
	fi
	@set -a; . ./.env.test; set +a; \
	$(MAKE) test-db-drop && \
	$(MAKE) test-db-create && \
	$(MAKE) test-migrate-up
	@echo "✓ Test databases reset complete"

test-migrate-up:
	@if [ ! -f .env.test ]; then \
		echo "❌ .env.test file not found"; \
		exit 1; \
	fi
	@set -a; . ./.env.test; set +a; \
	$(MAKE) migrate-pg-up \
		DB_PG_HOST=$$POSTGRES_HOST \
		DB_PG_PORT=$$POSTGRES_PORT \
		DB_PG_USER=$$POSTGRES_USER \
		DB_PG_PASSWORD=$$POSTGRES_PASSWORD \
		DB_PG_NAME=$$POSTGRES_DB \
		DB_PG_SSL_MODE=$$POSTGRES_SSL_MODE; \
	$(MAKE) migrate-ch-up \
		DB_CH_HOST=$${CLICKHOUSE_HOST:-$(CLICKHOUSE_HOST)} \
		DB_CH_PORT=$${CLICKHOUSE_PORT:-$(CLICKHOUSE_PORT)} \
		DB_CH_USER=$${CLICKHOUSE_USER:-$(CLICKHOUSE_USER)} \
		DB_CH_PASSWORD=$${CLICKHOUSE_PASSWORD:-$(CLICKHOUSE_PASSWORD)} \
		DB_CH_NAME=$${CLICKHOUSE_DB:-test_$(CLICKHOUSE_DB)}
	@echo "✓ All test migrations completed"

# ============================================================================
# Migration Commands (Combined)
# ============================================================================

migrate-up: migrate-pg-up migrate-ch-up
	@echo "✓ All migrations completed"

migrate-down: migrate-pg-down migrate-ch-down
	@echo "✓ All rollbacks completed"

migrate-status: migrate-pg-status migrate-ch-status

migrate-force:
	@echo "Select database:"
	@echo "  1) PostgreSQL"
	@echo "  2) ClickHouse"
	@read -p "Enter choice (1 or 2): " db_choice; \
	read -p "Enter version to force: " version; \
	if [ "$$db_choice" = "1" ]; then \
		migrate -database "postgres://$(DB_PG_USER):$(DB_PG_PASSWORD)@$(DB_PG_HOST):$(DB_PG_PORT)/$(DB_PG_NAME)?sslmode=$(DB_PG_SSL_MODE)" -path migrations/postgres force $$version; \
	elif [ "$$db_choice" = "2" ]; then \
		migrate -database "clickhouse://$(DB_CH_HOST):$(DB_CH_PORT)?database=$(DB_CH_NAME)&username=$(DB_CH_USER)&password=$(DB_CH_PASSWORD)&x-multi-statement=true" -path migrations/clickhouse force $$version; \
	else \
		echo "Invalid choice"; \
		exit 1; \
	fi

migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations/postgres -seq $$name
	@echo "✓ Migration files created in migrations/postgres/"

# Security
gen-encryption-key:
	@openssl rand -base64 32

# Code quality
lint:
	golangci-lint run ./...

fmt:
	gofmt -w -s .
	goimports -w .

# Testing
test:
	@echo "Running all tests (unit + integration)..."
	go test -v -race -coverprofile=coverage.out ./...

test-short:
	@echo "Running unit tests only (skipping integration tests)..."
	go test -v -short -race ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-integration:
	@echo "Running integration tests (requires test database)..."
	@echo "Make sure test database is ready: make test-db-reset"
	@if [ -f .env.test ]; then \
		set -a; . ./.env.test; set +a; \
		go test -v -race ./internal/repository/postgres/...; \
	else \
		echo "❌ .env.test file not found"; \
		exit 1; \
	fi

test-repo:
	@echo "Running repository integration tests..."
	@if [ -f .env.test ]; then \
		set -a; . ./.env.test; set +a; \
		go test -v ./internal/repository/postgres -run $(TEST); \
	else \
		echo "❌ .env.test file not found"; \
		exit 1; \
	fi

# Build
build:
	go build -o bin/prometheus ./cmd/main.go

build-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/prometheus-linux ./cmd/main.go

# Run
run:
	go run ./cmd/main.go

run-dev:
	air

# Dependencies
deps:
	go mod download
	go mod tidy
	go mod verify

# Protobuf generation
proto-gen:
	@echo "Generating protobuf code..."
	protoc --go_out=. --go_opt=paths=source_relative \
		internal/events/proto/events.proto
	@echo "✓ Protobuf code generated"

# Clean
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean -cache

# Development setup
setup: deps docker-up db-create migrate-up kafka-topics-create
	@echo "✓ Development environment ready!"
	@echo "  - PostgreSQL + ClickHouse databases created"
	@echo "  - Migrations applied"
	@echo "  - Kafka topics created (10 domain-level topics)"
	@echo ""
	@echo "Run 'make run' to start the application"

# Production build
build-prod:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o bin/prometheus ./cmd/main.go

# Docker build
docker-build:
	docker build -t prometheus:latest .

docker-run:
	docker run --env-file .env -p 8080:8080 prometheus:latest

# ============================================================================
# Server Control (with PID management)
# ============================================================================

PID_FILE ?= /tmp/prometheus.pid

start:
	@if [ -f $(PID_FILE) ]; then \
		if ps -p $$(cat $(PID_FILE)) > /dev/null 2>&1; then \
			echo "⚠️  Server is already running (PID: $$(cat $(PID_FILE)))"; \
			echo "Use 'make stop' to stop it first, or 'make restart' to restart"; \
			exit 1; \
		else \
			echo "🧹 Cleaning up stale PID file..."; \
			rm -f $(PID_FILE); \
		fi; \
	fi
	@echo "🔨 Building server..."
	@$(MAKE) build
	@echo "🚀 Starting server in background..."
	@nohup ./bin/prometheus > /tmp/prometheus.log 2>&1 & echo $$! > $(PID_FILE)
	@sleep 1
	@if ps -p $$(cat $(PID_FILE)) > /dev/null 2>&1; then \
		echo "✅ Server started successfully (PID: $$(cat $(PID_FILE)))"; \
		echo "📋 Logs: tail -f /tmp/prometheus.log"; \
		echo "🛑 Stop: make stop"; \
	else \
		echo "❌ Failed to start server. Check logs: tail /tmp/prometheus.log"; \
		rm -f $(PID_FILE); \
		exit 1; \
	fi

stop:
	@if [ ! -f $(PID_FILE) ]; then \
		echo "⚠️  PID file not found. Server may not be running."; \
		exit 1; \
	fi
	@PID=$$(cat $(PID_FILE)); \
	if ps -p $$PID > /dev/null 2>&1; then \
		echo "🛑 Stopping server (PID: $$PID)..."; \
		kill $$PID; \
		sleep 2; \
		if ps -p $$PID > /dev/null 2>&1; then \
			echo "⚠️  Server didn't stop gracefully, forcing..."; \
			kill -9 $$PID; \
		fi; \
		rm -f $(PID_FILE); \
		echo "✅ Server stopped"; \
	else \
		echo "⚠️  Server is not running (stale PID file)"; \
		rm -f $(PID_FILE); \
	fi

restart: stop start
	@echo "✅ Server restarted"

status:
	@if [ ! -f $(PID_FILE) ]; then \
		echo "❌ Server is not running (no PID file)"; \
		exit 1; \
	fi
	@PID=$$(cat $(PID_FILE)); \
	if ps -p $$PID > /dev/null 2>&1; then \
		echo "✅ Server is running (PID: $$PID)"; \
		ps -p $$PID -o pid,ppid,%cpu,%mem,etime,command; \
	else \
		echo "❌ Server is not running (stale PID file)"; \
		rm -f $(PID_FILE); \
		exit 1; \
	fi
