package transfer

import "testing"

func TestPlan(t *testing.T) {
	commands := Plan("main", "feature", "abc123", "def456", "Message")
	if len(commands) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(commands))
	}
	expected := []string{
		"git checkout feature",
		"git cherry-pick --no-commit abc123^..def456",
		"git commit -m \"Message\"",
	}
	for i, cmd := range expected {
		if commands[i] != cmd {
			t.Fatalf("command %d mismatch: expected %q got %q", i, cmd, commands[i])
		}
	}
}
