BINARY_NAME := garmin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build install test coverage coverage-check lint clean

## build: Build the binary
build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/garmin/

## install: Install to $GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/garmin/

## test: Run all tests
test:
	go test ./... -v

## coverage: Run tests with coverage output
coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out | awk '/^total:/ {print $$0}'

## coverage-check: Fail if coverage is below COVERAGE_MIN (default: 80)
coverage-check: coverage
	bash scripts/check_coverage.sh $${COVERAGE_MIN:-80} coverage.out

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## clean: Remove build artifacts
clean:
	rm -rf bin/

## help: Show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
