# Event Ticketing Platform Makefile

.PHONY: build run dev test clean help install setup css css-watch migrate

# Install dependencies
install:
	go mod download
	go mod tidy
	bun install

# Setup development environment
setup: install
	@echo "Setting up development environment..."
	@if not exist "tmp" mkdir tmp
	@if not exist "uploads" mkdir uploads
	@echo "Development environment ready!"

# Build the application
build:
	go build -o ./tmp/main.exe ./cmd/server

# Run the application
run: build
	./tmp/main.exe

# Start development server with hot reloading and CSS watching
dev:
	@echo "Starting development mode..."
	@echo "Run 'make css-watch' in another terminal for CSS watching"
	air

# Start development server with Air only
air:
	air

# Build CSS for production
css:
	bun run build-css-prod

# Watch CSS changes
css-watch:
	bun run build-css

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	@if exist "tmp" rmdir /s /q tmp
	@if exist "*.exe" del /q *.exe
	go clean
	@echo "Cleaned build artifacts"

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Run database migrations
migrate:
	go run ./cmd/migrate

# Show help
help:
	@echo "Available commands:"
	@echo "  install    - Install Go and Node.js dependencies"
	@echo "  setup      - Setup development environment"
	@echo "  build      - Build the application"
	@echo "  run        - Build and run the application"
	@echo "  dev        - Start development mode (Air + CSS watching)"
	@echo "  air        - Start Air only (hot reloading)"
	@echo "  css        - Build CSS for production"
	@echo "  css-watch  - Watch CSS changes"
	@echo "  test       - Run tests"
	@echo "  clean      - Clean build artifacts"
	@echo "  fmt        - Format code"
	@echo "  lint       - Run linter"
	@echo "  migrate    - Run database migrations"
	@echo "  help       - Show this help message"