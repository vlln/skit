---
name: search-skills
description: Find, evaluate, inspect, and install agent skills with the skit CLI. Use when the user asks to find a skill, search installable skills, add a capability, install a skill globally or for a project, inspect a skill source, restore skills from skit.lock, or diagnose installed skills.
metadata:
  skit:
    version: 0.1.0
    requires:
      bins:
        - skit
---

# Search Skills

Use this skill when the user wants an installable agent skill or asks whether a skill exists for a task.

## skit Availability

If `skit` is missing, install it from the skit repository:

```sh
curl -fsSLO https://raw.githubusercontent.com/vlln/skit/main/install.sh
less install.sh
```

After reviewing the installer, follow its instructions if it is acceptable.

Use release binaries or package manager installs when available. If working from a local checkout:

```sh
go install ./cmd/skit
```

Ensure the install directory, usually `~/.local/bin`, is on `PATH`.

## Workflow

1. Clarify the task only if the query is too vague to search.
2. Search with `skit search <query>`.
3. Prefer results with clear names, reputable sources, and higher install counts.
4. Inspect likely candidates before recommending or installing when practical:

```sh
skit inspect <source@skill>
```

5. Present the best option with its source and install command.
6. Install only when the user agrees, or when they directly asked you to install it.

Project install:

```sh
skit install <source@skill>
```

Global install:

```sh
skit install --global <source@skill>
```

After installing, verify:

```sh
skit list
skit doctor
```

For global installs:

```sh
skit list --global
skit doctor --global
```

## Selection Guidance

- Prefer exact source selectors from search output, such as `owner/repo@skill-name`.
- Use `--skill <name>` when the source or ref contains `@` and the selector would be ambiguous.
- Use `--all` only when the user explicitly wants every skill from a source.
- Use `--full-depth` when a source is known to contain nested skills that normal discovery misses.
- Use `--no-active` when the user wants to update the store and lock without creating active symlinks.
- Avoid `--force` unless the user agrees to replace an existing non-symlink active path.

## Quality Checks

Before recommending a skill, consider:

- Does the name and description match the user request?
- Is the source recognizable or inspectable?
- Does `skit inspect` show suspicious warnings?
- Does the skill declare missing local requirements?

If no good result exists, say that no suitable skill was found and offer to help directly or create a new skill.

## Common Commands

```sh
skit search "react testing"
skit inspect vercel-labs/skills@find-skills
skit install --global vercel-labs/skills@find-skills
skit install
skit update --global search-skills
skit remove --global search-skills
```
