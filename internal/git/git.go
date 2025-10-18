package git

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Runner executes git commands against an optional working directory.
type Runner struct {
	Dir   string
	Stdio bool
}

// Run executes the git binary with the provided arguments.
func (r *Runner) Run(args ...string) (string, string, error) {
	cmd := exec.Command("git", args...)

	if r != nil && r.Dir != "" {
		cmd.Dir = r.Dir
	}

	cmd.Env = withNoPrompt(os.Environ())

	var stdoutBuf, stderrBuf bytes.Buffer

	if r != nil && r.Stdio {
		cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
		cmd.Stdin = os.Stdin
	} else {
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf
	}

	err := cmd.Run()
	return stdoutBuf.String(), stderrBuf.String(), err
}

// CurrentBranch returns the current checked-out branch name.
func CurrentBranch() (string, error) {
	stdout, stderr, err := runGit("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", commandError(err, stderr)
	}
	return strings.TrimSpace(stdout), nil
}

// IsClean reports whether the working tree has no staged or unstaged changes.
func IsClean() (bool, error) {
	stdout, stderr, err := runGit("status", "--porcelain")
	if err != nil {
		return false, commandError(err, stderr)
	}
	return strings.TrimSpace(stdout) == "", nil
}

// Fetch updates remote tracking branches. When prune is true, prunes removed refs.
func Fetch(prune bool) error {
	args := []string{"fetch"}
	if prune {
		args = append(args, "--prune")
	}
	_, stderr, err := runGit(args...)
	if err != nil {
		return commandError(err, stderr)
	}
	return nil
}

// ListBranches returns the short names of local branches.
func ListBranches() ([]string, error) {
	stdout, stderr, err := runGit("branch", "--format=%(refname:short)")
	if err != nil {
		return nil, commandError(err, stderr)
	}

	lines := splitLines(stdout)
	branches := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		branches = append(branches, line)
	}
	return branches, nil
}

// Commit represents metadata about a single Git commit.
type Commit struct {
	Hash    string
	Author  string
	Date    string
	Message string
	Files   []string
}

// CommitsBetween returns commits reachable from head but not base.
func CommitsBetween(base, head string) ([]Commit, error) {
	spec := fmt.Sprintf("%s..%s", strings.TrimSpace(base), strings.TrimSpace(head))
	if strings.HasPrefix(spec, "..") || strings.HasSuffix(spec, "..") {
		return nil, errors.New("base and head must be provided")
	}

	stdout, stderr, err := runGit("rev-list", "--reverse", spec)
	if err != nil {
		return nil, commandError(err, stderr)
	}

	hashes := splitLines(stdout)
	commits := make([]Commit, 0, len(hashes))
	for _, hash := range hashes {
		hash = strings.TrimSpace(hash)
		if hash == "" {
			continue
		}

		info, infoErr, infoRunErr := runGit("log", "-1", "--name-only", "--date=iso-strict", "--pretty=format:%H\x1f%an\x1f%ad\x1f%s", hash)
		if infoRunErr != nil {
			return nil, commandError(infoRunErr, infoErr)
		}

		lines := splitLines(info)
		if len(lines) == 0 {
			continue
		}

		header := strings.SplitN(lines[0], "\x1f", 4)
		if len(header) != 4 {
			return nil, fmt.Errorf("unexpected git show format for %s", hash)
		}

		files := make([]string, 0, len(lines)-1)
		for _, file := range lines[1:] {
			file = strings.TrimSpace(file)
			if file == "" {
				continue
			}
			files = append(files, filepath.ToSlash(file))
		}

		commits = append(commits, Commit{
			Hash:    header[0],
			Author:  header[1],
			Date:    header[2],
			Message: header[3],
			Files:   files,
		})
	}

	return commits, nil
}

// PatchID returns the stable patch identifier for a commit.
func PatchID(hash string) (string, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return "", errors.New("hash is required")
	}

	showOut, showErr, err := runGit("show", hash, "--pretty=format:", "--patch")
	if err != nil {
		return "", commandError(err, showErr)
	}

	cmd := exec.Command("git", "patch-id", "--stable")
	cmd.Env = withNoPrompt(os.Environ())
	cmd.Stdin = strings.NewReader(showOut)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		return "", commandError(err, stderrBuf.String())
	}

	result := strings.Fields(stdoutBuf.String())
	if len(result) == 0 {
		return "", errors.New("patch-id returned no output")
	}
	return result[0], nil
}

func runGit(args ...string) (string, string, error) {
	var runner Runner
	return runner.Run(args...)
}

func withNoPrompt(env []string) []string {
	const key = "GIT_TERMINAL_PROMPT"

	base := make([]string, 0, len(env)+1)
	for _, item := range env {
		if strings.HasPrefix(item, key+"=") {
			continue
		}
		base = append(base, item)
	}

	base = append(base, key+"=0")
	return base
}

func splitLines(input string) []string {
	if input == "" {
		return nil
	}

	normalized := strings.ReplaceAll(input, "\r\n", "\n")
	normalized = strings.TrimSuffix(normalized, "\n")
	if normalized == "" {
		return nil
	}
	return strings.Split(normalized, "\n")
}

func commandError(err error, stderr string) error {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, stderr)
}
