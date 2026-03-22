// gitinfo.go provides lightweight git repository metadata for a working directory.
// It shells out to git with strict timeouts so a slow or missing repo never
// blocks session discovery.
package discovery

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// gitTimeout is the maximum time any single git command may run.
const gitTimeout = 2 * time.Second

// GitInfo holds git repository metadata for a session's working directory.
type GitInfo struct {
	// Branch is the current branch name (e.g. "main", "feat/foo").
	// Empty if not a git repo or on a detached HEAD.
	Branch string

	// IsDirty is true when the working tree has uncommitted changes
	// (staged, unstaged, or untracked files).
	IsDirty bool

	// Ahead is the number of commits ahead of the upstream tracking branch.
	// Zero if no upstream is configured or git fails.
	Ahead int

	// Behind is the number of commits behind the upstream tracking branch.
	// Zero if no upstream is configured or git fails.
	Behind int

	// LastCommit is the short hash and subject of the most recent commit
	// (e.g. "abc1234 Fix the thing"). Empty if there are no commits.
	LastCommit string
}

// GetGitInfo returns git metadata for the repository at cwd.
// It returns a zero GitInfo if cwd is empty, does not exist, or is not
// inside a git repository. All errors are swallowed -- this function
// is designed to be purely defensive like the rest of the discovery package.
func GetGitInfo(cwd string) GitInfo {
	if cwd == "" {
		return GitInfo{}
	}

	// Quick check: is this even a git repo?
	if _, err := runGit(cwd, "rev-parse", "--git-dir"); err != nil {
		return GitInfo{}
	}

	var info GitInfo

	// Branch name.
	if out, err := runGit(cwd, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		branch := strings.TrimSpace(out)
		if branch != "" && branch != "HEAD" {
			info.Branch = branch
		}
	}

	// Dirty state: any output from status --porcelain means dirty.
	if out, err := runGit(cwd, "status", "--porcelain"); err == nil {
		info.IsDirty = strings.TrimSpace(out) != ""
	}

	// Last commit: short hash + subject.
	if out, err := runGit(cwd, "log", "-1", "--format=%h %s"); err == nil {
		info.LastCommit = strings.TrimSpace(out)
	}

	// Ahead/behind: parse rev-list --left-right --count HEAD...@{upstream}.
	// This fails silently if there is no upstream configured, which is fine.
	if out, err := runGit(cwd, "rev-list", "--left-right", "--count", "HEAD...@{upstream}"); err == nil {
		info.Ahead, info.Behind = parseAheadBehind(out)
	}

	return info
}

// gitCommand creates an exec.Cmd for running git in the given directory.
// Exported only to tests in the same package (lowercase g would work, but
// the test helper needs it).
func gitCommand(cwd string, args ...string) *exec.Cmd {
	fullArgs := append([]string{"-C", cwd}, args...)
	return exec.Command("git", fullArgs...)
}

// runGit executes a git command in cwd with a timeout. Returns trimmed stdout
// on success or an error.
func runGit(cwd string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	fullArgs := append([]string{"-C", cwd}, args...)
	cmd := exec.CommandContext(ctx, "git", fullArgs...)

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// parseAheadBehind parses the output of `git rev-list --left-right --count`.
// The output format is "AHEAD\tBEHIND\n".
func parseAheadBehind(output string) (ahead, behind int) {
	parts := strings.Fields(strings.TrimSpace(output))
	if len(parts) != 2 {
		return 0, 0
	}

	// Simple integer parse without importing strconv for two small numbers.
	for _, c := range parts[0] {
		if c >= '0' && c <= '9' {
			ahead = ahead*10 + int(c-'0')
		}
	}
	for _, c := range parts[1] {
		if c >= '0' && c <= '9' {
			behind = behind*10 + int(c-'0')
		}
	}
	return ahead, behind
}
