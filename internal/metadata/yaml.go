package metadata

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type YAMLMap map[string]any

// ParseYAML parses a YAML string into a YAMLMap. It requires the document
// to be a mapping at the top level.
func ParseYAML(src string) (YAMLMap, error) {
	if strings.TrimSpace(src) == "" {
		return YAMLMap{}, nil
	}
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(src), &raw); err != nil {
		return nil, err
	}
	value, ok := normalizeYAML(raw).(YAMLMap)
	if !ok {
		return nil, fmt.Errorf("YAML document must be a mapping")
	}
	return value, nil
}

// ParseYAMLLenient attempts to parse YAML that may contain common errors
// (e.g. unquoted values with colons). It is a best-effort fallback for
// ParseYAML.
func ParseYAMLLenient(src string) (YAMLMap, error) {
	repaired := repairYAML(src)
	return ParseYAML(repaired)
}

// repairYAML applies simple heuristics to fix common YAML issues:
// - Unquoted values containing colons (e.g. "description: Use when: user asks")
var yamlColonLine = regexp.MustCompile(`^(\s*)([\w-]+)\s*:\s*(.+)$`)

func repairYAML(src string) string {
	lines := strings.Split(src, "\n")
	var out []string
	for _, line := range lines {
		matches := yamlColonLine.FindStringSubmatch(line)
		if matches == nil {
			out = append(out, line)
			continue
		}
		indent := matches[1]
		key := matches[2]
		value := matches[3]

		// Skip if value is already quoted, empty, or looks like a nested key
		if strings.HasPrefix(value, `"`) || strings.HasPrefix(value, `'`) {
			out = append(out, line)
			continue
		}
		if value == "" || value == "|" || value == ">" || strings.HasPrefix(value, "|") || strings.HasPrefix(value, ">") {
			out = append(out, line)
			continue
		}
		// If value contains a colon and isn't already quoted, wrap it
		if strings.Contains(value, ":") {
			out = append(out, fmt.Sprintf(`%s%s: "%s"`, indent, key, escapeYAMLValue(value)))
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func escapeYAMLValue(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func normalizeYAML(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := YAMLMap{}
		for k, v := range t {
			out[k] = normalizeYAML(v)
		}
		return out
	case map[any]any:
		out := YAMLMap{}
		for k, v := range t {
			out[fmt.Sprint(k)] = normalizeYAML(v)
		}
		return out
	case []any:
		out := make([]any, 0, len(t))
		for _, item := range t {
			out = append(out, normalizeYAML(item))
		}
		return out
	default:
		return v
	}
}

func AsMap(v any) (YAMLMap, bool) {
	m, ok := v.(YAMLMap)
	return m, ok
}

func AsString(v any) (string, bool) {
	s, ok := v.(string)
	return s, ok
}
