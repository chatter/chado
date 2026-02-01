.PHONY: all build clean test test-verbose test-cover test-race lint install

# Binary name
BINARY := chado
VERSION := 0.1.0

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOVET := $(GOCMD) vet

# Build flags
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

all: clean build test

build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY) .

build-debug:
	$(GOBUILD) -o $(BINARY) .

clean:
	$(GOCLEAN)
	rm -f $(BINARY)
	rm -f coverage.out

test:
	$(GOTEST) ./... -count=1

test-verbose:
	$(GOTEST) ./... -v -count=1

test-cover:
	$(GOTEST) ./... -cover -coverprofile=coverage.out
	$(GOCMD) tool cover -func=coverage.out

test-cover-html: test-cover
	$(GOCMD) tool cover -html=coverage.out

test-race:
	$(GOTEST) ./... -race -count=1

lint:
	$(GOVET) ./...
	@if command -v staticcheck > /dev/null; then \
		staticcheck ./...; \
	else \
		echo "staticcheck not installed, skipping"; \
	fi

install:
	$(GOCMD) install $(LDFLAGS) .

deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Development helpers
run: build
	./$(BINARY)

watch:
	@echo "Watching for changes..."
	@while true; do \
		make build 2>&1; \
		fswatch -1 -r --include='\.go$$' --exclude='.*' . ; \
	done

help:
	@echo "Available targets:"
	@echo "  all          - Clean, build, and test"
	@echo "  build        - Build the binary (optimized)"
	@echo "  build-debug  - Build the binary (with debug symbols)"
	@echo "  clean        - Remove binary and coverage files"
	@echo "  test         - Run tests"
	@echo "  test-verbose - Run tests with verbose output"
	@echo "  test-cover   - Run tests with coverage report"
	@echo "  test-race    - Run tests with race detector"
	@echo "  lint         - Run go vet and staticcheck"
	@echo "  install      - Install binary to GOPATH/bin"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  run          - Build and run"
	@echo "  help         - Show this help"
