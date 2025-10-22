package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rivo/tview"
	"github.com/stretchr/testify/require"

	"github.com/julianchen24/gitcherry/internal/config"
	"github.com/julianchen24/gitcherry/internal/git"
	"github.com/julianchen24/gitcherry/internal/logs"
)

func stubColorSupport(t *testing.T, enabled bool) {
	original := colorSupportFn
	colorSupportFn = func() bool { return enabled }
	t.Cleanup(func() { colorSupportFn = original })
}

func withStubBranches(t *testing.T, branches []string, err error) {
	original := listBranchesFunc
	listBranchesFunc = func() ([]string, error) {
		return branches, err
	}
	t.Cleanup(func() {
		listBranchesFunc = original
	})
}

func withStubCommits(t *testing.T, commits []git.Commit, err error) {
	original := commitsBetweenFunc
	commitsBetweenFunc = func(base, head string) ([]git.Commit, error) {
		return commits, err
	}
	t.Cleanup(func() {
		commitsBetweenFunc = original
	})
}

func TestNewAppInitializes(t *testing.T) {
	withStubBranches(t, []string{"main"}, nil)
	withStubCommits(t, nil, nil)

	cfg := config.Default()
	cfg.AutoRefresh = false

	stubColorSupport(t, true)
	app := NewApp(nil, cfg, logs.NewAuditLog())
	app.fetchFn = func() error { return nil }
	app.duplicateFn = func(string, []git.Commit) ([]git.Commit, error) { return nil, nil }
	require.NotNil(t, app)
	require.NotNil(t, app.BranchList)
	require.NotNil(t, app.CommitList)
	require.NotNil(t, app.previewFrame)
	require.NotNil(t, app.previewEditor)
	require.NotNil(t, app.previewActions)
	require.NotNil(t, app.HelpModal)
	require.NotNil(t, app.refreshBanner)
	require.False(t, app.HelpVisible())
	require.Equal(t, 1, app.BranchList.GetItemCount())
}

func TestToggleHelp(t *testing.T) {
	withStubBranches(t, []string{"main"}, nil)
	withStubCommits(t, nil, nil)

	stubColorSupport(t, true)
	app := NewApp(nil, config.Default(), logs.NewAuditLog())
	app.fetchFn = func() error { return nil }
	app.duplicateFn = func(string, []git.Commit) ([]git.Commit, error) { return nil, nil }
	require.False(t, app.HelpVisible())

	app.ToggleHelp()
	require.True(t, app.HelpVisible())

	app.ToggleHelp()
	require.False(t, app.HelpVisible())
}

func TestBranchSelectionFlowAndCommitRange(t *testing.T) {
	withStubBranches(t, []string{"main", "feature"}, nil)
	commits := []git.Commit{
		{Hash: "c1", Message: "First"},
		{Hash: "c2", Message: "Second"},
		{Hash: "c3", Message: "Third"},
	}
	withStubCommits(t, commits, nil)

	cfg := config.Default()
	stubColorSupport(t, true)
	app := NewApp(nil, cfg, logs.NewAuditLog())
	app.fetchFn = func() error { return nil }
	app.duplicateFn = func(string, []git.Commit) ([]git.Commit, error) { return nil, nil }
	require.Equal(t, 2, app.BranchList.GetItemCount())
	require.Equal(t, 0, app.branchStage)

	app.handleBranchSelection("main")
	require.Equal(t, "main", app.branchSource)
	require.Equal(t, 1, app.branchStage)
	msg, _ := app.CommitList.GetItemText(0)
	require.Contains(t, msg, "Select target branch")

	app.handleBranchSelection("feature")
	require.Equal(t, "feature", app.branchTarget)
	require.Equal(t, 2, app.branchStage)
	require.Equal(t, len(commits), app.CommitList.GetItemCount())

	app.markCommitStart(0)
	app.confirmCommitRange(2)

	start, end, ok := app.SelectedRange()
	require.True(t, ok)
	require.Equal(t, "c1", start)
	require.Equal(t, "c3", end)
	require.True(t, app.previewVisible)
	info := app.previewInfo.GetText(false)
	require.Contains(t, info, "feature")
	require.Equal(t, len(commits)+1, app.previewTable.GetRowCount())
	expected := strings.ReplaceAll(cfg.MessageTemplate, "{source}", "main")
	expected = strings.ReplaceAll(expected, "{target}", "feature")
	expected = strings.ReplaceAll(expected, "{range}", "c1..c3")
	require.Equal(t, expected, app.previewEditor.GetText())
}

func TestPreviewTemplateActions(t *testing.T) {
	withStubBranches(t, []string{"main", "feature"}, nil)
	commits := []git.Commit{
		{Hash: "c1", Message: "First"},
		{Hash: "c2", Message: "Second"},
	}
	withStubCommits(t, commits, nil)

	cfg := config.Default()
	cfg.MessageTemplate = "Transfer {source}->{target} range {range}"
	stubColorSupport(t, true)
	app := NewApp(nil, cfg, logs.NewAuditLog())
	app.fetchFn = func() error { return nil }
	app.duplicateFn = func(string, []git.Commit) ([]git.Commit, error) { return nil, nil }
	app.handleBranchSelection("main")
	app.handleBranchSelection("feature")
	app.markCommitStart(0)
	app.confirmCommitRange(1)

	app.previewEditor.SetText("custom message", true)
	app.applySuggestedMessage()
	expected := "Transfer main->feature range c1..c2"
	require.Equal(t, expected, app.previewEditor.GetText())

	app.previewEditor.SetText("edited", true)
	app.editPreviewMessage()
	require.Equal(t, "edited", app.previewEditor.GetText())
}

func TestManualRefreshInvokesFetch(t *testing.T) {
	withStubBranches(t, []string{"main"}, nil)
	withStubCommits(t, nil, nil)

	stubColorSupport(t, true)
	app := NewApp(nil, config.Default(), logs.NewAuditLog())
	count := 0
	app.fetchFn = func() error {
		count++
		return nil
	}
	app.loadBranchesWithFetch(true)
	require.Equal(t, 1, count)
}

func TestDuplicatePromptShown(t *testing.T) {
	withStubBranches(t, []string{"main", "feature"}, nil)
	commits := []git.Commit{
		{Hash: "c1", Message: "First"},
		{Hash: "c2", Message: "Second"},
	}
	withStubCommits(t, commits, nil)

	stubColorSupport(t, true)
	app := NewApp(nil, config.Default(), logs.NewAuditLog())
	app.fetchFn = func() error { return nil }
	app.duplicateFn = func(target string, selection []git.Commit) ([]git.Commit, error) {
		return selection[:1], nil
	}

	app.handleBranchSelection("main")
	app.handleBranchSelection("feature")
	app.markCommitStart(0)
	app.confirmCommitRange(1)
	require.True(t, app.duplicateVisible)
	require.False(t, app.previewVisible)
	app.hideDuplicatePrompt()
}

func TestDetectColorSupportRespectsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("TERM", "xterm")
	require.False(t, detectColorSupport())
}

func TestDetectColorSupportHandlesDumbTerm(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	require.False(t, detectColorSupport())
}

func TestDetectColorSupportDefaultsToColor(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")
	require.True(t, detectColorSupport())
}

func TestAppSnapshots(t *testing.T) {
	withStubBranches(t, []string{"main", "feature", "bugfix"}, nil)
	commits := []git.Commit{
		{Hash: "c1ffee0", Author: "Alice", Message: "Fix lint warnings"},
		{Hash: "deadbeef", Author: "Bob", Message: "Improve logging"},
		{Hash: "faceb00c", Author: "Carol", Message: "Add telemetry hooks"},
	}
	withStubCommits(t, commits, nil)
	stubColorSupport(t, false)

	app := NewApp(nil, config.Default(), logs.NewAuditLog())
	app.fetchFn = func() error { return nil }
	app.duplicateFn = func(string, []git.Commit) ([]git.Commit, error) { return nil, nil }

	app.handleBranchSelection("main")
	app.handleBranchSelection("feature")
	app.markCommitStart(0)
	app.confirmCommitRange(1)

	snapshot := snapshotAppViews(t, app)
	assertGolden(t, "tui_main_views.golden", snapshot)
}

func snapshotAppViews(t *testing.T, app *App) string {
	var sb strings.Builder
	sb.WriteString("Branches:\n")
	sb.WriteString(renderList(app.BranchList))
	sb.WriteString("\nCommits:\n")
	sb.WriteString(renderList(app.CommitList))
	sb.WriteString("\nPreview Info:\n")
	info := app.previewInfo.GetText(false)
	sb.WriteString(info)
	sb.WriteString("\n\nPreview Table:\n")
	sb.WriteString(renderTable(app.previewTable))
	sb.WriteString("\nPreview Message:\n")
	sb.WriteString(app.previewEditor.GetText())
	return sb.String()
}

func renderList(list *tview.List) string {
	var sb strings.Builder
	current := list.GetCurrentItem()
	for i := 0; i < list.GetItemCount(); i++ {
		main, secondary := list.GetItemText(i)
		prefix := " "
		if i == current {
			prefix = ">"
		}
		sb.WriteString(prefix)
		sb.WriteString(" ")
		sb.WriteString(main)
		if secondary != "" {
			sb.WriteString(" | ")
			sb.WriteString(secondary)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func renderTable(table *tview.Table) string {
	var sb strings.Builder
	rows := table.GetRowCount()
	cols := table.GetColumnCount()
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			cell := table.GetCell(r, c)
			if cell == nil {
				continue
			}
			if c > 0 {
				sb.WriteString(" | ")
			}
			sb.WriteString(cell.Text)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func assertGolden(t *testing.T, name, actual string) {
	t.Helper()
	path := filepath.Join("..", "..", "tests", "golden", name)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(actual), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	require.Equal(t, string(data), actual)
}
