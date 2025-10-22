package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/julianchen24/gitcherry/internal/config"
	"github.com/julianchen24/gitcherry/internal/git"
	"github.com/julianchen24/gitcherry/internal/logs"
	"github.com/julianchen24/gitcherry/internal/ops"
	"github.com/julianchen24/gitcherry/internal/ops/restore"
	"github.com/julianchen24/gitcherry/internal/ops/revert"
	"github.com/julianchen24/gitcherry/internal/ops/transfer"
	"github.com/julianchen24/gitcherry/internal/tui"
)

const dirtyWorktreeMessage = "Uncommitted changes detected. Please commit or stash before proceeding."

var (
	transferPlanFn             = transfer.Plan
	transferDetectDuplicatesFn = transfer.DetectDuplicates
	commitRangeFn              = collectCommitsForRange
	editMessageFn              = editMessage
	revertPlanFn               = revert.Plan
	restorePlanFn              = restore.Plan
	logsWriteOperationFn       = logs.WriteOperation
	logsPushUndoFn             = logs.PushUndo
	logsUndoFn                 = logs.Undo
	logsRedoFn                 = logs.Redo
)

func main() {
	rootCmd := newRootCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	var (
		flagRefresh     bool
		flagApply       bool
		flagNoPreview   bool
		flagTUI         bool
		flagOnDuplicate string
	)

	cmd := &cobra.Command{
		Use:   "gitcherry",
		Short: "Interactive helper for cherry-picking Git commits.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(".")
			if err != nil {
				return err
			}

			merged := *cfg
			if flagNoPreview {
				merged.Preview = false
			}
			if flagRefresh {
				merged.AutoRefresh = true
			}

			effectiveDuplicate := strings.TrimSpace(flagOnDuplicate)
			effectiveDuplicate = strings.ToLower(effectiveDuplicate)
			if effectiveDuplicate == "" {
				effectiveDuplicate = strings.ToLower(strings.TrimSpace(merged.OnDuplicate))
				if effectiveDuplicate == "" {
					effectiveDuplicate = "ask"
				}
				if !cmd.Flags().Changed("on-duplicate") && !isInteractive(os.Stdin) {
					effectiveDuplicate = "skip"
				}
			}
			switch effectiveDuplicate {
			case "ask", "skip", "apply":
			default:
				return fmt.Errorf("invalid value for --on-duplicate: %s", effectiveDuplicate)
			}
			merged.OnDuplicate = effectiveDuplicate

			clean, err := git.IsClean()
			if err != nil {
				return err
			}
			if !clean {
				return fmt.Errorf(dirtyWorktreeMessage)
			}

			if flagRefresh {
				if err := git.Fetch(true, true); err != nil {
					return err
				}
			}

			ctx := cmd.Context()
			ctx = context.WithValue(ctx, ctxConfigKey{}, &merged)
			ctx = context.WithValue(ctx, ctxApplyKey{}, flagApply)
			ctx = context.WithValue(ctx, ctxRefreshKey{}, flagRefresh)
			ctx = context.WithValue(ctx, ctxTUIKey{}, flagTUI)
			ctx = context.WithValue(ctx, ctxDuplicateKey{}, effectiveDuplicate)
			cmd.SetContext(ctx)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			cfg := configFromContext(ctx)
			if cfg == nil {
				return fmt.Errorf("configuration not initialised")
			}

			audit := logs.NewAuditLog()
			audit.Record(logs.Entry{Summary: "session started"})

			gitRunner := &git.Runner{}
			app := tui.NewApp(gitRunner, cfg, audit)
			runner := ops.NewRunner(app, cfg, audit)

			if !isApply(ctx) {
				printDryRunNotice("session")
			}

			return runner.Run(ctx)
		},
	}

	cmd.PersistentFlags().BoolVar(&flagRefresh, "refresh", false, "Fetch latest remote refs before operations")
	cmd.PersistentFlags().BoolVar(&flagApply, "apply", false, "Execute operations instead of dry-run")
	cmd.PersistentFlags().BoolVar(&flagNoPreview, "no-preview", false, "Disable preview before applying changes")
	cmd.PersistentFlags().BoolVar(&flagTUI, "tui", false, "Launch the interactive TUI")
	cmd.PersistentFlags().StringVar(&flagOnDuplicate, "on-duplicate", "", "Duplicate handling strategy: ask|skip|apply")

	cmd.AddCommand(newTransferCmd())
	cmd.AddCommand(newRevertCmd())
	cmd.AddCommand(newRestoreCmd())
	cmd.AddCommand(newUndoCmd())
	cmd.AddCommand(newRedoCmd())

	cmd.SetContext(context.Background())
	cmd.SilenceUsage = true

	return cmd
}

type ctxConfigKey struct{}
type ctxApplyKey struct{}
type ctxRefreshKey struct{}
type ctxTUIKey struct{}
type ctxDuplicateKey struct{}

func configFromContext(ctx context.Context) *config.Config {
	if ctx == nil {
		return nil
	}
	if cfg, ok := ctx.Value(ctxConfigKey{}).(*config.Config); ok {
		return cfg
	}
	return nil
}

func newTransferCmd() *cobra.Command {
	var (
		flagFrom    string
		flagTo      string
		flagRange   string
		flagMessage string
		flagEdit    bool
		flagAuto    bool
	)

	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "Transfer commits between branches",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg := configFromContext(ctx)
			if cfg == nil {
				return errors.New("configuration not available")
			}

			if flagFrom == "" || flagTo == "" || flagRange == "" {
				return errors.New("--from, --to, and --range are required")
			}

			startHash, endHash, err := parseRangeSpec(flagRange, false)
			if err != nil {
				return err
			}

			if flagMessage != "" && (flagEdit || flagAuto) {
				return errors.New("--message cannot be combined with --edit or --auto-message")
			}
			if flagEdit && flagAuto {
				return errors.New("--edit and --auto-message cannot be used together")
			}

			runner := &git.Runner{}
			commits, err := commitRangeFn(runner, startHash, endHash)
			if err != nil {
				return err
			}

			mode := duplicateMode(ctx)
			if mode == "" {
				mode = "ask"
			}
			if len(commits) > 0 {
				dups, err := transferDetectDuplicatesFn(runner, flagTo, commits)
				if err != nil {
					return err
				}
				if len(dups) > 0 {
					proceed, err := handleDuplicateChoice(cmd, mode, dups)
					if err != nil {
						return err
					}
					if !proceed {
						fmt.Fprintln(cmd.OutOrStdout(), "Skipping transfer due to duplicate patches.")
						return nil
					}
				}
			}

			rangeSpec := fmt.Sprintf("%s..%s", startHash, endHash)
			message, err := resolveTransferMessage(cmd, cfg, flagMessage, flagEdit, flagAuto, flagFrom, flagTo, rangeSpec)
			if err != nil {
				return err
			}

			commands := transferPlanFn(flagFrom, flagTo, startHash, endHash, message)
			if !isApply(ctx) {
				printPlan(cmd, commands)
				return nil
			}

			beforeHead, err := currentHead(runner, flagTo)
			if err != nil {
				return err
			}

			if err := runCommands(cmd, runner, commands); err != nil {
				return err
			}

			afterHead, err := currentHead(runner, flagTo)
			if err != nil {
				return err
			}

			op := logs.Operation{
				Source:    flagFrom,
				Target:    flagTo,
				StartHash: startHash,
				EndHash:   endHash,
				Message:   message,
				Commands:  commands,
			}
			if err := logsWriteOperationFn(op); err != nil {
				return err
			}

			undo := logs.UndoEntry{
				Source:     flagTo,
				Target:     flagTo,
				BeforeHead: beforeHead,
				AfterHead:  afterHead,
			}
			if err := logsPushUndoFn(undo); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Transfer applied successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&flagFrom, "from", "", "Source branch")
	cmd.Flags().StringVar(&flagTo, "to", "", "Target branch")
	cmd.Flags().StringVar(&flagRange, "range", "", "Commit range (a..b)")
	cmd.Flags().StringVar(&flagMessage, "message", "", "Commit message to use")
	cmd.Flags().BoolVar(&flagEdit, "edit", false, "Edit commit message before applying")
	cmd.Flags().BoolVar(&flagAuto, "auto-message", false, "Generate commit message from template")
	cmd.MarkFlagsMutuallyExclusive("message", "edit")
	cmd.MarkFlagsMutuallyExclusive("message", "auto-message")
	cmd.MarkFlagsMutuallyExclusive("edit", "auto-message")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")
	_ = cmd.MarkFlagRequired("range")
	cmd.SilenceUsage = true
	return cmd
}

func newRevertCmd() *cobra.Command {
	var (
		flagOn      string
		flagRange   string
		flagMessage string
	)

	cmd := &cobra.Command{
		Use:   "revert",
		Short: "Revert commits on a branch",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if flagOn == "" {
				return errors.New("--on is required")
			}
			if flagRange == "" {
				return errors.New("--range is required")
			}

			startHash, endHash, err := parseRangeSpec(flagRange, true)
			if err != nil {
				return err
			}

			message := flagMessage
			if message == "" {
				message = fmt.Sprintf("Revert %s on %s", flagRange, flagOn)
			}

			commands := revertPlanFn(flagOn, flagOn, startHash, endHash, message)
			if !isApply(ctx) {
				printPlan(cmd, commands)
				return nil
			}

			runner := &git.Runner{}
			beforeHead, err := currentHead(runner, flagOn)
			if err != nil {
				return err
			}

			if err := revert.Execute(ctx, runner, flagOn, startHash, endHash, message); err != nil {
				return err
			}

			afterHead, err := currentHead(runner, flagOn)
			if err != nil {
				return err
			}

			op := logs.Operation{
				Source:    flagOn,
				Target:    flagOn,
				StartHash: startHash,
				EndHash:   endHash,
				Message:   message,
				Commands:  commands,
			}
			if err := logsWriteOperationFn(op); err != nil {
				return err
			}

			undo := logs.UndoEntry{
				Source:     flagOn,
				Target:     flagOn,
				BeforeHead: beforeHead,
				AfterHead:  afterHead,
			}
			if err := logsPushUndoFn(undo); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Revert applied successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&flagOn, "on", "", "Branch to revert on")
	cmd.Flags().StringVar(&flagRange, "range", "", "Commit or range to revert (a or a..b)")
	cmd.Flags().StringVar(&flagMessage, "message", "", "Commit message for the revert")
	_ = cmd.MarkFlagRequired("on")
	_ = cmd.MarkFlagRequired("range")
	cmd.SilenceUsage = true
	return cmd
}

func newRestoreCmd() *cobra.Command {
	var (
		flagCommit string
		flagBranch string
	)

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Create a branch at a previous commit",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagCommit == "" || flagBranch == "" {
				return errors.New("--at and --branch-name are required")
			}

			commands := restorePlanFn(flagBranch, flagCommit)
			if !isApply(cmd.Context()) {
				printPlan(cmd, commands)
				return nil
			}

			audit := logs.NewAuditLog()
			if err := restore.Execute(cmd.Context(), &git.Runner{}, flagBranch, flagCommit, audit); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Restore completed successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&flagCommit, "at", "", "Commit to restore")
	cmd.Flags().StringVar(&flagBranch, "branch-name", "", "Branch name to create")
	_ = cmd.MarkFlagRequired("at")
	_ = cmd.MarkFlagRequired("branch-name")
	cmd.SilenceUsage = true
	return cmd
}

func newUndoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "undo",
		Short: "Undo the last GitCherry operation",
		RunE: func(cmd *cobra.Command, args []string) error {
			entry, ok, err := logsUndoFn()
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(cmd.OutOrStdout(), "No undo information available.")
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Undo entry: branch=%s before=%s after=%s\n", entry.Source, entry.BeforeHead, entry.AfterHead)
			fmt.Fprintln(cmd.OutOrStdout(), "Please manually reset your repository as needed (e.g., git reset --hard).")
			return nil
		},
	}
	cmd.SilenceUsage = true
	return cmd
}

func newRedoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "redo",
		Short: "Redo the last undone GitCherry operation",
		RunE: func(cmd *cobra.Command, args []string) error {
			entry, ok, err := logsRedoFn()
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(cmd.OutOrStdout(), "No redo information available.")
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Redo entry: branch=%s before=%s after=%s\n", entry.Source, entry.BeforeHead, entry.AfterHead)
			fmt.Fprintln(cmd.OutOrStdout(), "Please manually adjust your repository as needed (e.g., git reset --hard).")
			return nil
		},
	}
	cmd.SilenceUsage = true
	return cmd
}

func isApply(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	if apply, ok := ctx.Value(ctxApplyKey{}).(bool); ok {
		return apply
	}
	return false
}

func isRefresh(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	if refresh, ok := ctx.Value(ctxRefreshKey{}).(bool); ok {
		return refresh
	}
	return false
}

func isTUI(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	if tui, ok := ctx.Value(ctxTUIKey{}).(bool); ok {
		return tui
	}
	return false
}

func duplicateMode(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if mode, ok := ctx.Value(ctxDuplicateKey{}).(string); ok {
		return mode
	}
	return ""
}

func isInteractive(file *os.File) bool {
	if file == nil {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func printDryRunNotice(subject string) {
	fmt.Printf("[dry-run] %s; use --apply to execute\n", subject)
}

func printPlan(cmd *cobra.Command, commands []string) {
	out := cmd.OutOrStdout()
	if len(commands) == 0 {
		fmt.Fprintln(out, "No commands to execute.")
		return
	}
	fmt.Fprintln(out, "Planned commands:")
	for _, c := range commands {
		fmt.Fprintf(out, "  %s\n", c)
	}
}

func resolveTransferMessage(cmd *cobra.Command, cfg *config.Config, explicit string, edit bool, auto bool, from, to, rangeSpec string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}

	initial := cfg.MessageTemplate
	if initial == "" {
		initial = "[Transfer] {source} -> {target} {range}"
	}
	message := renderTemplate(initial, from, to, rangeSpec)

	if auto {
		return message, nil
	}
	if edit {
		return editMessageFn(message)
	}
	return message, nil
}

func renderTemplate(template, source, target, rangeSpec string) string {
	replacements := map[string]string{
		"{source}": source,
		"{target}": target,
		"{range}":  rangeSpec,
	}
	result := template
	for key, value := range replacements {
		result = strings.ReplaceAll(result, key, value)
	}
	return result
}

func editMessage(initial string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	file, err := os.CreateTemp("", "gitcherry-msg-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(file.Name())

	if _, err := file.WriteString(initial + "\n"); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}

	cmd := exec.Command(editor, file.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}

	data, err := os.ReadFile(file.Name())
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func parseRangeSpec(spec string, allowSingle bool) (string, string, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", "", errors.New("range must be provided")
	}

	if strings.Contains(spec, "..") {
		parts := strings.SplitN(spec, "..", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("invalid range: %s", spec)
		}
		return parts[0], parts[1], nil
	}

	if !allowSingle {
		return "", "", fmt.Errorf("range must include '..': %s", spec)
	}

	return spec, spec, nil
}

func runCommands(cmd *cobra.Command, runner *git.Runner, commands []string) error {
	for _, command := range commands {
		args, err := splitCommand(command)
		if err != nil {
			return err
		}
		if len(args) == 0 {
			continue
		}
		if args[0] != "git" {
			return fmt.Errorf("unsupported command: %s", command)
		}

		stdout, stderr, err := runner.Run(args[1:]...)
		if err != nil {
			return fmt.Errorf("%s failed: %v (%s)", command, err, strings.TrimSpace(stderr))
		}
		if out := strings.TrimSpace(stdout); out != "" {
			fmt.Fprintln(cmd.OutOrStdout(), out)
		}
	}
	return nil
}

func splitCommand(command string) ([]string, error) {
	var (
		args     []string
		current  strings.Builder
		inQuotes bool
	)

	runes := []rune(command)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch r {
		case '"':
			inQuotes = !inQuotes
		case ' ':
			if !inQuotes {
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
				continue
			}
			current.WriteRune(r)
		case '\\':
			if inQuotes && i+1 < len(runes) {
				i++
				current.WriteRune(runes[i])
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}

	if inQuotes {
		return nil, errors.New("unterminated quote in command")
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args, nil
}

func collectCommitsForRange(runner *git.Runner, start, end string) ([]git.Commit, error) {
	if runner == nil {
		runner = &git.Runner{}
	}

	specs := []string{fmt.Sprintf("%s^..%s", start, end), fmt.Sprintf("%s..%s", start, end)}
	var hashes []string
	for idx, spec := range specs {
		out, stderr, err := runner.Run("rev-list", "--reverse", spec)
		if err != nil {
			if idx == len(specs)-1 {
				return nil, fmt.Errorf("git rev-list %s failed: %v (%s)", spec, err, strings.TrimSpace(stderr))
			}
			continue
		}

		lines := strings.Fields(strings.TrimSpace(out))
		if len(lines) == 0 && idx == 0 {
			continue
		}
		hashes = lines
		if idx == 1 && (len(hashes) == 0 || hashes[0] != start) {
			hashes = append([]string{start}, hashes...)
		}
		break
	}

	if len(hashes) == 0 {
		hashes = []string{start}
		if start != end {
			hashes = append(hashes, end)
		}
	}

	commits := make([]git.Commit, 0, len(hashes))
	for _, hash := range hashes {
		commits = append(commits, git.Commit{Hash: hash})
	}
	return commits, nil
}

func handleDuplicateChoice(cmd *cobra.Command, mode string, duplicates []git.Commit) (bool, error) {
	switch mode {
	case "skip":
		fmt.Fprintf(cmd.OutOrStdout(), "Detected %d duplicate patches; skipping.\n", len(duplicates))
		return false, nil
	case "apply":
		return true, nil
	case "ask":
		if !isInteractive(os.Stdin) {
			fmt.Fprintln(cmd.OutOrStdout(), "Detected duplicate patches but cannot prompt; skipping.")
			return false, nil
		}
		example := shortHash(duplicates[0].Hash)
		return promptYesNo(fmt.Sprintf("Detected %d duplicate patches already on target (e.g., %s). Apply anyway? [y/N]: ", len(duplicates), example))
	default:
		return false, fmt.Errorf("unknown duplicate mode: %s", mode)
	}
}

func promptYesNo(prompt string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprint(os.Stdout, prompt)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func shortHash(hash string) string {
	if len(hash) <= 6 {
		return hash
	}
	return hash[:6]
}

func currentHead(runner *git.Runner, ref string) (string, error) {
	stdout, stderr, err := runner.Run("rev-parse", ref)
	if err != nil {
		return "", fmt.Errorf("git rev-parse %s failed: %v (%s)", ref, err, strings.TrimSpace(stderr))
	}
	return strings.TrimSpace(stdout), nil
}
