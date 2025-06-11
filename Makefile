# Shell Reserve (XSL) - Build System
# Digital Gold for Central Banks

.PHONY: all build test clean deps randomx install help

# Build configuration
BINARY_NAME=shell
GO_VERSION=$(shell go version | cut -d' ' -f3)
BUILD_TIME=$(shell date -u +%Y%m%d.%H%M%S)
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
VERSION=0.24.2-beta

# Build flags
LDFLAGS=-ldflags "-X main.appBuild=$(BUILD_TIME).$(GIT_COMMIT)"
BUILD_FLAGS=-trimpath $(LDFLAGS)

# Default target
all: deps build test

help: ## Show this help message
	@echo "Shell Reserve (XSL) Build System"
	@echo "================================"
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

deps: ## Install dependencies and build RandomX
	@echo "Installing Go dependencies..."
	go mod download
	go mod verify
	@echo "Building RandomX mining library..."
	$(MAKE) -C mining/randomx

build: deps ## Build Shell Reserve binary
	@echo "Building Shell Reserve v$(VERSION)..."
	@echo "Go version: $(GO_VERSION)"
	@echo "Build time: $(BUILD_TIME)"
	@echo "Git commit: $(GIT_COMMIT)"
	go build $(BUILD_FLAGS) -o $(BINARY_NAME) .

build-race: deps ## Build with race detection (for development)
	@echo "Building Shell Reserve with race detection..."
	go build $(BUILD_FLAGS) -race -o $(BINARY_NAME)-race .

test: ## Run all tests
	@echo "Running Shell Reserve test suite..."
	go test -v -timeout 30m ./...

test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	go test -v -race -timeout 30m ./...

test-coverage: ## Run tests with coverage report
	@echo "Running test coverage analysis..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	go test -v -timeout 60m -tags=integration ./test/

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

lint: ## Run linters
	@echo "Running linters..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	golangci-lint run ./...

format: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

randomx: ## Build only RandomX library
	$(MAKE) -C mining/randomx

randomx-clean: ## Clean RandomX build artifacts
	$(MAKE) -C mining/randomx clean

# Network configurations
mainnet: build ## Run Shell Reserve on mainnet (PRODUCTION)
	@echo "⚠️  MAINNET MODE - PRODUCTION READY"
	@echo "Launch date: January 1, 2026, 00:00 UTC"
	./$(BINARY_NAME) --configfile=shell-mainnet.conf

testnet: build ## Run Shell Reserve on testnet
	@echo "Running Shell Reserve on testnet..."
	./$(BINARY_NAME) --testnet --configfile=shell-testnet.conf

regtest: build ## Run Shell Reserve in regression test mode
	@echo "Running Shell Reserve in regression test mode..."
	./$(BINARY_NAME) --regtest --configfile=shell-regtest.conf

simnet: build ## Run Shell Reserve on simulation network
	@echo "Running Shell Reserve on simulation network..."
	./$(BINARY_NAME) --simnet --configfile=shell-simnet.conf

# Development tools
dev: build-race ## Build and run in development mode
	@echo "Starting Shell Reserve in development mode..."
	./$(BINARY_NAME)-race --regtest --debuglevel=debug

mine: build ## Start CPU mining on regtest
	@echo "Starting CPU mining on regtest network..."
	./$(BINARY_NAME) --regtest --generate --miningaddr=<MINING_ADDRESS>

# Installation and distribution
install: build ## Install Shell Reserve binary
	@echo "Installing Shell Reserve to $(GOPATH)/bin/$(BINARY_NAME)..."
	cp $(BINARY_NAME) $(GOPATH)/bin/

# Docker support
docker-build: ## Build Docker image
	@echo "Building Shell Reserve Docker image..."
	docker build -t shell-reserve:$(VERSION) .

docker-run: ## Run Shell Reserve in Docker
	@echo "Running Shell Reserve in Docker..."
	docker run -p 8533:8533 -p 8534:8534 shell-reserve:$(VERSION)

# Security and auditing
audit: ## Run security audit
	@echo "Running security audit..."
	@command -v gosec >/dev/null 2>&1 || { echo "gosec not installed. Run: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; exit 1; }
	gosec ./...

vuln-check: ## Check for known vulnerabilities
	@echo "Checking for known vulnerabilities..."
	@command -v govulncheck >/dev/null 2>&1 || { echo "govulncheck not installed. Run: go install golang.org/x/vuln/cmd/govulncheck@latest"; exit 1; }
	govulncheck ./...

# Genesis and launch preparation
genesis: build ## Generate genesis block (for January 1, 2026 launch)
	@echo "Generating Shell Reserve genesis block..."
	@echo "Target launch: January 1, 2026, 00:00:00 UTC"
	./$(BINARY_NAME) --regtest --generate-genesis

# Cleanup
clean: randomx-clean ## Clean all build artifacts
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME) $(BINARY_NAME)-race
	rm -f coverage.out coverage.html
	go clean ./...
	go clean -cache
	go clean -testcache

clean-all: clean ## Clean everything including dependencies
	@echo "Cleaning all artifacts and dependencies..."
	go clean -modcache

# Release preparation
release-check: test lint vet audit ## Run all checks required for release
	@echo "All release checks passed! ✅"
	@echo "Shell Reserve v$(VERSION) ready for institutional deployment"

release-notes: ## Generate release notes
	@echo "Shell Reserve v$(VERSION) Release Notes"
	@echo "======================================"
	@echo ""
	@echo "## Institutional Features"
	@echo "✅ RandomX CPU mining (decentralized)"  
	@echo "✅ Confidential Transactions (amounts hidden)"
	@echo "✅ Standard multisig custody (2-of-3, 3-of-5, 11-of-15)"
	@echo "✅ Time locks (nLockTime, CLTV)"
	@echo "✅ Document hash commitments (audit trails)"
	@echo "✅ Claimable balances (Stellar-style escrow)"
	@echo "✅ Bilateral payment channels"
	@echo "✅ ISO 20022 SWIFT compatibility"
	@echo "✅ Atomic swaps (cross-chain settlement)"
	@echo ""
	@echo "## Network Specifications"
	@echo "- Supply Cap: 100,000,000 XSL"
	@echo "- Block Time: 5 minutes"
	@echo "- Block Size: 500KB (1MB emergency)"
	@echo "- Minimum Transaction: 1 XSL (institutional focus)"
	@echo "- No premine, no special privileges"
	@echo ""
	@echo "## Launch Information"
	@echo "- Target Launch: January 1, 2026, 00:00:00 UTC"
	@echo "- Pure proof-of-work, fair launch"
	@echo "- Digital gold for central banks"

# Documentation
docs: ## Generate documentation
	@echo "Generating documentation..."
	godoc -http=:6060 &
	@echo "Documentation server started at http://localhost:6060"

# Quick start for institutions
quick-start: ## Show quick start guide for institutions
	@echo "Shell Reserve Quick Start for Institutions"
	@echo "========================================"
	@echo ""
	@echo "1. Build and test:"
	@echo "   make build test"
	@echo ""
	@echo "2. Run on testnet:"
	@echo "   make testnet"
	@echo ""
	@echo "3. Configure institutional custody:"
	@echo "   - Generate multisig addresses (2-of-3, 3-of-5)"
	@echo "   - Set up cold storage with time locks"
	@echo "   - Configure bilateral channels with counterparties"
	@echo ""
	@echo "4. Trade documentation:"
	@echo "   - Use OP_DOC_HASH for immutable audit trails"
	@echo "   - Commit Bills of Lading, Letters of Credit"
	@echo ""
	@echo "5. SWIFT integration:"
	@echo "   - Configure ISO 20022 message mapping"
	@echo "   - Generate settlement finality proofs"
	@echo ""
	@echo "For production deployment, wait until mainnet launch:"
	@echo "January 1, 2026, 00:00:00 UTC"

# Version information
version: ## Show version information
	@echo "Shell Reserve v$(VERSION)"
	@echo "Go version: $(GO_VERSION)"
	@echo "Build time: $(BUILD_TIME)"
	@echo "Git commit: $(GIT_COMMIT)"
	@echo ""
	@echo "Target mainnet launch: January 1, 2026, 00:00:00 UTC"
	@echo "Digital gold for the 21st century" 