package app

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vlln/skit/internal/hash"
	"github.com/vlln/skit/internal/lockfile"
	"github.com/vlln/skit/internal/skill"
	"github.com/vlln/skit/internal/store"
)

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
	return storePathFor(paths.Root, entry.Hashes.Tree, entry.Name)
}

func storePathFor(root, treeHash, name string) string {
	return filepath.Join(root, treeHash, name)
}

func activePath(paths store.Paths, entry lockfile.Entry) string {
	return filepath.Join(paths.Active, entry.Name)
}

func activate(paths store.Paths, entry lockfile.Entry, target string, force bool) (string, error) {
	return activateInDir(paths.Active, entry, target, force)
}

func activateInDir(activeDir string, entry lockfile.Entry, target string, force bool) (string, error) {
	path := filepath.Join(activeDir, entry.Name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			current, err := os.Readlink(path)
			if err != nil {
				return "", err
			}
			resolved := current
			if !filepath.IsAbs(resolved) {
				resolved = filepath.Join(filepath.Dir(path), current)
			}
			if samePath(resolved, target) {
				return path, nil
			}
			if err := os.Remove(path); err != nil {
				return "", err
			}
		} else {
			if !force {
				return "", fmt.Errorf("active path exists and is not a skit symlink: %s", path)
			}
			if err := os.RemoveAll(path); err != nil {
				return "", err
			}
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	rel, err := filepath.Rel(filepath.Dir(path), target)
	if err != nil {
		rel = target
	}
	if err := os.Symlink(rel, path); err != nil {
		return "", err
	}
	return path, nil
}

func samePath(a, b string) bool {
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return a == b
	}
	return aa == bb
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
func writeLock(paths store.Paths, scope Scope, cwd string, lock lockfile.Lock) error {
	if err := lockfile.Write(paths.Lock, lock); err != nil {
		return err
	}
	if scope == Project {
		if err := lockfile.Write(projectLockIndexPath(paths.Root, cwd), lock); err != nil {
			return err
		}
	}
	return nil
}

func projectLockIndexPath(storeRoot, cwd string) string {
	return filepath.Join(filepath.Dir(storeRoot), "locks", hashPath(cwd)+".lock")
}

func hashPath(path string) string {
	sum := sha256.Sum256([]byte(path))
	return fmt.Sprintf("%x", sum)
}
