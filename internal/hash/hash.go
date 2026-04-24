package hash

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Result struct {
	Tree    string
	SkillMD string
}

type entry struct {
	rel  string
	mode string
	size int64
	sum  string
}

func Tree(root, skillFile string) (Result, error) {
	var result Result
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return result, err
	}
	skillAbs, err := filepath.Abs(skillFile)
	if err != nil {
		return result, err
	}

	var entries []entry
	err = filepath.WalkDir(rootAbs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == rootAbs {
			return nil
		}
		rel, err := filepath.Rel(rootAbs, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if unsafeRel(rel) {
			return fmt.Errorf("unsafe relative path %q", rel)
		}
		if d.IsDir() {
			if excludedDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular file rejected: %s", rel)
		}
		sum, err := fileSHA256(path)
		if err != nil {
			return err
		}
		if path == skillAbs {
			result.SkillMD = encodeDigestBytes(sum)
		}
		entries = append(entries, entry{
			rel:  rel,
			mode: normalizedMode(info.Mode()),
			size: info.Size(),
			sum:  hex.EncodeToString(sum),
		})
		return nil
	})
	if err != nil {
		return result, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].rel < entries[j].rel })

	h := sha256.New()
	for _, e := range entries {
		fmt.Fprintf(h, "file %s\n", e.rel)
		fmt.Fprintf(h, "mode %s\n", e.mode)
		fmt.Fprintf(h, "size %d\n", e.size)
		fmt.Fprintf(h, "sha256 %s\n\n", e.sum)
	}
	result.Tree = encodeDigestBytes(h.Sum(nil))
	if result.SkillMD == "" {
		return result, fmt.Errorf("skill marker %s was not hashed", skillFile)
	}
	return result, nil
}

func fileSHA256(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func encodeDigestBytes(sum []byte) string {
	return "sha256-" + base64.RawURLEncoding.EncodeToString(sum)
}

func normalizedMode(mode fs.FileMode) string {
	if mode&0111 != 0 {
		return "0755"
	}
	return "0644"
}

func excludedDir(name string) bool {
	return name == ".git" || name == ".skit" || name == ".clawhub" || name == ".clawdhub"
}

func unsafeRel(rel string) bool {
	if rel == "" || strings.HasPrefix(rel, "/") || strings.Contains(rel, "\\") {
		return true
	}
	for _, part := range strings.Split(rel, "/") {
		if part == "" || part == "." || part == ".." {
			return true
		}
	}
	return false
}
