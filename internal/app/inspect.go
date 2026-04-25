package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/vlln/skit/internal/diagnose"
	"github.com/vlln/skit/internal/hash"
	"github.com/vlln/skit/internal/lockfile"
	"github.com/vlln/skit/internal/skill"
	"github.com/vlln/skit/internal/source"
	"github.com/vlln/skit/internal/store"
)

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
func requiresFrom(parsed skill.Skill) SkillRequires {
	return SkillRequires{
		Bins:    parsed.Skit.Requires.Bins,
		AnyBins: parsed.Skit.Requires.AnyBins,
		Env:     parsed.Skit.Requires.Env,
		Config:  parsed.Skit.Requires.Config,
	}
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
