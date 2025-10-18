package ops

import (
	"context"

	"github.com/julianchen24/gitcherry/internal/config"
	"github.com/julianchen24/gitcherry/internal/logs"
	"github.com/julianchen24/gitcherry/internal/tui"
)

// Runner wires together the configuration, audit trail, and user interface.
type Runner struct {
	app   tui.App
	cfg   *config.Config
	audit *logs.AuditLog
}

// NewRunner constructs a Runner using the provided collaborators.
func NewRunner(app tui.App, cfg *config.Config, audit *logs.AuditLog) *Runner {
	if cfg == nil {
		cfg = config.Default()
	}
	return &Runner{
		app:   app,
		cfg:   cfg,
		audit: audit,
	}
}

// Run boots the terminal UI. Additional orchestration will be added later.
func (r *Runner) Run(ctx context.Context) error {
	if r.audit != nil {
		r.audit.Record(logs.Entry{Summary: "runner started"})
	}

	if r.app == nil {
		return nil
	}
	return r.app.Run(ctx)
}
