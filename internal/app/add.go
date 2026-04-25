package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vlln/skit/internal/diagnose"
	"github.com/vlln/skit/internal/gitfetch"
	"github.com/vlln/skit/internal/lockfile"
	"github.com/vlln/skit/internal/metadata"
	"github.com/vlln/skit/internal/skill"
	"github.com/vlln/skit/internal/source"
	"github.com/vlln/skit/internal/store"
)

type AddRequest struct {
	Context    context.Context
	CWD        string
	Scope      Scope
	Source     string
	Skill      string
	Skills     []string
	All        bool
	IgnoreDeps bool
	FullDepth  bool
	NoActive   bool
	Force      bool
}
type AddResult struct {
	Entries              []lockfile.Entry `json:"entries"`
	StorePaths           []string         `json:"storePaths,omitempty"`
	DependencyEntries    []lockfile.Entry `json:"dependencyEntries,omitempty"`
	DependencyStorePaths []string         `json:"dependencyStorePaths,omitempty"`
	Warnings             []string         `json:"warnings,omitempty"`
	ActivePaths          []string         `json:"activePaths,omitempty"`
}

func Add(req AddRequest) (AddResult, error) {
	var result AddResult
	cwd := cleanCWD(req.CWD)
	paths := store.PathsFor(req.Scope, cwd)
	src, err := source.Parse(req.Source, source.WithSkill(req.Skill))
	if err != nil {
		return result, err
	}
	wanted := req.Skills
	if req.Skill != "" {
		wanted = append([]string{req.Skill}, wanted...)
	}
	if src.Skill != "" && len(wanted) > 0 {
		src.Warnings = append(src.Warnings, "inline skill selector ignored because --skill was provided")
	}
	if src.Skill != "" && len(wanted) == 0 {
		wanted = []string{src.Skill}
	}
	workRoot := src.Locator
	cleanup := func() {}
	resolvedRef := ""
	if isGitProvider(src.Type) {
		if err := resolveTreeSource(ctx(req.Context), &src); err != nil {
			return result, err
		}
		clone, err := gitfetch.Clone(ctx(req.Context), src.URL, src.Ref, paths.Tmp)
		if err != nil {
			return result, err
		}
		workRoot = clone.Dir
		src.Ref = clone.Ref
		resolvedRef = clone.ResolvedRef
		cleanup = func() { _ = gitfetch.RemoveUnder(clone.Dir, paths.Tmp) }
		defer cleanup()
	} else if src.Type != source.Local {
		return result, fmt.Errorf("provider %q is not implemented for install yet", src.Type)
	}

	discoverRoot := workRoot
	if src.Subpath != "" {
		discoverRoot = filepath.Join(workRoot, filepath.FromSlash(src.Subpath))
	}
	discoverOpts := parseOptionsForSource(src)
	discoverOpts.IncludeInternal = len(wanted) > 0
	discoverOpts.FullDepth = req.FullDepth
	discovered, warnings, err := skill.DiscoverWithOptions(discoverRoot, discoverOpts)
	if err != nil {
		return result, err
	}
	result.Warnings = append(result.Warnings, warnings...)
	selected, err := selectSkills(discovered, wanted, req.All)
	if err != nil {
		if len(result.Warnings) > 0 {
			return result, fmt.Errorf("%w; %s", err, strings.Join(result.Warnings, "; "))
		}
		return result, err
	}

	lock, err := lockfile.Read(paths.Lock)
	if err != nil {
		return result, err
	}
	session := addSession{
		ctx:        ctx(req.Context),
		paths:      paths,
		lock:       lock,
		result:     &result,
		ignoreDeps: req.IgnoreDeps,
		noActive:   req.NoActive,
		force:      req.Force,
		visiting:   map[string]bool{},
	}
	for _, parsed := range selected {
		entry, storePath, err := session.installParsed(src, parsed, workRoot, resolvedRef)
		if err != nil {
			return result, err
		}
		result.Entries = append(result.Entries, entry)
		result.StorePaths = append(result.StorePaths, storePath)
	}
	if err := writeLock(paths, req.Scope, cleanCWD(req.CWD), session.lock); err != nil {
		return result, err
	}
	return result, nil
}

type addSession struct {
	ctx        context.Context
	paths      store.Paths
	lock       lockfile.Lock
	result     *AddResult
	ignoreDeps bool
	noActive   bool
	force      bool
	replace    bool
	visiting   map[string]bool
}

func (s *addSession) installParsed(src source.Source, parsed skill.Skill, workRoot, resolvedRef string) (lockfile.Entry, string, error) {
	if s.visiting[parsed.Name] {
		return lockfile.Entry{}, "", fmt.Errorf("circular dependency involving %q", parsed.Name)
	}
	s.visiting[parsed.Name] = true
	defer delete(s.visiting, parsed.Name)

	var dependencyEdges []lockfile.Dependency
	var dependencyWarnings []string
	if len(parsed.Skit.Dependencies) > 0 {
		if s.ignoreDeps {
			warning := fmt.Sprintf("dependencies skipped for %s", parsed.Name)
			dependencyWarnings = appendUnique(dependencyWarnings, warning)
			s.result.Warnings = appendUnique(s.result.Warnings, warning)
		} else {
			edges, warnings, err := s.installDependencies(parsed)
			if err != nil {
				return lockfile.Entry{}, "", err
			}
			dependencyEdges = edges
			dependencyWarnings = warnings
		}
	}

	installed, err := store.InstallSnapshot(s.paths, parsed)
	if err != nil {
		return lockfile.Entry{}, "", err
	}
	entry := entryFor(src, parsed, installed.Hashes.Tree, installed.Hashes.SkillMD, workRoot)
	entry.Source.ResolvedRef = resolvedRef
	entry.Dependencies = dependencyEdges
	safetyWarnings, err := diagnose.SafetyWarnings(parsed.Root)
	if err != nil {
		return lockfile.Entry{}, "", err
	}
	entry.Warnings = appendUnique(entry.Warnings, parsed.Warnings...)
	entry.Warnings = appendUnique(entry.Warnings, src.Warnings...)
	entry.Warnings = appendUnique(entry.Warnings, dependencyWarnings...)
	entry.Warnings = appendUnique(entry.Warnings, safetyWarnings...)
	if s.replace {
		s.lock = lockfile.Put(s.lock, entry)
	} else {
		s.lock, err = lockfile.Add(s.lock, entry)
		if err != nil {
			return lockfile.Entry{}, "", err
		}
	}
	if !s.noActive {
		activePath, err := activate(s.paths, entry, installed.Path, s.force)
		if err != nil {
			return lockfile.Entry{}, "", err
		}
		s.result.ActivePaths = append(s.result.ActivePaths, activePath)
	}
	s.result.Warnings = appendUnique(s.result.Warnings, src.Warnings...)
	s.result.Warnings = appendUnique(s.result.Warnings, safetyWarnings...)
	return entry, installed.Path, nil
}

func (s *addSession) installDependencies(parent skill.Skill) ([]lockfile.Dependency, []string, error) {
	deps, err := normalizeDependencies(parent.Skit.Dependencies)
	if err != nil {
		return nil, nil, fmt.Errorf("%s dependencies: %w", parent.Name, err)
	}
	var edges []lockfile.Dependency
	var warnings []string
	for _, dep := range deps {
		entry, err := s.installDependency(dep)
		if err != nil {
			if dep.Optional {
				warning := fmt.Sprintf("optional dependency for %s failed: %v", parent.Name, err)
				warnings = appendUnique(warnings, warning)
				s.result.Warnings = appendUnique(s.result.Warnings, warning)
				continue
			}
			return nil, nil, fmt.Errorf("dependency for %s failed: %w", parent.Name, err)
		}
		edges = append(edges, lockfile.Dependency{Name: entry.Name, Source: entry.Source, Optional: dep.Optional})
	}
	return edges, warnings, nil
}

func (s *addSession) installDependency(dep metadata.Dependency) (lockfile.Entry, error) {
	src, err := source.Parse(dep.Source, source.WithSkill(dep.Skill))
	if err != nil {
		return lockfile.Entry{}, err
	}
	if dep.Ref != "" {
		if src.Ref != "" && src.Ref != dep.Ref {
			return lockfile.Entry{}, fmt.Errorf("dependency ref conflict for %s", dep.Source)
		}
		src.Ref = dep.Ref
	}
	parsed, srcOut, workRoot, resolvedRef, cleanup, err := resolveOneForInstall(s.ctx, s.paths, src)
	if err != nil {
		return lockfile.Entry{}, err
	}
	defer cleanup()
	entry, storePath, err := s.installParsed(srcOut, parsed, workRoot, resolvedRef)
	if err == nil {
		s.result.DependencyEntries = append(s.result.DependencyEntries, entry)
		s.result.DependencyStorePaths = append(s.result.DependencyStorePaths, storePath)
	}
	return entry, err
}

func normalizeDependencies(deps []metadata.Dependency) ([]metadata.Dependency, error) {
	seen := map[string]metadata.Dependency{}
	var out []metadata.Dependency
	for _, dep := range deps {
		key := dep.Source + "\x00" + dep.Skill
		if prev, ok := seen[key]; ok {
			if prev.Ref != dep.Ref {
				return nil, fmt.Errorf("conflicting refs for dependency %s", dep.Source)
			}
			if prev.Optional != dep.Optional {
				prev.Optional = prev.Optional && dep.Optional
				seen[key] = prev
			}
			continue
		}
		seen[key] = dep
		out = append(out, dep)
	}
	for i, dep := range out {
		if normalized, ok := seen[dep.Source+"\x00"+dep.Skill]; ok {
			out[i] = normalized
		}
	}
	return out, nil
}
