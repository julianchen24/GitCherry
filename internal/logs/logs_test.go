package logs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWriteOperationPersistsJSON(t *testing.T) {
	dir := t.TempDir()
	SetBasePath(dir)
	t.Cleanup(func() { SetBasePath("") })

	op := Operation{
		Source:    "main",
		Target:    "feature",
		StartHash: "abc",
		EndHash:   "def",
		Message:   "Merge commits",
		Commands:  []string{"git checkout feature"},
	}

	require.NoError(t, WriteOperation(op))

	logDir := filepath.Join(dir, ".gitcherry", "logs")
	entries, err := os.ReadDir(logDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	data, err := os.ReadFile(filepath.Join(logDir, entries[0].Name()))
	require.NoError(t, err)

	var stored Operation
	require.NoError(t, json.Unmarshal(data, &stored))
	require.Equal(t, op.Source, stored.Source)
	require.Equal(t, op.Target, stored.Target)
	require.Equal(t, op.StartHash, stored.StartHash)
	require.NotZero(t, stored.Timestamp)
}

func TestUndoRedoLifecycle(t *testing.T) {
	dir := t.TempDir()
	SetBasePath(dir)
	t.Cleanup(func() { SetBasePath("") })

	entry1 := UndoEntry{Source: "main", Target: "feature", BeforeHead: "a1", AfterHead: "b1"}
	entry2 := UndoEntry{Source: "main", Target: "feature", BeforeHead: "a2", AfterHead: "b2"}

	require.NoError(t, PushUndo(entry1))
	require.NoError(t, PushUndo(entry2))

	undoEntry, ok, err := Undo()
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, entry2.BeforeHead, undoEntry.BeforeHead)
	require.WithinDuration(t, time.Now(), undoEntry.Timestamp, time.Second)

	undoEntry, ok, err = Undo()
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, entry1.BeforeHead, undoEntry.BeforeHead)

	_, ok, err = Undo()
	require.NoError(t, err)
	require.False(t, ok)

	redoEntry, ok, err := Redo()
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, entry1.AfterHead, redoEntry.AfterHead)

	redoEntry, ok, err = Redo()
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, entry2.AfterHead, redoEntry.AfterHead)

	_, ok, err = Redo()
	require.NoError(t, err)
	require.False(t, ok)
}
