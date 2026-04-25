package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vlln/skit/internal/gitfetch"
	"github.com/vlln/skit/internal/lockfile"
	"github.com/vlln/skit/internal/skill"
	"github.com/vlln/skit/internal/source"
	"github.com/vlln/skit/internal/store"
)

func sourceFromLock(src lockfile.Source) (source.Source, error) {
	if src.Type == "" {
		return source.Source{}, fmt.Errorf("lock source type is required")
	}
	out := source.Source{
		Type:        source.Type(src.Type),
		Locator:     src.Locator,
		URL:         src.URL,
		Ref:         src.Ref,
		Subpath:     src.Subpath,
		Skill:       src.Skill,
		Implemented: src.Type == string(source.Local) || isGitProvider(source.Type(src.Type)),
	}
	if out.Type == source.Local && out.Locator == "" {
		return out, fmt.Errorf("local source locator is required")
	}
	if isGitProvider(out.Type) && out.URL == "" {
		return out, fmt.Errorf("%s source url is required", out.Type)
	}
	return out, nil
}
func selectSkills(skills []skill.Skill, wanted []string, all bool) ([]skill.Skill, error) {
	if all {
		if len(skills) == 0 {
			return nil, fmt.Errorf("no skills found")
		}
		return skills, nil
	}
	if len(wanted) > 0 {
		byName := map[string]skill.Skill{}
		for _, s := range skills {
			byName[s.Name] = s
		}
		var selected []skill.Skill
		for _, name := range wanted {
			s, ok := byName[name]
			if !ok {
				return nil, fmt.Errorf("skill %q not found", name)
			}
			selected = append(selected, s)
		}
		return selected, nil
	}
	if len(skills) == 1 {
		return skills, nil
	}
	if len(skills) == 0 {
		return nil, fmt.Errorf("no skills found")
	}
	return nil, fmt.Errorf("source contains multiple skills; use source@skill, --skill <name...>, or --all")
}

func resolveOne(ctx context.Context, paths store.Paths, src source.Source) (skill.Skill, source.Source, func(), error) {
	parsed, srcOut, _, _, cleanup, err := resolveOneForInstall(ctx, paths, src)
	return parsed, srcOut, cleanup, err
}

func resolveOneForInstall(ctx context.Context, paths store.Paths, src source.Source) (skill.Skill, source.Source, string, string, func(), error) {
	workRoot := src.Locator
	resolvedRef := ""
	cleanup := func() {}
	if isGitProvider(src.Type) {
		if err := resolveTreeSource(ctx, &src); err != nil {
			return skill.Skill{}, src, workRoot, resolvedRef, cleanup, err
		}
		clone, err := gitfetch.Clone(ctx, src.URL, src.Ref, paths.Tmp)
		if err != nil {
			return skill.Skill{}, src, workRoot, resolvedRef, cleanup, err
		}
		workRoot = clone.Dir
		src.Ref = clone.Ref
		resolvedRef = clone.ResolvedRef
		cleanup = func() { _ = gitfetch.RemoveUnder(clone.Dir, paths.Tmp) }
	} else if src.Type != source.Local {
		return skill.Skill{}, src, workRoot, resolvedRef, cleanup, fmt.Errorf("provider %q is not implemented yet", src.Type)
	}
	discoverRoot := workRoot
	if src.Subpath != "" {
		discoverRoot = filepath.Join(workRoot, filepath.FromSlash(src.Subpath))
	}
	discovered, _, err := skill.DiscoverWithOptions(discoverRoot, parseOptionsForSource(src))
	if err != nil {
		cleanup()
		return skill.Skill{}, src, workRoot, resolvedRef, func() {}, err
	}
	var wanted []string
	if src.Skill != "" {
		wanted = []string{src.Skill}
	}
	selected, err := selectSkills(discovered, wanted, false)
	if err != nil {
		cleanup()
		return skill.Skill{}, src, workRoot, resolvedRef, func() {}, err
	}
	return selected[0], src, workRoot, resolvedRef, cleanup, nil
}

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

func entryFor(src source.Source, parsed skill.Skill, treeHash, skillMDHash, workRoot string) lockfile.Entry {
	lockSource := lockfile.Source{
		Type:    string(src.Type),
		Locator: src.Locator,
		URL:     src.URL,
		Ref:     src.Ref,
		Subpath: sourceSubpath(src, parsed.Root, workRoot),
		Skill:   parsed.Name,
	}
	return lockfile.Entry{
		Name:        parsed.Name,
		Description: parsed.Description,
		Source:      lockSource,
		Hashes:      lockfile.Hashes{Tree: treeHash, SkillMD: skillMDHash},
	}
}

func sourceSubpath(src source.Source, skillRoot, workRoot string) string {
	if src.Type == source.Local {
		return relativeSubpath(src.Locator, skillRoot)
	}
	return relativeSubpath(workRoot, skillRoot)
}

func sourceSubpathForInspect(src source.Source, skillRoot string) string {
	if src.Subpath != "" {
		return src.Subpath
	}
	if src.Type == source.Local {
		return relativeSubpath(src.Locator, skillRoot)
	}
	return ""
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

func relativeSubpath(root, child string) string {
	rel, err := filepath.Rel(root, child)
	if err != nil || rel == "." {
		return ""
	}
	return filepath.ToSlash(rel)
}
func refForInstall(src lockfile.Source) string {
	if src.ResolvedRef != "" {
		return src.ResolvedRef
	}
	return src.Ref
}
