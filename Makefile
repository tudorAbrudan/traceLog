VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY  := tracelog
GOFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: all build dev lint test fmt clean

all: build

## build: Build the binary with embedded frontend
build: web-build
	go build $(GOFLAGS) -o $(BINARY) ./cmd/tracelog

## dev: Run in development mode (no embedded frontend)
dev:
	go run ./cmd/tracelog serve

## web-build: Build the Svelte frontend
web-build:
	@if [ -f web/package.json ]; then \
		cd web && npm ci && npm run build; \
	fi

## web-dev: Run frontend dev server
web-dev:
	cd web && npm run dev

## lint: Run linters (web-build required: hub uses go:embed dist)
lint: web-build
	golangci-lint run ./...
	@if [ -f web/package.json ]; then \
		cd web && npm run check && npx eslint . ; \
	fi

## test: Run all tests (web-build required: hub uses go:embed dist)
test: web-build
	go test -race -count=1 ./...
	@if [ -f web/package.json ]; then \
		cd web && npm run check && npx vitest run --passWithNoTests; \
	fi

## fmt: Format code
fmt:
	gofmt -w .
	goimports -w .
	@if [ -f web/package.json ]; then \
		cd web && npx prettier --write .; \
	fi

## clean: Remove build artifacts
clean:
	rm -f $(BINARY)
	rm -rf dist/
	rm -rf web/dist/
	rm -rf internal/hub/dist/

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //' | column -t -s ':'
