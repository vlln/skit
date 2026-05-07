package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vlln/skit/internal/metadata"
)

var namePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func ValidName(name string) bool {
	return len(name) > 0 && len(name) <= 64 && namePattern.MatchString(name)
}

type Skill struct {
	Root          string
	File          string
	Name          string
	Description   string
	License       string
	Compatibility string
	AllowedTools  string
	Internal      bool
	Frontmatter   metadata.YAMLMap
	Skit          metadata.Skit
	Warnings      []string
}

func ParseDir(root string) (Skill, error) {
	return ParseDirWithOptions(root, ParseOptions{})
}

type ParseOptions struct {
	// Kept for callers that still distinguish ecosystem sources. Name mismatches
	// are always warnings in v1.
	AllowNameMismatch bool
	ExpectedBasename  string
	IncludeInternal   bool
	FullDepth         bool
	IgnoreInvalid     bool
}

func ParseDirWithOptions(root string, opts ParseOptions) (Skill, error) {
	var out Skill
	abs, err := filepath.Abs(root)
	if err != nil {
		return out, err
	}
	out.Root = abs

	file, warnings, err := markerFile(abs)
	if err != nil {
		return out, err
	}
	out.File = file
	out.Warnings = append(out.Warnings, warnings...)

	raw, err := os.ReadFile(file)
	if err != nil {
		return out, err
	}
	fmText, _, err := splitFrontmatter(string(raw))
	if err != nil {
		return out, err
	}
	fm, err := metadata.ParseYAML(fmText)
	if err != nil {
		return out, fmt.Errorf("parse frontmatter: %w", err)
	}
	out.Frontmatter = fm
	if err := decodeStandard(&out, opts); err != nil {
		return out, err
	}

	manifest, hasManifest, err := readManifest(abs)
	if err != nil {
		return out, err
	}
	out.Skit, err = metadata.FromCarriers(fm, manifest, hasManifest)
	if err != nil {
		return out, err
	}
	return out, nil
}

func markerFile(root string) (string, []string, error) {
	upper := filepath.Join(root, "SKILL.md")
	lower := filepath.Join(root, "skill.md")
	upperExists := exists(upper)
	lowerExists := exists(lower)
	switch {
	case upperExists:
		var warnings []string
		if lowerExists {
			warnings = append(warnings, "skill.md ignored because SKILL.md exists")
		}
		return upper, warnings, nil
	case lowerExists:
		return lower, []string{"lowercase skill.md is accepted for compatibility"}, nil
	default:
		return "", nil, fmt.Errorf("SKILL.md not found in %s", root)
	}
}

func splitFrontmatter(src string) (string, string, error) {
	src = strings.TrimPrefix(src, "\ufeff")
	if !strings.HasPrefix(src, "---\n") && !strings.HasPrefix(src, "---\r\n") {
		return "", "", fmt.Errorf("SKILL.md must start with YAML frontmatter")
	}
	lines := strings.SplitAfter(src, "\n")
	var fm strings.Builder
	for i := 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "---" {
			body := strings.Join(lines[i+1:], "")
			return fm.String(), body, nil
		}
		fm.WriteString(lines[i])
	}
	return "", "", fmt.Errorf("frontmatter closing delimiter not found")
}

func decodeStandard(s *Skill, opts ParseOptions) error {
	name, ok := metadata.AsString(s.Frontmatter["name"])
	if !ok || name == "" {
		return fmt.Errorf("name is required")
	}
	description, ok := metadata.AsString(s.Frontmatter["description"])
	if !ok || description == "" {
		return fmt.Errorf("description is required")
	}
	if !ValidName(name) {
		return fmt.Errorf("invalid skill name %q", name)
	}
	basename := filepath.Base(s.Root)
	if opts.ExpectedBasename != "" {
		basename = opts.ExpectedBasename
	}
	if name != basename {
		s.Warnings = append(s.Warnings, fmt.Sprintf("skill name %q does not match directory basename %q", name, basename))
	}
	if len(description) > 1024 {
		return fmt.Errorf("description must be at most 1024 characters")
	}
	if compatibility, ok := metadata.AsString(s.Frontmatter["compatibility"]); ok {
		if compatibility == "" || len(compatibility) > 500 {
			return fmt.Errorf("compatibility must be 1-500 characters when present")
		}
		s.Compatibility = compatibility
	}
	if meta, ok := s.Frontmatter["metadata"]; ok {
		metaMap, ok := metadata.AsMap(meta)
		if !ok {
			return fmt.Errorf("metadata must be a mapping")
		}
		if internal, ok := metaMap["internal"].(bool); ok {
			s.Internal = internal
		}
	}
	if allowedTools, ok := metadata.AsString(s.Frontmatter["allowed-tools"]); ok {
		s.AllowedTools = allowedTools
	}
	s.Name = name
	s.Description = description
	s.License, _ = metadata.AsString(s.Frontmatter["license"])
	return nil
}

func readManifest(root string) (metadata.YAMLMap, bool, error) {
	path := filepath.Join(root, "skill.yaml")
	if !exists(path) {
		return nil, false, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}
	m, err := metadata.ParseYAML(string(raw))
	if err != nil {
		return nil, false, fmt.Errorf("parse skill.yaml: %w", err)
	}
	return m, true, nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
