.PHONY: test build clean coverage

# Test the module
test:
	go test -v -race ./...

# Run tests with coverage
coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Build the module
build:
	go build ./...

# Clean build artifacts
clean:
	rm -f coverage.out coverage.html

# Run go mod tidy
tidy:
	go mod tidy

# Run all checks
check: tidy build test

# Install dependencies
deps:
	go mod download
