# GitCherry Usage Guide

## Quickstart

1. Ensure you have a clean git working tree. GitCherry refuses to operate when unstaged changes are present.
2. Optionally fetch the latest refs before starting: `git fetch --prune --tags`.
3. Launch the TUI with `gitcherry --tui`, or use the CLI subcommands described below.
4. For dry-runs, omit `--apply`; GitCherry will print the planned git commands instead of executing them.

## TUI Walkthrough

1. **Branch Selection**
   - The left panel lists local branches. Use the arrow keys to choose a source branch; press `Enter` to mark it.
   - Select a second branch to designate it as the target. GitCherry will automatically load commits that are on the source but not on the target.
   - Press `r` at any time to fetch remote updates (`git fetch --prune --tags`) and refresh the lists.

2. **Commit List**
   - Navigate the commit list with the arrow keys.
   - Press `Space` to mark the start of the range. Move to the desired end commit and press `Enter`.
   - GitCherry checks for duplicate patches on the target branch. If duplicates are detected you can skip, proceed, or (in the TUI) answer the prompt.
   - Press `b` to open the restore modal and create a branch from the currently highlighted commit.

3. **Preview**
   - The preview screen summarises the selected commits, displays the suggested commit message, and shows the target branch.
   - Use `[A] Use suggested message` to reapply the template, or `[E] Edit` to open the message for editing.
   - Press `Esc` to return to the commit list without applying changes.

4. **Apply**
   - To execute the transfer, run GitCherry with `--apply` (either via CLI or when launching the TUI).
   - Without `--apply`, GitCherry remains in dry-run mode and simply shows the planned commands.
   - After successful execution, GitCherry records the operation in `.gitcherry/logs/` and stores undo metadata.

Keybindings:

| Key | Action |
| --- | --- |
| `?` | Toggle help modal |
| `q` | Quit |
| `r` | Fetch remotes and refresh |
| `Space` | Mark start commit |
| `Enter` | Confirm commit range |
| `b` | Restore branch at highlighted commit |
| `Esc` | Close modals / preview |

## CLI Examples

### Transfer commits

Dry run (show planned commands):

```bash
gitcherry transfer \
  --from main \
  --to release \
  --range a1b2c3..d4e5f6 \
  --message "Cherry-pick hotfix to release"
```

Apply changes:

```bash
gitcherry transfer \
  --from main \
  --to release \
  --range a1b2c3..d4e5f6 \
  --auto-message \
  --apply
```

Use `--edit` to open your `$EDITOR` and adjust the message before applying.

### Revert commits

Dry run:

```bash
gitcherry revert \
  --on main \
  --range a1b2c3..d4e5f6
```

Apply:

```bash
gitcherry revert \
  --on main \
  --range a1b2c3..d4e5f6 \
  --message "Revert hotfix range" \
  --apply
```

Use `--range <hash>` for single-commit reverts.

### Restore a branch

Dry run:

```bash
gitcherry restore \
  --at abcdef1 \
  --branch-name backup-main
```

Apply:

```bash
gitcherry restore \
  --at abcdef1 \
  --branch-name backup-main \
  --apply
```

### Undo and Redo

List the latest undo entry (dry-run by design):

```bash
gitcherry undo
```

Redo the last undone operation:

```bash
gitcherry redo
```

> The undo/redo commands print the stored head hashes so you can perform the appropriate git resets yourself.

## Handling Conflicts

- If `git cherry-pick` or `git revert` encounters conflicts during `--apply`, GitCherry surfaces the failure and indicates the git command that stopped.
- Resolve the conflicts manually, then run:
  - `git cherry-pick --continue` (for transfers)
  - `git revert --continue` (for reverts)
- If you wish to abandon the operation, use:
  - `git cherry-pick --abort`
  - `git revert --abort`
- After completing or aborting, you can re-run GitCherry to continue with other tasks. If an operation partially succeeded, consider using `gitcherry undo` (which prints the before/after heads) to guide any additional cleanup.
