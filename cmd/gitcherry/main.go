package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/julianchen24/gitcherry/internal/config"
	"github.com/julianchen24/gitcherry/internal/git"
	"github.com/julianchen24/gitcherry/internal/logs"
	"github.com/julianchen24/gitcherry/internal/ops"
	"github.com/julianchen24/gitcherry/internal/tui"
)

const dirtyWorktreeMessage = "Uncommitted changes detected. Please commit or stash before proceeding."

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
	return newStubCommand("transfer", "Transfer commits between branches")
}

func newRevertCmd() *cobra.Command {
	return newStubCommand("revert", "Revert previously transferred commits")
}

func newRestoreCmd() *cobra.Command {
	return newStubCommand("restore", "Restore workspace state to a previous snapshot")
}

func newUndoCmd() *cobra.Command {
	return newStubCommand("undo", "Undo the last GitCherry operation")
}

func newRedoCmd() *cobra.Command {
	return newStubCommand("redo", "Redo the last undone GitCherry operation")
}

func newStubCommand(name, description string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name,
		Short: description,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := configFromContext(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("configuration not available")
			}

			apply := isApply(cmd.Context())
			refresh := isRefresh(cmd.Context())

			if !apply {
				printDryRunNotice(name)
				fmt.Printf("[dry-run] %s planned with preview=%v autoRefresh=%v args=%s\n",
					name, cfg.Preview, cfg.AutoRefresh, strings.Join(args, " "))
				return nil
			}

			fmt.Printf("%s execution (refresh=%v) not yet implemented\n", name, refresh)
			return nil
		},
	}
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
