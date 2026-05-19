.PHONY: build test bench clean run lint fmt help cross-build install dev-install version

BINARY_NAME := fastlane
GO          := go
GOFLAGS     := -v
OUT_DIR     := ./bin
PKG         := github.com/atharvwasthere/Fastlane/cmd

# Build-time metadata. VERSION defaults to the latest tag or "dev"; COMMIT to
# the short SHA; DATE to UTC ISO-8601. Override on the command line:
#   make build VERSION=0.2.0
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -ldflags "-s -w \
  -X '$(PKG).Version=$(VERSION)' \
  -X '$(PKG).Commit=$(COMMIT)' \
  -X '$(PKG).BuildDate=$(DATE)'"

help:
	@echo "Fastlane - Network Benchmarking CLI"
	@echo ""
	@echo "Targets:"
	@echo "  build       Build $(BINARY_NAME) (with ldflags-injected version)"
	@echo "  test        Run all tests with coverage"
	@echo "  bench       Run benchmarks"
	@echo "  lint        Run golangci-lint if installed"
	@echo "  fmt         gofmt all packages"
	@echo "  run         Build and run"
	@echo "  cross-build linux-amd64, darwin-arm64, windows-amd64"
	@echo "  clean       Remove $(OUT_DIR)"
	@echo ""
	@echo "Current build metadata:"
	@echo "  VERSION=$(VERSION)"
	@echo "  COMMIT=$(COMMIT)"
	@echo "  DATE=$(DATE)"

build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(OUT_DIR)/$(BINARY_NAME) .

test:
	@echo "Running tests..."
	$(GO) test $(GOFLAGS) -cover ./...

bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

clean:
	@echo "Cleaning..."
	$(GO) clean
	@rm -rf $(OUT_DIR)

fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed"; exit 1)
	@golangci-lint run ./...

run: build
	./$(OUT_DIR)/$(BINARY_NAME)

dev-install:
	$(GO) mod download
	$(GO) mod tidy

cross-build:
	@echo "Cross-building $(VERSION)..."
	GOOS=linux   GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(OUT_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=darwin  GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(OUT_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(OUT_DIR)/$(BINARY_NAME)-darwin-arm64 .
	GOOS=linux   GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(OUT_DIR)/$(BINARY_NAME)-linux-arm64 .
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(OUT_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@echo "Binaries in $(OUT_DIR)/"

version:
	@./$(OUT_DIR)/$(BINARY_NAME) version

.DEFAULT_GOAL := help
