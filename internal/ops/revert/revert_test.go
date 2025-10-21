package revert

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/julianchen24/gitcherry/internal/git"
	"github.com/julianchen24/gitcherry/tests/repohelper"
)

func TestPlanCommands(t *testing.T) {
	commands := Plan("main", "feature", "abc", "def", "Revert message")
	require.Equal(t, []string{
		"git checkout feature",
		"git revert --no-commit abc^..def",
		"git commit -m \"Revert message\"",
	}, commands)
}

func TestExecuteRevert(t *testing.T) {
	repo := repohelper.Init(t)

	repo.MustRun(t, "checkout", "-b", "feature")
	repo.CommitFile(t, "file1.txt", "first\n", "first commit")
	repo.CommitFile(t, "file2.txt", "second\n", "second commit")

	commitsOutput := repo.MustRun(t, "log", "--pretty=%H", "-n", "2")
	lines := strings.Split(strings.TrimSpace(commitsOutput), "\n")
	require.Len(t, lines, 2)
	start := lines[1]
	end := lines[0]

	runner := &git.Runner{Dir: repo.Path}
	ctx := context.Background()
	require.NoError(t, Execute(ctx, runner, "feature", start, end, "Revert range"))

	message := strings.TrimSpace(repo.MustRun(t, "log", "-1", "--pretty=%s"))
	require.Equal(t, "Revert range", message)

	status := strings.TrimSpace(repo.MustRun(t, "status", "--porcelain"))
	require.Empty(t, status)
}
