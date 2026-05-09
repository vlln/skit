package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const sourcesSchema = "skit.sources/v1"

var sourceNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

type SourcesFile struct {
	Schema  string         `json:"schema"`
	Sources []SearchSource `json:"sources"`
}

type SearchSource struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Source  string `json:"source,omitempty"`
	URL     string `json:"url,omitempty"`
	Enabled bool   `json:"enabled"`
}

type SourceAddRequest struct {
	Name   string
	Type   string
	Source string
}

func SourcesPath() string {
	return filepath.Join(dataRoot(), "sources.json")
}

func DefaultSearchSources() []SearchSource {
	return []SearchSource{{
		Name:    "skills-sh",
		Type:    "registry",
		URL:     "https://skills.sh",
		Enabled: true,
	}}
}

func ListSearchSources() ([]SearchSource, error) {
	file, err := readSourcesFile()
	if err != nil {
		return nil, err
	}
	out := append([]SearchSource(nil), file.Sources...)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func AddSearchSource(req SourceAddRequest) ([]SearchSource, error) {
	name := strings.TrimSpace(req.Name)
	if !sourceNamePattern.MatchString(name) {
		return nil, fmt.Errorf("invalid source name %q", req.Name)
	}
	typ := strings.ToLower(strings.TrimSpace(req.Type))
	if typ != "registry" && typ != "repo" && typ != "json" && typ != "local" {
		return nil, fmt.Errorf("unsupported source type %q", req.Type)
	}
	locator := strings.TrimSpace(req.Source)
	if locator == "" {
		return nil, fmt.Errorf("source locator is required")
	}
	file, err := readSourcesFile()
	if err != nil {
		return nil, err
	}
	entry := SearchSource{Name: name, Type: typ, Enabled: true}
	if typ == "registry" || strings.HasPrefix(locator, "http://") || strings.HasPrefix(locator, "https://") {
		entry.URL = locator
	} else {
		entry.Source = locator
	}
	replaced := false
	for i := range file.Sources {
		if file.Sources[i].Name == name {
			file.Sources[i] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		file.Sources = append(file.Sources, entry)
	}
	if err := writeSourcesFile(file); err != nil {
		return nil, err
	}
	return ListSearchSources()
}

func RemoveSearchSource(name string) ([]SearchSource, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("source name is required")
	}
	file, err := readSourcesFile()
	if err != nil {
		return nil, err
	}
	out := file.Sources[:0]
	found := false
	for _, entry := range file.Sources {
		if entry.Name == name {
			found = true
			continue
		}
		out = append(out, entry)
	}
	if !found {
		return nil, fmt.Errorf("source %q not found", name)
	}
	file.Sources = out
	if err := writeSourcesFile(file); err != nil {
		return nil, err
	}
	return ListSearchSources()
}

func readSourcesFile() (SourcesFile, error) {
	path := SourcesPath()
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return SourcesFile{Schema: sourcesSchema, Sources: DefaultSearchSources()}, nil
	}
	if err != nil {
		return SourcesFile{}, err
	}
	var file SourcesFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return SourcesFile{}, err
	}
	if file.Schema == "" {
		file.Schema = sourcesSchema
	}
	return normalizeSourcesFile(file), nil
}

func writeSourcesFile(file SourcesFile) error {
	file = normalizeSourcesFile(file)
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	path := SourcesPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0644)
}

func EnableSearchSource(name string) ([]SearchSource, error) {
	return setSourceEnabled(name, true)
}

func DisableSearchSource(name string) ([]SearchSource, error) {
	return setSourceEnabled(name, false)
}

func setSourceEnabled(name string, enabled bool) ([]SearchSource, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("source name is required")
	}
	file, err := readSourcesFile()
	if err != nil {
		return nil, err
	}
	found := false
	for i := range file.Sources {
		if file.Sources[i].Name == name {
			file.Sources[i].Enabled = enabled
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("source %q not found", name)
	}
	if err := writeSourcesFile(file); err != nil {
		return nil, err
	}
	return ListSearchSources()
}

func normalizeSourcesFile(file SourcesFile) SourcesFile {
	if file.Schema == "" {
		file.Schema = sourcesSchema
	}
	if file.Sources == nil {
		file.Sources = []SearchSource{}
	}
	for i := range file.Sources {
		file.Sources[i].Type = strings.ToLower(strings.TrimSpace(file.Sources[i].Type))
		if !file.Sources[i].Enabled {
			continue
		}
	}
	sort.SliceStable(file.Sources, func(i, j int) bool {
		return file.Sources[i].Name < file.Sources[j].Name
	})
	return file
}
