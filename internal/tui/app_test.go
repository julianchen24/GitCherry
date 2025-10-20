package tui

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/julianchen24/gitcherry/internal/config"
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

func TestNewAppInitializes(t *testing.T) {
	withStubBranches(t, []string{"main"}, nil)

	cfg := config.Default()
	cfg.AutoRefresh = false

	app := NewApp(nil, cfg)
	require.NotNil(t, app)
	require.NotNil(t, app.BranchList)
	require.NotNil(t, app.CommitList)
	require.NotNil(t, app.PreviewModal)
	require.NotNil(t, app.HelpModal)
	require.NotNil(t, app.refreshBanner)
	require.False(t, app.HelpVisible())
	require.Equal(t, 1, app.BranchList.GetItemCount())
}

func TestToggleHelp(t *testing.T) {
	withStubBranches(t, []string{"main"}, nil)

	app := NewApp(nil, config.Default())
	require.False(t, app.HelpVisible())

	app.ToggleHelp()
	require.True(t, app.HelpVisible())

	app.ToggleHelp()
	require.False(t, app.HelpVisible())
}

func TestBranchSelectionFlow(t *testing.T) {
	withStubBranches(t, []string{"main", "feature"}, nil)

	cfg := config.Default()
	app := NewApp(nil, cfg)

	require.Equal(t, 2, app.BranchList.GetItemCount())
	require.Equal(t, 0, app.branchStage)

	app.handleBranchSelection("main")
	require.Equal(t, "main", app.branchSource)
	require.Equal(t, 1, app.branchStage)
	mainText, _ := app.CommitList.GetItemText(0)
	require.Contains(t, mainText, "Select target branch")
	require.Contains(t, mainText, "main")

	app.handleBranchSelection("feature")
	require.Equal(t, "feature", app.branchTarget)
	require.Equal(t, 2, app.branchStage)
	mainText, _ = app.CommitList.GetItemText(0)
	require.Contains(t, mainText, "Commits for main")
	require.Contains(t, mainText, "feature")
}
