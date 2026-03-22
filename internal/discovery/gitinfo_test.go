package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetGitInfo_RealRepo tests against the OpsDeck repo itself.
// This is a live repo, so we know it has a branch and commits.
func TestGetGitInfo_RealRepo(t *testing.T) {
	// Walk up from the test file to find the repo root.
	// We're in internal/discovery/, so go up two levels.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() failed: %v", err)
	}
	repoRoot := filepath.Dir(filepath.Dir(wd))

	info := GetGitInfo(repoRoot)

	if info.Branch == "" {
		t.Error("Branch should not be empty for a real git repo")
	}
	if info.LastCommit == "" {
		t.Error("LastCommit should not be empty for a real git repo")
	}
	// LastCommit should contain a short hash (7+ hex chars) followed by a space.
	if len(info.LastCommit) < 9 {
		t.Errorf("LastCommit too short: %q (expected hash + message)", info.LastCommit)
	}
}

// TestGetGitInfo_NonExistentDir tests that a non-existent path returns zero GitInfo.
func TestGetGitInfo_NonExistentDir(t *testing.T) {
	info := GetGitInfo("/this/path/does/not/exist/at/all")

	if info.Branch != "" {
		t.Errorf("Branch = %q, want empty for non-existent dir", info.Branch)
	}
	if info.IsDirty {
		t.Error("IsDirty = true, want false for non-existent dir")
	}
	if info.Ahead != 0 {
		t.Errorf("Ahead = %d, want 0 for non-existent dir", info.Ahead)
	}
	if info.Behind != 0 {
		t.Errorf("Behind = %d, want 0 for non-existent dir", info.Behind)
	}
	if info.LastCommit != "" {
		t.Errorf("LastCommit = %q, want empty for non-existent dir", info.LastCommit)
	}
}

// TestGetGitInfo_NonGitDir tests a directory that exists but is not a git repo.
func TestGetGitInfo_NonGitDir(t *testing.T) {
	tmpDir := t.TempDir()

	info := GetGitInfo(tmpDir)

	if info.Branch != "" {
		t.Errorf("Branch = %q, want empty for non-git dir", info.Branch)
	}
	if info.IsDirty {
		t.Error("IsDirty = true, want false for non-git dir")
	}
	if info.LastCommit != "" {
		t.Errorf("LastCommit = %q, want empty for non-git dir", info.LastCommit)
	}
}

// TestGetGitInfo_EmptyString tests that empty CWD returns zero GitInfo.
func TestGetGitInfo_EmptyString(t *testing.T) {
	info := GetGitInfo("")

	if info.Branch != "" {
		t.Errorf("Branch = %q, want empty for empty CWD", info.Branch)
	}
}

// TestGetGitInfo_DirtyState tests that uncommitted changes are detected.
func TestGetGitInfo_DirtyState(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize a git repo, create a commit, then make a dirty file.
	runGit := func(args ...string) {
		t.Helper()
		cmd := gitCommand(tmpDir, args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	runGit("init")
	runGit("config", "user.email", "test@test.com")
	runGit("config", "user.name", "Test")

	// Create initial commit so HEAD exists.
	initial := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(initial, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit("add", ".")
	runGit("commit", "-m", "initial")

	// Clean state first.
	info := GetGitInfo(tmpDir)
	if info.Branch == "" {
		t.Fatal("Branch should not be empty after git init + commit")
	}
	if info.IsDirty {
		t.Error("IsDirty should be false right after commit")
	}

	// Now make it dirty.
	dirty := filepath.Join(tmpDir, "dirty.txt")
	if err := os.WriteFile(dirty, []byte("uncommitted"), 0644); err != nil {
		t.Fatal(err)
	}

	info = GetGitInfo(tmpDir)
	if !info.IsDirty {
		t.Error("IsDirty should be true after creating untracked file")
	}
}

// TestGitInfo_ZeroValue tests that the zero value of GitInfo is safe.
func TestGitInfo_ZeroValue(t *testing.T) {
	var info GitInfo
	if info.Branch != "" {
		t.Error("zero Branch should be empty")
	}
	if info.IsDirty {
		t.Error("zero IsDirty should be false")
	}
	if info.Ahead != 0 {
		t.Error("zero Ahead should be 0")
	}
	if info.Behind != 0 {
		t.Error("zero Behind should be 0")
	}
	if info.LastCommit != "" {
		t.Error("zero LastCommit should be empty")
	}
}
