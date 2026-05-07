package app

import (
	"fmt"
	"os"
	"path/filepath"
)

type RemoveRequest struct {
	CWD    string
	Scope  Scope
	Name   string
	Agents []string
	Keep   bool

	// Deprecated pre-1.0 flag.
	Prune bool
}

type RemoveResult struct {
	Removed      bool     `json:"removed"`
	Unlinked     []string `json:"unlinked,omitempty"`
	Deleted      []string `json:"deleted,omitempty"`
	Skipped      []string `json:"skipped,omitempty"`
	StillTracked bool     `json:"stillTracked,omitempty"`
}

func Remove(req RemoveRequest) (RemoveResult, error) {
	var result RemoveResult
	manifest, err := readManifest()
	if err != nil {
		return result, err
	}
	entry, ok := manifest.Skills[req.Name]
	if !ok {
		return result, nil
	}
	installDir := filepath.Join(dataRoot(), filepath.FromSlash(entry.Path))
	if len(req.Agents) > 0 {
		for _, agent := range req.Agents {
			dir, err := activeDirForManifestAgent(agent)
			if err != nil {
				return result, err
			}
			link := filepath.Join(dir, req.Name)
			removed, skipped, err := unlinkManaged(link, installDir)
			if err != nil {
				return result, err
			}
			if removed {
				result.Unlinked = append(result.Unlinked, link)
			}
			if skipped != "" {
				result.Skipped = append(result.Skipped, skipped)
			}
			entry.Agents = withoutAgent(entry.Agents, agent)
		}
		manifest.Skills[req.Name] = entry
		if err := writeManifestFile(manifest); err != nil {
			return result, err
		}
		result.Removed = true
		result.StillTracked = true
		return result, nil
	}
	dirs, err := manifestActiveDirs(entry.Agents)
	if err != nil {
		return result, err
	}
	for _, dir := range dirs {
		link := filepath.Join(dir, req.Name)
		removed, skipped, err := unlinkManaged(link, installDir)
		if err != nil {
			return result, err
		}
		if removed {
			result.Unlinked = append(result.Unlinked, link)
		}
		if skipped != "" {
			result.Skipped = append(result.Skipped, skipped)
		}
	}
	if !req.Keep {
		if err := removeInstalledSkill(req.Name, installDir); err != nil {
			return result, err
		}
		result.Deleted = append(result.Deleted, installDir)
	}
	delete(manifest.Skills, req.Name)
	if err := writeManifestFile(manifest); err != nil {
		return result, err
	}
	result.Removed = true
	return result, nil
}

func activeDirForManifestAgent(agent string) (string, error) {
	if agent == "universal" || agent == "" {
		dirs, _ := manifestActiveDirs([]string{"universal"})
		return dirs[0], nil
	}
	return agentActiveDir(Global, "", agent)
}

func unlinkManaged(link, target string) (bool, string, error) {
	info, err := os.Lstat(link)
	if os.IsNotExist(err) {
		return false, "missing " + link, nil
	}
	if err != nil {
		return false, "", err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return false, "not a skit symlink " + link, nil
	}
	if !linkPointsTo(link, target) {
		return false, "points elsewhere " + link, nil
	}
	if err := os.Remove(link); err != nil {
		return false, "", err
	}
	return true, "", nil
}

func removeInstalledSkill(name, dir string) error {
	root := installedSkillsRoot()
	expected, err := safeChild(root, name)
	if err != nil {
		return err
	}
	if !samePath(expected, dir) {
		return fmt.Errorf("refusing to remove unexpected path %s", dir)
	}
	return os.RemoveAll(dir)
}
