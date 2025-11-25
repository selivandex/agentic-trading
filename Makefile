.PHONY: help build run test clean docker-up docker-down migrate-up migrate-down deps lint

# Default target
help:
	@echo "Available targets:"
	@echo "  make build        - Build the application"
	@echo "  make run          - Run the application"
	@echo "  make test         - Run tests"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make docker-up    - Start Docker services"
	@echo "  make docker-down  - Stop Docker services"
	@echo "  make migrate-up   - Run database migrations"
	@echo "  make migrate-down - Rollback database migrations"
	@echo "  make deps         - Download dependencies"
	@echo "  make lint         - Run linter"

# Build the application
build:
	@echo "Building prometheus..."
	@go build -o bin/prometheus cmd/main.go

# Run the application
run:
	@echo "Running prometheus..."
	@go run cmd/main.go

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# Start Docker services
docker-up:
	@echo "Starting Docker services..."
	@docker-compose up -d

# Stop Docker services
docker-down:
	@echo "Stopping Docker services..."
	@docker-compose down

# Run database migrations (TODO: implement migration tool)
migrate-up:
	@echo "Running migrations..."
	@echo "TODO: Implement migration runner"

# Rollback database migrations
migrate-down:
	@echo "Rolling back migrations..."
	@echo "TODO: Implement migration rollback"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Generate mocks (for testing)
generate:
	@echo "Generating mocks..."
	@go generate ./...

# Install development tools
install-tools:
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Create encryption key
gen-encryption-key:
	@echo "Generating encryption key..."
	@openssl rand -base64 32


# Default target
help:
	@echo "Available targets:"
	@echo "  make build        - Build the application"
	@echo "  make run          - Run the application"
	@echo "  make test         - Run tests"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make docker-up    - Start Docker services"
	@echo "  make docker-down  - Stop Docker services"
	@echo "  make migrate-up   - Run database migrations"
	@echo "  make migrate-down - Rollback database migrations"
	@echo "  make deps         - Download dependencies"
	@echo "  make lint         - Run linter"

# Build the application
build:
	@echo "Building prometheus..."
	@go build -o bin/prometheus cmd/main.go

# Run the application
run:
	@echo "Running prometheus..."
	@go run cmd/main.go

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# Start Docker services
docker-up:
	@echo "Starting Docker services..."
	@docker-compose up -d

# Stop Docker services
docker-down:
	@echo "Stopping Docker services..."
	@docker-compose down

# Run database migrations (TODO: implement migration tool)
migrate-up:
	@echo "Running migrations..."
	@echo "TODO: Implement migration runner"

# Rollback database migrations
migrate-down:
	@echo "Rolling back migrations..."
	@echo "TODO: Implement migration rollback"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Generate mocks (for testing)
generate:
	@echo "Generating mocks..."
	@go generate ./...

# Install development tools
install-tools:
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Create encryption key
gen-encryption-key:
	@echo "Generating encryption key..."
	@openssl rand -base64 32

