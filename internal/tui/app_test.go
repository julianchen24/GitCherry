package tui

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/julianchen24/gitcherry/internal/config"
)

func TestNewAppInitializes(t *testing.T) {
	app := NewApp(nil, config.Default())
	require.NotNil(t, app)
	require.NotNil(t, app.BranchList)
	require.NotNil(t, app.CommitList)
	require.NotNil(t, app.PreviewModal)
	require.NotNil(t, app.HelpModal)
	require.False(t, app.HelpVisible())
}

func TestToggleHelp(t *testing.T) {
	app := NewApp(nil, config.Default())
	require.False(t, app.HelpVisible())

	app.ToggleHelp()
	require.True(t, app.HelpVisible())

	app.ToggleHelp()
	require.False(t, app.HelpVisible())
}
