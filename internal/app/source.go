package app

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/vlln/skit/internal/gitfetch"
	"github.com/vlln/skit/internal/skill"
	"github.com/vlln/skit/internal/source"
)

func resolveTreeSource(ctx context.Context, src *source.Source) error {
	if src.UnresolvedTreePath == "" {
		return nil
	}
	resolved, err := gitfetch.ResolveTreePath(ctx, src.URL, src.UnresolvedTreePath)
	if err != nil {
		return err
	}
	if !resolved.Found {
		src.Warnings = appendUnique(src.Warnings, "could not resolve full tree ref; using first path segment as ref")
	}
	src.Ref = resolved.Ref
	src.Subpath = resolved.Subpath
	return nil
}

func sourceSubpath(src source.Source, skillRoot, workRoot string) string {
	if src.Type == source.Local {
		return relativeSubpath(src.Locator, skillRoot)
	}
	return relativeSubpath(workRoot, skillRoot)
}

func parseOptionsForSource(src source.Source) skill.ParseOptions {
	opts := skill.ParseOptions{
		AllowNameMismatch: isGitProvider(src.Type),
		IncludeInternal:   true,
	}
	if isGitProvider(src.Type) && src.Subpath == "" {
		parts := strings.Split(src.Locator, "/")
		if len(parts) > 0 && parts[len(parts)-1] != "" {
			opts.ExpectedBasename = strings.TrimSuffix(parts[len(parts)-1], ".git")
		}
	}
	return opts
}

func isGitProvider(t source.Type) bool {
	return t == source.GitHub || t == source.GitLab || t == source.Git
}

func cloneOptions(src source.Source) gitfetch.CloneOptions {
	if isGitProvider(src.Type) && src.Subpath != "" {
		return gitfetch.CloneOptions{SparsePaths: []string{src.Subpath}}
	}
	return gitfetch.CloneOptions{}
}

func relativeSubpath(root, child string) string {
	rel, err := filepath.Rel(root, child)
	if err != nil || rel == "." {
		return ""
	}
	return filepath.ToSlash(rel)
}
