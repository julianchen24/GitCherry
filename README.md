# GitCherry

GitCherry is a terminal-first assistant for orchestrating cherry-pick workflows on top of Git.

## Getting Started

```bash
go mod tidy
make build
```

- `make build`: compile the project into a binary.
- `make test`: run unit tests (`go test ./...`).
- `make lint`: run `golangci-lint` if installed; install it from https://golangci-lint.run/usage/install/ when needed.
- `make run`: start the interactive CLI (`go run ./cmd/gitcherry`).

If you do not have `make`, the equivalent `go build ./...` and `go test ./...` commands work as well.

## Documentation

See the [design spec](docs/design_spec.md) for project direction and roadmap.
