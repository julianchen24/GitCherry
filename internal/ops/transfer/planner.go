package transfer

import "fmt"

// Plan describes the shell commands required to move commits from the source
// branch onto the target branch.
func Plan(source, target, startHash, endHash, message string) []string {
	rangeSpec := fmt.Sprintf("%s^..%s", startHash, endHash)
	return []string{
		fmt.Sprintf("git checkout %s", target),
		fmt.Sprintf("git cherry-pick --no-commit %s", rangeSpec),
		fmt.Sprintf("git commit -m %q", message),
	}
}
