package gitfetch

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const cloneTimeout = 60 * time.Second

var fullSHA = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

type CloneResult struct {
	Dir         string
	Ref         string
	ResolvedRef string
}

type CloneOptions struct {
	SparsePaths []string
}

type TreeResolution struct {
	Ref     string
	Subpath string
	Found   bool
}

type CloneError struct {
	URL     string
	Message string
	Timeout bool
	Auth    bool
}

func (e *CloneError) Error() string {
	return e.Message
}

func Clone(ctx context.Context, url, ref, tmpParent string) (CloneResult, error) {
	return CloneWithOptions(ctx, url, ref, tmpParent, CloneOptions{})
}

func CloneWithOptions(ctx context.Context, url, ref, tmpParent string, opts CloneOptions) (CloneResult, error) {
	var result CloneResult
	if _, err := exec.LookPath("git"); err != nil {
		return result, fmt.Errorf("git executable not found")
	}
	if err := os.MkdirAll(tmpParent, 0755); err != nil {
		return result, err
	}
	dir, err := os.MkdirTemp(tmpParent, "git-*")
	if err != nil {
		return result, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(dir)
		}
	}()

	cloneCtx, cancel := context.WithTimeout(ctx, cloneTimeout)
	defer cancel()
	args := []string{"clone"}
	sparsePaths := cleanSparsePaths(opts.SparsePaths)
	if len(sparsePaths) > 0 {
		args = append(args, "--filter=blob:none", "--sparse", "--no-checkout")
	}
	if !fullSHA.MatchString(ref) {
		args = append(args, "--depth", "1", "--single-branch")
	}
	if ref != "" && !fullSHA.MatchString(ref) {
		args = append(args, "--branch", ref)
	}
	args = append(args, url, dir)
	run := runGit
	output := outputGit
	var cloneErr error
	if isGitHubHTTPSURL(url) {
		run = runGitWithoutGitHubAuth
		output = outputGitWithoutGitHubAuth
		cloneErr = run(cloneCtx, "", args...)
		if cloneErr != nil && shouldRetryWithGitHubAuth(ctx, cloneErr) {
			if resetErr := resetCloneDir(dir); resetErr != nil {
				return result, resetErr
			}
			cloneErr = runGit(cloneCtx, "", args...)
			if cloneErr == nil {
				run = runGit
				output = outputGit
			}
		}
	} else {
		cloneErr = run(cloneCtx, "", args...)
	}
	if cloneErr != nil {
		return result, classifyCloneError(url, cloneErr, cloneCtx.Err())
	}
	if len(sparsePaths) > 0 {
		if err := run(ctx, dir, append([]string{"sparse-checkout", "set", "--no-cone", "--"}, sparsePatterns(sparsePaths)...)...); err != nil {
			return result, fmt.Errorf("configure sparse checkout: %w", err)
		}
		if !fullSHA.MatchString(ref) {
			if err := run(ctx, dir, "checkout", "--force"); err != nil {
				return result, fmt.Errorf("checkout sparse paths: %w", err)
			}
		}
	}
	if fullSHA.MatchString(ref) {
		if err := run(ctx, dir, "checkout", "--detach", ref); err != nil {
			if fetchErr := run(ctx, dir, "fetch", "--depth", "1", "origin", ref); fetchErr != nil {
				return result, fmt.Errorf("checkout %s: %w; fetch by commit also failed: %v", ref, err, fetchErr)
			}
			if err := run(ctx, dir, "checkout", "--detach", ref); err != nil {
				return result, fmt.Errorf("checkout %s after fetch: %w", ref, err)
			}
		}
	}
	resolved, err := output(ctx, dir, "rev-parse", "HEAD")
	if err != nil {
		return result, err
	}
	actualRef := ref
	if actualRef == "" {
		actualRef = defaultBranch(ctx, dir, output)
	}
	cleanup = false
	return CloneResult{Dir: dir, Ref: actualRef, ResolvedRef: strings.TrimSpace(resolved)}, nil
}

func cleanSparsePaths(paths []string) []string {
	var cleaned []string
	seen := map[string]bool{}
	for _, path := range paths {
		path = strings.Trim(strings.ReplaceAll(path, "\\", "/"), "/")
		if path == "" || path == "." || strings.HasPrefix(path, "../") || strings.Contains(path, "/../") {
			continue
		}
		if !seen[path] {
			cleaned = append(cleaned, path)
			seen[path] = true
		}
	}
	return cleaned
}

func sparsePatterns(paths []string) []string {
	patterns := make([]string, 0, len(paths))
	for _, path := range paths {
		patterns = append(patterns, "/"+strings.Trim(path, "/")+"/")
	}
	return patterns
}

func ResolveTreePath(ctx context.Context, url, treePath string) (TreeResolution, error) {
	var result TreeResolution
	treePath = strings.Trim(treePath, "/")
	if treePath == "" {
		return result, nil
	}
	refs, err := remoteRefs(ctx, url)
	if err != nil {
		return result, err
	}
	parts := strings.Split(treePath, "/")
	for i := len(parts); i >= 1; i-- {
		candidate := strings.Join(parts[:i], "/")
		if refs[candidate] || fullSHA.MatchString(candidate) {
			result.Ref = candidate
			result.Subpath = strings.Join(parts[i:], "/")
			result.Found = true
			return result, nil
		}
	}
	result.Ref = parts[0]
	if len(parts) > 1 {
		result.Subpath = strings.Join(parts[1:], "/")
	}
	return result, nil
}

func classifyCloneError(url string, err error, ctxErr error) error {
	msg := err.Error()
	lower := strings.ToLower(msg)
	isTimeout := ctxErr == context.DeadlineExceeded || strings.Contains(lower, "timed out") || strings.Contains(lower, "timeout")
	isAuth := isAuthGitError(err)
	if isTimeout {
		msg = "clone timed out after 60s; check network access and repository authentication"
	} else if isAuth {
		msg = "authentication failed or repository is not accessible; check repository access and git credentials"
	} else {
		msg = "failed to clone " + url + ": " + msg
	}
	return &CloneError{URL: url, Message: msg, Timeout: isTimeout, Auth: isAuth}
}

func isAuthGitError(err error) bool {
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "authentication failed") ||
		strings.Contains(lower, "could not read username") ||
		strings.Contains(lower, "permission denied") ||
		strings.Contains(lower, "repository not found")
}

func shouldRetryWithGitHubAuth(ctx context.Context, err error) bool {
	return githubToken(ctx) != "" && isAuthGitError(err)
}

func isGitHubHTTPSURL(rawURL string) bool {
	u, parseErr := url.Parse(rawURL)
	return parseErr == nil && u.Scheme == "https" && strings.EqualFold(u.Hostname(), "github.com")
}

func resetCloneDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return os.MkdirAll(dir, 0755)
}

func Remove(dir string) error {
	return RemoveUnder(dir, os.TempDir())
}

func RemoveUnder(dir, parent string) error {
	if dir == "" {
		return nil
	}
	clean, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	base, err := filepath.Abs(parent)
	if err != nil {
		return err
	}
	if clean != base && !strings.HasPrefix(clean, base+string(filepath.Separator)) {
		return fmt.Errorf("refusing to remove git temp dir outside parent: %s", dir)
	}
	return os.RemoveAll(clean)
}

func runGit(ctx context.Context, dir string, args ...string) error {
	return runGitWithEnv(ctx, dir, gitEnv(ctx), args...)
}

func runGitWithoutGitHubAuth(ctx context.Context, dir string, args ...string) error {
	return runGitWithEnv(ctx, dir, gitEnvWithoutGitHubAuth(), args...)
}

func runGitWithEnv(ctx context.Context, dir string, env []string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func outputGit(ctx context.Context, dir string, args ...string) (string, error) {
	return outputGitWithEnv(ctx, dir, gitEnv(ctx), args...)
}

func outputGitWithoutGitHubAuth(ctx context.Context, dir string, args ...string) (string, error) {
	return outputGitWithEnv(ctx, dir, gitEnvWithoutGitHubAuth(), args...)
}

func outputGitWithEnv(ctx context.Context, dir string, env []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return string(out), nil
}

func gitEnv(ctx context.Context) []string {
	env := gitEnvWithoutGitHubAuth()
	token := githubToken(ctx)
	if token == "" {
		return env
	}
	return append(env,
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=http.https://github.com/.extraheader",
		"GIT_CONFIG_VALUE_0=AUTHORIZATION: bearer "+token,
	)
}

func gitEnvWithoutGitHubAuth() []string {
	return append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_LFS_SKIP_SMUDGE=1")
}

func githubToken(ctx context.Context) string {
	if ctx == nil {
		ctx = context.Background()
	}
	for _, key := range []string{"SKIT_GITHUB_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	if _, err := exec.LookPath("gh"); err != nil {
		return ""
	}
	tokenCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(tokenCtx, "gh", "auth", "token")
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func remoteRefs(ctx context.Context, url string) (map[string]bool, error) {
	out, err := outputGitWithoutGitHubAuth(ctx, "", "ls-remote", "--heads", "--tags", url)
	if err != nil {
		return nil, err
	}
	refs := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		name := fields[1]
		for _, prefix := range []string{"refs/heads/", "refs/tags/"} {
			if strings.HasPrefix(name, prefix) {
				refs[strings.TrimPrefix(name, prefix)] = true
			}
		}
	}
	return refs, nil
}

func defaultBranch(ctx context.Context, dir string, output func(context.Context, string, ...string) (string, error)) string {
	out, err := output(ctx, dir, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}
