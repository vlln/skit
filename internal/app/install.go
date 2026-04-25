package app

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/vlln/skit/internal/gitfetch"
	"github.com/vlln/skit/internal/lockfile"
	"github.com/vlln/skit/internal/skill"
	"github.com/vlln/skit/internal/source"
	"github.com/vlln/skit/internal/store"
)

type InstallRequest struct {
	CWD    string
	Scope  Scope
	Agents []string
}

type InstallResult struct {
	Restored    []lockfile.Entry
	Skipped     []lockfile.Entry
	ActivePaths []string
}

func Install(req InstallRequest) (InstallResult, error) {
	var result InstallResult
	cwd := cleanCWD(req.CWD)
	paths := store.PathsFor(req.Scope, cwd)
	activeDirs, err := activeDirs(paths, req.Scope, cwd, req.Agents)
	if err != nil {
		return result, err
	}
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
			for _, dir := range activeDirs {
				activePath, err := activateInDir(dir, entry, storePath(paths, entry), true)
				if err != nil {
					return result, err
				}
				result.ActivePaths = append(result.ActivePaths, activePath)
			}
			result.Restored = append(result.Restored, entry)
			continue
		}
		root := entry.Source.Locator
		cleanup := func() {}
		src, err := sourceFromLock(entry.Source)
		if err != nil {
			return result, err
		}
		if isGitProvider(source.Type(entry.Source.Type)) {
			clone, err := gitfetch.CloneWithOptions(ctx(context.Background()), entry.Source.URL, refForInstall(entry.Source), paths.Tmp, cloneOptions(src))
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
		for _, dir := range activeDirs {
			activePath, err := activateInDir(dir, entry, installed.Path, true)
			if err != nil {
				return result, err
			}
			result.ActivePaths = append(result.ActivePaths, activePath)
		}
		result.Restored = append(result.Restored, entry)
	}
	return result, nil
}
