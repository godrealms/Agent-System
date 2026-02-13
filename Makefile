# Makefile for Claude Agent System

.PHONY: build test clean install example

# Default target
all: build

# Build the agent binary
build:
	go build -o bin/agent cmd/agent/main.go

# Install dependencies
install:
	go mod tidy
	go mod download

# Run tests
test:
	go test ./...

# Run tests with coverage
test-cover:
	go test -cover ./...

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Run example project initialization
example:
	chmod +x examples/init-claude-clone.sh
	./examples/init-claude-clone.sh claude-example

# Run agent on example project
run-example: build example
	bin/agent -project-dir=./claude-example -init -project-type=web-chat-app
	bin/agent -project-dir=./claude-example -sessions=3

# Generate documentation
docs:
	go doc ./...

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Install the agent globally
install-global: build
	sudo cp bin/agent /usr/local/bin/claude-agent

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Build the agent binary"
	@echo "  install      - Install dependencies"
	@echo "  test         - Run tests"
	@echo "  test-cover   - Run tests with coverage"
	@echo "  clean        - Clean build artifacts"
	@echo "  example      - Create example project"
	@echo "  run-example  - Build and run example"
	@echo "  docs         - Generate documentation"
	@echo "  fmt          - Format code"
	@echo "  vet          - Vet code"
	@echo "  install-global - Install agent globally"
	@echo "  help         - Show this help"