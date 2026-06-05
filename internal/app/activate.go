package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vlln/skit/internal/skill"
)

type ActivateInstalledRequest struct {
	CWD    string
	Scope  Scope
	Name   string
	Agents []string
	Force  bool
}

func ActivateInstalled(req ActivateInstalledRequest) (AddResult, error) {
	var result AddResult
	if !skill.ValidName(req.Name) {
		return result, fmt.Errorf("invalid skill name %q", req.Name)
	}
	manifest, err := readManifest()
	if err != nil {
		return result, err
	}
	entry, ok := manifest.Skills[req.Name]
	if !ok {
		return result, fmt.Errorf("%s is not installed", req.Name)
	}
	installDir := filepath.Join(dataRoot(), filepath.FromSlash(entry.Path))
	if _, err := os.Stat(installDir); err != nil {
		if os.IsNotExist(err) {
			return result, fmt.Errorf("%s is installed but local skill directory is missing: %s", req.Name, installDir)
		}
		return result, err
	}
	if _, err := skill.ParseDir(installDir); err != nil {
		return result, fmt.Errorf("%s is installed but local skill is invalid: %w", req.Name, err)
	}

	agents := entry.Agents
	if len(req.Agents) > 0 {
		agents = appendUnique(agents, uniqueAgents(req.Agents)...)
	}
	if len(agents) == 0 {
		agents = uniqueAgents(nil)
	}
	activeDirs, err := manifestActiveDirs(req.Scope, req.CWD, agents)
	if err != nil {
		return result, err
	}
	for _, dir := range activeDirs {
		activePath, err := activateNameInDir(dir, req.Name, installDir, req.Force)
		if err != nil {
			return result, err
		}
		result.ActivePaths = append(result.ActivePaths, activePath)
	}
	entry.Agents = agents
	manifest.Skills[req.Name] = entry
	if err := writeManifestFile(manifest); err != nil {
		return result, err
	}
	result.Entries = append(result.Entries, entry)
	return result, nil
}
