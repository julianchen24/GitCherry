package main

import (
	"context"
	"log"
	"os"
	"os/signal"
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
	cmd := &cobra.Command{
		Use:   "gitcherry",
		Short: "Interactive helper for cherry-picking Git commits.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
			defer stop()

			cfg, err := config.Load(".")
			if err != nil {
				return err
			}

			audit := logs.NewAuditLog()
			audit.Record(logs.Entry{Summary: "session started"})

			app := tui.NewApp()
			runner := ops.NewRunner(app, cfg, audit)
			return runner.Run(ctx)
		},
	}

	cmd.SetContext(context.Background())

	return cmd
}
