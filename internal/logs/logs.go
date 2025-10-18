package logs

import "sync"

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
