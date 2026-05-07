# <repo-name>

Agent Skills for <short-purpose>.

This repository stores skills under `skills/`. Each skill follows the
[Agent Skills specification](https://agentskills.io/specification) and can be
used by skills-compatible agents.

## Skills

| Skill | Description |
|-------|-------------|
| [`<skill-name>`](skills/<skill-name>) | <one-sentence description> |

## Quick Start

Paste this into your AI agent (Claude Code, Cursor, OpenAI Assistants, etc.):

```text
Install the Agent Skills from https://raw.githubusercontent.com/<owner>/<repo>/main/README.md
```

## Installation

Recommended: install these skills with
`skit`. It fetches skills from the published repository, records them in a local
manifest, and activates them for local agents.

### skit

Install `skit` with Homebrew:

```sh
brew install vlln/tap/skit
```

For other platforms, see the `skit` installation instructions.

Install one skill:

```sh
skit install <owner>/<repo>/skills/<skill-name>
```

Install all skills in this repository:

```sh
skit install <owner>/<repo> --all
```

### npx skills

```sh
npx skills add <owner>/<repo>
```

### Manual

Copy the desired skill directory from `skills/<skill-name>` into your agent's
skills directory, then restart the agent if required.

Common locations:

- Codex CLI: `~/.codex/skills`
- Claude Code: `.claude/skills` in the project, or the configured user skills directory
- OpenCode: `~/.opencode/skills/<repo-name>`

## Requirements

<runtime, tool, account, or platform requirements. Remove this section if none.>

## License

<license>
