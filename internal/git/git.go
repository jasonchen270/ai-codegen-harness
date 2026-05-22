// Package git wraps the local git operations the harness relies on. Git is the
// enforcement boundary: a run's writes are only made permanent by a commit, and
// a rejected run is wiped by reverting the working tree to the pre-run snapshot.
//
// Commits use the local git identity; no remote is configured.
package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// run executes a git command in dir and returns its stdout.
func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s",
			strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// Init runs `git init` in dir.
func Init(dir string) error {
	_, err := run(dir, "init")
	return err
}

// IsClean reports whether the working tree has no pending changes.
func IsClean(dir string) (bool, error) {
	out, err := run(dir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "", nil
}

// HasCommits reports whether HEAD resolves (i.e. at least one commit exists).
func HasCommits(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = dir
	return cmd.Run() == nil
}

// Change is a single working-tree change as reported by `git status`.
type Change struct {
	Status string // porcelain XY code, e.g. "A ", " M", "??", " D"
	Path   string
}

func (c Change) IsDelete() bool { return strings.Contains(c.Status, "D") }
func (c Change) IsModify() bool { return strings.Contains(c.Status, "M") }

// Changes returns all working-tree changes vs HEAD (untracked files included,
// so newly written files are seen).
func Changes(dir string) ([]Change, error) {
	out, err := run(dir, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	var changes []Change
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		code, path := line[:2], strings.TrimSpace(line[3:])
		// Renames "old -> new": the new path is the change.
		if i := strings.Index(path, " -> "); i >= 0 {
			path = path[i+len(" -> "):]
		}
		path = strings.Trim(path, `"`)
		changes = append(changes, Change{Status: code, Path: path})
	}
	return changes, nil
}

// RevertToClean discards all working-tree changes: tracked files back to HEAD,
// untracked files removed. This is the "reject" action that wipes a violating
// prompt's writes so the tree returns to the pre-prompt snapshot.
func RevertToClean(dir string) error {
	if HasCommits(dir) {
		if _, err := run(dir, "reset", "--hard", "HEAD"); err != nil {
			return err
		}
	}
	_, err := run(dir, "clean", "-fd")
	return err
}

// CommitAll stages everything and commits with the given message.
func CommitAll(dir, message string) error {
	if _, err := run(dir, "add", "-A"); err != nil {
		return err
	}
	_, err := run(dir, "commit", "-m", message)
	return err
}
