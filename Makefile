.PHONY: build test clean run lint fmt help

BINARY_NAME=fastlane
GO=go
GOFLAGS=-v
OUT_DIR=./bin

help:
	@echo "Fastlane - Network Benchmarking CLI"
	@echo ""
	@echo "Available targets:"
	@echo "  build       - Build fastlane binary"
	@echo "  test        - Run all tests"
	@echo "  bench       - Run benchmarks"
	@echo "  lint        - Run linters (if available)"
	@echo "  fmt         - Format code with gofmt"
	@echo "  run         - Build and run fastlane"
	@echo "  clean       - Remove build artifacts"
	@echo "  help        - Show this message"

build:
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) -o $(OUT_DIR)/$(BINARY_NAME) .

test:
	@echo "Running tests..."
	$(GO) test $(GOFLAGS) -cover ./...

bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

clean:
	@echo "Cleaning..."
	$(GO) clean
	@rm -f $(OUT_DIR)/$(BINARY_NAME)
	@rm -rf $(OUT_DIR)

fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

lint:
	@echo "Linting code..."
	@which golangci-lint > /dev/null || echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
	@golangci-lint run ./... 2>/dev/null || true

run: build
	@echo "Running $(BINARY_NAME)..."
	./$(OUT_DIR)/$(BINARY_NAME)

# Development helpers
dev-install:
	@echo "Installing development dependencies..."
	$(GO) mod download
	$(GO) mod tidy

cross-build: 
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 $(GO) build -o $(OUT_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=arm64 $(GO) build -o $(OUT_DIR)/$(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 $(GO) build -o $(OUT_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@echo "Binaries built in $(OUT_DIR)/"

version:
	@./$(OUT_DIR)/$(BINARY_NAME) version

.DEFAULT_GOAL := help
