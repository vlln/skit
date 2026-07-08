package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/vlln/skit/internal/metadata"
)

// ValidName checks whether name conforms to the naming rules:
// 1-64 characters, Unicode lowercase letters/digits/hyphens, no leading/trailing
// or consecutive hyphens. Callers should NFKC-normalize the name first.
func ValidName(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return false
	}
	if strings.Contains(name, "--") {
		return false
	}
	for _, r := range name {
		if unicode.IsDigit(r) || r == '-' {
			continue
		}
		if unicode.IsLetter(r) {
			if unicode.IsUpper(r) {
				return false
			}
			continue
		}
		return false
	}
	return true
}

// NormalizeName applies NFKC normalization.
func NormalizeName(name string) string {
	return norm.NFKC.String(name)
}

// Skill holds the fields skit cares about. All fields except Root and File
// are optional. Frontmatter preserves the complete raw YAML for pass-through.
type Skill struct {
	Root        string            // absolute path to skill directory
	File        string            // path to SKILL.md (or skill.md)
	Name        string            // defaults to directory basename
	Description string            // recommended
	License     string            // optional
	Metadata    map[string]string // optional annotations (author, version, ...)
	Requires    metadata.Requires // structured runtime requirements
	Frontmatter metadata.YAMLMap  // complete raw frontmatter
	Warnings    []string
}

func ParseDir(root string) (Skill, error) {
	return ParseDirWithOptions(root, ParseOptions{})
}

type ParseOptions struct {
	ExpectedBasename string
	IncludeInternal  bool
	FullDepth        bool
	IgnoreInvalid    bool
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
		fm2, err2 := metadata.ParseYAMLLenient(fmText)
		if err2 != nil {
			return out, fmt.Errorf("parse frontmatter: %w", err)
		}
		fm = fm2
		out.Warnings = append(out.Warnings, "frontmatter YAML was repaired (e.g. unquoted colons)")
	}
	out.Frontmatter = fm

	decodeStandard(&out, opts)
	out.Requires = metadata.FromCarriers(fm).Requires
	return out, nil
}

func markerFile(root string) (string, []string, error) {
	upper := filepath.Join(root, "SKILL.md")
	lower := filepath.Join(root, "skill.md")
	upperExists := exists(upper)
	lowerExists := exists(lower)
	// On case-insensitive filesystems (e.g. macOS default), os.Stat("skill.md")
	// matches "SKILL.md", so we must verify via a directory listing that both
	// names actually exist as separate entries.
	if upperExists && lowerExists {
		entries, readErr := os.ReadDir(root)
		if readErr == nil {
			hasUpper, hasLower := false, false
			for _, e := range entries {
				switch e.Name() {
				case "SKILL.md":
					hasUpper = true
				case "skill.md":
					hasLower = true
				}
			}
			upperExists = hasUpper
			lowerExists = hasLower
		}
	}
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

// decodeStandard extracts the fields skit cares about. All are optional.
// Warnings are emitted only for quality issues that may indicate mistakes.
func decodeStandard(s *Skill, opts ParseOptions) {
	basename := filepath.Base(s.Root)
	if opts.ExpectedBasename != "" {
		basename = opts.ExpectedBasename
	}

	// name — optional, defaults to directory name
	if name, ok := metadata.AsString(s.Frontmatter["name"]); ok && name != "" {
		s.Name = NormalizeName(name)
	} else {
		s.Name = basename
	}
	if !ValidName(s.Name) {
		s.Warnings = append(s.Warnings, fmt.Sprintf(
			"name %q is invalid (must be 1-64 lowercase letters/digits/hyphens, no leading/trailing/consecutive hyphens)", s.Name))
	}
	if s.Name != basename {
		s.Warnings = append(s.Warnings, fmt.Sprintf(
			"name %q does not match directory basename %q", s.Name, basename))
	}

	// description — optional
	if desc, ok := metadata.AsString(s.Frontmatter["description"]); ok {
		s.Description = desc
		if len(desc) > 1024 {
			s.Warnings = append(s.Warnings, fmt.Sprintf("description exceeds 1024 characters (%d)", len(desc)))
		}
	}

	// license — optional
	if v, ok := metadata.AsString(s.Frontmatter["license"]); ok {
		s.License = v
	}

	// metadata — optional, map[string]string
	if meta, ok := s.Frontmatter["metadata"]; ok {
		s.Metadata = toStringMap(meta)
	}
}

// toStringMap converts a frontmatter value to map[string]string.
func toStringMap(v any) map[string]string {
	switch t := v.(type) {
	case metadata.YAMLMap:
		out := make(map[string]string, len(t))
		for k, val := range t {
			out[k] = fmt.Sprint(val)
		}
		return out
	case map[string]any:
		out := make(map[string]string, len(t))
		for k, val := range t {
			out[k] = fmt.Sprint(val)
		}
		return out
	default:
		return nil
	}
}

// ValidateSkill performs strict validation for publish/validate workflows.
func ValidateSkill(s Skill) []string {
	var errs []string

	if s.Name == "" {
		errs = append(errs, "name is required")
	} else if !ValidName(s.Name) {
		errs = append(errs, fmt.Sprintf("invalid name %q", s.Name))
	}

	basename := filepath.Base(s.Root)
	if s.Name != "" && s.Name != basename {
		errs = append(errs, fmt.Sprintf("name %q must match directory name %q", s.Name, basename))
	}

	if s.Description == "" {
		errs = append(errs, "description is recommended")
	}

	return errs
}

// isInternal checks whether the skill is marked as internal via metadata.
func isInternal(s Skill) bool {
	return s.Metadata["internal"] == "true"
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
