# GitCherry — Developer Specification

## 1. Project Summary

**Purpose:**
GitCherry is a lightweight, cross-platform **CLI/TUI tool written in Go** that automates complex cherry-pick and history-cleaning operations in Git. It removes the need to manually track commit SHAs and ensures atomic, consistent commit transfers between branches.

**Goal:**
Streamline professional Git workflows by allowing users to visually select commits (or commit ranges) and perform operations like **Consolidated Commit Transfers**, **Selective Rollbacks**, and **Branch Restorations** — all without ever typing a SHA manually.

**Core Philosophy:**

* Minimal, reliable, and terminal-native
* Zero magic — all actions transparent and reversible
* Always safe: user confirms every potentially destructive step

---

## 2. Target User & Licensing

* **Primary Users:** Developers and engineers working in professional team environments who need deterministic cherry-picking and history management without SHA complexity
* **Use Case:** Day-to-day team branching, backports, and feature management
* **License:** MIT License (open-source, simple reuse)
* **Supported Platforms:**

  * GitHub and Bitbucket (primary)
  * GitLab (planned, interface-compatible)

---

## 3. Technical Problem & Solution

### Problem:

Manual `git cherry-pick` and `git revert` workflows are:

* Error-prone (SHA-based)
* Redundant (reintroduce commits)
* Non-atomic (requires multiple manual commands)
* Opaque (no visual confirmation)

### Solution:

GitCherry provides:

* **Visual commit selection** (TUI)
* **Automated operation sequencing** (single keypress execution)
* **Consistent commit consolidation** (auto-squash into one clean commit)
* **Safe rollback & restoration** (structured undo/redo)

By enforcing structured operations, GitCherry eliminates fragmented history, redundant commits, and lost SHAs

---

## 4. Core Features

### A. Consolidated Commit Transfer

**Purpose:** Move a contiguous range of commits from a source branch to a target branch, and squash them into one atomic commit
**Workflow:**

1. User selects source and target branches
2. TUI displays commits (chronological, contiguous only)
3. User highlights range → preview table → confirm
4. GitCherry executes:

   ```bash
   git checkout <target>
   git cherry-pick --no-commit <start>^..<end>
   git commit -m "<user or auto message>"
   ```
5. Conflict handling (if any):

   * Show “Merge conflict detected. Resolve or abort?”
   * On resolve, continue; on abort, restore pre-op state
6. Write operation details to `.gitcherry/logs/YYYYMMDD_HHMM.json`

### B. Selective Rollback

**Purpose:** Revert one or more contiguous commits from the current branch
**Workflow:**

1. User highlights commits to revert
2. TUI shows preview → “Will undo 3 commits”
3. GitCherry runs:

   ```bash
   git revert --no-commit <start>^..<end>
   git commit -m "<user or auto message>"
   ```
4. If conflict → TUI prompt for manual resolve/abort

### C. Branch Restoration

**Purpose:** Recreate a branch from any historical commit
**Workflow:**

1. User selects a commit from the log
2. Prompt: “Enter new branch name” (default suggestion `feature/resurrected-<oldbranch>`)
3. Execute:

   ```bash
   git branch <newname> <commitSHA>
   ```

---

## 5. CLI + TUI Interaction Model

### Modes

* **TUI (default):**

  * Clean, keyboard-only navigation (`tview`)
  * Color-coded commit states
  * Built-in help (`?`) showing key shortcuts
  * Previews of commit summaries and diffs
  * Prompts for all confirmations (conflicts, messages, etc.)
* **CLI:**

  * Mirrors all operations (`transfer`, `revert`, `restore`, `undo`, `redo`)
  * Flags:

    ```
    --apply       # execute (default dry-run)
    --refresh     # run git fetch before
    --message/-m  # custom message
    --edit        # open editor for message
    --auto-message
    --on-duplicate=ask|skip|apply
    --dirty=fail|stash|ask
    ```
  * Example:

    ```bash
    gitcherry transfer --from main --to release/1.8 --range a1..b9 --apply
    ```

---

## 6. Configuration & Defaults

**Optional Config File:** `.gitcherry.yml`
Default behavior:

```yaml
on_duplicate: ask
preview: true
autorefresh: false
default_branch: release/main
message_template: "[Transfer] Moved commits from {source} → {target}\nRange: {range}"
```

**Never required**, but supported for convenience

---

## 7. Internal Data Structures (Go)

### Commit

```go
type Commit struct {
    Hash        string
    Author      string
    Date        time.Time
    Message     string
    Files       []string
}
```

### Operation

```go
type Operation struct {
    ID          string
    Type        string // "transfer" | "revert" | "restore"
    Source      string
    Target      string
    Commits     []Commit
    Timestamp   time.Time
    Status      string
    LogPath     string
}
```

### TUIState

```go
type TUIState struct {
    CurrentBranch string
    SelectedCommits []Commit
    Mode string // "transfer" | "revert" | "restore"
    HelpVisible bool
    MessageDraft string
}
```

---

## 8. Error Handling & Recovery

### Merge Conflicts

* Pause operation and prompt:

  ```
  Merge conflict detected. Resolve manually or abort?
  [Resolve] [Abort]
  ```
* On resolve → user presses “Continue” to finalize
* On abort → rollback with:

  ```bash
  git cherry-pick --abort
  ```

  or

  ```bash
  git revert --abort
  ```

### Workspace Not Clean

* Abort early with message:
  *“Uncommitted changes detected. Please commit or stash before using GitCherry”*

### Duplicate Commits

* Detect with `git patch-id`
* Warn + ask (default behavior)

### Unexpected Errors

* Log error + operation snapshot in `.gitcherry/errors/YYYYMMDD_HHMM.log`
* Always exit with descriptive error codes

---

## 9. Logging & Audit Trail

* Logs stored in `.gitcherry/`

  * `/logs` → JSON records of all operations
  * `/undo.json` → undo/redo queue
* Each log includes:

  ```json
  {
    "type": "transfer",
    "source": "main",
    "target": "release/1.8",
    "commits": ["a1b2c3", "b4c5d6"],
    "result": "success",
    "timestamp": "2025-10-16T21:33:00Z"
  }
  ```
* Undo: `gitcherry undo`
* Redo: `gitcherry redo`

---

## 10. Color & TUI Design

| Color     | Meaning                    |
| --------- | -------------------------- |
| 🟩 Green  | Selected commits / success |
| 🟨 Yellow | Pending / warning          |
| 🟥 Red    | Conflicts or errors        |
| 🟦 Blue   | Current branch focus       |

Monochrome fallback auto-detected for basic terminals.

---

## 11. Testing Plan

### Unit Tests

* Core logic: git command wrapper, conflict detection, patch-id comparison
* Data structures: operation serialization/deserialization
* Config parsing and defaults

### Integration Tests

* Use a temporary git repo fixture
* Test all major flows:

  * Transfer success and failure
  * Rollback
  * Restoration
  * Undo/redo
* Validate audit logs generated correctly

### TUI Tests

* Snapshot-based tests (using `expect` or `termtest`)
* Verify key commands, navigation, and prompts

### Performance Tests

* Ensure TUI rendering stays under 50ms update latency on large commit lists (≥1000 commits)

### Error Recovery Tests

* Simulate merge conflicts, abort paths, duplicate commits, and bad configs

---

## 12. Project Structure

```
/cmd/gitcherry/         → CLI entrypoint
/internal/git/          → Git command wrappers
/internal/tui/          → tview-based UI components
/internal/ops/          → transfer, revert, restore logic
/internal/config/       → config loader + defaults
/internal/logs/         → audit + undo/redo
/tests/                 → integration fixtures
/docs/                  → usage, design_spec, examples
```

---

## 13. Future Extensions (v1.1+)

* Optional non-contiguous commit support (with dependency detection)
* Multi-repo orchestration
* GitLab API integration
* Persistent session memory (last viewed branch)
* GUI wrapper (optional)

