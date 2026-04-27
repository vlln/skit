# skit

`skit` is a small, reproducible Skill manager for agent ecosystems.

It works with `SKILL.md` packages as defined by [agentskills.io](../agentskills.io/):
discover Skills, fetch them from local and git sources, store immutable
snapshots, write deterministic locks, activate Skills through symlinks, and
diagnose declared local requirements.

> Status: v0.1. The CLI and lock format are usable, but still allowed to change
> before a first stable release.

## Features

- Standards-oriented `SKILL.md` parsing with ecosystem metadata.
- Content-addressed global store under XDG data directories.
- Project activation via `.agents/skills` and global activation via `~/.agents/skills`.
- Deterministic `skit.lock` files stored next to active Skills.
- Local, GitHub, GitLab, SSH git, and generic git source parsing/fetching.
- Dependency locking for `metadata.skit.dependencies`.
- Requirement diagnostics for declared binaries, environment variables, config,
  platforms, and stored warnings.
- JSON output for automation-friendly commands.
- No Skill code execution during install, inspect, update, or restore.

## Installation

Recommended install path for users is a prebuilt binary from GitHub Releases.
Release artifacts are published for macOS, Linux, and Windows, with a checksum
file. macOS and Linux artifacts use `.tar.gz`; Windows artifacts use `.zip`.
Install macOS/Linux with:

```sh
curl -fsSL https://raw.githubusercontent.com/vlln/skit/main/install.sh | sh
```

The installer detects the platform, downloads the matching release asset,
verifies checksums, and places `skit` in `~/.local/bin` or `SKIT_INSTALL_DIR`.
Downloads fail instead of hanging indefinitely: by default the installer uses a
10s connect timeout, 300s total transfer timeout, 30s low-speed timeout, and 3
retries. These can be adjusted with `SKIT_CONNECT_TIMEOUT`, `SKIT_MAX_TIME`,
`SKIT_SPEED_LIMIT`, `SKIT_SPEED_TIME`, and `SKIT_RETRY`.

For restricted networks, point the installer at a release mirror or pre-staged
asset directory with `SKIT_DOWNLOAD_BASE`:

```sh
curl -fsSL https://raw.githubusercontent.com/vlln/skit/main/install.sh |
  SKIT_DOWNLOAD_BASE=https://example.com/skit/releases/<version> sh
```

Package-manager distribution can layer on top of the same release artifacts:

```sh
brew install vlln/tap/skit
```

Uninstall:

```sh
rm -f "${SKIT_INSTALL_DIR:-$HOME/.local/bin}/skit"
```

If installed with Homebrew:

```sh
brew uninstall skit
```

From a local checkout, for development:

```sh
go install ./cmd/skit
```

Or build a local binary:

```sh
go build -o skit ./cmd/skit
./skit version
```

Requirements:

- `git` for remote git sources
- Go 1.23+ only when building from source

## Quick Start

Create a Skill:

```sh
skit init my-skill
```

Install a local Skill into the current project:

```sh
skit install ./my-skill
```

Install a Skill from a GitHub repository:

```sh
skit install github:owner/repo --skill skill-name
```

Install more than one Skill from the same source:

```sh
skit install github:owner/repo --skill skill-one skill-two
```

Restore active symlinks from the lock:

```sh
skit install
```

Also activate a Skill for Codex:

```sh
skit install ./my-skill --agent codex
skit install --global ./my-skill --agent codex
```

Search for published Skills:

```sh
skit search "skill create"
```

Inspect and diagnose:

```sh
skit inspect skill-name
skit doctor
```

## Bundled Skills

Install bundled skills directly from this repository with `skit`:

```sh
skit install --global vlln/skit/skills/<skill-name>
```

| Skill | Description |
|-------|-------------|
| [`search-skills`](skills/search-skills) | Find, evaluate, inspect, and install agent skills with the `skit` CLI. |
| [`make-skill`](skills/make-skill) | Create or revise Agent Skills with precise frontmatter, concise instructions, validation checks, and skit-friendly metadata. |

## Commands

```text
skit install [source...]  Install sources, or restore from skit.lock
skit search <query>       Search for Skills
skit list                 List locked Skills
skit remove <name...>     Remove locked and active Skills
skit uninstall <name...>  Alias for remove
skit gc                   Prune unreferenced store snapshots
skit update [name]        Refresh locked Skills from their sources
skit inspect <target>     Inspect a locked Skill or source
skit doctor               Check lock, store, hashes, and requirements
skit init [name]          Create a SKILL.md template
skit import-lock <kind>   Import a compatible lock file
skit version              Print the CLI version
skit version --check      Check GitHub Releases for a newer skit version
```

Common flags:

```text
--project          Use project scope (default)
--global           Use global scope
--agent <names...> Also activate for specific agents, such as codex
--skill <names...> Select one or more Skills from a single source
--all              Install every discovered non-internal Skill from a source
--full-depth       Search recursively for Skills when installing a source
--ignore-deps      Skip declared Skill dependencies
--no-active        Write store/lock only; do not create active symlinks
--force            Replace an existing non-symlink active path
--prune            With remove, delete unreferenced store snapshots
--json             Print JSON for supported commands
```

`--agent` keeps the skit lock and content-addressed store as the source of
truth, then creates extra symlinks for selected agents. Supported agent names:
`codex`, `claude-code`, `cursor`, `gemini-cli`, and `opencode`. For Codex,
project installs target `.agents/skills` and global installs target
`${CODEX_HOME:-~/.codex}/skills`. The default skit active roots are already
handled by `--project` and `--global`, so there is no separate universal agent
target.

`--skill` may be provided once. It can contain multiple space-separated Skill
names for one source. For multiple sources, use inline selectors:

```sh
skit install owner/repo@skill-one other/repo@skill-two
```

`skit install` and `skit update` check for newer skit releases at most once per
day and print a short update hint when one is available. Set
`SKIT_UPDATE_CHECK=0` to disable the automatic check.

## Paths

Project scope:

```text
.agents/skills/<skill-name>  -> symlink to global store snapshot
.agents/skills/skit.lock     deterministic project lock
```

Global scope:

```text
~/.agents/skills/<skill-name> -> symlink to global store snapshot
~/.agents/skills/skit.lock    deterministic global lock
```

Store and temporary files:

```text
${XDG_DATA_HOME:-~/.local/share}/skit/store/<tree-hash>/<skill-name>/
${XDG_CACHE_HOME:-~/.cache}/skit/tmp/
```

The store is shared across project and global scopes. Active Skills are symlinks
to immutable store snapshots.

## Sources

Supported source forms include:

```text
./skill
/absolute/path
owner/repo
github:owner/repo
owner/repo/path/to/skill
https://github.com/owner/repo/tree/ref/path
gitlab:group/subgroup/repo
https://gitlab.com/group/repo/-/tree/ref/path
git@github.com:owner/repo.git
https://example.com/owner/repo.git
```

Selectors:

```text
source#ref
source#ref@skill-name
owner/repo@skill-name
```

Use `--skill <name>` when a source locator or git ref contains `@` and would be
ambiguous.

Git sources with an explicit subpath, such as `vlln/skit/skills/search-skills`,
use a sparse checkout when possible so install does not need the whole worktree.

## Discovery

Discovery is bounded and deterministic:

- A source root containing `SKILL.md` is one Skill.
- Direct children and common Skill roots are checked next, including
  `skills/`, `.agents/skills`, `.codex/skills`, `.claude/skills`,
  `.opencode/skills`, and `.windsurf/skills`.
- `--full-depth` enables depth-limited recursive discovery.
- `metadata.internal: true` Skills are skipped unless selected explicitly with
  `--skill`.

Lowercase `skill.md` is accepted for ecosystem interoperability and recorded as a warning.

## Metadata

`skit` reads standard `SKILL.md` frontmatter and `metadata.skit` extensions:

```yaml
---
name: pdf-tools
description: Extract, merge, compress, and inspect PDF files.
metadata:
  skit:
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

Ecosystem metadata such as `metadata.openclaw.requires` is preserved for
inspection and diagnostics. It is not executed.

## Safety

`skit install`, `skit inspect`, `skit update`, and `skit doctor` do not execute
Skill code.

The CLI rejects unsafe source subpaths, rejects non-regular files while copying
snapshots, normalizes executable modes, verifies store hashes, and records
warnings for suspicious content such as piping network downloads into a shell.

## Ecosystem Imports

Existing ecosystem lock files can be imported:

```sh
skit import-lock skills
skit import-lock clawhub
```

Lossy imports are marked `incomplete: true` because the source lock may not
contain enough information for reproducible restore. Reinstall the Skill with
`skit install <source>` to make it fully restorable.

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
