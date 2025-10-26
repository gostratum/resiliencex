## Consolidated Makefile for resiliencex
.PHONY: test build clean coverage tidy check deps help test-coverage lint install-tools \
	version validate-version update-deps bump-patch bump-minor bump-major \
	release release-dry-run release-patch release-minor release-major

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Test parameters
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

VERSION := $(shell cat .version 2>/dev/null || echo "0.0.0")

# Default target: show help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Development setup
deps: ## Download module dependencies
	go mod download

build: ## Build the project
	go build ./...

# Testing
test: ## Run tests
	@echo "Running tests..."
	@$(GOTEST) -v -race ./...

test-coverage: ## Run tests with coverage and generate HTML report
	@echo "Running tests with coverage..."
	@$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

# Generate HTML coverage report
coverage: test-coverage ## Alias for test-coverage

# Code quality
lint: ## Run linter (golangci-lint)
	@echo "Running linters..."
	@GOLANGCI_BIN=$(go env GOPATH)/bin/golangci-lint; \
	if [ -x "$$GOLANGCI_BIN" ]; then \
		"$$GOLANGCI_BIN" run ./...; \
	elif command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Run: make install-tools"; exit 1; \
	fi

# Tidy dependencies
tidy: ## Tidy dependencies
	@$(GOMOD) tidy

# Clean up
clean: ## Clean up build artifacts
	@echo "Cleaning..."
	@$(GOCLEAN) -cache
	rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)

# Run all checks
check: tidy build test ## Run all checks (tidy, build, test)

# Install development tools
install-tools: ## Install development tools used by the project
	@echo "Installing development tools..."
	@command -v golangci-lint >/dev/null 2>&1 || \
		(echo "Installing golangci-lint..." && \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@echo "Tools installed successfully"

# Version management helpers
version: ## Print current version
	@echo "Current version: v$(VERSION)"

validate-version: ## Validate .version file if scripts exist
	@if [ -x ./scripts/validate-version.sh ]; then ./scripts/validate-version.sh; else echo "No validate-version script present"; fi

update-deps: ## Run update-deps script if present
	@if [ -x ./scripts/update-deps.sh ]; then ./scripts/update-deps.sh; else echo "No update-deps script present"; fi

bump-patch: ## Bump patch version
	@./scripts/bump-version.sh patch

bump-minor: ## Bump minor version
	@./scripts/bump-version.sh minor

bump-major: ## Bump major version
	@./scripts/bump-version.sh major

# Release management (delegated to scripts if present)
release: ## Run release script (default: patch)
	@./scripts/release.sh $(or $(TYPE),patch)

release-dry-run: ## Test release without committing
	@DRY_RUN=true ./scripts/release.sh $(or $(TYPE),patch)

release-patch: ## Create patch release
	@./scripts/release.sh patch

release-minor: ## Create minor release
	@./scripts/release.sh minor

release-major: ## Create major release
	@./scripts/release.sh major


