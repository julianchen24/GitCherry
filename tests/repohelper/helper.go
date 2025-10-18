package repohelper

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Repo represents a temporary Git repository used in tests.
type Repo struct {
	Path string
}

// Init creates a new repository with an initial commit.
func Init(t *testing.T) *Repo {
	t.Helper()

	dir := t.TempDir()

	if _, _, err := run(dir, "git", "init", "--initial-branch=main"); err != nil {
		t.Fatalf("git init: %v", err)
	}

	mustRun(t, dir, "git", "config", "user.name", "Test User")
	mustRun(t, dir, "git", "config", "user.email", "test@example.com")

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatalf("write initial file: %v", err)
	}

	mustRun(t, dir, "git", "add", ".")
	mustRun(t, dir, "git", "commit", "-m", "initial")

	return &Repo{Path: dir}
}

// Run runs a git command inside the repository.
func (r *Repo) Run(args ...string) (string, string, error) {
	return run(r.Path, "git", args...)
}

// MustRun runs a git command and fails the test on error, returning stdout.
func (r *Repo) MustRun(t *testing.T, args ...string) string {
	t.Helper()
	stdout, stderr, err := r.Run(args...)
	if err != nil {
		t.Fatalf("git %v: %v (%s)", strings.Join(args, " "), err, strings.TrimSpace(stderr))
	}
	return stdout
}

// CommitFile creates/updates a file, commits it, and returns the new hash.
func (r *Repo) CommitFile(t *testing.T, name, content, message string) string {
	t.Helper()

	path := filepath.Join(r.Path, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	r.MustRun(t, "add", name)
	r.MustRun(t, "commit", "-m", message)
	return strings.TrimSpace(r.MustRun(t, "rev-parse", "HEAD"))
}

// WriteFile writes a file relative to the repository path.
func (r *Repo) WriteFile(name, content string) error {
	path := filepath.Join(r.Path, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// Chdir changes the working directory to the repository and restores it afterwards.
func Chdir(t *testing.T, dir string) {
	t.Helper()

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			panic(fmt.Sprintf("restore cwd: %v", err))
		}
	})
}

func run(dir, command string, args ...string) (string, string, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	cmd.Env = append(withoutPrompt(os.Environ()), "GIT_TERMINAL_PROMPT=0")

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	return stdoutBuf.String(), stderrBuf.String(), err
}

func mustRun(t *testing.T, dir, command string, args ...string) {
	t.Helper()
	if _, stderr, err := run(dir, command, args...); err != nil {
		t.Fatalf("%s %v: %v (%s)", command, strings.Join(args, " "), err, strings.TrimSpace(stderr))
	}
}

func withoutPrompt(env []string) []string {
	const key = "GIT_TERMINAL_PROMPT"
	result := make([]string, 0, len(env))
	for _, item := range env {
		if strings.HasPrefix(item, key+"=") {
			continue
		}
		result = append(result, item)
	}
	return result
}
