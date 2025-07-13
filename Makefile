# CLI tools
TOOLS := gocker ginx

.PHONY: all build test clean

# Default target
all: test build

# Build all tools
build:
	@for tool in $(TOOLS); do \
		echo "Building $$tool..."; \
		go build -o bin/$$tool ./cmd/$$tool; \
	done

# Run tests
test:
	@go test ./...

# Clean build artifacts
clean:
	@rm -rf bin/

# Build specific tool (usage: make tool1)
$(TOOLS):
	@go build -o bin/$@ ./cmd/$@