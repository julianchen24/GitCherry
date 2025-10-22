GO ?= go
BINARY ?= gitcherry
GOOSARCH ?= darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64

.PHONY: build test lint run regen-golden build-all

build:
	$(GO) build ./...

test:
	$(GO) test ./...

lint:
	golangci-lint run ./...

run:
	$(GO) run ./cmd/$(BINARY)

regen-golden:
	UPDATE_GOLDEN=1 $(GO) test ./internal/tui

build-all:
	@mkdir -p dist
	@for target in $(GOOSARCH); do \
		GOOS=$${target%/*}; \
		GOARCH=$${target##*/}; \
		OUTPUT=dist/$(BINARY)-$${GOOS}-$${GOARCH}; \
		if [ "$$GOOS" = "windows" ]; then OUTPUT=$${OUTPUT}.exe; fi; \
		GOOS=$$GOOS GOARCH=$$GOARCH $(GO) build -o $$OUTPUT ./cmd/$(BINARY); \
		echo "built $$OUTPUT"; \
	done
