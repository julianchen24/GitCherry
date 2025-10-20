package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/julianchen24/gitcherry/internal/config"
	"github.com/julianchen24/gitcherry/internal/git"
)

var (
	listBranchesFunc   = git.ListBranches
	commitsBetweenFunc = git.CommitsBetween
)

// App represents the terminal UI for GitCherry.
type App struct {
	runner *git.Runner
	config *config.Config

	ui    *tview.Application
	pages *tview.Pages
	mu    sync.RWMutex

	BranchList   *tview.List
	CommitList   *tview.List
	PreviewModal *tview.Modal
	HelpModal    *tview.Modal

	refreshBanner *tview.TextView

	branchStage  int
	branchSource string
	branchTarget string

	commits     []git.Commit
	commitStart int
	commitEnd   int

	helpVisible bool
}

// NewApp constructs a new TUI application.
func NewApp(runner *git.Runner, cfg *config.Config) *App {
	if runner == nil {
		runner = &git.Runner{}
	}
	if cfg == nil {
		cfg = config.Default()
	}

	app := &App{
		runner:      runner,
		config:      cfg,
		ui:          tview.NewApplication(),
		commitStart: -1,
		commitEnd:   -1,
	}

	app.initialiseViews()
	app.initialiseLayout()
	app.bindKeys()
	app.loadBranches()

	return app
}

// Run launches the UI and blocks until completion or context cancellation.
func (a *App) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.ui.Run()
	}()

	select {
	case <-ctx.Done():
		a.Stop()
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// Stop halts the underlying tview application.
func (a *App) Stop() {
	a.ui.Stop()
}

// ToggleHelp shows or hides the help modal.
func (a *App) ToggleHelp() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.helpVisible {
		a.pages.HidePage("help")
		a.helpVisible = false
		a.ui.SetFocus(a.BranchList)
		return
	}

	a.pages.ShowPage("help")
	a.helpVisible = true
	a.ui.SetFocus(a.HelpModal)
}

// HelpVisible reports whether the help modal is currently shown.
func (a *App) HelpVisible() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.helpVisible
}

func (a *App) initialiseViews() {
	a.BranchList = tview.NewList()
	a.BranchList.ShowSecondaryText(false)
	a.BranchList.SetTitle("Branches")
	a.BranchList.SetBorder(true)
	a.BranchList.SetSelectedBackgroundColor(tcell.ColorBlue)
	a.BranchList.SetSelectedTextColor(tcell.ColorWhite)
	a.BranchList.SetSelectedFocusOnly(true)
	a.BranchList.AddItem("(loading branches...)", "", 0, nil)
	a.BranchList.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		a.handleBranchSelection(mainText)
	})

	a.CommitList = tview.NewList()
	a.CommitList.ShowSecondaryText(true)
	a.CommitList.SetTitle("Commits")
	a.CommitList.SetBorder(true)
	a.CommitList.SetSelectedBackgroundColor(tcell.ColorGreen)
	a.CommitList.SetSelectedTextColor(tcell.ColorBlack)
	a.CommitList.AddItem("(select a branch)", "", 0, nil)
	a.CommitList.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		a.confirmCommitRange(index)
	})

	a.PreviewModal = tview.NewModal().
		SetText("Preview not available yet").
		AddButtons([]string{"Close"})
	a.PreviewModal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		a.pages.HidePage("preview")
	})

	a.HelpModal = tview.NewModal().
		SetText("GitCherry Help\n\nq: quit\n?: toggle help").
		AddButtons([]string{"Close"})
	a.HelpModal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		a.ToggleHelp()
	})
}

func (a *App) initialiseLayout() {
	left := tview.NewFlex().SetDirection(tview.FlexRow)
	if !a.config.AutoRefresh {
		a.refreshBanner = tview.NewTextView().
			SetText("Press 'r' to refresh remote refs").
			SetDynamicColors(false)
		left.AddItem(a.refreshBanner, 1, 0, false)
	}
	left.AddItem(a.BranchList, 0, 1, true)

	mainContent := tview.NewFlex().
		AddItem(left, 0, 1, true).
		AddItem(a.CommitList, 0, 2, false)

	a.pages = tview.NewPages().
		AddPage("main", mainContent, true, true).
		AddPage("preview", a.PreviewModal, true, false).
		AddPage("help", a.HelpModal, true, false)

	a.ui.SetRoot(a.pages, true)
	a.ui.SetFocus(a.BranchList)
}

func (a *App) bindKeys() {
	a.ui.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event == nil {
			return nil
		}

		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case '?':
				a.ToggleHelp()
				return nil
			case 'q', 'Q':
				a.Stop()
				return nil
			case ' ':
				if a.ui.GetFocus() == a.CommitList {
					index := a.CommitList.GetCurrentItem()
					a.markCommitStart(index)
					return nil
				}
			}
		case tcell.KeyCtrlC:
			a.Stop()
			return nil
		}

		return event
	})
}

func (a *App) loadBranches() {
	branches, err := listBranchesFunc()
	a.BranchList.Clear()
	if err != nil {
		a.BranchList.AddItem(fmt.Sprintf("Error loading branches: %v", err), "", 0, nil)
		return
	}
	if len(branches) == 0 {
		a.BranchList.AddItem("No local branches found", "", 0, nil)
		return
	}
	for _, branch := range branches {
		branch = strings.TrimSpace(branch)
		if branch == "" {
			continue
		}
		a.BranchList.AddItem(branch, "", 0, nil)
	}
	a.BranchList.SetCurrentItem(0)
}

func (a *App) handleBranchSelection(branch string) {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return
	}

	switch a.branchStage {
	case 0:
		a.branchSource = branch
		a.branchTarget = ""
		a.branchStage = 1
		a.previewCommitSelectionPrompt()
	case 1:
		a.branchTarget = branch
		a.branchStage = 2
		a.showCommitListForSource()
	default:
		a.branchSource = branch
		a.branchTarget = ""
		a.branchStage = 1
		a.previewCommitSelectionPrompt()
	}
}

func (a *App) previewCommitSelectionPrompt() {
	a.CommitList.Clear()
	message := fmt.Sprintf("Select target branch (source: %s)", a.branchSource)
	a.CommitList.AddItem(message, "", 0, nil)
	a.ui.SetFocus(a.BranchList)
}

func (a *App) showCommitListForSource() {
	a.commitStart = -1
	a.commitEnd = -1
	a.commitTargetReset()
	a.ui.SetFocus(a.CommitList)
}

func (a *App) commitTargetReset() {
	a.CommitList.Clear()
	commits, err := commitsBetweenFunc(a.branchTarget, a.branchSource)
	a.commits = commits

	if err != nil {
		a.CommitList.AddItem(fmt.Sprintf("Error loading commits: %v", err), "", 0, nil)
		return
	}

	if len(commits) == 0 {
		a.CommitList.AddItem(fmt.Sprintf("No commits to transfer from %s to %s", a.branchSource, a.branchTarget), "", 0, nil)
		return
	}

	for _, commit := range commits {
		title := commit.Message
		if strings.TrimSpace(title) == "" {
			title = commit.Hash
		}
		secondary := commit.Hash
		a.CommitList.AddItem(title, secondary, 0, nil)
	}
	a.CommitList.SetCurrentItem(0)
}

func (a *App) markCommitStart(index int) {
	if index < 0 || index >= len(a.commits) {
		return
	}
	a.commitStart = index
	a.commitEnd = index
}

func (a *App) confirmCommitRange(index int) {
	if len(a.commits) == 0 {
		return
	}
	if index < 0 || index >= len(a.commits) {
		index = a.CommitList.GetCurrentItem()
	}
	if index < 0 || index >= len(a.commits) {
		return
	}
	if a.commitStart < 0 || a.commitStart >= len(a.commits) {
		a.markCommitStart(index)
	}

	if index < a.commitStart {
		a.commitEnd = a.commitStart
		a.commitStart = index
	} else {
		a.commitEnd = index
	}

	startCommit := a.commits[a.commitStart]
	endCommit := a.commits[a.commitEnd]

	text := fmt.Sprintf("Planned range:\n%s → %s\nTotal commits: %d",
		startCommit.Hash, endCommit.Hash, a.commitEnd-a.commitStart+1)
	a.PreviewModal.SetText(text)
	a.pages.ShowPage("preview")
	a.ui.SetFocus(a.PreviewModal)
}

// SelectedRange returns the hash bounds for the current selection if available.
func (a *App) SelectedRange() (string, string, bool) {
	if a.commitStart < 0 || a.commitEnd < 0 ||
		a.commitStart >= len(a.commits) || a.commitEnd >= len(a.commits) {
		return "", "", false
	}
	return a.commits[a.commitStart].Hash, a.commits[a.commitEnd].Hash, true
}
