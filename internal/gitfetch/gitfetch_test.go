package gitfetch

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCloneLocalRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	repo := t.TempDir()
	git(t, repo, "init", "-b", "main")
	git(t, repo, "config", "user.email", "test@example.com")
	git(t, repo, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("demo\n"), 0644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "init")

	result, err := Clone(context.Background(), repo, "", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer Remove(result.Dir)
	if result.Ref != "main" || result.ResolvedRef == "" {
		t.Fatalf("result = %#v", result)
	}
	if _, err := os.Stat(filepath.Join(result.Dir, "README.md")); err != nil {
		t.Fatal(err)
	}
}

func TestResolveTreePathPrefersLongestRef(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	repo := t.TempDir()
	git(t, repo, "init", "-b", "main")
	git(t, repo, "config", "user.email", "test@example.com")
	git(t, repo, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("demo\n"), 0644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "init")
	git(t, repo, "checkout", "-b", "feature/foo")

	got, err := ResolveTreePath(context.Background(), repo, "feature/foo/skills/bar")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Found || got.Ref != "feature/foo" || got.Subpath != "skills/bar" {
		t.Fatalf("got = %#v", got)
	}
}

func TestRemoveUnderAllowsProjectTmp(t *testing.T) {
	parent := filepath.Join(t.TempDir(), ".skit", "tmp")
	dir := filepath.Join(parent, "git-demo")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := RemoveUnder(dir, parent); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("dir still exists or stat failed: %v", err)
	}
}

func TestClassifyCloneErrorAuth(t *testing.T) {
	err := classifyCloneError("https://github.com/o/r.git", errors.New("fatal: Authentication failed"), nil)
	var cloneErr *CloneError
	if !errors.As(err, &cloneErr) || !cloneErr.Auth {
		t.Fatalf("err = %#v", err)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}
