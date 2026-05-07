# skit v1.0.0 Design

`skit` is an agent-first local Skills manager. It installs `SKILL.md`
directories into one user data directory, records them in one manifest, and
activates that shared inventory through symlinks that agents can read.

## Non-Goals

- No content-addressed store.
- No tree hashes or integrity drift checks.
- No reproducible lockfile semantics.
- No garbage collection command.
- No dependency resolution.
- No source-specific directory layout.

Skills are lightweight instructions. The practical agent questions are: what is
installed, where did it come from, how can I activate it, and how do I update,
export, or remove it without prompting a human?

Default text output is concise rather than chatty. It should be readable by an
AI agent, but it should avoid repeated explanations and long next-step blocks.
Detailed data belongs in `--json`, `help`, `check`, or follow-up commands.

## Paths

```text
${XDG_DATA_HOME:-~/.local/share}/skit/
├── manifest.json
└── skills/
    └── <name>/
        └── SKILL.md
```

Default active root:

```text
~/.agents/skills/<name> -> ~/.local/share/skit/skills/<name>
```

Agent-specific roots are added with `--agent`, for example Codex uses
`${CODEX_HOME:-~/.codex}/skills`.

## Manifest

`manifest.json` is the local installation database. `skit export` writes the
same format to `skit.json`, which is the repository-friendly sharing file.
Sharing that file lets another agent install the same Skill setup.

```json
{
  "schema": "skit.manifest/v1",
  "skills": {
    "review-helper": {
      "name": "review-helper",
      "description": "Review pull requests.",
      "source": {
        "type": "github",
        "locator": "example/skills",
        "url": "https://github.com/example/skills.git",
        "ref": "main",
        "subpath": "skills/review-helper",
        "skill": "upstream-review-helper"
      },
      "path": "skills/review-helper",
      "agents": ["universal", "codex"]
    }
  }
}
```

The manifest does not record hashes. If a Skill changes upstream, `skit update`
re-installs it from the recorded source.

## Commands

```text
skit search [query]
skit install <source> [--skill <name>] [--name <folder-name>] [--agent <agent>]
skit install [--dry-run]          # applies ./skit.json when present
skit export [path]
skit list [--all] [--json]
skit update [name]
skit remove <name> [--agent <agent>] [--keep]
skit check [--json]
skit init <name>
```

`doctor` is an alias for `check` during the transition.

## Install

Install discovers one or more Skills from a local or git source, copies each
selected Skill into `~/.local/share/skit/skills/<name>`, writes the manifest,
and creates active symlinks.

With no source, `install` applies `./skit.json` when present. `--dry-run`
previews manifest installation without changing local state.

`--name` is only valid when exactly one Skill is selected. It controls the local
folder name, manifest key, and active link name. The upstream `SKILL.md` name is
kept as source metadata only.

## List

`list` reads the manifest and checks whether expected active links exist. Text
output is `name<TAB>state<TAB>description`. It is not a lockfile view and does
not show unmanaged agent Skills by default.

`list --all` scans all supported agent skill directories, including project
directories and global/user directories, and merges those externally visible
Skills with manifest entries. This is an environment view, not the shareable
configuration view.

## Export

`export` writes the local manifest to a shareable file. The default path is
`./skit.json`; `bundle` is an alias. This is intended for agent and repository
automation, not as a second source of truth.

## Update

`update` re-installs from recorded sources and repairs active links. No hash
comparison is performed.

## Remove

Default removal unlinks active roots, deletes the local Skill directory, and
removes the manifest entry.

`--agent` only removes that agent's active link and updates the manifest's agent
list. `--keep` removes manifest/link state while leaving the local Skill
directory on disk.

## Check

`check` is intentionally thin. It verifies:

- manifest parses;
- installed Skill directory exists;
- `SKILL.md` exists;
- active links exist;
- active links point to the expected local Skill directory.
