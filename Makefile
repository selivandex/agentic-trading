.PHONY: help docker-up docker-down migrate-up migrate-down migrate-status migrate-force migrate-create gen-encryption-key lint fmt test build run clean deps proto-gen

# Load environment variables from .env file (if exists)
-include .env
export

# Default target
help:
	@echo "Available commands:"
	@echo "  make docker-up          - Start all Docker services"
	@echo "  make docker-down        - Stop all Docker services"
	@echo "  make migrate-up         - Apply all migrations"
	@echo "  make migrate-down       - Rollback 1 migration"
	@echo "  make migrate-status     - Show migration status"
	@echo "  make migrate-force      - Force migration version (fix dirty state)"
	@echo "  make migrate-create     - Create new migration file"
	@echo "  make gen-encryption-key - Generate encryption key"
	@echo "  make lint               - Run linter"
	@echo "  make fmt                - Format code"
	@echo "  make test               - Run tests"
	@echo "  make build              - Build application"
	@echo "  make run                - Run application"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make deps               - Install dependencies"
	@echo "  make proto-gen          - Generate protobuf code"

# Docker commands
docker-up:
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 5

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# Database migrations
migrate-up:
	@echo "Running PostgreSQL migrations..."
	@migrate -database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=$(POSTGRES_SSL_MODE)" -path migrations/postgres up
	@echo "✓ PostgreSQL migrations completed"
	@echo "Running ClickHouse migrations..."
	@migrate -database "clickhouse://$(CLICKHOUSE_HOST):$(CLICKHOUSE_PORT)?database=$(CLICKHOUSE_DB)&username=$(CLICKHOUSE_USER)&password=$(CLICKHOUSE_PASSWORD)" -path migrations/clickhouse up
	@echo "✓ ClickHouse migrations completed"

migrate-down:
	@echo "Rolling back PostgreSQL migrations (1 step)..."
	@migrate -database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=$(POSTGRES_SSL_MODE)" -path migrations/postgres down 1
	@echo "✓ PostgreSQL rollback completed"

migrate-status:
	@echo "PostgreSQL migration status:"
	@migrate -database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=$(POSTGRES_SSL_MODE)" -path migrations/postgres version
	@echo "\nClickHouse migration status:"
	@migrate -database "clickhouse://$(CLICKHOUSE_HOST):$(CLICKHOUSE_PORT)?database=$(CLICKHOUSE_DB)&username=$(CLICKHOUSE_USER)&password=$(CLICKHOUSE_PASSWORD)" -path migrations/clickhouse version

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
	go test -v -race -coverprofile=coverage.out ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-integration:
	go test -v -tags=integration ./...

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
setup: deps docker-up migrate-up
	@echo "Development environment ready!"
	@echo "Run 'make run' to start the application"

# Production build
build-prod:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o bin/prometheus ./cmd/main.go

# Docker build
docker-build:
	docker build -t prometheus:latest .

docker-run:
	docker run --env-file .env -p 8080:8080 prometheus:latest
