package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/julianchen24/gitcherry/internal/config"
	"github.com/julianchen24/gitcherry/internal/logs"
	"github.com/julianchen24/gitcherry/internal/ops"
	"github.com/julianchen24/gitcherry/internal/tui"
)

func main() {
	rootCmd := newRootCommand()
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func newRootCommand() *cobra.Command {
	var (
		flagRefresh   bool
		flagApply     bool
		flagNoPreview bool
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

			ctx := context.WithValue(cmd.Context(), ctxConfigKey{}, &merged)
			ctx = context.WithValue(ctx, ctxApplyKey{}, flagApply)
			ctx = context.WithValue(ctx, ctxRefreshKey{}, flagRefresh)
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

			app := tui.NewApp()
			runner := ops.NewRunner(app, cfg, audit)

			if !flagApply {
				printDryRunNotice("session")
			}

			return runner.Run(ctx)
		},
	}

	cmd.PersistentFlags().BoolVar(&flagRefresh, "refresh", false, "Fetch latest remote refs before operations")
	cmd.PersistentFlags().BoolVar(&flagApply, "apply", false, "Execute operations instead of dry-run")
	cmd.PersistentFlags().BoolVar(&flagNoPreview, "no-preview", false, "Disable preview before applying changes")

	cmd.AddCommand(newTransferCmd())
	cmd.AddCommand(newRevertCmd())
	cmd.AddCommand(newRestoreCmd())
	cmd.AddCommand(newUndoCmd())
	cmd.AddCommand(newRedoCmd())

	cmd.SetContext(context.Background())

	return cmd
}

type ctxConfigKey struct{}
type ctxApplyKey struct{}
type ctxRefreshKey struct{}

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

func printDryRunNotice(subject string) {
	fmt.Printf("[dry-run] %s; use --apply to execute\n", subject)
}
