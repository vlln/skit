package lockfile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const Schema = "skit.lock/v1"

type Lock struct {
	Schema string           `json:"schema"`
	Skills map[string]Entry `json:"skills"`
}

type Entry struct {
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	Source       Source       `json:"source"`
	Hashes       Hashes       `json:"hashes"`
	Dependencies []Dependency `json:"dependencies,omitempty"`
	Incomplete   bool         `json:"incomplete,omitempty"`
	Warnings     []string     `json:"warnings,omitempty"`
}

type Source struct {
	Type        string `json:"type"`
	Locator     string `json:"locator"`
	URL         string `json:"url,omitempty"`
	Ref         string `json:"ref,omitempty"`
	ResolvedRef string `json:"resolvedRef,omitempty"`
	Subpath     string `json:"subpath,omitempty"`
	Skill       string `json:"skill,omitempty"`
}

type Hashes struct {
	Tree    string `json:"tree"`
	SkillMD string `json:"skillMd"`
}

type Dependency struct {
	Name     string `json:"name"`
	Source   Source `json:"source"`
	Optional bool   `json:"optional"`
}

func New() Lock {
	return Lock{Schema: Schema, Skills: map[string]Entry{}}
}

func Read(path string) (Lock, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return New(), nil
	}
	if err != nil {
		return Lock{}, err
	}
	var lock Lock
	if err := json.Unmarshal(raw, &lock); err != nil {
		return Lock{}, err
	}
	if lock.Schema == "" {
		lock.Schema = Schema
	}
	if lock.Schema != Schema {
		return Lock{}, fmt.Errorf("unsupported lock schema %q", lock.Schema)
	}
	if lock.Skills == nil {
		lock.Skills = map[string]Entry{}
	}
	return lock, nil
}

func Write(path string, lock Lock) error {
	if lock.Schema == "" {
		lock.Schema = Schema
	}
	if lock.Skills == nil {
		lock.Skills = map[string]Entry{}
	}
	for name, entry := range lock.Skills {
		sort.Slice(entry.Dependencies, func(i, j int) bool {
			left := entry.Dependencies[i].Source.Locator + "\x00" + entry.Dependencies[i].Source.Skill
			right := entry.Dependencies[j].Source.Locator + "\x00" + entry.Dependencies[j].Source.Skill
			return left < right
		})
		lock.Skills[name] = entry
	}
	raw, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".lock-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	_, writeErr := tmp.Write(raw)
	closeErr := tmp.Close()
	if writeErr != nil {
		os.Remove(tmpName)
		return writeErr
	}
	if closeErr != nil {
		os.Remove(tmpName)
		return closeErr
	}
	return os.Rename(tmpName, path)
}

func Add(lock Lock, entry Entry) (Lock, error) {
	if lock.Schema == "" {
		lock.Schema = Schema
	}
	if lock.Skills == nil {
		lock.Skills = map[string]Entry{}
	}
	existing, ok := lock.Skills[entry.Name]
	if ok && !sameIdentity(existing, entry) {
		return lock, fmt.Errorf("skill %q already exists with different source or hash", entry.Name)
	}
	lock.Skills[entry.Name] = entry
	return lock, nil
}

func Put(lock Lock, entry Entry) Lock {
	if lock.Schema == "" {
		lock.Schema = Schema
	}
	if lock.Skills == nil {
		lock.Skills = map[string]Entry{}
	}
	lock.Skills[entry.Name] = entry
	return lock
}

func Remove(lock Lock, name string) (Lock, bool) {
	if lock.Skills == nil {
		return lock, false
	}
	if _, ok := lock.Skills[name]; !ok {
		return lock, false
	}
	delete(lock.Skills, name)
	return lock, true
}

func Names(lock Lock) []string {
	names := make([]string, 0, len(lock.Skills))
	for name := range lock.Skills {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sameIdentity(a, b Entry) bool {
	return a.Source.Type == b.Source.Type &&
		a.Source.Locator == b.Source.Locator &&
		a.Source.Ref == b.Source.Ref &&
		a.Source.Subpath == b.Source.Subpath &&
		a.Hashes.Tree == b.Hashes.Tree
}

func EqualJSON(a, b []byte) bool {
	return bytes.Equal(bytes.TrimSpace(a), bytes.TrimSpace(b))
}
