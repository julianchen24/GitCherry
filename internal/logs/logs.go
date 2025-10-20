package logs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry captures a single auditable action that GitCherry performed.
type Entry struct {
	Summary  string
	Metadata map[string]string
}

// AuditLog stores chronological actions and supports undo/redo navigation.
type AuditLog struct {
	mu       sync.Mutex
	entries  []Entry
	position int
}

// NewAuditLog returns an empty in-memory audit log.
func NewAuditLog() *AuditLog {
	return &AuditLog{}
}

// Record appends a new entry and truncates any redo history.
func (a *AuditLog) Record(entry Entry) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.position < len(a.entries) {
		a.entries = append([]Entry{}, a.entries[:a.position]...)
	}
	a.entries = append(a.entries, entry)
	a.position = len(a.entries)
}

// Undo steps back in the history and returns the entry if available.
func (a *AuditLog) Undo() (Entry, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.position == 0 {
		return Entry{}, false
	}
	a.position--
	return a.entries[a.position], true
}

// Redo steps forward in the history when possible.
func (a *AuditLog) Redo() (Entry, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.position >= len(a.entries) {
		return Entry{}, false
	}
	entry := a.entries[a.position]
	a.position++
	return entry, true
}

// Operation describes a transfer that GitCherry performed.
type Operation struct {
	Source    string    `json:"source"`
	Target    string    `json:"target"`
	StartHash string    `json:"start_hash"`
	EndHash   string    `json:"end_hash"`
	Message   string    `json:"message"`
	Commands  []string  `json:"commands"`
	Timestamp time.Time `json:"timestamp"`
}

// UndoEntry captures metadata required to restore repository state.
type UndoEntry struct {
	Source     string    `json:"source"`
	Target     string    `json:"target"`
	BeforeHead string    `json:"before_head"`
	AfterHead  string    `json:"after_head"`
	Timestamp  time.Time `json:"timestamp"`
}

var (
	storageMu sync.Mutex
	basePath  = "."
)

// SetBasePath overrides the root used for persisting log data. Use only in tests.
func SetBasePath(path string) {
	storageMu.Lock()
	defer storageMu.Unlock()
	if path == "" {
		basePath = "."
		return
	}
	basePath = path
}

// WriteOperation persists the provided operation to the on-disk log.
func WriteOperation(op Operation) error {
	storageMu.Lock()
	defer storageMu.Unlock()

	if op.Timestamp.IsZero() {
		op.Timestamp = time.Now().UTC()
	}

	dir := filepath.Join(basePath, ".gitcherry", "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	baseName := op.Timestamp.Format("20060102T150405Z0700")
	path := filepath.Join(dir, baseName+".json")
	if _, err := os.Stat(path); err == nil {
		for i := 1; ; i++ {
			candidate := filepath.Join(dir, fmt.Sprintf("%s_%d.json", baseName, i))
			if _, statErr := os.Stat(candidate); errors.Is(statErr, fs.ErrNotExist) {
				path = candidate
				break
			}
		}
	}

	data, err := json.MarshalIndent(op, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// PushUndo appends a new undo entry to the persistent stack.
func PushUndo(entry UndoEntry) error {
	storageMu.Lock()
	defer storageMu.Unlock()

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	state, err := loadUndoStateLocked()
	if err != nil {
		return err
	}

	if state.Position < len(state.History) {
		state.History = append([]UndoEntry{}, state.History[:state.Position]...)
	}
	state.History = append(state.History, entry)
	state.Position = len(state.History)

	return saveUndoStateLocked(state)
}

// Undo steps backwards in the persistent undo stack.
func Undo() (UndoEntry, bool, error) {
	storageMu.Lock()
	defer storageMu.Unlock()

	state, err := loadUndoStateLocked()
	if err != nil {
		return UndoEntry{}, false, err
	}

	if state.Position == 0 {
		return UndoEntry{}, false, nil
	}
	state.Position--
	entry := state.History[state.Position]

	if err := saveUndoStateLocked(state); err != nil {
		return UndoEntry{}, false, err
	}

	return entry, true, nil
}

// Redo re-applies the most recently undone entry if available.
func Redo() (UndoEntry, bool, error) {
	storageMu.Lock()
	defer storageMu.Unlock()

	state, err := loadUndoStateLocked()
	if err != nil {
		return UndoEntry{}, false, err
	}

	if state.Position >= len(state.History) {
		return UndoEntry{}, false, nil
	}
	entry := state.History[state.Position]
	state.Position++

	if err := saveUndoStateLocked(state); err != nil {
		return UndoEntry{}, false, err
	}

	return entry, true, nil
}

type undoState struct {
	History  []UndoEntry `json:"history"`
	Position int         `json:"position"`
}

func loadUndoStateLocked() (undoState, error) {
	path := undoStatePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return undoState{}, nil
		}
		return undoState{}, err
	}
	if len(data) == 0 {
		return undoState{}, nil
	}
	var state undoState
	if err := json.Unmarshal(data, &state); err != nil {
		return undoState{}, err
	}
	if state.Position < 0 || state.Position > len(state.History) {
		state.Position = len(state.History)
	}
	return state, nil
}

func saveUndoStateLocked(state undoState) error {
	dir := filepath.Join(basePath, ".gitcherry")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(undoStatePath(), data, 0o600)
}

func undoStatePath() string {
	return filepath.Join(basePath, ".gitcherry", "undo.json")
}
