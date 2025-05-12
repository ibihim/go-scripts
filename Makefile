# Get a list of all directories in cmd/
CMDS := $(notdir $(wildcard cmd/*))

# Set GOBIN to a default if not set in the environment
GOBIN ?= $(shell go env GOPATH)/bin

# Define go build flags
GOBUILD_FLAGS := -trimpath

.PHONY: all clean install lint test help

# Build and install all commands
all: install

# Install all commands to $GOBIN
install:
	@echo "Installing binaries to $(GOBIN)..."
	@for cmd in $(CMDS); do \
		echo "Building and installing $$cmd..."; \
		go install $(GOBUILD_FLAGS) ./cmd/$$cmd; \
	done
	@echo "Installation complete."

# Clean build artifacts and binaries
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(addprefix $(GOBIN)/, $(CMDS))
	@go clean ./...
	@echo "Clean complete."

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run go vet and staticcheck if available
lint:
	@echo "Running linters..."
	@go vet ./...
	@if command -v staticcheck > /dev/null; then \
		staticcheck ./...; \
	else \
		echo "Staticcheck not installed. Skipping."; \
	fi

# Show available commands
help:
	@echo "Available make targets:"
	@echo "  all      - Default target, same as 'install'"
	@echo "  install  - Build and install all commands to \$$GOBIN (currently $(GOBIN))"
	@echo "  clean    - Remove build artifacts and installed binaries"
	@echo "  test     - Run all tests"
	@echo "  lint     - Run linters (go vet and staticcheck if installed)"
	@echo "  help     - Show this help message"
	@echo ""
	@echo "Available commands to install:"
	@for cmd in $(CMDS); do \
		echo "  $$cmd"; \
	done
