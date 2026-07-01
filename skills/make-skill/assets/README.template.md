<h1 align="center"><skill-title></h1>

<p align="center">
  <strong>One-line value proposition.</strong><br/>
  Two to three lines describing what the skills do, the target audience,
  and the key benefit.
</p>

<p align="center">
  <a href="https://github.com/<owner>/<repo>/stargazers"><img src="https://badgen.net/github/stars/<owner>/<repo>?label=%E2%98%85" alt="GitHub stars" /></a>
  <img src="https://badgen.net/badge/license/<license>/blue" alt="<license>" />
  <img src="https://badgen.net/badge/spec/Agent%20Skills/8257D0" alt="Agent Skills spec" />
</p>

<p align="center">
  <sub><a href="README.md">English</a> · <a href="docs/readme/README.zh-CN.md">中文</a></sub>
</p>

---

## Installation

### [skit](https://github.com/vlln/skit) (Recommended)

```bash
skit install ./<repo-name> --all
```

### [skills](https://github.com/bananaml/skills)

```bash
npx skills add git@github.com:<owner>/<repo>.git
```

### Manually

| Agent | Command |
|-------|---------|
| **Claude Code** | `cp -r skills/<skill-name> .claude/skills/` |
| **Codex** | `cp -r skills/<skill-name> ~/.codex/skills/` |
| **OpenCode** | `git clone https://github.com/<owner>/<repo>.git ~/.opencode/skills/<repo-name>` |
| **Kimi** | `cp -r skills/<skill-name> ~/.kimi/skills/` |

---

## Skills

| Skill | Description |
|-------|-------------|
| [<skill-name>](skills/<skill-name>/SKILL.md) | One-line description. |

## Requirements

<runtime, tool, account, or platform requirements. Remove this section if none.>

## License

<license>