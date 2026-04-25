package app

import (
	"context"
	"fmt"

	"github.com/vlln/skit/internal/lockfile"
	"github.com/vlln/skit/internal/store"
)

type UpdateRequest struct {
	Context    context.Context
	CWD        string
	Scope      Scope
	Name       string
	Agents     []string
	IgnoreDeps bool
}

type UpdateResult = AddResult

func Update(req UpdateRequest) (UpdateResult, error) {
	var result UpdateResult
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
		activeDirs: activeDirs,
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
	if err := writeLock(paths, req.Scope, cleanCWD(req.CWD), session.lock); err != nil {
		return result, err
	}
	return result, nil
}
