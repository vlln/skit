package app

import (
	"os"
	"path/filepath"
)

type DoctorRequest struct {
	CWD   string
	Scope Scope
}

type DoctorResult struct {
	Checks []DoctorCheck `json:"checks"`
}

type DoctorCheck struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Skill    string `json:"skill,omitempty"`
	Message  string `json:"message"`
}

func Doctor(req DoctorRequest) (DoctorResult, error) {
	return Check(req)
}

func Check(req DoctorRequest) (DoctorResult, error) {
	var result DoctorResult
	manifest, err := readManifest()
	if err != nil {
		return result, err
	}
	if len(manifest.Skills) == 0 {
		result.Checks = append(result.Checks, DoctorCheck{Severity: "info", Code: "empty", Message: "no skills are installed"})
		return result, nil
	}
	for _, name := range manifestNames(manifest) {
		entry := manifest.Skills[name]
		installDir := filepath.Join(dataRoot(), filepath.FromSlash(entry.Path))
		if _, err := os.Stat(installDir); err != nil {
			result.Checks = append(result.Checks, DoctorCheck{Severity: "error", Code: "missing-skill", Skill: name, Message: "installed skill directory is missing"})
			continue
		}
		if _, err := os.Stat(filepath.Join(installDir, "SKILL.md")); err != nil {
			result.Checks = append(result.Checks, DoctorCheck{Severity: "error", Code: "missing-skill-md", Skill: name, Message: "SKILL.md is missing"})
		}
		dirs, err := manifestActiveDirs(entry.Agents)
		if err != nil {
			return result, err
		}
		for _, dir := range dirs {
			link := filepath.Join(dir, name)
			info, err := os.Lstat(link)
			if os.IsNotExist(err) {
				result.Checks = append(result.Checks, DoctorCheck{Severity: "error", Code: "missing-active-link", Skill: name, Message: link})
				continue
			}
			if err != nil {
				result.Checks = append(result.Checks, DoctorCheck{Severity: "error", Code: "active-link-error", Skill: name, Message: err.Error()})
				continue
			}
			if info.Mode()&os.ModeSymlink == 0 {
				result.Checks = append(result.Checks, DoctorCheck{Severity: "error", Code: "active-path-conflict", Skill: name, Message: link})
				continue
			}
			if !linkPointsTo(link, installDir) {
				result.Checks = append(result.Checks, DoctorCheck{Severity: "error", Code: "active-link-target", Skill: name, Message: link})
			}
		}
	}
	return result, nil
}
