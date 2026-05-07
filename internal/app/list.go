package app

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/vlln/skit/internal/skill"
)

type ListRequest struct {
	CWD    string
	Scope  Scope
	Agents []string
	All    bool
}

type ListEntry struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Source      ManifestSource `json:"source"`
	Path        string         `json:"path"`
	Agents      []string       `json:"agents,omitempty"`
	Active      []string       `json:"active,omitempty"`
	Missing     bool           `json:"missing,omitempty"`
	Managed     bool           `json:"managed,omitempty"`
	Scope       string         `json:"scope,omitempty"`
}

func List(req ListRequest) ([]ListEntry, error) {
	manifest, err := readManifest()
	if err != nil {
		return nil, err
	}
	byName := map[string]*ListEntry{}
	for _, name := range manifestNames(manifest) {
		entry := manifest.Skills[name]
		installDir := filepath.Join(dataRoot(), filepath.FromSlash(entry.Path))
		item := ListEntry{
			Name:        entry.Name,
			Description: entry.Description,
			Source:      entry.Source,
			Path:        installDir,
			Agents:      entry.Agents,
			Managed:     true,
			Scope:       "managed",
		}
		if _, err := os.Stat(filepath.Join(installDir, "SKILL.md")); err != nil {
			item.Missing = true
		}
		dirs, err := manifestActiveDirs(entry.Agents)
		if err != nil {
			return nil, err
		}
		for _, dir := range dirs {
			link := filepath.Join(dir, name)
			if linkPointsTo(link, installDir) {
				item.Active = append(item.Active, link)
			}
		}
		byName[item.Name] = &item
	}
	if req.All {
		if err := scanVisibleSkills(req, byName); err != nil {
			return nil, err
		}
	}
	out := make([]ListEntry, 0, len(byName))
	for _, entry := range byName {
		sort.Strings(entry.Active)
		sort.Strings(entry.Agents)
		out = append(out, *entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func scanVisibleSkills(req ListRequest, byName map[string]*ListEntry) error {
	for _, root := range listAllRoots(req.CWD) {
		entries, err := os.ReadDir(root.Dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			dir := filepath.Join(root.Dir, entry.Name())
			parsed, err := skill.ParseDirWithOptions(dir, skill.ParseOptions{AllowNameMismatch: true})
			if err != nil {
				continue
			}
			name := entry.Name()
			item, ok := byName[name]
			if !ok {
				item = &ListEntry{
					Name:        name,
					Description: parsed.Description,
					Source:      ManifestSource{Type: "external", Locator: dir},
					Path:        dir,
					Scope:       root.Scope,
				}
				byName[name] = item
			}
			for _, agent := range root.Agents {
				item.Agents = appendUnique(item.Agents, agent)
			}
			item.Active = appendUnique(item.Active, dir)
		}
	}
	return nil
}

type listRoot struct {
	Dir    string
	Agents []string
	Scope  string
}

func listAllRoots(cwd string) []listRoot {
	if cwd == "" {
		cwd = "."
	}
	byDir := map[string]*listRoot{}
	add := func(dir, agent, scope string) {
		clean := filepath.Clean(dir)
		root, ok := byDir[clean]
		if ok {
			root.Agents = appendUnique(root.Agents, agent)
			return
		}
		byDir[clean] = &listRoot{Dir: clean, Agents: []string{agent}, Scope: scope}
	}
	add(filepath.Join(userHome(), ".agents", "skills"), "universal", "global")
	for _, agent := range sortedAgents() {
		target := agentTargets[agent]
		projectDir := target.Project
		if !filepath.IsAbs(projectDir) {
			projectDir = filepath.Join(cwd, projectDir)
		}
		add(projectDir, agent, "project")
		add(target.Global(), agent, "global")
	}
	roots := make([]listRoot, 0, len(byDir))
	for _, root := range byDir {
		sort.Strings(root.Agents)
		roots = append(roots, *root)
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].Dir < roots[j].Dir })
	return roots
}

func sortedAgents() []string {
	names := make([]string, 0, len(agentTargets))
	for name := range agentTargets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func linkPointsTo(link, target string) bool {
	got, err := os.Readlink(link)
	if err != nil {
		return false
	}
	if !filepath.IsAbs(got) {
		got = filepath.Join(filepath.Dir(link), got)
	}
	return samePath(got, target)
}
