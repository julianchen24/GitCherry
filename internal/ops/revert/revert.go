package revert

import (
	"context"
	"fmt"
)

import "github.com/julianchen24/gitcherry/internal/git"

// Plan returns the shell commands required to revert a range of commits.
func Plan(source, target, startHash, endHash, message string) []string {
	rangeSpec := fmt.Sprintf("%s^..%s", startHash, endHash)
	return []string{
		fmt.Sprintf("git checkout %s", target),
		fmt.Sprintf("git revert --no-commit %s", rangeSpec),
		fmt.Sprintf("git commit -m %q", message),
	}
}

// Execute performs the revert using the provided git runner.
func Execute(ctx context.Context, runner *git.Runner, target, startHash, endHash, message string) error {
	if runner == nil {
		runner = &git.Runner{}
	}

	if _, stderr, err := runner.Run("checkout", target); err != nil {
		return fmt.Errorf("git checkout %s failed: %v (%s)", target, err, stderr)
	}

	rangeSpec := fmt.Sprintf("%s^..%s", startHash, endHash)
	if _, stderr, err := runner.Run("revert", "--no-commit", rangeSpec); err != nil {
		return fmt.Errorf("git revert --no-commit %s failed: %v (%s). Resolve conflicts, then run 'git revert --continue' or 'git revert --abort'",
			rangeSpec, err, stderr)
	}

	if _, stderr, err := runner.Run("commit", "-m", message); err != nil {
		return fmt.Errorf("git commit failed: %v (%s)", err, stderr)
	}

	return nil
}
