package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

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
