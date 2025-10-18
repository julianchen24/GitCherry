# TODO — GitCherry

Comprehensive, checkable list for implementing GitCherry (Go, CLI/TUI)

---

## Milestone 0 — Bootstrap & Scaffolding

* [ ] Initialize repo & module

  * [ ] `go mod init github.com/<you>/gitcherry`
  * [ ] Add deps: `tview`, `cobra`, `testify` (and optionally `golangci-lint`)
  * [ ] Create `Makefile` (build/test/lint/run)
  * [ ] Add CI workflow (`.github/workflows/ci.yml`): build + `go test ./...`
  * [ ] Add `LICENSE` (MIT)
  * [ ] Add `.gitignore` (Go, editor, `.gitcherry/` logs)
  * [ ] Seed `README.md` with brief intro + link to `docs/design_spec.md`
* [ ] Docs folder

* [ ] Project layout

  * [ ] `/cmd/gitcherry`
  * [ ] `/internal/git`
  * [ ] `/internal/tui`
  * [ ] `/internal/ops`
  * [ ] `/internal/config`
  * [ ] `/internal/logs`
  * [ ] `/tests` (fixtures/helpers)

## Milestone 0.1 — Config loader

* [ ] Implement `internal/config`

  * [ ] `Config` struct with defaults (on_duplicate, preview, autorefresh, default_branch, message_template)
  * [ ] Loader: repo `.gitcherry.yml` → `$HOME/.config/gitcherry/config.yml` → defaults
  * [ ] Unit tests: missing file, partial override, full override

## Milestone 0.2 — Git runner

* [ ] Implement `internal/git`

  * [ ] `Runner` with `Run(args ...string)` (env `GIT_TERMINAL_PROMPT=0`)
  * [ ] Helpers: `CurrentBranch`, `IsClean`, `Fetch`, `ListBranches`, `CommitsBetween`, `PatchID`
  * [ ] `Commit` struct (Hash, Author, Date, Message, Files)
  * [ ] Temp-repo fixtures under `/tests` and unit tests for helpers

---

## Milestone 1 — CLI Skeleton (dry-run default)

* [ ] Cobra root command in `/cmd/gitcherry`

  * [ ] Global flags: `--apply` (default false), `--refresh`, `--no-preview`
  * [ ] Subcommands: `transfer`, `revert`, `restore`, `undo`, `redo` (stubs)
  * [ ] Pre-run: require clean workspace via `git.IsClean()`
  * [ ] Tests: help output, unknown flag handling, dirty workspace exit

---

## Milestone 2 — TUI Skeleton (tview)

* [ ] App shell

  * [ ] `App` struct with refs to git runner + config
  * [ ] Views: BranchList, CommitList, PreviewModal, HelpModal
  * [ ] Keybindings: `?` help, `q` quit
  * [ ] Tests: init + help toggle
* [ ] Branch selection view

  * [ ] List local branches (blue highlight for focus)
  * [ ] First select = source; second = target → open CommitList
  * [ ] Optional banner: “Press `r` to refresh remotes” (wire later)
  * [ ] Tests: selection flow state
* [ ] Commit selection (contiguous only)

  * [ ] Show commits from `CommitsBetween(target, source)`
  * [ ] Start mark with Space, end confirm with Enter
  * [ ] Selected range highlighted green
  * [ ] Tests: start/end capture

---

## Milestone 3 — Transfer (Consolidated Commit Transfer)

* [ ] Preview + message UX

  * [ ] Table summary (hash short, author, subject)
  * [ ] Display target branch and “→ Will become 1 new commit”
  * [ ] Actions: **Edit message** (multiline), **Use suggested message** (template)
  * [ ] Tests: template rendering, action selection
* [ ] Planner (dry-run)

  * [ ] Generate steps: checkout target → cherry-pick `--no-commit` range → commit
  * [ ] CLI dry-run prints steps; TUI shows scrollable preview
  * [ ] Tests: sequence generation
* [ ] Executor (apply) with conflicts

  * [ ] Run planned steps; detect conflicts; expose `ErrConflict`
  * [ ] TUI modal: “Resolve manually or Abort?” with Continue/Abort paths
  * [ ] CLI prints guidance and exits nonzero on conflict (when `--apply`)
  * [ ] Integration tests: resolve path + abort path
* [ ] Audit log + undo/redo enqueue

  * [ ] `.gitcherry/logs/<timestamp>.json` writer
  * [ ] `.gitcherry/undo.json` queue (push on success)
  * [ ] Tests: serialization + queue behavior

---

## Milestone 4 — Rollback & Restore

* [ ] `revert` operation

  * [ ] Planner: `git revert --no-commit <start>^..<end>` → `git commit -m ...`
  * [ ] Conflict handling: continue/abort
  * [ ] Reuse preview + messaging
  * [ ] Integration tests
* [ ] `restore` operation

  * [ ] Commit picker → prompt branch name → `git branch <name> <commit>`
  * [ ] Audit log + undo entry (optional delete branch on undo)
  * [ ] Tests: unit + integration

---

## Milestone 5 — Duplicates & Refresh

* [ ] Duplicate detection

  * [ ] Compute `patch-id` for each selected commit; compare on target
  * [ ] TUI: Ask on duplicates (Yes/No)
  * [ ] CLI: `--on-duplicate=ask|skip|apply` (default ask; skip when non-interactive)
  * [ ] Tests with crafted commits
* [ ] Refresh controls

  * [ ] `--refresh` flag triggers `git fetch --prune --tags`
  * [ ] TUI key `r` performs fetch and reloads lists
  * [ ] Tests: mocked fetch

---

## Milestone 6 — CLI Parity & Polish

* [ ] Wire all cobra handlers fully

  * [ ] `transfer --from --to --range a..b [--message|--edit|--auto-message] [--apply]`
  * [ ] `revert --on HEAD --range a..b`
  * [ ] `restore --at <commit> --branch-name <name>`
  * [ ] `undo`, `redo`
  * [ ] Tests: arg parsing, planner wiring
* [ ] Help panel & colors

  * [ ] `?` shows keybindings
  * [ ] ANSI-safe color constants with monochrome fallback
  * [ ] Tests: color guard logic
* [ ] Usage docs

  * [ ] Create `docs/USAGE.md` (Quickstart, TUI flow, CLI examples, conflict handling)

---

## Milestone 7 — Test Hardening & Release

* [ ] TUI golden snapshot tests (`/tests/golden`)

  * [ ] Deterministic dataset
  * [ ] `make regen-golden` target
* [ ] Cross-platform builds

  * [ ] `Makefile` targets for darwin/linux/windows amd64/arm64
  * [ ] (Optional) `goreleaser` config
* [ ] README updates

  * [ ] Install instructions (binaries or `go install`)
  * [ ] Link to `docs/USAGE.md`

---

## Nice-to-haves / Backlog

* [ ] Non-contiguous selection (advanced) with dependency warnings
* [ ] GitLab API integration
* [ ] Multi-repo mode (workspace orchestrator)
* [ ] Persist last-viewed branch/session
* [ ] External difftool handoff (`D`)

---

## Meta

* [ ] Keep PRs small (per milestone/step)
* [ ] Ensure CI stays green before moving to next milestone

