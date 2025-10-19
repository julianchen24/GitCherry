package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandShowsHelp(t *testing.T) {
	root := newRootCommand()
	root.SetArgs([]string{"--help"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Interactive helper for cherry-picking Git commits") {
		t.Fatalf("expected help output, got %q", output)
	}
}

func TestRootCommandUnknownFlag(t *testing.T) {
	root := newRootCommand()
	root.SetArgs([]string{"--does-not-exist"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	err := root.Execute()
	if err == nil {
		t.Fatalf("expected error for unknown flag")
	}

	if !strings.Contains(buf.String(), "unknown flag") {
		t.Fatalf("expected unknown flag message, got %q", buf.String())
	}
}
