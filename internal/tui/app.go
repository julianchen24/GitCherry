package tui

import (
	"context"

	"github.com/rivo/tview"
)

// App defines the minimal API the ops layer needs from the TUI.
type App interface {
	Run(ctx context.Context) error
	Stop()
}

type application struct {
	ui *tview.Application
}

// NewApp creates a placeholder tview application until the real UI is ready.
func NewApp() App {
	app := tview.NewApplication()
	text := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("GitCherry TUI coming soon")
	app.SetRoot(text, true)
	return &application{ui: app}
}

// Run launches the tview application and watches for context cancellation.
func (a *application) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.ui.Run()
	}()

	select {
	case <-ctx.Done():
		a.ui.Stop()
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// Stop halts the underlying tview application.
func (a *application) Stop() {
	a.ui.Stop()
}
