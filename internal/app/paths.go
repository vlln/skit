package app

import (
	"fmt"
	"os"
	"path/filepath"
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

func activateNameInDir(activeDir, name, target string, force bool) (string, error) {
	path, err := safeChild(activeDir, name)
	if err != nil {
		return "", err
	}
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
