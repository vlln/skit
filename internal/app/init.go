package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vlln/skit/internal/skill"
)

type InitRequest struct {
	CWD  string
	Name string
}

type InitResult struct {
	Name     string   `json:"name"`
	RepoName string   `json:"repoName"`
	Path     string   `json:"path"`
	README   string   `json:"readme"`
	Created  []string `json:"created,omitempty"`
}

func Init(req InitRequest) (InitResult, error) {
	cwd := cleanCWD(req.CWD)
	base := req.Name
	if base == "" {
		base = filepath.Base(cwd)
	}
	name := skillNameFromRepo(base)
	if !skill.ValidName(name) {
		return InitResult{}, fmt.Errorf("invalid skill name %q", name)
	}
	repoName := repoNameForSkill(base)
	if !skill.ValidName(repoName) {
		return InitResult{}, fmt.Errorf("invalid repo name %q", repoName)
	}
	repoDir := filepath.Join(cwd, repoName)
	skillDir := filepath.Join(repoDir, "skills", name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return InitResult{}, err
	}
	readmePath := filepath.Join(repoDir, "README.md")
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if exists(readmePath) {
		return InitResult{}, fmt.Errorf("README.md already exists in %s", repoDir)
	}
	if exists(skillPath) {
		return InitResult{}, fmt.Errorf("SKILL.md already exists in %s", skillDir)
	}
	if err := os.WriteFile(readmePath, []byte(readmeTemplate(repoName, name)), 0644); err != nil {
		return InitResult{}, err
	}
	if err := os.WriteFile(skillPath, []byte(skillTemplate(name)), 0644); err != nil {
		return InitResult{}, err
	}
	return InitResult{
		Name:     name,
		RepoName: repoName,
		Path:     skillPath,
		README:   readmePath,
		Created:  []string{repoDir, readmePath, skillPath},
	}, nil
}

func repoNameForSkill(name string) string {
	if filepath.Base(name) != name {
		name = filepath.Base(name)
	}
	if strings.HasSuffix(name, "-skill") {
		return name
	}
	return name + "-skill"
}

func skillNameFromRepo(name string) string {
	if filepath.Base(name) != name {
		name = filepath.Base(name)
	}
	return strings.TrimSuffix(name, "-skill")
}

func skillTemplate(name string) string {
	return "---\n" +
		"name: " + name + "\n" +
		"description: \"TODO: describe " + name + ".\"\n" +
		"# Optional: uncomment fields that apply.\n" +
		"# license: MIT\n" +
		"# compatibility: Requires TODO tool/account/platform.\n" +
		"metadata:\n" +
		"  skit:\n" +
		"    version: 0.1.0\n" +
		"    # Optional: keywords help search and discovery.\n" +
		"    # keywords:\n" +
		"    #   - TODO-keyword\n" +
		"    # Optional: requirements the agent can diagnose before use.\n" +
		"    # requires:\n" +
		"    #   bins:\n" +
		"    #     - TODO-command\n" +
		"    #   env:\n" +
		"    #     - TODO_ENV_VAR\n" +
		"    #   skills:\n" +
		"    #     - github:owner/repo@required-skill\n" +
		"    #   platforms:\n" +
		"    #     os:\n" +
		"    #       - linux\n" +
		"    #       - darwin\n" +
		"---\n" +
		"# " + name + "\n" +
		"\n" +
		"Use this skill when TODO: describe the task or situation where an agent should use it.\n" +
		"\n" +
		"## Workflow\n" +
		"\n" +
		"1. TODO: identify the user's goal and relevant inputs.\n" +
		"2. TODO: inspect the required files, tools, or context.\n" +
		"3. TODO: perform the task using the repository or domain conventions.\n" +
		"4. TODO: validate the result and report the outcome clearly.\n" +
		"\n" +
		"## Notes\n" +
		"\n" +
		"- Keep instructions concise and procedural.\n" +
		"- Put human-facing installation and repository overview text in the root README.md, not here.\n"
}

func readmeTemplate(repoName, skillName string) string {
	return "# " + repoName + "\n" +
		"\n" +
		"Agent Skills for TODO: describe the repository purpose.\n" +
		"\n" +
		"This repository stores skills under `skills/`. Each skill follows the\n" +
		"[Agent Skills specification](https://agentskills.io/specification) and can be\n" +
		"used by skills-compatible agents.\n" +
		"\n" +
		"## Skills\n" +
		"\n" +
		"| Skill | Description |\n" +
		"|-------|-------------|\n" +
		"| [`" + skillName + "`](skills/" + skillName + ") | TODO: one-sentence description. |\n" +
		"\n" +
		"## Installation\n" +
		"\n" +
		"Install one skill with `skit`:\n" +
		"\n" +
		"```sh\n" +
		"skit install <owner>/<repo>/skills/" + skillName + "\n" +
		"```\n" +
		"\n" +
		"Install all skills in this repository:\n" +
		"\n" +
		"```sh\n" +
		"skit install <owner>/<repo> --all\n" +
		"```\n" +
		"\n" +
		"## Requirements\n" +
		"\n" +
		"TODO: list runtime tools, accounts, or platform requirements. Remove this section if none.\n" +
		"\n" +
		"## License\n" +
		"\n" +
		"TODO: add a license.\n"
}
