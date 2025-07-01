.PHONY: all build test lint fmt clean help ci pre-commit pr-ready security test-coverage e2e-test setup-e2e

# Default target
all: build

# Build the binary
build:
	@echo "Building katago-mcp..."
	@./build.sh

# Run tests
test:
	@echo "Running tests..."
	@go test -race ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -race -coverprofile=coverage.txt -covermode=atomic ./...
	@go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Setup e2e test environment
setup-e2e:
	@echo "Setting up e2e test environment..."
	@chmod +x scripts/setup-katago-test.sh
	@./scripts/setup-katago-test.sh

# Run e2e tests with KataGo
e2e-test: setup-e2e
	@echo "Running e2e tests..."
	@if [ -z "$$KATAGO_TEST_MODEL" ] || [ -z "$$KATAGO_TEST_CONFIG" ]; then \
		echo "Setting up test environment..."; \
		eval "$$(./scripts/setup-katago-test.sh | tail -2)"; \
	fi; \
	go test -tags=e2e ./e2e/... -v

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
	else \
		echo "golangci-lint not installed. Install from https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Code formatting complete"

# Run security scan
security:
	@echo "Running security scan..."
	@if command -v trivy >/dev/null 2>&1; then \
		trivy fs .; \
	else \
		echo "Trivy not installed. Install from https://github.com/aquasecurity/trivy"; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f katago-mcp
	@rm -f coverage.txt coverage.html
	@echo "Clean complete"

# Run all CI checks locally
ci: fmt lint test build
	@echo "All CI checks passed!"

# Pre-commit checks
pre-commit: fmt lint test
	@echo "Pre-commit checks passed!"

# Ensure code is ready for PR
pr-ready: ci
	@echo "Checking git status..."
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Error: Working directory not clean. Commit or stash changes."; \
		git status --short; \
		exit 1; \
	fi
	@echo "Code is ready for PR!"

# Show available commands
help:
	@echo "Available targets:"
	@echo "  all          - Build the binary (default)"
	@echo "  build        - Build the katago-mcp binary"
	@echo "  test         - Run tests with race detection"
	@echo "  test-coverage- Run tests and generate coverage report"
	@echo "  e2e-test     - Run end-to-end tests with real KataGo"
	@echo "  setup-e2e    - Setup e2e test environment"
	@echo "  lint         - Run golangci-lint"
	@echo "  fmt          - Format code with go fmt"
	@echo "  security     - Run security scan with Trivy"
	@echo "  clean        - Remove build artifacts"
	@echo "  ci           - Run all CI checks locally"
	@echo "  pre-commit   - Run pre-commit checks"
	@echo "  pr-ready     - Ensure code is ready for PR"
	@echo "  help         - Show this help message"