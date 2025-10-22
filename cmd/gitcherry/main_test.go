package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/julianchen24/gitcherry/internal/config"
	"github.com/julianchen24/gitcherry/internal/git"
	"github.com/julianchen24/gitcherry/tests/repohelper"
)

func TestRootCommandShowsHelp(t *testing.T) {
	repo := repohelper.Init(t)
	repohelper.Chdir(t, repo.Path)

	root := newRootCommand()
	root.SetArgs([]string{"--help"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	err := root.Execute()
	require.NoError(t, err)
	require.Contains(t, buf.String(), "Interactive helper for cherry-picking Git commits")
}

func TestRootCommandUnknownFlag(t *testing.T) {
	repo := repohelper.Init(t)
	repohelper.Chdir(t, repo.Path)

	root := newRootCommand()
	root.SilenceErrors = true
	root.SetArgs([]string{"--does-not-exist"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	err := root.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown flag")
}

func TestRootCommandFailsWhenDirty(t *testing.T) {
	repo := repohelper.Init(t)
	repohelper.Chdir(t, repo.Path)

	err := repo.WriteFile("dirty.txt", "dirty\n")
	require.NoError(t, err)

	root := newRootCommand()
	root.SilenceErrors = true

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	err = root.Execute()
	require.Error(t, err)
	require.EqualError(t, err, dirtyWorktreeMessage)
}

func TestTransferDryRunUsesPlan(t *testing.T) {
	origPlan := transferPlanFn
	defer func() { transferPlanFn = origPlan }()
	origRange := commitRangeFn
	defer func() { commitRangeFn = origRange }()
	origDup := transferDetectDuplicatesFn
	defer func() { transferDetectDuplicatesFn = origDup }()

	var captured struct {
		from, to, start, end, message string
	}
	transferPlanFn = func(from, to, start, end, message string) []string {
		captured = struct {
			from, to, start, end, message string
		}{from, to, start, end, message}
		return []string{"git checkout " + to}
	}
	commitRangeFn = func(*git.Runner, string, string) ([]git.Commit, error) {
		return []git.Commit{{Hash: "a"}, {Hash: "b"}}, nil
	}
	transferDetectDuplicatesFn = func(*git.Runner, string, []git.Commit) ([]git.Commit, error) {
		return nil, nil
	}

	cmd := newTransferCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	ctx := context.Background()
	cfg := config.Default()
	ctx = context.WithValue(ctx, ctxConfigKey{}, cfg)
	ctx = context.WithValue(ctx, ctxApplyKey{}, false)
	ctx = context.WithValue(ctx, ctxDuplicateKey{}, "skip")
	cmd.SetContext(ctx)

	require.NoError(t, cmd.Flags().Set("from", "main"))
	require.NoError(t, cmd.Flags().Set("to", "feature"))
	require.NoError(t, cmd.Flags().Set("range", "a..b"))
	require.NoError(t, cmd.Flags().Set("message", "custom"))

	require.NoError(t, cmd.Execute())
	require.Equal(t, "main", captured.from)
	require.Equal(t, "feature", captured.to)
	require.Equal(t, "a", captured.start)
	require.Equal(t, "b", captured.end)
	require.Equal(t, "custom", captured.message)
	require.Contains(t, buf.String(), "Planned commands")
}

func TestTransferSkipsWhenDuplicatesAndModeSkip(t *testing.T) {
	origPlan := transferPlanFn
	defer func() { transferPlanFn = origPlan }()
	origRange := commitRangeFn
	defer func() { commitRangeFn = origRange }()
	origDup := transferDetectDuplicatesFn
	defer func() { transferDetectDuplicatesFn = origDup }()

	called := false
	transferPlanFn = func(from, to, start, end, message string) []string {
		called = true
		return nil
	}
	commitRangeFn = func(*git.Runner, string, string) ([]git.Commit, error) {
		return []git.Commit{{Hash: "a"}}, nil
	}
	transferDetectDuplicatesFn = func(*git.Runner, string, []git.Commit) ([]git.Commit, error) {
		return []git.Commit{{Hash: "dup"}}, nil
	}

	cmd := newTransferCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	ctx := context.Background()
	cfg := config.Default()
	ctx = context.WithValue(ctx, ctxConfigKey{}, cfg)
	ctx = context.WithValue(ctx, ctxApplyKey{}, false)
	ctx = context.WithValue(ctx, ctxDuplicateKey{}, "skip")
	cmd.SetContext(ctx)

	require.NoError(t, cmd.Flags().Set("from", "main"))
	require.NoError(t, cmd.Flags().Set("to", "feature"))
	require.NoError(t, cmd.Flags().Set("range", "a..a"))

	require.NoError(t, cmd.Execute())
	require.False(t, called)
	require.Contains(t, buf.String(), "Skipping transfer")
}

func TestRevertDryRunUsesPlan(t *testing.T) {
	origPlan := revertPlanFn
	defer func() { revertPlanFn = origPlan }()

	var captured struct {
		start, end, message string
	}
	revertPlanFn = func(source, target, start, end, message string) []string {
		captured = struct {
			start, end, message string
		}{start, end, message}
		return []string{"git revert --no-commit"}
	}

	cmd := newRevertCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxApplyKey{}, false)
	cmd.SetContext(ctx)

	require.NoError(t, cmd.Flags().Set("on", "main"))
	require.NoError(t, cmd.Flags().Set("range", "a..b"))
	require.NoError(t, cmd.Execute())
	require.Equal(t, "a", captured.start)
	require.Equal(t, "b", captured.end)
	require.Contains(t, buf.String(), "Planned commands")
}

func TestRestoreDryRunUsesPlan(t *testing.T) {
	origPlan := restorePlanFn
	defer func() { restorePlanFn = origPlan }()

	var captured struct{ branch, commit string }
	restorePlanFn = func(branchName, commitHash string) []string {
		captured = struct{ branch, commit string }{branchName, commitHash}
		return []string{"git branch " + branchName + " " + commitHash}
	}

	cmd := newRestoreCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxApplyKey{}, false)
	cmd.SetContext(ctx)

	require.NoError(t, cmd.Flags().Set("at", "abc"))
	require.NoError(t, cmd.Flags().Set("branch-name", "backup"))
	require.NoError(t, cmd.Execute())
	require.Equal(t, "backup", captured.branch)
	require.Equal(t, "abc", captured.commit)
	require.Contains(t, buf.String(), "Planned commands")
}
