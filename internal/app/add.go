package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vlln/skit/internal/gitfetch"
	"github.com/vlln/skit/internal/skill"
	"github.com/vlln/skit/internal/source"
)

type AddRequest struct {
	Context   context.Context
	CWD       string
	Scope     Scope
	Source    string
	Name      string
	Skill     string
	Skills    []string
	Agents    []string
	All       bool
	FullDepth bool
	Force     bool
	Progress  func(string)

	// Deprecated pre-1.0 flags.
	IgnoreDeps bool
	NoActive   bool
}

type AddResult struct {
	Entries     []ManifestSkill `json:"entries"`
	ActivePaths []string        `json:"activePaths,omitempty"`
	Warnings    []string        `json:"warnings,omitempty"`
}

func Add(req AddRequest) (AddResult, error) {
	var result AddResult
	src, err := source.Parse(req.Source, source.WithSkill(req.Skill))
	if err != nil {
		return result, err
	}
	progress := func(message string) {
		if req.Progress != nil {
			req.Progress(message)
		}
	}
	wanted := req.Skills
	if req.Skill != "" {
		wanted = append([]string{req.Skill}, wanted...)
	}
	if src.Skill != "" && len(wanted) > 0 {
		result.Warnings = append(result.Warnings, "inline skill selector ignored because --skill was provided")
	}
	if src.Skill != "" && len(wanted) == 0 {
		wanted = []string{src.Skill}
	}

	workRoot := src.Locator
	cleanup := func() {}
	if isGitProvider(src.Type) {
		if err := resolveTreeSource(ctx(req.Context), &src); err != nil {
			return result, err
		}
		progress("fetch " + src.Locator)
		clone, err := gitfetch.CloneWithOptions(ctx(req.Context), src.URL, src.Ref, tmpRoot(), cloneOptions(src))
		if err != nil {
			return result, err
		}
		workRoot = clone.Dir
		src.Ref = clone.Ref
		cleanup = func() { _ = gitfetch.RemoveUnder(clone.Dir, tmpRoot()) }
		defer cleanup()
	} else if src.Type != source.Local {
		return result, fmt.Errorf("provider %q is not implemented for install yet", src.Type)
	}

	discoverRoot := workRoot
	if src.Subpath != "" {
		discoverRoot = filepath.Join(workRoot, filepath.FromSlash(src.Subpath))
	}
	progress("discover")
	discoverOpts := parseOptionsForSource(src)
	discoverOpts.IncludeInternal = len(wanted) > 0
	discoverOpts.FullDepth = req.FullDepth
	discoverOpts.IgnoreInvalid = len(wanted) > 0
	discovered, warnings, err := skill.DiscoverWithOptions(discoverRoot, discoverOpts)
	if err != nil {
		return result, err
	}
	selected, err := selectSkills(discovered, wanted, req.All)
	if err != nil {
		if len(result.Warnings) > 0 {
			return result, fmt.Errorf("%w; %s", err, strings.Join(result.Warnings, "; "))
		}
		return result, err
	}
	if len(wanted) > 0 {
		for _, parsed := range selected {
			result.Warnings = append(result.Warnings, parsed.Warnings...)
		}
	} else {
		result.Warnings = append(result.Warnings, warnings...)
	}
	if req.Name != "" && len(selected) != 1 {
		return result, fmt.Errorf("--name can only be used when installing exactly one skill")
	}

	manifest, err := readManifest()
	if err != nil {
		return result, err
	}
	agents := uniqueAgents(req.Agents)
	activeDirs, err := manifestActiveDirs(agents)
	if err != nil {
		return result, err
	}
	for _, parsed := range selected {
		name := parsed.Name
		if req.Name != "" {
			name = req.Name
		}
		progress("copy " + name)
		installDir, err := safeChild(installedSkillsRoot(), name)
		if err != nil {
			return result, err
		}
		if err := copySkillTree(parsed.Root, installDir); err != nil {
			return result, err
		}
		for _, dir := range activeDirs {
			activePath, err := activateNameInDir(dir, name, installDir, req.Force)
			if err != nil {
				return result, err
			}
			result.ActivePaths = append(result.ActivePaths, activePath)
		}
		entry := ManifestSkill{
			Name:        name,
			Description: parsed.Description,
			Source: ManifestSource{
				Type:    string(src.Type),
				Locator: src.Locator,
				URL:     src.URL,
				Ref:     src.Ref,
				Subpath: sourceSubpath(src, parsed.Root, workRoot),
				Skill:   parsed.Name,
			},
			Path:   filepath.ToSlash(filepath.Join("skills", name)),
			Agents: agents,
		}
		manifest.Skills[name] = entry
		result.Entries = append(result.Entries, entry)
	}
	if err := writeManifestFile(manifest); err != nil {
		return result, err
	}
	return result, nil
}

func manifestActiveDirs(agents []string) ([]string, error) {
	var dirs []string
	seen := map[string]bool{}
	for _, agent := range agents {
		var dir string
		if agent == "universal" {
			dir = filepath.Join(userHome(), ".agents", "skills")
		} else {
			got, err := agentActiveDir(Global, "", agent)
			if err != nil {
				return nil, err
			}
			dir = got
		}
		clean := filepath.Clean(dir)
		if seen[clean] {
			continue
		}
		seen[clean] = true
		dirs = append(dirs, clean)
	}
	if len(dirs) == 0 {
		dirs = append(dirs, filepath.Join(userHome(), ".agents", "skills"))
	}
	return dirs, nil
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
