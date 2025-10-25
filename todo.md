# TODO - GitCherry

Comprehensive, checkable list for implementing GitCherry (Go, CLI/TUI)

---

## Milestone 0 - Bootstrap & Scaffolding

* [ ] Initialize repo & module
  * [x] `go mod init github.com/<you>/gitcherry`
  * [x] Add deps: `tview`, `cobra`, `testify` (and optionally `golangci-lint`)
  * [x] Create `Makefile` (build/test/lint/run)
  * [x] Add CI workflow (`.github/workflows/ci.yml`): build + `go test ./...`
  * [x] Add `LICENSE` (MIT)
  * [ ] Add `.gitignore` (Go, editor, `.gitcherry/` logs)
  * [x] Seed `README.md` with brief intro + link to `docs/design_spec.md`
* [x] Docs folder
* [x] Project layout
  * [x] `/cmd/gitcherry`
  * [x] `/internal/git`
  * [x] `/internal/tui`
  * [x] `/internal/ops`
  * [x] `/internal/config`
  * [x] `/internal/logs`
  * [x] `/tests` (fixtures/helpers)

## Milestone 0.1 - Config loader

* [x] Implement `internal/config`
  * [x] `Config` struct with defaults (on_duplicate, preview, autorefresh, default_branch, message_template)
  * [x] Loader: repo `.gitcherry.yml` -> `$HOME/.config/gitcherry/config.yml` -> defaults (+ env vars)
  * [x] Unit tests: missing file, partial override, full override

## Milestone 0.2 - Git runner

* [x] Implement `internal/git`
  * [x] `Runner` with `Run(args ...string)` (env `GIT_TERMINAL_PROMPT=0`)
  * [x] Helpers: `CurrentBranch`, `IsClean`, `Fetch`, `ListBranches`, `CommitsBetween`, `PatchID`
  * [x] `Commit` struct (Hash, Author, Date, Message, Files)
  * [x] Temp-repo fixtures under `/tests` and unit tests for helpers

---

## Milestone 1 - CLI Skeleton (dry-run default)

* [x] Cobra root command in `/cmd/gitcherry`
  * [x] Global flags: `--apply`, `--refresh`, `--no-preview`
  * [x] Subcommands: `transfer`, `revert`, `restore`, `undo`, `redo`
  * [x] Pre-run: require clean workspace via `git.IsClean()`
  * [x] Tests: help output, unknown flag handling, dirty workspace exit

---

## Milestone 2 - TUI Skeleton (tview)

* [x] App shell
  * [x] `App` struct with refs to git runner + config
  * [x] Views: BranchList, CommitList, PreviewModal, HelpModal
  * [x] Keybindings: `?` help, `q` quit
  * [x] Tests: init + help toggle
* [x] Branch selection view
  * [x] List local branches (focus highlight)
  * [x] First select = source; second = target -> open CommitList
  * [x] Banner: "Press `r` to refresh remotes" (wired to fetch)
  * [x] Tests: selection flow state
* [x] Commit selection (contiguous only)
  * [x] Show commits from `CommitsBetween(target, source)`
  * [x] Start mark with Space, end confirm with Enter
  * [x] Selected range highlighted
  * [x] Tests: start/end capture

---

## Milestone 3 - Transfer (Consolidated Commit Transfer)

* [x] Preview + message UX
  * [x] Table summary (hash short, author, subject)
  * [x] Display target branch and "-> Will become 1 new commit"
  * [x] Actions: **Edit message**, **Use suggested message**
  * [x] Tests: template rendering, action selection
* [x] Planner (dry-run)
  * [x] Generate steps: checkout target -> cherry-pick `--no-commit` range -> commit
  * [x] CLI dry-run prints steps; TUI shows preview
  * [x] Tests: sequence generation
* [ ] Executor (apply) with conflicts
  * [x] Run planned steps
  * [ ] Detect conflicts; expose `ErrConflict`
  * [ ] TUI modal: "Resolve manually or Abort" flow
  * [ ] CLI conflict guidance & non-zero exit
  * [ ] Integration tests: resolve path + abort path
* [x] Audit log + undo/redo enqueue
  * [x] `.gitcherry/logs/<timestamp>.json` writer
  * [x] `.gitcherry/undo.json` queue (push on success)
  * [x] Tests: serialization + queue behavior

---

## Milestone 4 - Rollback & Restore

* [ ] `revert` operation
  * [x] Planner: `git revert --no-commit <start>^..<end>` -> `git commit -m ...`
  * [ ] Conflict handling: continue/abort
  * [ ] Reuse preview + messaging
  * [x] Integration tests (basic success path)
* [x] `restore` operation
  * [x] Commit picker -> prompt branch name -> `git branch <name> <commit>`
  * [x] Audit log + undo entry (delete via manual undo guidance)
  * [x] Tests: unit + integration

---

## Milestone 5 - Duplicates & Refresh

* [x] Duplicate detection
  * [x] Compute `patch-id` for selected commits; compare on target
  * [x] TUI: ask on duplicates (Yes/No)
  * [x] CLI: `--on-duplicate=ask|skip|apply` defaulting to skip when non-interactive
  * [x] Tests with crafted commits
* [x] Refresh controls
  * [x] `--refresh` flag triggers `git fetch --prune --tags`
  * [x] TUI key `r` performs fetch and reloads lists
  * [x] Tests: mocked fetch

---

## Milestone 6 - CLI Parity & Polish

* [x] Wire all cobra handlers fully
  * [x] `transfer --from --to --range a..b [--message|--edit|--auto-message] [--apply]`
  * [x] `revert --on HEAD --range a..b`
  * [x] `restore --at <commit> --branch-name <name>`
  * [x] `undo`, `redo`
  * [x] Tests: arg parsing, planner wiring
* [x] Help panel & colors
  * [x] `?` shows keybindings
  * [x] ANSI-safe color constants with monochrome fallback
  * [x] Tests: color guard logic
* [x] Usage docs
  * [x] Create `docs/USAGE.md` (Quickstart, TUI flow, CLI examples, conflict handling)

---

## Milestone 7 - Test Hardening & Release
s
  * [x] Create `docs/USAGE.md` (Quickstart, TUI flow, CLI examples, conflict handling)

---

## Milestone 7 � Test Hardening & Release

* [x] TUI golden snapshot tests (`/tests/golden`)
  * [x] Deterministic dataset
  * [x] `make regen-golden` target
* [x] Cross-platform builds
  * [x] `Makefile` targets for darwin/linux/windows amd64/arm64
  * [ ] (Optional) `goreleaser` config
* [x] README updates
  * [x] Install instructions (binaries or `go install`)
  * [x] Link to `docs/USAGE.md`

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
