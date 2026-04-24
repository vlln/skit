package store

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/vlln/skit/internal/hash"
	"github.com/vlln/skit/internal/skill"
	"github.com/vlln/skit/internal/xdg"
)

type Scope string

const (
	Project Scope = "project"
	Global  Scope = "global"
)

type Paths struct {
	Root string
	Lock string
	Tmp  string
}

func PathsFor(scope Scope, cwd string) Paths {
	if scope == Global {
		return Paths{
			Root: filepath.Join(xdg.DataHome(), "skit", "store"),
			Lock: filepath.Join(xdg.StateHome(), "skit", "lock.json"),
			Tmp:  filepath.Join(xdg.DataHome(), "skit", "tmp"),
		}
	}
	return Paths{
		Root: filepath.Join(cwd, ".skit", "store"),
		Lock: filepath.Join(cwd, ".skit", "lock.json"),
		Tmp:  filepath.Join(cwd, ".skit", "tmp"),
	}
}

type InstallResult struct {
	Hashes hash.Result
	Path   string
	Reused bool
}

func InstallSnapshot(paths Paths, parsed skill.Skill) (InstallResult, error) {
	var out InstallResult
	hashes, err := hash.Tree(parsed.Root, parsed.File)
	if err != nil {
		return out, err
	}
	finalPath := filepath.Join(paths.Root, hashes.Tree, parsed.Name)
	out.Hashes = hashes
	out.Path = finalPath

	if _, err := os.Stat(finalPath); err == nil {
		existing, err := hash.Tree(finalPath, filepath.Join(finalPath, filepath.Base(parsed.File)))
		if err != nil {
			return out, err
		}
		if existing.Tree != hashes.Tree {
			return out, fmt.Errorf("store path exists with different content: %s", finalPath)
		}
		out.Reused = true
		return out, nil
	} else if !os.IsNotExist(err) {
		return out, err
	}

	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return out, err
	}
	if err := os.MkdirAll(paths.Tmp, 0755); err != nil {
		return out, err
	}
	tmp, err := os.MkdirTemp(paths.Tmp, "install-*")
	if err != nil {
		return out, err
	}
	defer os.RemoveAll(tmp)

	tmpSkill := filepath.Join(tmp, parsed.Name)
	if err := copyTree(parsed.Root, tmpSkill); err != nil {
		return out, err
	}
	tmpHashes, err := hash.Tree(tmpSkill, filepath.Join(tmpSkill, filepath.Base(parsed.File)))
	if err != nil {
		return out, err
	}
	if tmpHashes.Tree != hashes.Tree || tmpHashes.SkillMD != hashes.SkillMD {
		return out, fmt.Errorf("copied snapshot hash mismatch")
	}
	if err := os.Rename(tmpSkill, finalPath); err != nil {
		return out, err
	}
	return out, nil
}

func copyTree(src, dst string) error {
	srcAbs, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	return filepath.WalkDir(srcAbs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == srcAbs {
			return nil
		}
		rel, err := filepath.Rel(srcAbs, path)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)
		if d.IsDir() {
			if excludedDir(d.Name()) {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular file rejected: %s", relSlash)
		}
		return copyFile(path, filepath.Join(dst, rel), info.Mode())
	})
}

func copyFile(src, dst string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, normalizedMode(mode))
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

func normalizedMode(mode fs.FileMode) fs.FileMode {
	if mode&0111 != 0 {
		return 0755
	}
	return 0644
}

func excludedDir(name string) bool {
	return name == ".git" || name == ".skit" || name == ".clawhub" || name == ".clawdhub" || strings.HasPrefix(name, string(filepath.Separator))
}
