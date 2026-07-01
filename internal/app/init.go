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
		"# when-to-use: \"TODO: additional trigger context.\"\n" +
		"# allowed-tools:\n" +
		"#   - Bash(git:*)\n" +
		"#   - Read\n" +
		"metadata:\n" +
		"  author: TODO\n" +
		"  version: \"0.1.0\"\n" +
		"# Optional: structured requirements for automated diagnostics.\n" +
		"# requires:\n" +
		"#   bins:\n" +
		"#     - TODO-command\n" +
		"#   env:\n" +
		"#     - TODO_ENV_VAR\n" +
		"#   skills:\n" +
		"#     - github:owner/repo@required-skill\n" +
		"#   platforms:\n" +
		"#     os:\n" +
		"#       - linux\n" +
		"#       - darwin\n" +
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
	return "<h1 align=\"center\">" + repoName + "</h1>\n" +
		"\n" +
		"<p align=\"center\">\n" +
		"  <strong>TODO: one-line value proposition.</strong><br/>\n" +
		"  TODO: two to three lines describing what the skills do, the target\n" +
		"  audience, and the key benefit.\n" +
		"</p>\n" +
		"\n" +
		"<p align=\"center\">\n" +
		"  <a href=\"https://github.com/<owner>/<repo>/stargazers\"><img src=\"https://badgen.net/github/stars/<owner>/<repo>?label=%E2%98%85\" alt=\"GitHub stars\" /></a>\n" +
		"  <img src=\"https://badgen.net/badge/license/<license>/blue\" alt=\"<license>\" />\n" +
		"  <img src=\"https://badgen.net/badge/spec/Agent%20Skills/8257D0\" alt=\"Agent Skills spec\" />\n" +
		"</p>\n" +
		"\n" +
		"<p align=\"center\">\n" +
		"  <sub><a href=\"README.md\">English</a> · <a href=\"docs/readme/README.zh-CN.md\">中文</a></sub>\n" +
		"</p>\n" +
		"\n" +
		"---\n" +
		"\n" +
		"## Installation\n" +
		"\n" +
		"### [skit](https://github.com/vlln/skit) (Recommended)\n" +
		"\n" +
		"```bash\n" +
		"skit install ./" + repoName + " --all\n" +
		"```\n" +
		"\n" +
		"### [skills](https://github.com/bananaml/skills)\n" +
		"\n" +
		"```bash\n" +
		"npx skills add git@github.com:<owner>/<repo>.git\n" +
		"```\n" +
		"\n" +
		"### Manually\n" +
		"\n" +
		"| Agent | Command |\n" +
		"|-------|---------|\n" +
		"| **Claude Code** | `cp -r skills/" + skillName + " .claude/skills/` |\n" +
		"| **Codex** | `cp -r skills/" + skillName + " ~/.codex/skills/` |\n" +
		"| **OpenCode** | `git clone https://github.com/<owner>/<repo>.git ~/.opencode/skills/" + repoName + "` |\n" +
		"| **Kimi** | `cp -r skills/" + skillName + " ~/.kimi/skills/` |\n" +
		"\n" +
		"---\n" +
		"\n" +
		"## Skills\n" +
		"\n" +
		"| Skill | Description |\n" +
		"|-------|-------------|\n" +
		"| [`" + skillName + "`](skills/" + skillName + "/SKILL.md) | TODO: one-sentence description. |\n" +
		"\n" +
		"## Requirements\n" +
		"\n" +
		"TODO: list runtime tools, accounts, or platform requirements. Remove this section if none.\n" +
		"\n" +
		"## License\n" +
		"\n" +
		"TODO: add a license.\n"
}
