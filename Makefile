.PHONY: help docker-up docker-down db-create db-drop db-reset migrate-up migrate-down migrate-status migrate-force migrate-create test-db-create test-db-drop test-db-reset test-migrate-up gen-encryption-key lint fmt test test-short test-integration build run clean deps proto-gen

# Load environment variables from .env file (if exists)
-include .env
export

# Load test environment variables when running test commands
TEST_ENV_FILE := .env.test
ifneq (,$(wildcard $(TEST_ENV_FILE)))
    include $(TEST_ENV_FILE)
    export
endif

# Default target
help:
	@echo "Available commands:"
	@echo ""
	@echo "🐳 Docker:"
	@echo "  make docker-up          - Start all Docker services"
	@echo "  make docker-down        - Stop all Docker services"
	@echo ""
	@echo "🗄️  Database:"
	@echo "  make db-create          - Create databases (PostgreSQL + ClickHouse)"
	@echo "  make db-drop            - Drop databases (with confirmation)"
	@echo "  make db-reset           - Drop, create, and migrate databases"
	@echo "  make migrate-up         - Apply all migrations"
	@echo "  make migrate-down       - Rollback 1 migration"
	@echo "  make migrate-status     - Show migration status"
	@echo "  make migrate-force      - Force migration version (fix dirty state)"
	@echo "  make migrate-create     - Create new migration file"
	@echo ""
	@echo "🧪 Test Database:"
	@echo "  make test-db-create     - Create test database"
	@echo "  make test-db-drop       - Drop test database"
	@echo "  make test-db-reset      - Reset test database (drop + create + migrate)"
	@echo "  make test-migrate-up    - Apply migrations to test database"
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

# Docker commands
docker-up:
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 5

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# Database initialization
db-create:
	@echo "Creating PostgreSQL database..."
	@PGPASSWORD=$(POSTGRES_PASSWORD) psql -h $(POSTGRES_HOST) -p $(POSTGRES_PORT) -U $(POSTGRES_USER) -d postgres -c "CREATE DATABASE $(POSTGRES_DB);" 2>/dev/null || echo "PostgreSQL database already exists"
	@echo "✓ PostgreSQL database ready"
	@echo "Creating ClickHouse database..."
	@docker exec flowly-clickhouse clickhouse-client --query "CREATE DATABASE IF NOT EXISTS $(CLICKHOUSE_DB);" 2>/dev/null || echo "ClickHouse database already exists"
	@echo "✓ ClickHouse database ready"

db-drop:
	@echo "⚠️  WARNING: This will DROP all databases!"
	@read -p "Are you sure? Type 'yes' to confirm: " confirm; \
	if [ "$$confirm" = "yes" ]; then \
		echo "Dropping PostgreSQL database..."; \
		PGPASSWORD=$(POSTGRES_PASSWORD) psql -h $(POSTGRES_HOST) -p $(POSTGRES_PORT) -U $(POSTGRES_USER) -d postgres -c "DROP DATABASE IF EXISTS $(POSTGRES_DB);"; \
		echo "Dropping ClickHouse database..."; \
		docker exec flowly-clickhouse clickhouse-client --query "DROP DATABASE IF EXISTS $(CLICKHOUSE_DB);"; \
		echo "✓ Databases dropped"; \
	else \
		echo "Cancelled."; \
	fi

db-reset: db-drop db-create migrate-up
	@echo "✓ Databases reset complete"

# Test database commands (use .env.test)
test-db-create:
	@echo "Creating test PostgreSQL database..."
	@if [ -f .env.test ]; then \
		set -a; . ./.env.test; set +a; \
		PGPASSWORD=$$POSTGRES_PASSWORD psql -h $$POSTGRES_HOST -p $$POSTGRES_PORT -U $$POSTGRES_USER -d postgres -c "CREATE DATABASE $$POSTGRES_DB;" 2>/dev/null || echo "Test database already exists"; \
		echo "✓ Test PostgreSQL database ready: $$POSTGRES_DB"; \
	else \
		echo "❌ .env.test file not found"; \
		exit 1; \
	fi

test-db-drop:
	@echo "⚠️  WARNING: This will DROP the test database!"
	@read -p "Are you sure? Type 'yes' to confirm: " confirm; \
	if [ "$$confirm" = "yes" ]; then \
		if [ -f .env.test ]; then \
			set -a; . ./.env.test; set +a; \
			echo "Dropping test PostgreSQL database: $$POSTGRES_DB..."; \
			PGPASSWORD=$$POSTGRES_PASSWORD psql -h $$POSTGRES_HOST -p $$POSTGRES_PORT -U $$POSTGRES_USER -d postgres -c "DROP DATABASE IF EXISTS $$POSTGRES_DB;"; \
			echo "✓ Test database dropped"; \
		else \
			echo "❌ .env.test file not found"; \
			exit 1; \
		fi; \
	else \
		echo "Cancelled."; \
	fi

test-db-reset: test-db-drop test-db-create test-migrate-up
	@echo "✓ Test database reset complete"

# Test database migrations
test-migrate-up:
	@echo "Running PostgreSQL migrations on test database..."
	@if [ -f .env.test ]; then \
		set -a; . ./.env.test; set +a; \
		migrate -database "postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@$$POSTGRES_HOST:$$POSTGRES_PORT/$$POSTGRES_DB?sslmode=$$POSTGRES_SSL_MODE" -path migrations/postgres up; \
		echo "✓ Test database migrations completed"; \
	else \
		echo "❌ .env.test file not found"; \
		exit 1; \
	fi

# Database migrations
migrate-up:
	@echo "Running PostgreSQL migrations..."
	@migrate -database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=$(POSTGRES_SSL_MODE)" -path migrations/postgres up
	@echo "✓ PostgreSQL migrations completed"
	@echo "Running ClickHouse migrations..."
	@migrate -database "clickhouse://$(CLICKHOUSE_HOST):$(CLICKHOUSE_PORT)?database=$(CLICKHOUSE_DB)&username=$(CLICKHOUSE_USER)&password=$(CLICKHOUSE_PASSWORD)&x-multi-statement=true" -path migrations/clickhouse up
	@echo "✓ ClickHouse migrations completed"

migrate-down:
	@echo "Rolling back PostgreSQL migrations (1 step)..."
	@migrate -database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=$(POSTGRES_SSL_MODE)" -path migrations/postgres down 1
	@echo "✓ PostgreSQL rollback completed"

migrate-status:
	@echo "PostgreSQL migration status:"
	@migrate -database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=$(POSTGRES_SSL_MODE)" -path migrations/postgres version
	@echo "\nClickHouse migration status:"
	@migrate -database "clickhouse://$(CLICKHOUSE_HOST):$(CLICKHOUSE_PORT)?database=$(CLICKHOUSE_DB)&username=$(CLICKHOUSE_USER)&password=$(CLICKHOUSE_PASSWORD)&x-multi-statement=true" -path migrations/clickhouse version

migrate-force:
	@read -p "Enter version to force: " version; \
	migrate -database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=$(POSTGRES_SSL_MODE)" -path migrations/postgres force $$version

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
setup: deps docker-up db-create migrate-up
	@echo "✓ Development environment ready!"
	@echo "Run 'make run' to start the application"

# Production build
build-prod:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o bin/prometheus ./cmd/main.go

# Docker build
docker-build:
	docker build -t prometheus:latest .

docker-run:
	docker run --env-file .env -p 8080:8080 prometheus:latest
