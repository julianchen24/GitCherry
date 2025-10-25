# GitCherry
### Cross-platform CLI/TUI for safe, scripted cherry-pick workflows

## Features
- Keyboard-driven tview UI for branch and commit selection
- Dry-run mode that prints planned git commands before execution
- Automated cherry-pick, revert, and branch restore operations
- Duplicate patch detection via `git patch-id`
- Audit logging with undo/redo metadata in `.gitcherry/`
- Configurable message templates and workflow defaults via YAML
- Color-aware UI with monochrome fallback
- Supported on Linux, macOS, and Windows (amd64/arm64)

## Problem & Solution
**Problem:** Manual cherry-picks require copying SHAs, chaining brittle git commands, and juggling conflict recovery by hand; repeated reverts or branch restoration amplify the risk of inconsistent history, especially when patches must be duplicated across branches.

**Solution:** GitCherry wraps those workflows in a deterministic CLI/TUI that previews commit batches, enforces clean working trees, logs every apply for auditing, and exposes undo metadata so cherry-pick, revert, and restore operations remain atomic and reproducible across Linux, macOS, and Windows.

## Demo

## Architecture Overview
```
┌────────────┐   commands/flags    ┌──────────────┐
│ CLI (cobra)│────────────────────>│ internal/ops │
└────────────┘                     │  transfer    │
                                   │  revert      │───────┐
┌────────────┐ key events/selects  │  restore     │       │git commands
│ TUI (tview)│────────────────────>└──────────────┘       ▼
└────────────┘                               ┌────────────────┐
                                             │ internal/git   │
                                             └────────────────┘
                         audit/undo ┌────────────────┐
                                    │ internal/logs  │
                                    └────────────────┘
                         defaults   ┌────────────────┐
                                    │ internal/config│
                                    └────────────────┘
```

- `/internal/git`: thin wrapper over `git` CLI for `fetch`, `rev-list`, `patch-id`, etc.
- `/internal/tui`: tview-based interface for keyboard-only navigation, previews, and dialogs.
- `/internal/ops`: business logic for transfer, revert, restore, duplicate detection, and planners.
- `/internal/logs`: audit log writers plus undo/redo queue persisted under `.gitcherry/`.
- `/internal/config`: layered YAML loader (repo → user config → env/defaults).

## Installation
```bash
# Install from source (Go 1.22+)
go install github.com/<you>/gitcherry/cmd/gitcherry@latest

# Build host binary
make build

# Produce cross-platform binaries under dist/
make build-all
```

The `dist/` directory will contain binaries named `gitcherry-<os>-<arch>` (Windows builds include `.exe`). Copy the binary matching your platform into a directory on your `$PATH`.

## Quickstart
```bash
# Launch the TUI (dry-run by default)
gitcherry --tui

# Perform a dry-run transfer via CLI
gitcherry transfer \
  --from main \
  --to release \
  --range a1b2c3..d4e5f6 \
  --auto-message

# Apply the transfer after reviewing the plan
gitcherry transfer ... --apply
```

TUI keybindings: `?` help, `q` quit, `r` refresh remotes, `Space` marks the start commit, `Enter` confirms the range, `b` restores a branch at the highlighted commit, `Esc` closes modals.

## Configuration
GitCherry works out of the box; optional overrides live in `.gitcherry.yml` (or `$HOME/.config/gitcherry/config.yml`). Supported fields:

```yaml
on_duplicate: ask      # ask | skip | apply
preview: true
auto_refresh: false
default_branch: main
message_template: |
  [Transfer] Moved commits from {source} → {target}
  Range: {range}
```

Environment variables `GITCHERRY_*` mirror these fields. When unset, defaults shown above are used.

## Command Reference
| Command | Description |
| --- | --- |
| `transfer --from <src> --to <dst> --range a..b [--message \| --edit \| --auto-message] [--apply]` | Cherry-picks the specified range onto the target branch. Dry-run prints `git checkout`, `git cherry-pick --no-commit`, and `git commit` steps. |
| `revert --on <branch> --range a..b [--message] [--apply]` | Reverts a commit or range on the given branch. Shows the planned `git revert --no-commit` + `git commit` commands unless `--apply` is provided. |
| `restore --at <commit> --branch-name <name> [--apply]` | Creates a new branch pointing at the specified commit. |
| `undo` | Displays the most recent recorded operation with before/after HEADs to guide manual resets. |
| `redo` | Displays the next redo entry, mirroring `undo`. |

All commands respect `--apply` for dry-run vs. execution and `--on-duplicate` (ask/skip/apply). The TUI and CLI both enforce a clean working tree before operating.

## Conflict Handling & Safety
- If `git cherry-pick` or `git revert` encounters conflicts during `--apply`, GitCherry surfaces the failure and indicates the git command that stopped.
- Resolve the conflicts manually, then run:
  - `git cherry-pick --continue` (for transfers)
  - `git revert --continue` (for reverts)
- To abandon the operation use the corresponding `--abort` form.
- Each successful apply writes an audit entry (`.gitcherry/logs/`) and updates the undo stack (`.gitcherry/undo.json`), enabling inspection or rollback.

## Development Guide
Prerequisites: Go 1.22+, `golangci-lint` (optional), make.

```
# Fetch deps
go mod tidy

# Build host binary
make build

# Run unit tests
make test

# Regenerate TUI golden snapshots
make regen-golden

# Start the CLI/TUI locally
make run
```

Repo layout follows the Go convention (`/cmd/gitcherry`, `/internal/...`, `/tests`). Contributions should keep PRs scoped, include tests, and ensure CI (`.github/workflows/ci.yml`) stays green.

Additional design notes live in:
- `docs/design_spec.md`
- `docs/prompt_plan.md`

## License
MIT License — see [`LICENSE`](LICENSE).

## Acknowledgements / Credits
- Built with Go 1.22+, Cobra, and tview.
- Snapshot testing inspired by existing golden-file patterns in the Go ecosystem.
- Thanks to the open-source Git tooling community for continual inspiration.
