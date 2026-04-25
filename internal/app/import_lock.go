package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vlln/skit/internal/lockfile"
	"github.com/vlln/skit/internal/store"
)

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
	if err := writeLock(paths, req.Scope, cleanCWD(req.CWD), lock); err != nil {
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
