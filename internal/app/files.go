package app

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func copySkillTree(src, dst string) error {
	srcAbs, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	dstAbs, err := filepath.Abs(dst)
	if err != nil {
		return err
	}
	if samePath(srcAbs, dstAbs) {
		return fmt.Errorf("refusing to install skill from its destination: %s", dst)
	}
	parent := filepath.Dir(dstAbs)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(parent, ".copy-*")
	if err != nil {
		return err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(tmp)
		}
	}()
	if err := copySkillTreeInto(srcAbs, tmp); err != nil {
		return err
	}
	if err := replaceDir(dstAbs, tmp); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func replaceDir(dst, next string) error {
	backup := ""
	if _, err := os.Lstat(dst); err == nil {
		tmp, err := os.MkdirTemp(filepath.Dir(dst), ".replace-*")
		if err != nil {
			return err
		}
		if err := os.Remove(tmp); err != nil {
			return err
		}
		backup = tmp
		if err := os.Rename(dst, backup); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(next, dst); err != nil {
		if backup != "" {
			_ = os.Rename(backup, dst)
		}
		return err
	}
	if backup != "" {
		_ = os.RemoveAll(backup)
	}
	return nil
}

func copySkillTreeInto(srcAbs, dst string) error {
	return filepath.WalkDir(srcAbs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == srcAbs {
			return os.MkdirAll(dst, 0755)
		}
		rel, err := filepath.Rel(srcAbs, path)
		if err != nil {
			return err
		}
		if unsafeRelPath(rel) {
			return fmt.Errorf("unsafe path in skill: %s", rel)
		}
		if d.IsDir() {
			if excludedSkillDir(d.Name()) {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular file rejected: %s", filepath.ToSlash(rel))
		}
		return copySkillFile(path, filepath.Join(dst, rel), info.Mode())
	})
}

func copySkillFile(src, dst string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, normalizedSkillMode(mode))
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func normalizedSkillMode(mode fs.FileMode) fs.FileMode {
	if mode&0111 != 0 {
		return 0755
	}
	return 0644
}

func excludedSkillDir(name string) bool {
	return name == ".git" || name == "node_modules" || name == "__pycache__" || strings.HasPrefix(name, ".skit")
}

func unsafeRelPath(rel string) bool {
	return rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func safeChild(base, name string) (string, error) {
	if name == "" || filepath.IsAbs(name) || strings.Contains(name, string(filepath.Separator)) || name == "." || name == ".." {
		return "", fmt.Errorf("invalid skill name %q", name)
	}
	path := filepath.Join(base, name)
	rel, err := filepath.Rel(base, path)
	if err != nil || unsafeRelPath(rel) {
		return "", fmt.Errorf("invalid skill name %q", name)
	}
	return path, nil
}
