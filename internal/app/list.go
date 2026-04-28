package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vlln/skit/internal/lockfile"
	"github.com/vlln/skit/internal/store"
)

type ListRequest struct {
	CWD    string
	Scope  Scope
	Agents []string
}

func List(req ListRequest) ([]lockfile.Entry, error) {
	paths, err := pathsForLockRequest(req.Scope, cleanCWD(req.CWD), req.Agents)
	if err != nil {
		return nil, err
	}
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

func pathsForLockRequest(scope Scope, cwd string, agents []string) (store.Paths, error) {
	if len(agents) == 0 {
		return store.PathsFor(scope, cwd), nil
	}
	if len(agents) > 1 {
		return store.Paths{}, fmt.Errorf("expected at most one --agent value")
	}
	return pathsForAgent(scope, cwd, agents[0])
}

type ListStoreRequest struct {
	CWD          string
	Names        []string
	IncludeLocks bool
}

type StoreListEntry struct {
	Name  string   `json:"name"`
	Tree  string   `json:"tree"`
	Use   []string `json:"use"`
	Locks []string `json:"locks,omitempty"`
}

func ListStore(req ListStoreRequest) ([]StoreListEntry, error) {
	cwd := cleanCWD(req.CWD)
	paths := store.PathsFor(Project, cwd)
	refs := referencedStoreKeys(cwd)
	active := activeStoreKeys(cwd)
	locks := map[string][]string{}
	if req.IncludeLocks {
		locks = lockOwners(cwd)
	}
	nameFilter := stringSet(req.Names)
	entries, err := storeEntries(paths.Root)
	if err != nil {
		return nil, err
	}
	out := make([]StoreListEntry, 0, len(entries))
	for _, entry := range entries {
		if len(nameFilter) > 0 && !nameFilter[entry.name] {
			continue
		}
		use := storeUse(entry.key, refs, active)
		item := StoreListEntry{
			Name: entry.name,
			Tree: shortTreeHash(entry.treeHash),
			Use:  use,
		}
		if req.IncludeLocks {
			item.Locks = locks[entry.key]
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Tree < out[j].Tree
	})
	return out, nil
}

func stringSet(items []string) map[string]bool {
	out := map[string]bool{}
	for _, item := range items {
		if item != "" {
			out[item] = true
		}
	}
	return out
}

func storeUse(key string, refs, active map[string]bool) []string {
	var use []string
	if active[key] {
		use = append(use, "active")
	}
	if refs[key] {
		use = append(use, "locked")
	}
	if len(use) == 0 {
		return []string{"orphan"}
	}
	return use
}

func shortTreeHash(treeHash string) string {
	const algorithmPrefix = "sha256-"
	const hexLen = 12
	if strings.HasPrefix(treeHash, algorithmPrefix) && len(treeHash) > len(algorithmPrefix)+hexLen {
		return treeHash[len(algorithmPrefix) : len(algorithmPrefix)+hexLen]
	}
	if len(treeHash) > 18 {
		return treeHash[:18]
	}
	return treeHash
}

func activeStoreKeys(cwd string) map[string]bool {
	project := store.PathsFor(Project, cwd)
	global := store.PathsFor(Global, cwd)
	return activeStoreKeysForDirs(project.Root, []string{project.Active, global.Active})
}

func lockOwners(cwd string) map[string][]string {
	owners := map[string][]string{}
	paths := store.PathsFor(Project, cwd)
	addLockOwners(owners, paths.Lock, "project")
	addLockOwners(owners, store.PathsFor(Global, cwd).Lock, "global")

	currentIndex := projectLockIndexPath(paths.Root, paths.Lock)
	for _, lockPath := range knownProjectLocks(paths.Root) {
		label := projectIndexLabel(lockPath)
		if samePath(lockPath, currentIndex) {
			label = lockOwnerLabel(projectLockIndexMeta{CWD: cwd, Lock: paths.Lock})
		}
		addLockOwners(owners, lockPath, label)
	}
	for key := range owners {
		sort.Strings(owners[key])
	}
	return owners
}

func addLockOwners(owners map[string][]string, path, label string) {
	lock, err := lockfile.Read(path)
	if err != nil {
		return
	}
	for _, entry := range lock.Skills {
		if entry.Hashes.Tree == "" || entry.Name == "" || entry.Incomplete {
			continue
		}
		key := storeKey(entry.Hashes.Tree, entry.Name)
		owners[key] = appendUnique(owners[key], label)
	}
}

func projectIndexLabel(path string) string {
	if meta := readProjectLockIndexMeta(path); meta.CWD != "" {
		return lockOwnerLabel(meta)
	}
	name := strings.TrimSuffix(filepath.Base(path), ".lock")
	if len(name) > 12 {
		name = name[:12]
	}
	if name == "" {
		return "unknown-project"
	}
	return "unknown-project:" + name
}

func lockOwnerLabel(meta projectLockIndexMeta) string {
	if meta.Lock == "" {
		return filepath.Clean(meta.CWD)
	}
	return filepath.Clean(meta.CWD) + ":" + filepath.ToSlash(relLockPath(meta.CWD, meta.Lock))
}

func relLockPath(cwd, lockPath string) string {
	rel, err := filepath.Rel(cwd, lockPath)
	if err != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return lockPath
	}
	return rel
}

func readProjectLockIndexMeta(lockPath string) projectLockIndexMeta {
	raw, err := os.ReadFile(projectLockIndexMetaPath(lockPath))
	if err != nil {
		return projectLockIndexMeta{}
	}
	var meta projectLockIndexMeta
	if json.Unmarshal(raw, &meta) != nil || meta.Version != 1 {
		return projectLockIndexMeta{}
	}
	return meta
}

func activeStoreKeysForDirs(storeRoot string, dirs []string) map[string]bool {
	keys := map[string]bool{}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink == 0 {
				continue
			}
			linkPath := filepath.Join(dir, entry.Name())
			target, err := os.Readlink(linkPath)
			if err != nil {
				continue
			}
			if !filepath.IsAbs(target) {
				target = filepath.Join(dir, target)
			}
			rel, err := filepath.Rel(storeRoot, filepath.Clean(target))
			if err != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
				continue
			}
			parts := strings.Split(filepath.ToSlash(rel), "/")
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				continue
			}
			keys[storeKey(parts[0], parts[1])] = true
		}
	}
	return keys
}
