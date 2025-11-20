.PHONY: build test test-race test-integration test-property lint clean help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Binary names
GRIMOPS_BINARY=grimops
NECROOPS_BINARY=necroops

# Build directory
BUILD_DIR=bin

# Linter
GOLANGCI_LINT=golangci-lint

all: test build

## build: Build all binaries
build: build-grimops build-necroops

## build-grimops: Build GrimOps binary
build-grimops:
	@echo "Building GrimOps..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(GRIMOPS_BINARY) ./cmd/grimops

## build-necroops: Build NecroOps binary
build-necroops:
	@echo "Building NecroOps..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(NECROOPS_BINARY) ./cmd/necroops

## test: Run unit tests
test:
	@echo "Running unit tests..."
	$(GOTEST) -v -cover ./...

## test-race: Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	$(GOTEST) -v -race ./...

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -tags=integration ./...

## test-property: Run property-based tests
test-property:
	@echo "Running property-based tests..."
	$(GOTEST) -v -tags=property ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## lint: Run linter
lint:
	@echo "Running linter..."
	@if command -v $(GOLANGCI_LINT) > /dev/null; then \
		$(GOLANGCI_LINT) run ./...; \
	else \
		echo "golangci-lint not installed. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		$(GOLANGCI_LINT) run ./...; \
	fi

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...
	@if command -v goimports > /dev/null; then \
		goimports -w .; \
	fi

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	$(GOCMD) clean

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

