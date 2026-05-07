---
name: search-skills
description: Find, evaluate, and install agent skills with the skit CLI. Use when the user asks to find a skill, search installable skills, add a capability, install a skill, list installed skills, update skills, remove skills, or check active links.
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

If `skit` is missing, prefer the project installer, release binaries, or a
package manager install. Do not require the user to have a Go toolchain.

```sh
curl -fsSL https://raw.githubusercontent.com/vlln/skit/main/install.sh | sh
```

## Workflow

1. Clarify the task only if the query is too vague to search.
2. Search with `skit search <query>`.
3. Prefer results with clear names, reputable sources, and higher install counts.
4. Present the best option with its source and install command.
5. Install only when the user agrees, or when they directly asked you to install it.

Install:

```sh
skit install <source@skill>
```

Install under a chosen local folder name:

```sh
skit install <source@skill> --name <local-name>
```

After installing, verify:

```sh
skit list
skit check
```

## Selection Guidance

- Prefer exact source selectors from search output, such as `owner/repo@skill-name`.
- Use `--skill <name>` when the source or ref contains `@` and the selector would be ambiguous.
- Use `--all` only when the user explicitly wants every skill from a source.
- Use `--full-depth` when a source is known to contain nested skills that normal discovery misses.
- Avoid `--force` unless the user agrees to replace an existing non-symlink active path.

## Quality Checks

Before recommending a skill, consider:

- Does the name and description match the user request?
- Is the source recognizable?
- Does the install command identify a specific source and skill?

If no good result exists, say that no suitable skill was found and offer to help directly or create a new skill.

## Common Commands

```sh
skit search "react testing"
skit install vercel-labs/skills@find-skills
skit list
skit check
skit update search-skills
skit remove search-skills
```
