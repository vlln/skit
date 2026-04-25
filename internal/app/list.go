package app

import (
	"github.com/vlln/skit/internal/lockfile"
	"github.com/vlln/skit/internal/store"
)

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
