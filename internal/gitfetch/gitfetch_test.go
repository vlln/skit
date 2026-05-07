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

func TestCloneWithOptionsSparsePath(t *testing.T) {
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
	skillDir := filepath.Join(repo, "skills", "demo")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: demo\ndescription: Demo.\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "init")

	result, err := CloneWithOptions(context.Background(), repo, "", t.TempDir(), CloneOptions{SparsePaths: []string{"skills/demo"}})
	if err != nil {
		t.Fatal(err)
	}
	defer Remove(result.Dir)
	if _, err := os.Stat(filepath.Join(result.Dir, "skills", "demo", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(result.Dir, "README.md")); !os.IsNotExist(err) {
		t.Fatalf("README.md should not be checked out in sparse clone: %v", err)
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

func TestGitHubTokenEnvPrecedence(t *testing.T) {
	t.Setenv("SKIT_GITHUB_TOKEN", "skit-token")
	t.Setenv("GITHUB_TOKEN", "github-token")
	t.Setenv("GH_TOKEN", "gh-token")
	if got := githubToken(context.Background()); got != "skit-token" {
		t.Fatalf("token = %q", got)
	}
}

func TestGitEnvAddsGitHubAuthHeader(t *testing.T) {
	t.Setenv("SKIT_GITHUB_TOKEN", "skit-token")
	env := gitEnv(context.Background())
	if !containsEnv(env, "GIT_CONFIG_KEY_0=http.https://github.com/.extraheader") {
		t.Fatalf("missing git config key: %#v", env)
	}
	if !containsEnv(env, "GIT_CONFIG_VALUE_0=AUTHORIZATION: bearer skit-token") {
		t.Fatalf("missing auth header: %#v", env)
	}
}

func TestGitHubHTTPSAuthRetryPredicates(t *testing.T) {
	t.Setenv("SKIT_GITHUB_TOKEN", "bad-token")
	err := errors.New("remote: repository not found")
	if !isGitHubHTTPSURL("https://github.com/owner/repo.git") {
		t.Fatal("expected GitHub HTTPS URL")
	}
	if !shouldRetryWithGitHubAuth(context.Background(), err) {
		t.Fatal("expected auth failure to retry with token")
	}
	if isGitHubHTTPSURL("https://gitlab.com/owner/repo.git") {
		t.Fatal("should not match non-GitHub URL")
	}
}

func containsEnv(env []string, want string) bool {
	for _, item := range env {
		if item == want {
			return true
		}
	}
	return false
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
