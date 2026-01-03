# ch - Claude Code History CLI
# Makefile for build, test, and install

BINARY_NAME := ch
INSTALL_PATH := ~/bin
GO := go
GOFLAGS := -trimpath -ldflags="-s -w"

.PHONY: all build install test lint clean help

## all: Build the binary (default)
all: build

## build: Build the binary in current directory
build:
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) ./cmd/ch

## install: Build and install to ~/bin
install:
	$(GO) build $(GOFLAGS) -o $(INSTALL_PATH)/$(BINARY_NAME) ./cmd/ch
	@echo "Installed to $(INSTALL_PATH)/$(BINARY_NAME)"

## test: Run all tests
test:
	$(GO) test ./...

## test-cover: Run tests with coverage
test-cover:
	$(GO) test -cover ./...

## test-verbose: Run tests with verbose output
test-verbose:
	$(GO) test -v ./...

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## fmt: Format code
fmt:
	$(GO) fmt ./...

## vet: Run go vet
vet:
	$(GO) vet ./...

## tidy: Tidy go.mod
tidy:
	$(GO) mod tidy

## clean: Remove build artifacts
clean:
	rm -f $(BINARY_NAME)
	$(GO) clean

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
