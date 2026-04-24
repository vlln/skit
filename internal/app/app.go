package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/vlln/skit/internal/diagnose"
	"github.com/vlln/skit/internal/gitfetch"
	"github.com/vlln/skit/internal/hash"
	"github.com/vlln/skit/internal/lockfile"
	"github.com/vlln/skit/internal/metadata"
	"github.com/vlln/skit/internal/search"
	"github.com/vlln/skit/internal/skill"
	"github.com/vlln/skit/internal/source"
	"github.com/vlln/skit/internal/store"
)

type Scope = store.Scope

const (
	Project = store.Project
	Global  = store.Global
)

type AddRequest struct {
	Context    context.Context
	CWD        string
	Scope      Scope
	Source     string
	Skill      string
	All        bool
	IgnoreDeps bool
	FullDepth  bool
}

type SearchRequest struct {
	Context context.Context
	Query   string
	Limit   int
}

type SearchResult = search.Result

func Search(req SearchRequest) ([]SearchResult, error) {
	client := search.NewClient()
	return client.Search(ctx(req.Context), req.Query, req.Limit)
}

type AddResult struct {
	Entries              []lockfile.Entry `json:"entries"`
	StorePaths           []string         `json:"storePaths,omitempty"`
	DependencyEntries    []lockfile.Entry `json:"dependencyEntries,omitempty"`
	DependencyStorePaths []string         `json:"dependencyStorePaths,omitempty"`
	Warnings             []string         `json:"warnings,omitempty"`
}

func Add(req AddRequest) (AddResult, error) {
	var result AddResult
	cwd := cleanCWD(req.CWD)
	paths := store.PathsFor(req.Scope, cwd)
	src, err := source.Parse(req.Source, source.WithSkill(req.Skill))
	if err != nil {
		return result, err
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
		return result, fmt.Errorf("provider %q is not implemented for add yet", src.Type)
	}

	discoverRoot := workRoot
	if src.Subpath != "" {
		discoverRoot = filepath.Join(workRoot, filepath.FromSlash(src.Subpath))
	}
	discoverOpts := parseOptionsForSource(src)
	discoverOpts.IncludeInternal = req.Skill != ""
	discoverOpts.FullDepth = req.FullDepth
	discovered, warnings, err := skill.DiscoverWithOptions(discoverRoot, discoverOpts)
	if err != nil {
		return result, err
	}
	result.Warnings = append(result.Warnings, warnings...)
	selected, err := selectSkills(discovered, src.Skill, req.All)
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
	if err := lockfile.Write(paths.Lock, session.lock); err != nil {
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

type ListRequest struct {
	CWD   string
	Scope Scope
}

func List(req ListRequest) ([]lockfile.Entry, error) {
	paths := store.PathsFor(req.Scope, cleanCWD(req.CWD))
	lock, err := lockfile.Read(paths.Lock)
	if err != nil {
		return nil, err
	}
	names := lockfile.Names(lock)
	out := make([]lockfile.Entry, 0, len(names))
	for _, name := range names {
		out = append(out, lock.Skills[name])
	}
	return out, nil
}

type UpdateRequest struct {
	Context    context.Context
	CWD        string
	Scope      Scope
	Name       string
	IgnoreDeps bool
}

type UpdateResult = AddResult

func Update(req UpdateRequest) (UpdateResult, error) {
	var result UpdateResult
	paths := store.PathsFor(req.Scope, cleanCWD(req.CWD))
	lock, err := lockfile.Read(paths.Lock)
	if err != nil {
		return result, err
	}
	names := lockfile.Names(lock)
	if req.Name != "" {
		if _, ok := lock.Skills[req.Name]; !ok {
			return result, fmt.Errorf("%s is not installed", req.Name)
		}
		names = []string{req.Name}
	}
	session := addSession{
		ctx:        ctx(req.Context),
		paths:      paths,
		lock:       lock,
		result:     &result,
		ignoreDeps: req.IgnoreDeps,
		replace:    true,
		visiting:   map[string]bool{},
	}
	for _, name := range names {
		current := session.lock.Skills[name]
		if current.Incomplete {
			result.Warnings = appendUnique(result.Warnings, fmt.Sprintf("skipped incomplete entry %s", name))
			continue
		}
		src, err := sourceFromLock(current.Source)
		if err != nil {
			return result, err
		}
		parsed, srcOut, workRoot, resolvedRef, cleanup, err := resolveOneForInstall(session.ctx, paths, src)
		if err != nil {
			cleanup()
			return result, err
		}
		entry, storePath, err := session.installParsed(srcOut, parsed, workRoot, resolvedRef)
		cleanup()
		if err != nil {
			return result, err
		}
		result.Entries = append(result.Entries, entry)
		result.StorePaths = append(result.StorePaths, storePath)
	}
	if err := lockfile.Write(paths.Lock, session.lock); err != nil {
		return result, err
	}
	return result, nil
}

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

type RemoveRequest struct {
	CWD   string
	Scope Scope
	Name  string
}

func Remove(req RemoveRequest) (bool, error) {
	paths := store.PathsFor(req.Scope, cleanCWD(req.CWD))
	lock, err := lockfile.Read(paths.Lock)
	if err != nil {
		return false, err
	}
	next, removed := lockfile.Remove(lock, req.Name)
	if !removed {
		return false, nil
	}
	return true, lockfile.Write(paths.Lock, next)
}

type InstallRequest struct {
	CWD   string
	Scope Scope
}

type InstallResult struct {
	Restored []lockfile.Entry
	Skipped  []lockfile.Entry
}

func Install(req InstallRequest) (InstallResult, error) {
	var result InstallResult
	paths := store.PathsFor(req.Scope, cleanCWD(req.CWD))
	lock, err := lockfile.Read(paths.Lock)
	if err != nil {
		return result, err
	}
	for _, name := range lockfile.Names(lock) {
		entry := lock.Skills[name]
		if entry.Incomplete {
			result.Skipped = append(result.Skipped, entry)
			continue
		}
		if exists(storePath(paths, entry)) {
			if err := verifyStoreEntry(paths, entry); err != nil {
				return result, err
			}
			result.Restored = append(result.Restored, entry)
			continue
		}
		root := entry.Source.Locator
		cleanup := func() {}
		if isGitProvider(source.Type(entry.Source.Type)) {
			clone, err := gitfetch.Clone(ctx(context.Background()), entry.Source.URL, refForInstall(entry.Source), paths.Tmp)
			if err != nil {
				return result, err
			}
			root = clone.Dir
			cleanup = func() { _ = gitfetch.RemoveUnder(clone.Dir, paths.Tmp) }
			defer cleanup()
		} else if entry.Source.Type != string(source.Local) {
			return result, fmt.Errorf("cannot restore %q: provider %q is not implemented", entry.Name, entry.Source.Type)
		}
		if entry.Source.Subpath != "" {
			root = filepath.Join(root, filepath.FromSlash(entry.Source.Subpath))
		}
		src, err := sourceFromLock(entry.Source)
		if err != nil {
			return result, err
		}
		parsed, err := skill.ParseDirWithOptions(root, parseOptionsForSource(src))
		if err != nil {
			return result, err
		}
		installed, err := store.InstallSnapshot(paths, parsed)
		if err != nil {
			return result, err
		}
		if installed.Hashes.Tree != entry.Hashes.Tree || installed.Hashes.SkillMD != entry.Hashes.SkillMD {
			return result, fmt.Errorf("hash mismatch restoring %q", entry.Name)
		}
		result.Restored = append(result.Restored, entry)
	}
	return result, nil
}

type InspectRequest struct {
	Context context.Context
	CWD     string
	Scope   Scope
	Target  string
	Skill   string
}

type InspectResult struct {
	Name         string                `json:"name"`
	Description  string                `json:"description"`
	Source       lockfile.Source       `json:"source"`
	Hashes       lockfile.Hashes       `json:"hashes"`
	Dependencies []lockfile.Dependency `json:"dependencies,omitempty"`
	StorePath    string                `json:"storePath,omitempty"`
	Files        []string              `json:"files,omitempty"`
	Requires     SkillRequires         `json:"requires"`
	Warnings     []string              `json:"warnings,omitempty"`
	FromLock     bool                  `json:"fromLock"`
}

type SkillRequires struct {
	Bins    []string `json:"bins,omitempty"`
	AnyBins []string `json:"anyBins,omitempty"`
	Env     []string `json:"env,omitempty"`
	Config  []string `json:"config,omitempty"`
}

func Inspect(req InspectRequest) (InspectResult, error) {
	var result InspectResult
	if req.Target == "" {
		return result, fmt.Errorf("inspect target is required")
	}
	cwd := cleanCWD(req.CWD)
	paths := store.PathsFor(req.Scope, cwd)
	lock, err := lockfile.Read(paths.Lock)
	if err != nil {
		return result, err
	}
	if entry, ok := lock.Skills[req.Target]; ok {
		result.FromLock = true
		result.Name = entry.Name
		result.Description = entry.Description
		result.Source = entry.Source
		result.Hashes = entry.Hashes
		result.Dependencies = entry.Dependencies
		result.Warnings = append(result.Warnings, entry.Warnings...)
		result.StorePath = storePath(paths, entry)
		if exists(result.StorePath) {
			parsed, err := skill.ParseDir(result.StorePath)
			if err != nil {
				return result, err
			}
			result.Requires = requiresFrom(parsed)
			result.Warnings = appendUnique(result.Warnings, parsed.Warnings...)
			result.Files, err = fileList(parsed.Root)
			if err != nil {
				return result, err
			}
			safetyWarnings, err := diagnose.SafetyWarnings(parsed.Root)
			if err != nil {
				return result, err
			}
			result.Warnings = appendUnique(result.Warnings, safetyWarnings...)
			hashes, err := hash.Tree(parsed.Root, parsed.File)
			if err != nil {
				return result, err
			}
			if hashes.Tree != entry.Hashes.Tree || hashes.SkillMD != entry.Hashes.SkillMD {
				result.Warnings = append(result.Warnings, "store content hash differs from lock")
			}
		} else {
			result.Warnings = append(result.Warnings, "store entry is missing; run skit install")
		}
		return result, nil
	}

	src, err := source.Parse(req.Target, source.WithSkill(req.Skill))
	if err != nil {
		return result, err
	}
	parsed, srcOut, cleanup, err := resolveOne(ctx(req.Context), paths, src)
	if err != nil {
		return result, err
	}
	defer cleanup()
	hashes, err := hash.Tree(parsed.Root, parsed.File)
	if err != nil {
		return result, err
	}
	result.Name = parsed.Name
	result.Description = parsed.Description
	result.Source = lockfile.Source{
		Type:    string(srcOut.Type),
		Locator: srcOut.Locator,
		URL:     srcOut.URL,
		Ref:     srcOut.Ref,
		Subpath: sourceSubpathForInspect(srcOut, parsed.Root),
		Skill:   parsed.Name,
	}
	result.Hashes = lockfile.Hashes{Tree: hashes.Tree, SkillMD: hashes.SkillMD}
	result.Requires = requiresFrom(parsed)
	result.Warnings = appendUnique(result.Warnings, srcOut.Warnings...)
	result.Warnings = appendUnique(result.Warnings, parsed.Warnings...)
	safetyWarnings, err := diagnose.SafetyWarnings(parsed.Root)
	if err != nil {
		return result, err
	}
	result.Warnings = appendUnique(result.Warnings, safetyWarnings...)
	result.Files, err = fileList(parsed.Root)
	if err != nil {
		return result, err
	}
	return result, nil
}

type DoctorRequest struct {
	CWD   string
	Scope Scope
}

type DoctorResult struct {
	Checks []DoctorCheck `json:"checks"`
}

type DoctorCheck struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Skill    string `json:"skill,omitempty"`
	Message  string `json:"message"`
}

func Doctor(req DoctorRequest) (DoctorResult, error) {
	var result DoctorResult
	paths := store.PathsFor(req.Scope, cleanCWD(req.CWD))
	lock, err := lockfile.Read(paths.Lock)
	if err != nil {
		return result, err
	}
	if len(lock.Skills) == 0 {
		result.Checks = append(result.Checks, DoctorCheck{Severity: "info", Code: "empty-lock", Message: "no skills are locked"})
		return result, nil
	}
	for _, name := range lockfile.Names(lock) {
		entry := lock.Skills[name]
		if entry.Incomplete {
			result.Checks = append(result.Checks, DoctorCheck{Severity: "warning", Code: "incomplete", Skill: name, Message: "entry is incomplete and cannot be restored automatically"})
			continue
		}
		path := storePath(paths, entry)
		if !exists(path) {
			result.Checks = append(result.Checks, DoctorCheck{Severity: "error", Code: "missing-store", Skill: name, Message: "store entry is missing; run skit install"})
			continue
		}
		parsed, err := skill.ParseDir(path)
		if err != nil {
			result.Checks = append(result.Checks, DoctorCheck{Severity: "error", Code: "invalid-skill", Skill: name, Message: err.Error()})
			continue
		}
		hashes, err := hash.Tree(parsed.Root, parsed.File)
		if err != nil {
			result.Checks = append(result.Checks, DoctorCheck{Severity: "error", Code: "hash-error", Skill: name, Message: err.Error()})
			continue
		}
		if hashes.Tree != entry.Hashes.Tree || hashes.SkillMD != entry.Hashes.SkillMD {
			result.Checks = append(result.Checks, DoctorCheck{Severity: "error", Code: "hash-mismatch", Skill: name, Message: "store content differs from lock"})
		}
		warnings := appendUnique(nil, entry.Warnings...)
		warnings = appendUnique(warnings, parsed.Warnings...)
		safetyWarnings, err := diagnose.SafetyWarnings(parsed.Root)
		if err != nil {
			result.Checks = append(result.Checks, DoctorCheck{Severity: "error", Code: "safety-scan-error", Skill: name, Message: err.Error()})
			continue
		}
		warnings = appendUnique(warnings, safetyWarnings...)
		for _, warning := range warnings {
			result.Checks = append(result.Checks, DoctorCheck{Severity: "warning", Code: "warning", Skill: name, Message: warning})
		}
		result.Checks = append(result.Checks, requirementChecks(name, parsed)...)
	}
	return result, nil
}

type InitRequest struct {
	CWD  string
	Name string
}

type InitResult struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func Init(req InitRequest) (InitResult, error) {
	cwd := cleanCWD(req.CWD)
	name := req.Name
	dir := cwd
	if name == "" {
		name = filepath.Base(cwd)
	} else {
		dir = filepath.Join(cwd, name)
	}
	if !skill.ValidName(name) {
		return InitResult{}, fmt.Errorf("invalid skill name %q", name)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return InitResult{}, err
	}
	path := filepath.Join(dir, "SKILL.md")
	if exists(path) {
		return InitResult{}, fmt.Errorf("SKILL.md already exists in %s", dir)
	}
	body := skillTemplate(name)
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		return InitResult{}, err
	}
	return InitResult{Name: name, Path: path}, nil
}

type ImportLockRequest struct {
	CWD   string
	Scope Scope
	Kind  string
}

type ImportLockResult struct {
	Entries  []lockfile.Entry `json:"entries"`
	Warnings []string         `json:"warnings,omitempty"`
}

func ImportLock(req ImportLockRequest) (ImportLockResult, error) {
	var result ImportLockResult
	paths := store.PathsFor(req.Scope, cleanCWD(req.CWD))
	lock, err := lockfile.Read(paths.Lock)
	if err != nil {
		return result, err
	}
	var entries []lockfile.Entry
	switch req.Kind {
	case "skills":
		entries, err = importSkillsLock(cleanCWD(req.CWD))
	case "clawhub":
		entries, err = importClawHubLock(cleanCWD(req.CWD))
	default:
		return result, fmt.Errorf("unsupported lock kind %q", req.Kind)
	}
	if err != nil {
		return result, err
	}
	for _, entry := range entries {
		lock = lockfile.Put(lock, entry)
		result.Entries = append(result.Entries, entry)
		result.Warnings = appendUnique(result.Warnings, entry.Warnings...)
	}
	if err := lockfile.Write(paths.Lock, lock); err != nil {
		return result, err
	}
	return result, nil
}

type skillsLockFile struct {
	Version int                        `json:"version"`
	Skills  map[string]skillsLockEntry `json:"skills"`
}

type skillsLockEntry struct {
	Source       string `json:"source"`
	Ref          string `json:"ref"`
	SourceType   string `json:"sourceType"`
	ComputedHash string `json:"computedHash"`
}

func importSkillsLock(cwd string) ([]lockfile.Entry, error) {
	path := filepath.Join(cwd, "skills-lock.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var imported skillsLockFile
	if err := json.Unmarshal(raw, &imported); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if imported.Skills == nil {
		return nil, fmt.Errorf("skills-lock.json missing skills")
	}
	var entries []lockfile.Entry
	for _, name := range sortedMapKeys(imported.Skills) {
		item := imported.Skills[name]
		sourceType := item.SourceType
		if sourceType == "" {
			sourceType = "unknown"
		}
		warnings := []string{"imported from skills lock without reproducible skit hash; re-add the Skill to make it restorable"}
		if item.ComputedHash != "" {
			warnings = append(warnings, "skills computedHash preserved as diagnostic only: "+item.ComputedHash)
		}
		entries = append(entries, lockfile.Entry{
			Name:       name,
			Source:     lockfile.Source{Type: sourceType, Locator: item.Source, Ref: item.Ref, Skill: name},
			Incomplete: true,
			Warnings:   warnings,
		})
	}
	return entries, nil
}

type clawHubLockFile struct {
	Version int                         `json:"version"`
	Skills  map[string]clawHubLockEntry `json:"skills"`
}

type clawHubLockEntry struct {
	Version     *string `json:"version"`
	InstalledAt int64   `json:"installedAt"`
}

type clawHubOrigin struct {
	Version          int    `json:"version"`
	Registry         string `json:"registry"`
	Slug             string `json:"slug"`
	InstalledVersion string `json:"installedVersion"`
}

func importClawHubLock(cwd string) ([]lockfile.Entry, error) {
	path := firstExisting(filepath.Join(cwd, ".clawhub", "lock.json"), filepath.Join(cwd, ".clawdhub", "lock.json"))
	if path == "" {
		return nil, fmt.Errorf(".clawhub/lock.json not found")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var imported clawHubLockFile
	if err := json.Unmarshal(raw, &imported); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if imported.Skills == nil {
		return nil, fmt.Errorf("clawhub lock missing skills")
	}
	var entries []lockfile.Entry
	for _, name := range sortedMapKeys(imported.Skills) {
		item := imported.Skills[name]
		registry := ""
		version := ""
		if item.Version != nil {
			version = *item.Version
		}
		if origin := readClawHubOrigin(cwd, name); origin != nil {
			registry = origin.Registry
			if origin.Slug != "" {
				name = origin.Slug
			}
			if origin.InstalledVersion != "" {
				version = origin.InstalledVersion
			}
		}
		locator := name
		if registry != "" {
			locator = registry + "/" + name
		}
		source := lockfile.Source{Type: "registry", Locator: locator, Ref: version, Skill: name}
		entries = append(entries, lockfile.Entry{
			Name:       name,
			Source:     source,
			Incomplete: true,
			Warnings:   []string{"imported from clawhub lock without source archive/hash; re-add the Skill to make it restorable"},
		})
	}
	return entries, nil
}

func readClawHubOrigin(cwd, slug string) *clawHubOrigin {
	candidates := []string{
		filepath.Join(cwd, "skills", slug, ".clawhub", "origin.json"),
		filepath.Join(cwd, "skills", slug, ".clawdhub", "origin.json"),
		filepath.Join(cwd, slug, ".clawhub", "origin.json"),
		filepath.Join(cwd, slug, ".clawdhub", "origin.json"),
	}
	for _, path := range candidates {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var origin clawHubOrigin
		if json.Unmarshal(raw, &origin) == nil && origin.Version == 1 && origin.Slug != "" {
			return &origin
		}
	}
	return nil
}

func selectSkills(skills []skill.Skill, wanted string, all bool) ([]skill.Skill, error) {
	if all {
		if len(skills) == 0 {
			return nil, fmt.Errorf("no skills found")
		}
		return skills, nil
	}
	if wanted != "" {
		for _, s := range skills {
			if s.Name == wanted {
				return []skill.Skill{s}, nil
			}
		}
		return nil, fmt.Errorf("skill %q not found", wanted)
	}
	if len(skills) == 1 {
		return skills, nil
	}
	if len(skills) == 0 {
		return nil, fmt.Errorf("no skills found")
	}
	return nil, fmt.Errorf("source contains multiple skills; pass --skill <name> or --all")
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
	selected, err := selectSkills(discovered, src.Skill, false)
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

func cleanCWD(cwd string) string {
	if cwd != "" {
		return cwd
	}
	got, err := os.Getwd()
	if err != nil {
		return "."
	}
	return got
}

func storePath(paths store.Paths, entry lockfile.Entry) string {
	return filepath.Join(paths.Root, entry.Hashes.Tree, entry.Name)
}

func verifyStoreEntry(paths store.Paths, entry lockfile.Entry) error {
	path := storePath(paths, entry)
	parsed, err := skill.ParseDir(path)
	if err != nil {
		return err
	}
	hashes, err := hash.Tree(parsed.Root, parsed.File)
	if err != nil {
		return err
	}
	if hashes.Tree != entry.Hashes.Tree || hashes.SkillMD != entry.Hashes.SkillMD {
		return fmt.Errorf("hash mismatch restoring %q", entry.Name)
	}
	return nil
}

func requiresFrom(parsed skill.Skill) SkillRequires {
	return SkillRequires{
		Bins:    parsed.Skit.Requires.Bins,
		AnyBins: parsed.Skit.Requires.AnyBins,
		Env:     parsed.Skit.Requires.Env,
		Config:  parsed.Skit.Requires.Config,
	}
}

func skillTemplate(name string) string {
	return "---\n" +
		"name: " + name + "\n" +
		"description: \"TODO: describe " + name + ".\"\n" +
		"metadata:\n" +
		"  skit:\n" +
		"    version: 0.1.0\n" +
		"---\n" +
		"# " + name + "\n" +
		"\n" +
		"Describe when to use this skill and what workflow it should follow.\n"
}

func fileList(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(files)
	return files, err
}

func requirementChecks(name string, parsed skill.Skill) []DoctorCheck {
	var checks []DoctorCheck
	for _, bin := range parsed.Skit.Requires.Bins {
		if _, err := exec.LookPath(bin); err != nil {
			checks = append(checks, DoctorCheck{Severity: "error", Code: "missing-bin", Skill: name, Message: "missing required binary: " + bin})
		}
	}
	if len(parsed.Skit.Requires.AnyBins) > 0 {
		found := false
		for _, bin := range parsed.Skit.Requires.AnyBins {
			if _, err := exec.LookPath(bin); err == nil {
				found = true
				break
			}
		}
		if !found {
			checks = append(checks, DoctorCheck{Severity: "error", Code: "missing-any-bin", Skill: name, Message: "missing one of required binaries: " + strings.Join(parsed.Skit.Requires.AnyBins, ", ")})
		}
	}
	for _, key := range parsed.Skit.Requires.Env {
		if _, ok := os.LookupEnv(key); !ok {
			checks = append(checks, DoctorCheck{Severity: "error", Code: "missing-env", Skill: name, Message: "missing environment variable: " + key})
		}
	}
	for _, cfg := range parsed.Skit.Requires.Config {
		if !exists(expandPath(cfg)) {
			checks = append(checks, DoctorCheck{Severity: "error", Code: "missing-config", Skill: name, Message: "missing config path: " + cfg})
		}
	}
	if len(parsed.Skit.Platforms.OS) > 0 && !contains(parsed.Skit.Platforms.OS, runtime.GOOS) {
		checks = append(checks, DoctorCheck{Severity: "error", Code: "unsupported-os", Skill: name, Message: "unsupported OS: " + runtime.GOOS})
	}
	if len(parsed.Skit.Platforms.Arch) > 0 && !contains(parsed.Skit.Platforms.Arch, runtime.GOARCH) {
		checks = append(checks, DoctorCheck{Severity: "error", Code: "unsupported-arch", Skill: name, Message: "unsupported architecture: " + runtime.GOARCH})
	}
	return checks
}

func expandPath(path string) string {
	path = os.ExpandEnv(path)
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func sortedMapKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func firstExisting(paths ...string) string {
	for _, path := range paths {
		if exists(path) {
			return path
		}
	}
	return ""
}

func appendUnique(dst []string, items ...string) []string {
	seen := make(map[string]bool, len(dst)+len(items))
	for _, item := range dst {
		seen[item] = true
	}
	for _, item := range items {
		if item == "" || seen[item] {
			continue
		}
		dst = append(dst, item)
		seen[item] = true
	}
	return dst
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ctx(c context.Context) context.Context {
	if c != nil {
		return c
	}
	return context.Background()
}

func refForInstall(src lockfile.Source) string {
	if src.ResolvedRef != "" {
		return src.ResolvedRef
	}
	return src.Ref
}
