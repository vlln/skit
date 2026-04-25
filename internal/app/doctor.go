package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vlln/skit/internal/diagnose"
	"github.com/vlln/skit/internal/hash"
	"github.com/vlln/skit/internal/lockfile"
	"github.com/vlln/skit/internal/skill"
	"github.com/vlln/skit/internal/store"
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
	var result DoctorResult
	paths := store.PathsFor(req.Scope, cleanCWD(req.CWD))
	lock, err := lockfile.Read(paths.Lock)
	if err != nil {
		return result, err
	}
	if len(lock.Skills) == 0 {
		result.Checks = append(result.Checks, DoctorCheck{Severity: "info", Code: "empty-lock", Message: "no skills are locked"})
		return result, nil
	}
	for _, name := range lockfile.Names(lock) {
		entry := lock.Skills[name]
		if entry.Incomplete {
			result.Checks = append(result.Checks, DoctorCheck{Severity: "warning", Code: "incomplete", Skill: name, Message: "entry is incomplete and cannot be restored automatically"})
			continue
		}
		path := storePath(paths, entry)
		if !exists(path) {
			result.Checks = append(result.Checks, DoctorCheck{Severity: "error", Code: "missing-store", Skill: name, Message: "store entry is missing; run skit install"})
			continue
		}
		parsed, err := skill.ParseDir(path)
		if err != nil {
			result.Checks = append(result.Checks, DoctorCheck{Severity: "error", Code: "invalid-skill", Skill: name, Message: err.Error()})
			continue
		}
		hashes, err := hash.Tree(parsed.Root, parsed.File)
		if err != nil {
			result.Checks = append(result.Checks, DoctorCheck{Severity: "error", Code: "hash-error", Skill: name, Message: err.Error()})
			continue
		}
		if hashes.Tree != entry.Hashes.Tree || hashes.SkillMD != entry.Hashes.SkillMD {
			result.Checks = append(result.Checks, DoctorCheck{Severity: "error", Code: "hash-mismatch", Skill: name, Message: "store content differs from lock"})
		}
		warnings := appendUnique(nil, entry.Warnings...)
		warnings = appendUnique(warnings, parsed.Warnings...)
		safetyWarnings, err := diagnose.SafetyWarnings(parsed.Root)
		if err != nil {
			result.Checks = append(result.Checks, DoctorCheck{Severity: "error", Code: "safety-scan-error", Skill: name, Message: err.Error()})
			continue
		}
		warnings = appendUnique(warnings, safetyWarnings...)
		for _, warning := range warnings {
			result.Checks = append(result.Checks, DoctorCheck{Severity: "warning", Code: "warning", Skill: name, Message: warning})
		}
		result.Checks = append(result.Checks, requirementChecks(name, parsed)...)
	}
	return result, nil
}
func requirementChecks(name string, parsed skill.Skill) []DoctorCheck {
	var checks []DoctorCheck
	for _, bin := range parsed.Skit.Requires.Bins {
		if _, err := exec.LookPath(bin); err != nil {
			checks = append(checks, DoctorCheck{Severity: "error", Code: "missing-bin", Skill: name, Message: "missing required binary: " + bin})
		}
	}
	if len(parsed.Skit.Requires.AnyBins) > 0 {
		found := false
		for _, bin := range parsed.Skit.Requires.AnyBins {
			if _, err := exec.LookPath(bin); err == nil {
				found = true
				break
			}
		}
		if !found {
			checks = append(checks, DoctorCheck{Severity: "error", Code: "missing-any-bin", Skill: name, Message: "missing one of required binaries: " + strings.Join(parsed.Skit.Requires.AnyBins, ", ")})
		}
	}
	for _, key := range parsed.Skit.Requires.Env {
		if _, ok := os.LookupEnv(key); !ok {
			checks = append(checks, DoctorCheck{Severity: "error", Code: "missing-env", Skill: name, Message: "missing environment variable: " + key})
		}
	}
	for _, cfg := range parsed.Skit.Requires.Config {
		if !exists(expandPath(cfg)) {
			checks = append(checks, DoctorCheck{Severity: "error", Code: "missing-config", Skill: name, Message: "missing config path: " + cfg})
		}
	}
	if len(parsed.Skit.Platforms.OS) > 0 && !contains(parsed.Skit.Platforms.OS, runtime.GOOS) {
		checks = append(checks, DoctorCheck{Severity: "error", Code: "unsupported-os", Skill: name, Message: "unsupported OS: " + runtime.GOOS})
	}
	if len(parsed.Skit.Platforms.Arch) > 0 && !contains(parsed.Skit.Platforms.Arch, runtime.GOARCH) {
		checks = append(checks, DoctorCheck{Severity: "error", Code: "unsupported-arch", Skill: name, Message: "unsupported architecture: " + runtime.GOARCH})
	}
	return checks
}

func expandPath(path string) string {
	path = os.ExpandEnv(path)
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
