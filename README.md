# skit

`skit` is a fast, reproducible Skill management CLI for agent ecosystems.

It is designed as a small Go-style toolchain for [agentskills.io](../agentskills.io/)
`SKILL.md` packages: parse Skills, fetch them from common sources, store immutable
snapshots, write deterministic lock files, and diagnose local requirements.

> Status: early v1 implementation. `skit` stores and locks Skills; it does not
> activate them into Claude Code, Codex, OpenCode, or other agent target
> directories yet.

## Why skit

- **Compatible first**: reads standard `SKILL.md` frontmatter and accepts common
  source syntax from the current Skill ecosystem.
- **Reproducible by default**: stores content-addressed snapshots and records
  source identity, resolved refs, and content hashes in `.skit/lock.json`.
- **No code execution on install**: install/fetch operations copy and hash files;
  Skill scripts are never run automatically.
- **Unix-shaped CLI**: explicit commands, deterministic JSON output where useful,
  and project/global scopes.
- **Go native**: single binary, fast startup, simple cross-platform distribution.

## Installation

From this repository:

```sh
go install ./cmd/skit
```

Or build a local binary:

```sh
go build -o skit ./cmd/skit
./skit version
```

Requirements:

- Go 1.23+
- `git` for GitHub, GitLab, SSH git, and generic `.git` sources

## Quick Start

Create a new Skill:

```sh
skit init my-skill
```

Add a local Skill to the project store and lock:

```sh
skit add ./my-skill
```

Add a Skill from a GitHub repository:

```sh
skit add github:owner/repo --skill skill-name
```

Restore locked Skills into the content-addressed store:

```sh
skit install
```

Inspect a locked Skill or a source:

```sh
skit inspect skill-name
skit inspect github:owner/repo --skill skill-name --json
```

Check store integrity and declared requirements:

```sh
skit doctor
```

Search for Skills:

```sh
skit search "skill create"
```

## Commands

```text
skit add <source>         Add a Skill source to the skit store and lock
skit search <query>       Search for Skills
skit install              Restore Skills from the lock file
skit list                 List locked Skills
skit remove <name>        Remove a Skill from the lock file
skit update [name]        Refresh locked Skills from their sources
skit inspect <target>     Inspect a locked Skill or source
skit doctor               Check lock, store, hashes, and declared requirements
skit init [name]          Create a SKILL.md template
skit import-lock <kind>   Import a compatible lock file
skit version              Print the CLI version
```

Common flags:

```text
--project       Use project scope (default)
--global        Use global scope
--skill <name>  Select one Skill from a multi-Skill source
--all           Add every discovered non-internal Skill from a source
--full-depth    Search recursively for Skills when adding a source
--ignore-deps   Skip declared Skill dependencies
--json          Print JSON for supported commands
```

## Supported Sources

`skit` currently fetches:

- Local paths: `./skill`, `/absolute/path`
- GitHub shorthand: `owner/repo`, `github:owner/repo`
- GitHub subpaths: `owner/repo/path/to/skill`
- GitHub tree URLs: `https://github.com/owner/repo/tree/ref/path`
- GitLab shorthand: `gitlab:group/subgroup/repo`
- GitLab tree URLs: `https://gitlab.com/group/repo/-/tree/ref/path`
- SSH git URLs: `git@github.com:owner/repo.git`
- Generic git URLs ending in `.git`

Selectors:

```text
source#ref
source#ref@skill-name
owner/repo@skill-name
```

For non-ambiguous automation, prefer `--skill <name>` over inline `@skill`
syntax.

Recognized but not fetched yet:

- `registry:<slug>`
- `clawhub:<slug>`
- non-git HTTP(S) URLs as `well-known` sources

These are parsed as future provider types so diagnostics are explicit instead
of accidentally attempting to clone ordinary websites as git repositories.

## Skill Discovery

Discovery is bounded and deterministic:

- A source root containing `SKILL.md` is treated as one Skill.
- Otherwise, `skit` checks direct children and common Skill roots such as
  `skills/`, `.agents/skills`, `.codex/skills`, `.claude/skills`,
  `.opencode/skills`, and `.windsurf/skills`.
- If nothing is found, `skit` falls back to a depth-limited recursive search.
- `--full-depth` forces depth-limited recursive discovery.
- `metadata.internal: true` Skills are skipped by default and can be installed
  explicitly with `--skill <name>`.

`SKILL.md` is canonical. Lowercase `skill.md` is accepted for ecosystem
compatibility and produces a warning.

## Lock and Store

Project scope uses:

```text
.skit/lock.json
.skit/store/
.skit/tmp/
```

Global scope uses XDG-style state/data directories.

Lock entries record:

- runtime Skill name and description
- source type, locator, URL, ref, resolved ref, subpath, and selected Skill
- content hashes for the full Skill tree and `SKILL.md`
- resolved dependency edges
- warnings relevant to the installed snapshot

Store paths are content-addressed:

```text
.skit/store/<tree-hash>/<skill-name>/
```

`skit install` verifies existing store entries before reporting them restored.
If store content differs from the lock, install fails with a hash mismatch.

## Metadata

`skit` reads standard `SKILL.md` frontmatter:

```yaml
---
name: pdf-tools
description: Extract, merge, compress, and inspect PDF files.
metadata:
  skit:
    version: 1.2.0
    dependencies:
      - source: github:example/pdf-core
        ref: v1.2.0
        skill: pdf-core
    requires:
      bins:
        - pdftotext
      env:
        - PDF_API_KEY
---
```

It also understands the future `skill.yaml` carrier and compatibility metadata
from current ClawHub/OpenClaw-style frontmatter. Compatibility install
declarations are preserved for inspection and diagnosis; they are not executed.

## Security Model

`skit add` and `skit install` do not execute Skill code.

The CLI rejects unsafe source subpaths, rejects non-regular files during store
copying, normalizes executable file modes, records immutable content hashes, and
warns about suspicious script patterns such as piping network downloads into a
shell.

System requirements declared by Skills are checked by:

```sh
skit doctor
```

`doctor` reports missing binaries, environment variables, config files, platform
mismatches, store hash mismatches, and stored warnings.

## Compatibility Imports

Existing ecosystem locks can be imported:

```sh
skit import-lock skills
skit import-lock clawhub
```

Lossy imports are marked `incomplete: true` because the original lock formats do
not contain enough source and hash information for fully reproducible restore.
Re-add the Skill with `skit add` to make it restorable.

## Roadmap

Near-term:

- registry and well-known provider fetching
- richer search providers beyond the current skills.sh-compatible API
- `skit tidy` for unused lock/store/tmp entries
- richer source and requirement diagnostics
- optional agent activation/sync as a separate command, without changing the
  lock file into agent state

Deferred:

- semver dependency solving and MVS-style selection
- automatic system package installation
- registry publishing and search

## Development

Run tests:

```sh
go test ./...
```

Run the CLI without installing:

```sh
go run ./cmd/skit --help
```

## License

MIT.
