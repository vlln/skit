package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Discover(root string) ([]Skill, []string, error) {
	return DiscoverWithOptions(root, ParseOptions{})
}

func DiscoverWithOptions(root string, opts ParseOptions) ([]Skill, []string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, nil, err
	}
	var skills []Skill
	var warnings []string
	seen := map[string]string{}
	if hasMarker(abs) {
		s, err := ParseDirWithOptions(abs, opts)
		if err != nil {
			return nil, nil, err
		}
		if s.Internal && !opts.IncludeInternal {
			return nil, append(s.Warnings, internalSkipWarning(s.Name, abs)), nil
		}
		if !opts.FullDepth {
			return []Skill{s}, s.Warnings, nil
		}
		seen[s.Name] = abs
		skills = append(skills, s)
		warnings = append(warnings, s.Warnings...)
	}

	candidates, err := discoveryCandidates(abs)
	if err != nil {
		return nil, nil, err
	}
	for _, candidate := range candidates {
		if !hasMarker(candidate) {
			continue
		}
		s, err := ParseDirWithOptions(candidate, opts)
		if err != nil {
			return nil, warnings, err
		}
		if s.Internal && !opts.IncludeInternal {
			warnings = append(warnings, internalSkipWarning(s.Name, candidate))
			continue
		}
		if prev, ok := seen[s.Name]; ok {
			return nil, warnings, &DuplicateNameError{Name: s.Name, First: prev, Second: candidate}
		}
		seen[s.Name] = candidate
		skills = append(skills, s)
		warnings = append(warnings, s.Warnings...)
	}
	if len(skills) == 0 || opts.FullDepth {
		more, err := recursiveCandidates(abs, candidates, 5)
		if err != nil {
			return nil, warnings, err
		}
		for _, candidate := range more {
			if !hasMarker(candidate) {
				continue
			}
			s, err := ParseDirWithOptions(candidate, opts)
			if err != nil {
				return nil, warnings, err
			}
			if s.Internal && !opts.IncludeInternal {
				warnings = append(warnings, internalSkipWarning(s.Name, candidate))
				continue
			}
			if prev, ok := seen[s.Name]; ok {
				if prev == candidate {
					continue
				}
				return nil, warnings, &DuplicateNameError{Name: s.Name, First: prev, Second: candidate}
			}
			seen[s.Name] = candidate
			skills = append(skills, s)
			warnings = append(warnings, s.Warnings...)
		}
	}
	return skills, warnings, nil
}

type DuplicateNameError struct {
	Name   string
	First  string
	Second string
}

func (e *DuplicateNameError) Error() string {
	return "duplicate skill name " + e.Name
}

func discoveryCandidates(root string) ([]string, error) {
	var out []string
	addChildren := func(dir string) error {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		for _, entry := range entries {
			if !entry.IsDir() || skipDir(entry.Name()) {
				continue
			}
			out = append(out, filepath.Join(dir, entry.Name()))
		}
		return nil
	}
	if err := addChildren(root); err != nil {
		return nil, err
	}
	for _, dir := range prioritySkillDirs(root) {
		if err := addChildren(dir); err != nil {
			return nil, err
		}
	}
	sort.Strings(out)
	return out, nil
}

func recursiveCandidates(root string, already []string, maxDepth int) ([]string, error) {
	seen := map[string]bool{}
	for _, path := range already {
		abs, err := filepath.Abs(path)
		if err == nil {
			seen[abs] = true
		}
	}
	var out []string
	err := walkSkillDirs(root, root, 0, maxDepth, seen, &out)
	sort.Strings(out)
	return out, err
}

func walkSkillDirs(root, dir string, depth, maxDepth int, seen map[string]bool, out *[]string) error {
	if depth > maxDepth {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || skipDir(entry.Name()) {
			continue
		}
		child := filepath.Join(dir, entry.Name())
		if hasMarker(child) {
			abs, err := filepath.Abs(child)
			if err != nil {
				return err
			}
			if !seen[abs] {
				seen[abs] = true
				*out = append(*out, abs)
			}
			continue
		}
		if err := walkSkillDirs(root, child, depth+1, maxDepth, seen, out); err != nil {
			return err
		}
	}
	return nil
}

func prioritySkillDirs(root string) []string {
	return []string{
		filepath.Join(root, "skills"),
		filepath.Join(root, "skills", ".system"),
		filepath.Join(root, "skills", ".curated"),
		filepath.Join(root, "skills", ".experimental"),
		filepath.Join(root, ".agents", "skills"),
		filepath.Join(root, ".codex", "skills"),
		filepath.Join(root, ".claude", "skills"),
		filepath.Join(root, ".cline", "skills"),
		filepath.Join(root, ".codebuddy", "skills"),
		filepath.Join(root, ".commandcode", "skills"),
		filepath.Join(root, ".continue", "skills"),
		filepath.Join(root, ".github", "skills"),
		filepath.Join(root, ".goose", "skills"),
		filepath.Join(root, ".iflow", "skills"),
		filepath.Join(root, ".junie", "skills"),
		filepath.Join(root, ".kilocode", "skills"),
		filepath.Join(root, ".kiro", "skills"),
		filepath.Join(root, ".mux", "skills"),
		filepath.Join(root, ".neovate", "skills"),
		filepath.Join(root, ".opencode", "skills"),
		filepath.Join(root, ".openhands", "skills"),
		filepath.Join(root, ".pi", "skills"),
		filepath.Join(root, ".qoder", "skills"),
		filepath.Join(root, ".roo", "skills"),
		filepath.Join(root, ".trae", "skills"),
		filepath.Join(root, ".windsurf", "skills"),
		filepath.Join(root, ".zencoder", "skills"),
	}
}

func hasMarker(dir string) bool {
	return exists(filepath.Join(dir, "SKILL.md")) || exists(filepath.Join(dir, "skill.md"))
}

func skipDir(name string) bool {
	return name == ".git" || name == ".skit" || name == "node_modules" || strings.HasPrefix(name, ".")
}

func internalSkipWarning(name, path string) string {
	return fmt.Sprintf("internal skill %q skipped at %s; pass --skill %s to install explicitly", name, path, name)
}
