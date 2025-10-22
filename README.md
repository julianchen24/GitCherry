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
- `make build-all`: produce cross-compiled binaries under `dist/` for Darwin, Linux, and Windows (amd64/arm64).
- `make regen-golden`: refresh snapshot test fixtures (`UPDATE_GOLDEN=1 go test ./internal/tui`).

If you do not have `make`, the equivalent `go build ./...` and `go test ./...` commands work as well. For cross-compilation, run `GOOS=<os> GOARCH=<arch> go build ./cmd/gitcherry` for each desired target.

### Installing Binaries

Run `make build` for a host build or `make build-all` to populate `dist/` with ready-to-use binaries:

```
dist/
  gitcherry-darwin-amd64
  gitcherry-darwin-arm64
  gitcherry-linux-amd64
  gitcherry-linux-arm64
  gitcherry-windows-amd64.exe
  gitcherry-windows-arm64.exe
```

Copy the binary for your platform into a directory on your `$PATH` (for Windows, include the `.exe` extension).

## Documentation

See the [design spec](docs/design_spec.md) for project direction and roadmap, and the [usage guide](docs/USAGE.md) for CLI and TUI walkthroughs.
