package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/julianchen24/gitcherry/internal/config"
	"github.com/julianchen24/gitcherry/internal/git"
	"github.com/julianchen24/gitcherry/internal/logs"
)

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

	app := NewApp(nil, cfg, logs.NewAuditLog())
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

	app := NewApp(nil, config.Default(), logs.NewAuditLog())
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
	app := NewApp(nil, cfg, logs.NewAuditLog())
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
	app := NewApp(nil, cfg, logs.NewAuditLog())
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
