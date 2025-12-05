# FlowGauge Makefile
# Usage: make [target]

# Variables
BINARY_NAME := flowgauge
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/lan-dot-party/flowgauge/pkg/version.Version=$(VERSION) \
	-X github.com/lan-dot-party/flowgauge/pkg/version.Commit=$(COMMIT) \
	-X github.com/lan-dot-party/flowgauge/pkg/version.BuildDate=$(BUILD_DATE)

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Directories
CMD_DIR := ./cmd/flowgauge
BUILD_DIR := ./bin

.PHONY: all build clean test deps lint run install docker help

# Default target
all: clean deps build

# Build the binary
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for current OS without version info (fast build for development)
build-dev:
	@echo "Building $(BINARY_NAME) (dev)..."
	$(GOBUILD) -o $(BINARY_NAME) $(CMD_DIR)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY_NAME)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Run linter
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed" && exit 1)
	golangci-lint run ./...

# Run the application
run: build-dev
	./$(BINARY_NAME) server

# Run a test
test-run: build-dev
	./$(BINARY_NAME) test

# Install to /usr/local/bin
install: build
	@echo "Installing to /usr/local/bin..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "Installed: /usr/local/bin/$(BINARY_NAME)"

# Uninstall from /usr/local/bin
uninstall:
	@echo "Uninstalling..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)

# Build Docker image
docker:
	@echo "Building Docker image..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(BINARY_NAME):$(VERSION) \
		-t $(BINARY_NAME):latest \
		.

# Build for multiple platforms
build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
	@echo "Built binaries in $(BUILD_DIR)/"
	@ls -la $(BUILD_DIR)/

# Release using GoReleaser (requires goreleaser)
release:
	@echo "Creating release..."
	goreleaser release --clean

# Snapshot release (no publish)
release-snapshot:
	@echo "Creating snapshot release..."
	goreleaser release --snapshot --clean

# Show version info
version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(BUILD_DATE)"

# Help
help:
	@echo "FlowGauge Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all              Clean, download deps, and build"
	@echo "  build            Build the binary with version info"
	@echo "  build-dev        Quick build for development"
	@echo "  build-all        Build for all platforms"
	@echo "  clean            Remove build artifacts"
	@echo "  deps             Download dependencies"
	@echo "  test             Run tests"
	@echo "  test-coverage    Run tests with coverage report"
	@echo "  lint             Run linter"
	@echo "  run              Build and run the server"
	@echo "  test-run         Build and run a speedtest"
	@echo "  install          Install to /usr/local/bin"
	@echo "  uninstall        Remove from /usr/local/bin"
	@echo "  docker           Build Docker image"
	@echo "  release          Create release with GoReleaser"
	@echo "  release-snapshot Create snapshot release"
	@echo "  version          Show version info"
	@echo "  help             Show this help"


