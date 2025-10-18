GO ?= go
BINARY ?= gitcherry

.PHONY: build test lint run

build:
	$(GO) build ./...

test:
	$(GO) test ./...

lint:
	golangci-lint run ./...

run:
	$(GO) run ./cmd/$(BINARY)
