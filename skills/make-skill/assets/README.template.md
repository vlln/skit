# <repo-name>

Agent Skills for <short-purpose>.

This repository stores skills under `skills/`. Each skill follows the
[Agent Skills specification](https://agentskills.io/specification) and can be
used by skills-compatible agents.

## Skills

| Skill | Description |
|-------|-------------|
| [`<skill-name>`](skills/<skill-name>) | <one-sentence description> |

## Installation

### skit

Install one skill:

```sh
skit install <owner>/<repo>/skills/<skill-name>
```

Install globally:

```sh
skit install --global <owner>/<repo>/skills/<skill-name>
```

Install all discoverable skills:

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

## Repository Layout

```text
skills/
  <skill-name>/
    SKILL.md
    assets/
    references/
    scripts/
```

Only `SKILL.md` is required. Add `assets/`, `references/`, or `scripts/` when
they keep the skill concise or provide reusable support files.

## Authoring Notes

- Keep each `SKILL.md` focused on one recurring task.
- Put large reference material in `references/`.
- Put reusable templates or static files in `assets/`.
- Put deterministic helper commands in `scripts/`.
- Validate skills before publishing:

```sh
skit inspect ./skills/<skill-name>
```

## License

<license>
