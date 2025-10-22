package restore

import (
	"context"
	"fmt"
	"time"

	"github.com/julianchen24/gitcherry/internal/git"
	"github.com/julianchen24/gitcherry/internal/logs"
)

// Plan returns the commands required to create a branch pointing at commitHash.
func Plan(branchName, commitHash string) []string {
	return []string{fmt.Sprintf("git branch %s %s", branchName, commitHash)}
}

// Execute runs the restore operation and records the bookkeeping artifacts.
func Execute(ctx context.Context, runner *git.Runner, branchName, commitHash string, audit *logs.AuditLog) error {
	_ = ctx
	if runner == nil {
		runner = &git.Runner{}
	}

	if _, stderr, err := runner.Run("branch", branchName, commitHash); err != nil {
		return fmt.Errorf("git branch %s %s failed: %v (%s)", branchName, commitHash, err, stderr)
	}

	if audit != nil {
		audit.Record(logs.Entry{
			Summary: fmt.Sprintf("restore branch %s", branchName),
			Metadata: map[string]string{
				"branch": branchName,
				"commit": commitHash,
			},
		})
	}

	plan := Plan(branchName, commitHash)
	op := logs.Operation{
		Source:    branchName,
		Target:    branchName,
		StartHash: commitHash,
		EndHash:   commitHash,
		Message:   fmt.Sprintf("Restore branch %s at %s", branchName, commitHash),
		Commands:  plan,
		Timestamp: time.Now().UTC(),
	}
	if err := logs.WriteOperation(op); err != nil {
		return err
	}

	undo := logs.UndoEntry{
		Source:    branchName,
		AfterHead: commitHash,
		Timestamp: time.Now().UTC(),
	}
	if err := logs.PushUndo(undo); err != nil {
		return err
	}

	return nil
}
