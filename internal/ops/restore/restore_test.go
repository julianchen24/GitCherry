package restore

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/julianchen24/gitcherry/internal/git"
	"github.com/julianchen24/gitcherry/internal/logs"
	"github.com/julianchen24/gitcherry/tests/repohelper"
)

func TestPlan(t *testing.T) {
	commands := Plan("backup", "abc123")
	require.Equal(t, []string{"git branch backup abc123"}, commands)
}

func TestExecuteCreatesBranchAndLogs(t *testing.T) {
	repo := repohelper.Init(t)
	repohelper.Chdir(t, repo.Path)

	logs.SetBasePath(repo.Path)
	t.Cleanup(func() { logs.SetBasePath("") })

	commit := strings.TrimSpace(repo.MustRun(t, "rev-parse", "HEAD"))
	branchName := "backup"

	runner := &git.Runner{Dir: repo.Path}
	audit := logs.NewAuditLog()

	require.NoError(t, Execute(context.Background(), runner, branchName, commit, audit))

	branches := repo.MustRun(t, "branch", "--list", branchName)
	require.Contains(t, branches, branchName)

	opDir := filepath.Join(repo.Path, ".gitcherry", "logs")
	entries, err := os.ReadDir(opDir)
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	undoEntry, ok, err := logs.Undo()
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, branchName, undoEntry.Source)
}
