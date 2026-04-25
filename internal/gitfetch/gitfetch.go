package gitfetch

import (
	"bytes"
	"context"
	"fmt"
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
	if !fullSHA.MatchString(ref) {
		args = append(args, "--depth", "1")
	}
	if ref != "" && !fullSHA.MatchString(ref) {
		args = append(args, "--branch", ref)
	}
	args = append(args, url, dir)
	if err := runGit(cloneCtx, "", args...); err != nil {
		return result, classifyCloneError(url, err, cloneCtx.Err())
	}
	if fullSHA.MatchString(ref) {
		if err := runGit(ctx, dir, "checkout", "--detach", ref); err != nil {
			if fetchErr := runGit(ctx, dir, "fetch", "--depth", "1", "origin", ref); fetchErr != nil {
				return result, fmt.Errorf("checkout %s: %w; fetch by commit also failed: %v", ref, err, fetchErr)
			}
			if err := runGit(ctx, dir, "checkout", "--detach", ref); err != nil {
				return result, fmt.Errorf("checkout %s after fetch: %w", ref, err)
			}
		}
	}
	resolved, err := outputGit(ctx, dir, "rev-parse", "HEAD")
	if err != nil {
		return result, err
	}
	actualRef := ref
	if actualRef == "" {
		actualRef = defaultBranch(ctx, dir)
	}
	cleanup = false
	return CloneResult{Dir: dir, Ref: actualRef, ResolvedRef: strings.TrimSpace(resolved)}, nil
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
	isAuth := strings.Contains(lower, "authentication failed") ||
		strings.Contains(lower, "could not read username") ||
		strings.Contains(lower, "permission denied") ||
		strings.Contains(lower, "repository not found")
	if isTimeout {
		msg = "clone timed out after 60s; check network access and repository authentication"
	} else if isAuth {
		msg = "authentication failed or repository is not accessible; check repository access and git credentials"
	} else {
		msg = "failed to clone " + url + ": " + msg
	}
	return &CloneError{URL: url, Message: msg, Timeout: isTimeout, Auth: isAuth}
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
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_LFS_SKIP_SMUDGE=1")
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
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_LFS_SKIP_SMUDGE=1")
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

func remoteRefs(ctx context.Context, url string) (map[string]bool, error) {
	out, err := outputGit(ctx, "", "ls-remote", "--heads", "--tags", url)
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

func defaultBranch(ctx context.Context, dir string) string {
	out, err := outputGit(ctx, dir, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}
