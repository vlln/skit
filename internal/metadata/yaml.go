package metadata

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type YAMLMap map[string]any

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
