BINARY_NAME := garmin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

GORELEASER := go run github.com/goreleaser/goreleaser/v2@latest

.PHONY: build install test lint clean snapshot release-check

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

## release-check: Validate the GoReleaser config
release-check:
	$(GORELEASER) check

## snapshot: Build a local cross-platform snapshot release (no publish)
snapshot:
	$(GORELEASER) release --snapshot --clean

## clean: Remove build artifacts
clean:
	rm -rf bin/ dist/

## help: Show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
