package vitals

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestRenderGitDirty(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// Empty cwd and a non-repo directory both report "no git".
	if got := renderGitDirty("", false); got != "no git" {
		t.Errorf("renderGitDirty(\"\") = %q", got)
	}
	if got := renderGitDirty(t.TempDir(), false); got != "no git" {
		t.Errorf("renderGitDirty(non-repo) = %q", got)
	}

	// Fresh clean repo.
	repo := t.TempDir()
	git(t, repo, "init", "-q")
	if got := renderGitDirty(repo, false); got != "✅ clean" {
		t.Errorf("renderGitDirty(clean) = %q, want ✅ clean", got)
	}

	// Untracked file.
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := renderGitDirty(repo, false); got != "📝 ?1" {
		t.Errorf("renderGitDirty(untracked) = %q, want 📝 ?1", got)
	}

	// Staged addition.
	git(t, repo, "add", "a.txt")
	if got := renderGitDirty(repo, false); got != "📝 +1" {
		t.Errorf("renderGitDirty(added) = %q, want 📝 +1", got)
	}

	// Commit, then modify -> "!1".
	git(t, repo, "commit", "-q", "-m", "init")
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := renderGitDirty(repo, false); got != "📝 !1" {
		t.Errorf("renderGitDirty(modified) = %q, want 📝 !1", got)
	}
}
