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
	"github.com/julianchen24/gitcherry/internal/logs"
	"github.com/julianchen24/gitcherry/internal/ops/restore"
)

var (
	listBranchesFunc   = git.ListBranches
	commitsBetweenFunc = git.CommitsBetween
)

// App represents the terminal UI for GitCherry.
type App struct {
	runner *git.Runner
	config *config.Config
	audit  *logs.AuditLog

	ui    *tview.Application
	pages *tview.Pages
	mu    sync.RWMutex

	BranchList *tview.List
	CommitList *tview.List
	HelpModal  *tview.Modal

	previewFrame   *tview.Frame
	previewTable   *tview.Table
	previewInfo    *tview.TextView
	previewEditor  *tview.TextArea
	previewActions *tview.List
	previewVisible bool

	restoreForm        *tview.Form
	restoreVisible     bool
	restoreCommitIndex int

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
func NewApp(runner *git.Runner, cfg *config.Config, audit *logs.AuditLog) *App {
	if runner == nil {
		runner = &git.Runner{}
	}
	if cfg == nil {
		cfg = config.Default()
	}

	app := &App{
		runner:             runner,
		config:             cfg,
		audit:              audit,
		ui:                 tview.NewApplication(),
		commitStart:        -1,
		commitEnd:          -1,
		restoreCommitIndex: -1,
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

	a.HelpModal = tview.NewModal().
		SetText("GitCherry Help\n\nq: quit\n?: toggle help").
		AddButtons([]string{"Close"})
	a.HelpModal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		a.ToggleHelp()
	})

	a.restoreForm = tview.NewForm().
		AddInputField("Branch name", "", 40, nil, nil).
		AddButton("Create", func() {
			a.submitRestore()
		}).
		AddButton("Cancel", func() {
			a.hideRestore()
		})
	a.restoreForm.SetBorder(true)
	a.restoreForm.SetTitle("Restore Branch")

	a.previewTable = tview.NewTable()
	a.previewTable.SetBorder(true)
	a.previewTable.SetTitle("Selected Commits")
	a.previewTable.SetSelectable(false, false)
	a.previewTable.SetFixed(1, 0)

	a.previewInfo = tview.NewTextView()
	a.previewInfo.SetDynamicColors(false)
	a.previewInfo.SetBorder(true)
	a.previewInfo.SetTitle("Summary")

	a.previewEditor = tview.NewTextArea()
	a.previewEditor.SetBorder(true)
	a.previewEditor.SetTitle("Commit Message")

	a.previewActions = tview.NewList().ShowSecondaryText(false)
	a.previewActions.SetBorder(true)
	a.previewActions.SetTitle("Actions")
	a.previewActions.AddItem("[E] Edit message", "", 'e', func() {
		a.editPreviewMessage()
	})
	a.previewActions.AddItem("[A] Use suggested message", "", 'a', func() {
		a.applySuggestedMessage()
	})

	body := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.previewInfo, 3, 0, false).
		AddItem(a.previewTable, 0, 3, false).
		AddItem(a.previewEditor, 0, 4, true).
		AddItem(a.previewActions, 7, 0, false)

	a.previewFrame = tview.NewFrame(body)
	a.previewFrame.SetBorder(true)
	a.previewFrame.SetTitle("Preview")
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
		AddPage("preview", a.previewFrame, true, false).
		AddPage("help", a.HelpModal, true, false).
		AddPage("restore", a.restoreForm, true, false)

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
			case 'b', 'B':
				if a.ui.GetFocus() == a.CommitList {
					index := a.CommitList.GetCurrentItem()
					a.openRestoreModal(index)
					return nil
				}
			}
		case tcell.KeyEscape:
			if a.previewVisible {
				a.hidePreview()
				return nil
			}
			if a.restoreVisible {
				a.hideRestore()
				return nil
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

	a.populatePreviewTable(a.commitStart, a.commitEnd)
	a.previewInfo.SetText(fmt.Sprintf("Target: %s\n→ Will become 1 new commit", a.branchTarget))

	suggested := a.renderSuggestedMessage(startCommit, endCommit)
	a.previewEditor.SetText(suggested, true)

	a.previewVisible = true
	a.pages.ShowPage("preview")
	a.ui.SetFocus(a.previewActions)
}

// SelectedRange returns the hash bounds for the current selection if available.
func (a *App) SelectedRange() (string, string, bool) {
	if a.commitStart < 0 || a.commitEnd < 0 ||
		a.commitStart >= len(a.commits) || a.commitEnd >= len(a.commits) {
		return "", "", false
	}
	return a.commits[a.commitStart].Hash, a.commits[a.commitEnd].Hash, true
}

func (a *App) populatePreviewTable(start, end int) {
	a.previewTable.Clear()
	a.previewTable.SetCell(0, 0, tview.NewTableCell("Hash").SetAttributes(tcell.AttrBold))
	a.previewTable.SetCell(0, 1, tview.NewTableCell("Author").SetAttributes(tcell.AttrBold))
	a.previewTable.SetCell(0, 2, tview.NewTableCell("Subject").SetAttributes(tcell.AttrBold))

	row := 1
	for i := start; i <= end && i < len(a.commits); i++ {
		commit := a.commits[i]
		hash := commit.Hash
		if len(hash) > 7 {
			hash = hash[:7]
		}
		a.previewTable.SetCell(row, 0, tview.NewTableCell(hash))
		a.previewTable.SetCell(row, 1, tview.NewTableCell(commit.Author))
		a.previewTable.SetCell(row, 2, tview.NewTableCell(commit.Message))
		row++
	}
}

func (a *App) renderSuggestedMessage(start, end git.Commit) string {
	if a.config == nil {
		return ""
	}
	rangeSpec := fmt.Sprintf("%s..%s", start.Hash, end.Hash)
	message := a.config.MessageTemplate
	replacements := map[string]string{
		"{source}": a.branchSource,
		"{target}": a.branchTarget,
		"{range}":  rangeSpec,
	}
	for key, val := range replacements {
		message = strings.ReplaceAll(message, key, val)
	}
	return message
}

func (a *App) applySuggestedMessage() {
	if a.commitStart < 0 || a.commitEnd < 0 || a.commitStart >= len(a.commits) || a.commitEnd >= len(a.commits) {
		return
	}
	msg := a.renderSuggestedMessage(a.commits[a.commitStart], a.commits[a.commitEnd])
	a.previewEditor.SetText(msg, true)
	a.ui.SetFocus(a.previewEditor)
}

func (a *App) editPreviewMessage() {
	a.ui.SetFocus(a.previewEditor)
}

func (a *App) hidePreview() {
	a.previewVisible = false
	a.pages.HidePage("preview")
	a.ui.SetFocus(a.CommitList)
}

func (a *App) openRestoreModal(index int) {
	if index < 0 || index >= len(a.commits) {
		return
	}
	a.restoreCommitIndex = index
	defaultName := fmt.Sprintf("%s-backup", a.branchSource)
	if input := a.restoreInput(); input != nil {
		input.SetText(defaultName)
	}
	a.restoreForm.SetTitle("Restore Branch")
	a.restoreVisible = true
	a.pages.ShowPage("restore")
	a.ui.SetFocus(a.restoreForm)
}

func (a *App) submitRestore() {
	input := a.restoreInput()
	if input == nil {
		return
	}
	value := strings.TrimSpace(input.GetText())
	if value == "" {
		a.restoreForm.SetTitle("Restore Branch (name required)")
		return
	}
	a.executeRestore(value)
}

func (a *App) hideRestore() {
	a.restoreVisible = false
	a.pages.HidePage("restore")
	a.restoreForm.SetTitle("Restore Branch")
	a.ui.SetFocus(a.CommitList)
}

func (a *App) restoreInput() *tview.InputField {
	if a.restoreForm == nil || a.restoreForm.GetFormItemCount() == 0 {
		return nil
	}
	if input, ok := a.restoreForm.GetFormItem(0).(*tview.InputField); ok {
		return input
	}
	return nil
}

func (a *App) executeRestore(branchName string) {
	if a.restoreCommitIndex < 0 || a.restoreCommitIndex >= len(a.commits) {
		a.restoreForm.SetTitle("Restore Branch (select a commit)")
		return
	}
	commit := a.commits[a.restoreCommitIndex].Hash
	if err := restore.Execute(context.Background(), a.runner, branchName, commit, a.audit); err != nil {
		a.restoreForm.SetTitle(fmt.Sprintf("Restore Branch (error: %v)", err))
		return
	}
	a.restoreForm.SetTitle("Restore Branch (created)")
	a.hideRestore()
	a.loadBranches()
}
