package transfer

import (
	"fmt"
	"strings"

	"github.com/julianchen24/gitcherry/internal/git"
)

// DetectDuplicates returns commits whose patch-ids already exist on the target branch.
func DetectDuplicates(runner *git.Runner, target string, commits []git.Commit) ([]git.Commit, error) {
	if len(commits) == 0 {
		return nil, nil
	}
	if runner == nil {
		runner = &git.Runner{}
	}

	out, stderr, err := runner.Run("log", "--pretty=%H", target)
	if err != nil {
		return nil, commandError(err, stderr)
	}

	targetHashes := strings.Fields(strings.TrimSpace(out))
	patches := make(map[string]struct{}, len(targetHashes))
	for _, hash := range targetHashes {
		pid, err := runner.PatchID(hash)
		if err != nil || pid == "" {
			continue
		}
		patches[pid] = struct{}{}
	}

	duplicates := make([]git.Commit, 0)
	for _, commit := range commits {
		pid, err := runner.PatchID(commit.Hash)
		if err != nil || pid == "" {
			continue
		}
		if _, ok := patches[pid]; ok {
			duplicates = append(duplicates, commit)
		}
	}
	return duplicates, nil
}

func commandError(err error, stderr string) error {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, stderr)
}
