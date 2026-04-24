package diagnose

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const maxSafetyScanBytes int64 = 1024 * 1024

var safetyPatterns = []struct {
	code string
	re   *regexp.Regexp
}{
	{
		code: "curl/wget piped to shell",
		re:   regexp.MustCompile(`(?i)\b(curl|wget)\b[^\n|;&]*\|[^\n]*(sh|bash|zsh)\b`),
	},
	{
		code: "base64 decode piped to shell",
		re:   regexp.MustCompile(`(?i)\bbase64\b[^\n|;&]*(--decode|-d)\b[^\n|;&]*\|[^\n]*(sh|bash|zsh)\b`),
	},
	{
		code: "eval of command substitution",
		re:   regexp.MustCompile(`(?i)\beval\b[^\n]*\$\(`),
	},
	{
		code: "shell -c command substitution",
		re:   regexp.MustCompile(`(?i)\b(sh|bash|zsh)\b[^\n]*\s-c\s+[^\n]*\$\(`),
	},
	{
		code: "eval of encoded payload",
		re:   regexp.MustCompile(`(?i)\beval\b[^\n]*(base64|openssl\s+enc)`),
	},
}

func SafetyWarnings(root string) ([]string, error) {
	var warnings []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
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
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if info.Mode()&0111 != 0 {
			warnings = append(warnings, fmt.Sprintf("executable file in Skill directory: %s", rel))
		}
		if info.Size() > maxSafetyScanBytes {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !looksText(raw) {
			return nil
		}
		text := string(raw)
		for _, pattern := range safetyPatterns {
			if pattern.re.MatchString(text) {
				warnings = append(warnings, fmt.Sprintf("suspicious content in %s: %s", rel, pattern.code))
			}
		}
		return nil
	})
	sort.Strings(warnings)
	return dedupe(warnings), err
}

func looksText(raw []byte) bool {
	if len(raw) == 0 {
		return true
	}
	return !bytes.Contains(raw, []byte{0})
}

func excludedDir(name string) bool {
	return name == ".git" || name == ".skit" || name == "node_modules" || strings.HasPrefix(name, ".")
}

func dedupe(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := items[:0]
	var last string
	for i, item := range items {
		if i == 0 || item != last {
			out = append(out, item)
			last = item
		}
	}
	return out
}
