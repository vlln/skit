# skit v1 Implementation Plan

> **Status**: Active  
> **Date**: 2026-04-25  
> **Scope**: Current v1 implementation shape, package boundaries, tests, and next work

This plan tracks the implementation that exists today. `spec-v1.md` remains the
behavior source of truth; this document explains how the codebase should stay
organized while implementing it.

---

## 1. Principles

- Keep the CLI thin: parse flags, call `internal/app`, format output.
- Keep core packages deterministic and independently testable.
- Prefer explicit data flow over package-level mutable state.
- Do not execute Skill code during install, inspect, update, restore, or doctor.
- Shell out to `git` for remote source resolution.
- Treat `skit install [source...]` as the install command.

---

## 2. Repository Layout

```text
cmd/skit/              binary entrypoint
internal/cli/          command parsing and output
internal/app/          use-case orchestration
internal/diagnose/     doctor checks and safety warnings
internal/gitfetch/     git clone/fetch helpers
internal/hash/         canonical Skill tree hashing
internal/lockfile/     deterministic skit.lock read/write/import
internal/metadata/     metadata.skit, skill.yaml, and ecosystem metadata
internal/search/       skills.sh-compatible search client
internal/skill/        SKILL.md discovery and parsing
internal/source/       source syntax parsing
internal/store/        global store writes
internal/xdg/          XDG path helpers
```

Dependency direction:

```text
cmd/skit -> internal/cli -> internal/app -> core packages
```

`internal/app` is the only layer that coordinates source parsing, discovery,
store writes, lock updates, activation, and diagnostics.

---

## 3. Commands

Implemented public commands:

| Command | App use case | Notes |
| :--- | :--- | :--- |
| `skit install [source...]` | `app.Add`, `app.Install` | Source install, or lock restore with no source |
| `skit search <query>` | `app.Search` | skills.sh-compatible API |
| `skit list` / `skit ls` | `app.List` | Lock-derived entries |
| `skit remove <name>` / `skit rm` | `app.Remove` | Removes lock entry and active symlink |
| `skit update [name]` | `app.Update` | Refreshes locked sources |
| `skit inspect <target>` | `app.Inspect` | Locked Skill or source |
| `skit doctor` | `app.Doctor` | Lock, store, requirements, and warnings |
| `skit init [name]` | `app.Init` | Creates a `SKILL.md` template |
| `skit import-lock <kind>` | `app.ImportLock` | Imports supported ecosystem lock files |

Flag rules:

- `--project` is the default scope.
- `--global` targets `~/.agent/skills`.
- `--skill <names...>` may appear once and only with one source.
- Multiple sources use inline selectors such as `owner/repo@skill-name`.
- `--skill` and `--all` are mutually exclusive.
- `--no-active` writes store and lock only.
- `--force` may replace an existing non-symlink active path.

---

## 4. Store, Lock, And Active Paths

The store is global and content-addressed:

```text
${XDG_DATA_HOME:-~/.local/share}/skit/store/<tree-hash>/<skill-name>/
```

Temporary writes use:

```text
${XDG_CACHE_HOME:-~/.cache}/skit/tmp/
```

Project scope:

```text
.agent/skills/<skill-name>  -> symlink to global store snapshot
.agent/skills/skit.lock
```

Global scope:

```text
~/.agent/skills/<skill-name> -> symlink to global store snapshot
~/.agent/skills/skit.lock
```

Rules:

- Store paths are immutable snapshots keyed by tree hash and Skill name.
- Lock files do not record store paths.
- Active entries are symlinks to store snapshots.
- `skit install` with no sources verifies store entries and recreates active
  symlinks from `skit.lock`.
- `skit remove` removes the lock entry and active symlink; store pruning is a
  separate future command.

---

## 5. Source And Discovery

Implemented source forms:

- local paths
- GitHub shorthand and GitHub tree URLs
- GitLab shorthand and GitLab tree URLs
- SSH git URLs
- generic `.git` URLs
- inline selectors: `owner/repo@skill-name`, `source#ref@skill-name`

Git sources with an explicit subpath use sparse checkout with blob filtering
when possible. Sources without an explicit subpath keep the regular shallow
clone behavior because discovery must inspect the repository layout.

Discovery order:

1. Source root containing `SKILL.md` or `skill.md`.
2. Direct children.
3. Known Skill roots such as `skills/`, `.agents/skills`, `.codex/skills`,
   `.claude/skills`, `.opencode/skills`, and `.windsurf/skills`.
4. Depth-limited recursive fallback, or forced recursion with `--full-depth`.

`SKILL.md` is canonical. Lowercase `skill.md` is accepted with a warning for
ecosystem interoperability.

---

## 6. Metadata

`internal/metadata` chooses exactly one skit metadata carrier:

- Prefer `metadata.skit` in `SKILL.md`.
- Read `skill.yaml` only when `metadata.skit` is absent.
- If both exist, fail with a validation error.

Ecosystem metadata such as `metadata.openclaw.requires` is read for inspection
and diagnostics. Install declarations from ecosystem metadata are never
executed.

---

## 7. Testing

Required local checks:

```sh
go test ./...
```

Important coverage areas:

- CLI command/flag behavior, including repeated `--skill` rejection.
- Store, lock, active symlink install/restore/remove loops.
- Source parsing, including ambiguous `@` and `#ref@skill` inputs.
- Skill discovery and lowercase marker warnings.
- Metadata carrier conflicts and ecosystem metadata mapping.
- Hash stability and rejection of symlinks/non-regular files.
- Lock JSON stability and import behavior.
- Doctor checks for missing bins/env/config and stored warnings.

Network integration tests should remain opt-in.

---

## 8. Next Work

- Add `skit tidy` or `skit store prune` for unreferenced store snapshots.
- Document that `skills.sh` is a search/leaderboard service, not a publish
  target. GitHub releases do not automatically register Skills there; public
  listing currently appears tied to the `skills` CLI ecosystem and its
  aggregated install telemetry.
- Add a `SearchProvider` layer before adding more install providers. Initial
  candidates:
  - `skills.sh`: search-only, returns GitHub-backed install hints.
  - `clawhub`: registry-backed search with richer metadata when configured.
- Add a conservative search aggregation and deduplication policy:
  - exact source identity: provider, canonical source, subpath, and Skill name;
  - content identity: tree hash, registry digest, or package digest when known;
  - display grouping: same Skill name or same repository groups results but
    does not collapse distinct install choices.
- Keep `skills.sh` and ClawHub semantics separate. `skills.sh` search results
  should continue to install through Git/GitHub source resolution; ClawHub or
  future registries may support version, digest, download, moderation, and
  security status.
- Add registry/well-known provider support after search aggregation is stable.
- Add release packaging after the CLI surface settles.
- Consider agent-specific symlink helpers only if `.agent/skills` is not enough
  for real workflows.
