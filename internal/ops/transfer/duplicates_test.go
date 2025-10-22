package transfer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/julianchen24/gitcherry/internal/git"
	"github.com/julianchen24/gitcherry/tests/repohelper"
)

func TestDetectDuplicatesFindsMatchingPatch(t *testing.T) {
	repo := repohelper.Init(t)

	// Prepare target branch with commit A.
	repo.MustRun(t, "checkout", "-b", "target")
	repo.CommitFile(t, "file.txt", "line1\n", "target commit")

	// Create source branch with duplicate patch.
	repo.MustRun(t, "checkout", "main")
	repo.MustRun(t, "checkout", "-b", "source")
	repo.MustRun(t, "cherry-pick", "target")

	dupHash := strings.TrimSpace(repo.MustRun(t, "rev-parse", "HEAD"))
	commits := []git.Commit{{Hash: dupHash}}

	duplicates, err := DetectDuplicates(&git.Runner{Dir: repo.Path}, "target", commits)
	require.NoError(t, err)
	require.Len(t, duplicates, 1)
	require.Equal(t, dupHash, duplicates[0].Hash)
}

func TestDetectDuplicatesIgnoresUniquePatch(t *testing.T) {
	repo := repohelper.Init(t)
	repo.MustRun(t, "checkout", "-b", "target")
	repo.CommitFile(t, "a.txt", "a\n", "unique target")

	repo.MustRun(t, "checkout", "main")
	repo.MustRun(t, "checkout", "-b", "source")
	repo.CommitFile(t, "b.txt", "b\n", "unique source")

	dupHash := strings.TrimSpace(repo.MustRun(t, "rev-parse", "HEAD"))
	commits := []git.Commit{{Hash: dupHash}}

	duplicates, err := DetectDuplicates(&git.Runner{Dir: repo.Path}, "target", commits)
	require.NoError(t, err)
	require.Len(t, duplicates, 0)
}
