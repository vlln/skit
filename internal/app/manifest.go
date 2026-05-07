package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/vlln/skit/internal/xdg"
)

const manifestSchema = "skit.manifest/v1"

type Manifest struct {
	Schema string                   `json:"schema"`
	Skills map[string]ManifestSkill `json:"skills"`
}

type ManifestSkill struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Source      ManifestSource `json:"source"`
	Path        string         `json:"path"`
	Agents      []string       `json:"agents,omitempty"`
}

type ManifestSource struct {
	Type    string `json:"type"`
	Locator string `json:"locator"`
	URL     string `json:"url,omitempty"`
	Ref     string `json:"ref,omitempty"`
	Subpath string `json:"subpath,omitempty"`
	Skill   string `json:"skill,omitempty"`
}

func dataRoot() string {
	return filepath.Join(xdg.DataHome(), "skit")
}

func manifestPath() string {
	return filepath.Join(dataRoot(), "manifest.json")
}

func installedSkillsRoot() string {
	return filepath.Join(dataRoot(), "skills")
}

func tmpRoot() string {
	return filepath.Join(xdg.CacheHome(), "skit", "tmp")
}

func readManifest() (Manifest, error) {
	path := manifestPath()
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return newManifest(), nil
	}
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return Manifest{}, err
	}
	if manifest.Schema == "" {
		manifest.Schema = manifestSchema
	}
	if manifest.Skills == nil {
		manifest.Skills = map[string]ManifestSkill{}
	}
	return manifest, nil
}

func writeManifestFile(manifest Manifest) error {
	raw, err := manifestJSON(manifest)
	if err != nil {
		return err
	}
	path := manifestPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".manifest-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

func manifestJSON(manifest Manifest) ([]byte, error) {
	if manifest.Schema == "" {
		manifest.Schema = manifestSchema
	}
	if manifest.Skills == nil {
		manifest.Skills = map[string]ManifestSkill{}
	}
	sorted := Manifest{Schema: manifest.Schema, Skills: map[string]ManifestSkill{}}
	for _, name := range manifestNames(manifest) {
		sorted.Skills[name] = manifest.Skills[name]
	}
	raw, err := json.MarshalIndent(sorted, "", "  ")
	if err != nil {
		return nil, err
	}
	raw = append(raw, '\n')
	return raw, nil
}

func newManifest() Manifest {
	return Manifest{Schema: manifestSchema, Skills: map[string]ManifestSkill{}}
}

func manifestNames(manifest Manifest) []string {
	names := make([]string, 0, len(manifest.Skills))
	for name := range manifest.Skills {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func uniqueAgents(agents []string) []string {
	seen := map[string]bool{}
	out := []string{"universal"}
	seen["universal"] = true
	for _, agent := range agents {
		if agent == "" || seen[agent] {
			continue
		}
		seen[agent] = true
		out = append(out, agent)
	}
	return out
}

func withoutAgent(agents []string, remove string) []string {
	var out []string
	seen := map[string]bool{}
	for _, agent := range agents {
		if agent == "" || agent == remove || seen[agent] {
			continue
		}
		seen[agent] = true
		out = append(out, agent)
	}
	return out
}
