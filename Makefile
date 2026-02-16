BINARY_NAME := garmin-cli
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build install test lint clean

## build: Build the binary
build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/garmin/

## install: Install to $GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/garmin/

## test: Run all tests
test:
	go test ./... -v

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## clean: Remove build artifacts
clean:
	rm -rf bin/

## help: Show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
