package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vlln/skit/internal/lockfile"
	"github.com/vlln/skit/internal/store"
)

type RemoveRequest struct {
	CWD   string
	Scope Scope
	Name  string
	Prune bool
}

type RemoveResult struct {
	Removed bool
	Pruned  []string
	Skipped []string
}

func Remove(req RemoveRequest) (RemoveResult, error) {
	var result RemoveResult
	paths := store.PathsFor(req.Scope, cleanCWD(req.CWD))
	lock, err := lockfile.Read(paths.Lock)
	if err != nil {
		return result, err
	}
	entry := lock.Skills[req.Name]
	next, removed := lockfile.Remove(lock, req.Name)
	if !removed {
		return result, nil
	}
	_ = os.Remove(activePath(paths, entry))
	if err := writeLock(paths, req.Scope, cleanCWD(req.CWD), next); err != nil {
		return result, err
	}
	result.Removed = true
	if req.Prune && entry.Hashes.Tree != "" {
		pruned, err := pruneStoreEntry(paths.Root, entry.Hashes.Tree, entry.Name, referencedStoreKeys(cleanCWD(req.CWD)))
		if err != nil {
			return result, err
		}
		if pruned {
			result.Pruned = append(result.Pruned, storePathFor(paths.Root, entry.Hashes.Tree, entry.Name))
		} else {
			result.Skipped = append(result.Skipped, storePathFor(paths.Root, entry.Hashes.Tree, entry.Name))
		}
	}
	return result, nil
}

type GCRequest struct {
	CWD string
}

type GCResult struct {
	Pruned []string `json:"pruned,omitempty"`
	Kept   []string `json:"kept,omitempty"`
}

func GC(req GCRequest) (GCResult, error) {
	var result GCResult
	cwd := cleanCWD(req.CWD)
	paths := store.PathsFor(Project, cwd)
	refs := referencedStoreKeys(cwd)
	entries, err := storeEntries(paths.Root)
	if err != nil {
		return result, err
	}
	for _, entry := range entries {
		if refs[entry.key] {
			result.Kept = append(result.Kept, entry.path)
			continue
		}
		if err := os.RemoveAll(entry.path); err != nil {
			return result, err
		}
		if entries, err := os.ReadDir(filepath.Dir(entry.path)); err == nil && len(entries) == 0 {
			_ = os.Remove(filepath.Dir(entry.path))
		}
		result.Pruned = append(result.Pruned, entry.path)
	}
	sort.Strings(result.Pruned)
	sort.Strings(result.Kept)
	return result, nil
}

type storeEntry struct {
	key  string
	path string
}

func storeEntries(root string) ([]storeEntry, error) {
	hashDirs, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []storeEntry
	for _, hashDir := range hashDirs {
		if !hashDir.IsDir() {
			continue
		}
		treeHash := hashDir.Name()
		skillRoot := filepath.Join(root, treeHash)
		skills, err := os.ReadDir(skillRoot)
		if err != nil {
			return nil, err
		}
		for _, skillDir := range skills {
			if !skillDir.IsDir() {
				continue
			}
			name := skillDir.Name()
			out = append(out, storeEntry{
				key:  storeKey(treeHash, name),
				path: storePathFor(root, treeHash, name),
			})
		}
	}
	return out, nil
}

func pruneStoreEntry(root, treeHash, name string, refs map[string]bool) (bool, error) {
	key := storeKey(treeHash, name)
	if refs[key] {
		return false, nil
	}
	path := storePathFor(root, treeHash, name)
	if err := os.RemoveAll(path); err != nil {
		return false, err
	}
	parent := filepath.Join(root, treeHash)
	if entries, err := os.ReadDir(parent); err == nil && len(entries) == 0 {
		_ = os.Remove(parent)
	}
	return true, nil
}

func referencedStoreKeys(cwd string) map[string]bool {
	refs := map[string]bool{}
	paths := store.PathsFor(Project, cwd)
	addLockRefs(refs, paths.Lock)
	addLockRefs(refs, store.PathsFor(Global, cwd).Lock)
	for _, lockPath := range knownProjectLocks(paths.Root) {
		addLockRefs(refs, lockPath)
	}
	return refs
}

func knownProjectLocks(storeRoot string) []string {
	locksRoot := filepath.Join(filepath.Dir(storeRoot), "locks")
	entries, err := os.ReadDir(locksRoot)
	if err != nil {
		return nil
	}
	var out []string
	for _, entry := range entries {
		if entry.Type().IsRegular() && strings.HasSuffix(entry.Name(), ".lock") {
			out = append(out, filepath.Join(locksRoot, entry.Name()))
		}
	}
	sort.Strings(out)
	return out
}

func addLockRefs(refs map[string]bool, path string) {
	lock, err := lockfile.Read(path)
	if err != nil {
		return
	}
	for _, entry := range lock.Skills {
		if entry.Hashes.Tree == "" || entry.Name == "" || entry.Incomplete {
			continue
		}
		refs[storeKey(entry.Hashes.Tree, entry.Name)] = true
	}
}

func storeKey(treeHash, name string) string {
	return treeHash + "\x00" + name
}
