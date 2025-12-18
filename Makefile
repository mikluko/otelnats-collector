.PHONY: build test lint clean tidy

# Build variables
BINARY_NAME := opentelemetry-collector-nats
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0-dev")
BUILD_DIR := ./bin
CMD_DIR := ./cmd/opentelemetry-collector-nats

# Go build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

## build: Build the collector binary
build: tidy
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

## test: Run tests
test:
	go test -race -v ./...

## lint: Run linter
lint:
	golangci-lint run ./...

## tidy: Tidy and verify go modules
tidy:
	go mod tidy

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)

## run: Run the collector with example config
run: build
	$(BUILD_DIR)/$(BINARY_NAME) --config examples/gateway/config.yaml

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
