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
	CWD   string
	Scope Scope
}

type InstallResult struct {
	Restored    []lockfile.Entry
	Skipped     []lockfile.Entry
	ActivePaths []string
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
			activePath, err := activate(paths, entry, storePath(paths, entry), true)
			if err != nil {
				return result, err
			}
			result.ActivePaths = append(result.ActivePaths, activePath)
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
		activePath, err := activate(paths, entry, installed.Path, true)
		if err != nil {
			return result, err
		}
		result.ActivePaths = append(result.ActivePaths, activePath)
		result.Restored = append(result.Restored, entry)
	}
	return result, nil
}
