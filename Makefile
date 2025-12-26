# Makefile for nuke

BINARY_NAME=nuke
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Platforms
PLATFORMS=darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 linux/arm freebsd/amd64

.PHONY: all build clean test coverage lint install uninstall release-dry release help

all: clean build

## Build the binary for current platform
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

## Build optimized binary for current platform
build-prod:
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

## Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -rf dist/

## Run tests
test:
	$(GOTEST) -v ./...

## Run tests with coverage
coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## Run linter
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: brew install golangci-lint"; \
	fi

## Install to /usr/local/bin
install: build
	sudo mv $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to /usr/local/bin/"

## Uninstall from /usr/local/bin
uninstall:
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstalled $(BINARY_NAME) from /usr/local/bin/"

## Build for all platforms
build-all:
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		output=$(BINARY_NAME)-$$os-$$arch; \
		echo "Building $$output..."; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $$output .; \
	done

## Dry run of goreleaser
release-dry:
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --snapshot --clean; \
	else \
		echo "goreleaser not installed. Install with: brew install goreleaser"; \
	fi

## Create a new release (requires git tag)
release:
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --clean; \
	else \
		echo "goreleaser not installed. Install with: brew install goreleaser"; \
	fi

## Tidy go modules
tidy:
	$(GOMOD) tidy

## Download dependencies
deps:
	$(GOMOD) download

## Show help
help:
	@echo "nuke - A safer, faster, and more user-friendly alternative to rm"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' Makefile | sed 's/## /  /'
	@echo ""
	@echo "Examples:"
	@echo "  make build        - Build for current platform"
	@echo "  make build-all    - Build for all platforms"
	@echo "  make install      - Install to /usr/local/bin"
	@echo "  make release-dry  - Test release process"
