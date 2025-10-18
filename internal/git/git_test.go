package git_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/julianchen24/gitcherry/internal/git"
	"github.com/julianchen24/gitcherry/tests/repohelper"
)

func TestRunnerRunUsesWorkingDirectory(t *testing.T) {
	repo := repohelper.Init(t)

	runner := &git.Runner{Dir: repo.Path}
	stdout, stderr, err := runner.Run("status", "--porcelain")
	require.NoError(t, err)
	require.Empty(t, strings.TrimSpace(stdout))
	require.Empty(t, strings.TrimSpace(stderr))
}

func TestCurrentBranch(t *testing.T) {
	repo := repohelper.Init(t)
	repohelper.Chdir(t, repo.Path)

	branch, err := git.CurrentBranch()
	require.NoError(t, err)
	require.Equal(t, "main", branch)
}

func TestIsClean(t *testing.T) {
	repo := repohelper.Init(t)
	repohelper.Chdir(t, repo.Path)

	clean, err := git.IsClean()
	require.NoError(t, err)
	require.True(t, clean)

	err = repo.WriteFile("untracked.txt", "hello\n")
	require.NoError(t, err)

	clean, err = git.IsClean()
	require.NoError(t, err)
	require.False(t, clean)
}

func TestListBranches(t *testing.T) {
	repo := repohelper.Init(t)
	repohelper.Chdir(t, repo.Path)

	repo.MustRun(t, "checkout", "-b", "feature")

	branches, err := git.ListBranches()
	require.NoError(t, err)
	require.Contains(t, branches, "main")
	require.Contains(t, branches, "feature")
}

func TestCommitsBetween(t *testing.T) {
	repo := repohelper.Init(t)
	repohelper.Chdir(t, repo.Path)

	initial := strings.TrimSpace(repo.MustRun(t, "rev-parse", "HEAD"))
	newHash := repo.CommitFile(t, "note.txt", "first note\n", "add note")

	commits, err := git.CommitsBetween(initial, newHash)
	require.NoError(t, err)
	require.Len(t, commits, 1)

	commit := commits[0]
	require.Equal(t, newHash, commit.Hash)
	require.Equal(t, "Test User", commit.Author)
	require.Equal(t, "add note", commit.Message)
	require.NotEmpty(t, commit.Date)
	require.Equal(t, []string{"note.txt"}, commit.Files)
}

func TestPatchID(t *testing.T) {
	repo := repohelper.Init(t)
	repohelper.Chdir(t, repo.Path)

	hash := repo.CommitFile(t, "patch.txt", "content\n", "patch commit")

	patchID, err := git.PatchID(hash)
	require.NoError(t, err)
	require.Len(t, patchID, 40)
}
