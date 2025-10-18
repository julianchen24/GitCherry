package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Run executes the git binary with the provided arguments and returns the
// combined stdout and stderr output.
func Run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		return buf.String(), fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return buf.String(), nil
}

// Version returns the Git version string, trimming whitespace for convenience.
func Version(ctx context.Context) (string, error) {
	out, err := Run(ctx, "--version")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
