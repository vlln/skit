# skit v1 Implementation Plan

> **Status**: Draft  
> **Date**: 2026-04-24  
> **Scope**: Go project layout, module boundaries, implementation order, testing strategy

This document explains how to turn `design.md` and `spec-v1.md` into code. The spec remains the source of truth for behavior; this plan is the source of truth for implementation shape.

---

## 1. Implementation Principles

- Keep the CLI thin. Commands parse flags, call internal services, and format output.
- Keep core packages deterministic. Parsing, hashing, lock writing, and path resolution must be testable without network access.
- Prefer explicit data flow over package-level global state.
- Return structured errors from internal packages; only the CLI decides how to print them.
- Shell out to `git` for v1 instead of implementing git protocol clients.
- Do not implement Agent target sync, system package installation, MVS, semver solving, or lock export in v1.

---

## 2. Technology Stack

### Language And Toolchain

- Development environment confirmed on 2026-04-24: Go 1.26.1, Git 2.43.0.
- `go.mod` should initially target Go 1.23 unless M1 discovers a concrete reason to require a newer version.
- Standard `go test ./...` test runner.
- `gofmt` and `go vet` as baseline checks.
- `git` must be discovered through `exec.LookPath("git")`; missing `git` is an `EnvironmentError`.

### Libraries

Use the standard library first. Add dependencies only where they remove real implementation risk.

Recommended initial dependencies:

- CLI: `github.com/spf13/cobra`
- YAML: `gopkg.in/yaml.v3`

Avoid in v1:

- Long-lived background daemons.
- Embedded scripting runtimes.
- Git libraries unless shelling out to `git` proves insufficient.
- Complex dependency injection frameworks.

---

## 3. Repository Layout

Recommended layout:

```text
skit/
├── cmd/
│   └── skit/
│       └── main.go
├── internal/
│   ├── app/
│   ├── cli/
│   ├── diagnose/
│   ├── hash/
│   ├── lockfile/
│   ├── metadata/
│   ├── source/
│   ├── store/
│   ├── skill/
│   └── xdg/
├── testdata/
│   ├── locks/
│   └── skills/
├── docs/
│   ├── design.md
│   ├── implementation-plan.md
│   └── spec-v1.md
├── go.mod
└── go.sum
```

Package intent:

- `cmd/skit`: binary entrypoint only.
- `internal/cli`: Cobra command tree, flag parsing, terminal output.
- `internal/app`: command orchestration use cases, such as Add, Install, List, Remove, Inspect, Doctor.
- `internal/skill`: `SKILL.md` discovery, frontmatter parsing, standard validation.
- `internal/metadata`: `metadata.skit`, `skill.yaml`, and ClawHub/OpenClaw compatibility mapping.
- `internal/source`: source string parsing, source identity, provider interfaces, local/GitHub resolution, and v1.x diagnostics for generic git/GitLab.
- `internal/store`: project/global store path resolution and content installation.
- `internal/lockfile`: deterministic lock read/write/import.
- `internal/hash`: canonical tree hash and file hash implementation.
- `internal/diagnose`: doctor checks for lock, metadata, bins, env, config, platform.
- `internal/xdg`: XDG config/data/state path helpers.

Do not create public Go packages until there is an external API requirement.

---

## 4. Module Boundaries

Dependency direction must stay acyclic:

```text
cmd/skit -> internal/cli -> internal/app -> core packages
```

Core package dependency rules:

- `skill` may depend on `metadata` only for shared frontmatter value types, but should not call source, store, hash, or lockfile.
- `metadata` must not depend on source, store, hash, lockfile, or app.
- `source` may depend on `skill` for Skill discovery types, but must not write stores or lock files.
- `store` may depend on `hash` for content-addressed paths, but must not depend on lockfile or source.
- `lockfile` must not depend on store or source provider implementations.
- `diagnose` may read from lockfile, skill, metadata, and hash, but must not mutate store or lock state.
- `app` is the only package that coordinates source, store, hash, metadata, skill, lockfile, and diagnose.

### CLI Layer

Allowed:

- Parse flags and args.
- Prompt only when the spec allows interaction.
- Convert structured errors into human-readable messages.
- Print JSON when `--json` is requested.

Not allowed:

- Compute hashes.
- Parse `SKILL.md` directly.
- Write lock files directly.
- Shell out to `git` directly.

### App Layer

The app layer coordinates packages but should contain little parsing logic.

Examples:

- `Add(ctx, AddRequest) (AddResult, error)`
- `Install(ctx, InstallRequest) (InstallResult, error)`
- `List(ctx, ListRequest) (ListResult, error)`
- `Remove(ctx, RemoveRequest) (RemoveResult, error)`
- `Inspect(ctx, InspectRequest) (InspectResult, error)`
- `Doctor(ctx, DoctorRequest) (DoctorResult, error)`

### Core Packages

Core packages should be independently testable:

- `skill.ParseDir(path)` returns parsed standard metadata and validation diagnostics.
- `metadata.Merge(frontmatter, manifest)` returns normalized skit metadata or conflict errors.
- `source.Parse(input)` returns an unresolved source object without network access.
- `source.Resolve(ctx, source)` may use network or `git`.
- `hash.Tree(path)` returns canonical hashes and rejected entry diagnostics.
- `lockfile.Write(path, lock)` writes stable JSON with sorted keys and a trailing newline.

---

## 5. Source Provider Design

The provider interface below is the target shape, not a requirement for the first parser implementation. M3 may start with concrete parser and resolver functions if that keeps the local and GitHub paths simpler.

```go
type Provider interface {
    Parse(input string) (Source, bool, error)
    List(ctx context.Context, src Source) ([]SkillRef, error)
    Resolve(ctx context.Context, ref SkillRef) (ResolvedSkill, error)
    Fetch(ctx context.Context, resolved ResolvedSkill, dst string) error
}
```

Provider order for source parsing:

1. Local path.
2. Explicit provider prefix, such as `github:`.
3. URL provider, such as GitHub/GitLab/generic git.
4. GitHub shorthand `owner/repo`.

v1 MVP providers:

- `local`: no network, direct directory validation.
- `github`: shorthand and GitHub tree URL handling.

v1.x providers:

- `git`: generic `git clone` / `git fetch` through `os/exec`.
- `gitlab`: GitLab tree URL handling.

Registry and well-known providers can be added after M6/v1.x. Keep interfaces open enough for them, but do not implement registry search/download, registry config, or source management commands as requirements for the local/GitHub v1 MVP.

---

## 6. Store And Lock Strategy

### Store

The store owns installed content. Lock files describe installed content.

Project paths:

- Store: `.skit/store/`
- Lock: `.skit/lock.json`

Global paths:

- Store: `$XDG_DATA_HOME/skit/store/`, fallback `~/.local/share/skit/store/`
- Lock: `$XDG_STATE_HOME/skit/lock.json`, fallback `~/.local/state/skit/lock.json`

Store entries are content-addressed in v1:

```text
<store>/<hashes.tree>/<skill-name>/
```

Rules:

- The tree hash directory is the canonical `hashes.tree` value from `spec-v1.md`, including the `sha256-` prefix.
- The final path component is the runtime Skill name from `SKILL.md`.
- The lock file must not record store paths.
- If two sources resolve to the same tree hash and Skill name, they may share the same store entry.
- If a store path already exists with different content, installation must fail with an `IntegrityError`.
- Store writes must be staged in a temporary directory under the same filesystem when possible, then moved into place atomically.
- Local path installs copy a snapshot into the store; the lock records the absolute local source path but restore uses the stored content and recorded hashes.
- A failed install must not leave a partial final store entry. Temporary directories may be cleaned best-effort.

### Lock

Lock writing rules live in `internal/lockfile` only.

The lock package must guarantee:

- No timestamps.
- No Agent targets.
- Stable key ordering.
- Trailing newline.
- Stable JSON indentation.
- Optional `source.skill`, `registry`, and `download` fields as defined in `spec-v1.md`.
- Dependency edges are serialized only after their dependency lock entries have been resolved.
- Entries imported without enough source/hash data must set `incomplete: true`; restore skips them with a clear diagnostic.
- Lossy imports report warnings.

### Metadata Carriers

`internal/metadata` must choose exactly one skit metadata carrier:

- Prefer `metadata.skit` in `SKILL.md`.
- If `metadata.skit` is absent, read `skill.yaml` when present.
- If both exist, return a `ValidationError`; do not merge them in v1.
- ClawHub/OpenClaw compatibility metadata is always read separately for diagnosis and inspect output, but it never conflicts with the chosen skit metadata carrier.

### Skill Marker Files

`internal/skill` must treat `SKILL.md` as canonical.

- If only `skill.md` exists, parse it and attach a warning.
- If both `SKILL.md` and `skill.md` exist, parse `SKILL.md` and attach a warning that `skill.md` was ignored.
- Frontmatter `name` is the runtime Skill identity.
- Directory/repository name mismatches must warn instead of failing, matching common ecosystem practice in `skills` and `clawhub`.
- New files created by `skit init` must use `SKILL.md`.

---

## 7. Error Model

Internal errors should carry a category:

- `UsageError`: invalid flags, invalid source syntax, missing required args.
- `ValidationError`: invalid `SKILL.md`, metadata conflict, unsupported platform.
- `NotFoundError`: missing files, missing Skill, missing lock entry.
- `IntegrityError`: hash mismatch, unsafe archive path, rejected symlink/non-regular file.
- `EnvironmentError`: missing binary, env var, config path, or `git`.
- `NetworkError`: remote fetch/resolve failure.
- `InternalError`: bug or unexpected invariant failure.

CLI exit codes:

- `0`: success.
- `1`: general failure.
- `2`: usage error.
- `3`: validation or integrity failure.
- `4`: environment failure.
- `5`: network failure.

These exit codes are implementation decisions for v1. They may change before the first tagged release, but must remain stable after v1.0.0.

---

## 8. Output And Result Formatting

Internal packages and the app layer return structured result types. They do not write to stdout or stderr.

CLI formatting rules:

- Human output is concise and stable enough for users, but not treated as a machine-readable API.
- `--json` output is the machine-readable form and should be backed by explicit response structs.
- JSON output must not include ANSI color or progress text.
- Errors are printed to stderr.
- Successful machine-readable results are printed to stdout.
- Prompts are allowed only for interactive commands and must be disabled by `--yes` or future `--no-input` style flags.
- `--global` and `--project` are mutually exclusive.
- `--all` and `--skill` are mutually exclusive.
- `--yes` skips confirmations only; it does not select every Skill.
- Non-interactive multi-Skill selection without `--all` or `--skill` is a usage error.

Initial JSON support:

- `list --json` returns lock-derived Skill entries.
- `inspect --json` returns source, metadata, hashes, requirements, and warnings.
- `doctor --json` returns checks grouped by severity.

---

## 9. Command-To-Package Map

| Command | App use case | Core packages |
| :--- | :--- | :--- |
| `skit add <source>` | `app.Add` | `source`, `skill`, `metadata`, `hash`, `store`, `lockfile` |
| `skit install` | `app.Install` | `lockfile`, `source`, `store`, `hash` |
| `skit list` | `app.List` | `lockfile` |
| `skit remove <name>` | `app.Remove` | `lockfile`, `store` |
| `skit update [name]` | `app.Update` | `lockfile`, `source`, `hash`, `store` |
| `skit init [name]` | `app.Init` | `skill` |
| `skit doctor` | `app.Doctor` | `diagnose`, `lockfile`, `skill`, `metadata`, `hash` |
| `skit inspect <source-or-name>` | `app.Inspect` | `source`, `skill`, `metadata`, `hash`, `lockfile` |
| `skit import-lock <kind>` | `app.ImportLock` | `lockfile`, `source` |

Commands from `design.md` that are not in v1 should remain hidden or absent until implemented.

---

## 10. Configuration And Environment

### XDG Paths

Path resolution belongs in `internal/xdg`.

Rules:

- Config: `$XDG_CONFIG_HOME/skit/config.yaml`, fallback `~/.config/skit/config.yaml`.
- Data/store: `$XDG_DATA_HOME/skit/store/`, fallback `~/.local/share/skit/store/`.
- State/global lock: `$XDG_STATE_HOME/skit/lock.json`, fallback `~/.local/state/skit/lock.json`.
- Project paths are always relative to the selected working directory.

### Environment

- `SKIT_CONFIG` may override the config file path in a later milestone; it is not required for M1-M5.
- `SKIT_ENABLE_NETWORK_TESTS=1` enables network integration tests.
- Standard proxy variables are inherited by `git` and HTTP clients.

### Platform Notes

- Use `filepath` for local filesystem paths.
- Use slash-normalized paths only for source subpaths, lock entries, and hashes.
- Windows support must not rely on symlink behavior in v1 because Skill symlinks are rejected.

---

## 11. Testing Strategy

### Unit Tests

- `skill`: frontmatter parsing and official validation rules.
- `skill`: lowercase `skill.md` compatibility warnings and `SKILL.md` precedence.
- `metadata`: `metadata.skit`, `skill.yaml`, mutual exclusion, ClawHub/OpenClaw compatibility mapping.
- `source`: table tests for every supported source syntax and ambiguous `@` cases.
- `hash`: golden tests with fixed fixture directories.
- `lockfile`: golden JSON tests for sorted output, dependency edges, incomplete imports, and import warnings.
- `xdg`: environment override tests.

### Integration Tests

Use temp directories and local fixtures.

- `skit add ./skill`.
- `skit list --json`.
- `skit remove <name>`.
- `skit install` from `.skit/lock.json`.
- `skit doctor` reports missing bins/env/config.
- `skit inspect ./skill`.

Network integration tests for GitHub/GitLab should be opt-in:

```text
SKIT_ENABLE_NETWORK_TESTS=1 go test ./...
```

### Testdata Rules

- Keep fixtures small and readable.
- Use explicit golden files for lock JSON.
- Include rejected fixtures: missing frontmatter, invalid names, symlink, path traversal archive entries.

---

## 12. Milestone Plan

### M1: CLI Skeleton

- Create Go module.
- Add Cobra command tree.
- Implement `skit --help` and `skit version`.
- Add baseline CI-equivalent local commands: `go test ./...`, `go vet ./...`.

### M2: Skill Parser

- Parse `SKILL.md` frontmatter.
- Validate official fields.
- Parse `metadata.skit`.
- Parse `skill.yaml`.
- Map ClawHub/OpenClaw compatibility metadata without executing install declarations.

### M3: Source Parser

- Implement deterministic source parser.
- Add local path normalization.
- Add GitHub/GitLab tree URL unresolved parsing.
- Reject unsafe subpaths.

### M4: Hash, Store, Lock

- Implement canonical tree hash.
- Implement project/global store path resolution.
- Implement lock read/write.
- Implement imports for `skills-lock.json` and `.clawhub/lock.json`.

### M5: Local Closed Loop

- Implement `add`, `list`, `remove`, `install` for local paths.
- Verify lock restore in temp project.
- Keep Agent target sync absent.

### M6: GitHub Closed Loop

- Implement git fetch/clone provider.
- Resolve branch/tag to commit.
- Support GitHub shorthand and tree URL subpaths.
- Hash fetched Skill directory and update lock.

### M7: Doctor And Inspect

- Implement metadata/source inspection.
- Implement lock/hash/environment diagnostics.
- Implement `list --json`, `inspect --json`, and `doctor --json`.
- Implement `skit init [name]` with non-overwriting `SKILL.md` generation.
- Add safety warnings for obvious shell execution patterns.

### M8: Update

- Implement `skit update [name]` for local and GitHub locked sources.
- Refresh hashes, store snapshots, resolved refs, warnings, and dependency edges.

### M9: Import Lock

- Implement conservative `import-lock skills` from `skills-lock.json`.
- Implement conservative `import-lock clawhub` from `.clawhub/lock.json` and legacy `.clawdhub/lock.json`.
- Mark imported entries incomplete unless skit can prove a reproducible source and hash.

---

## 13. Deferred Work

Do not implement these in v1 unless the spec changes:

- Agent target sync.
- `--agent` and `--copy`.
- Lock export to `skills` or `clawhub`.
- Registry publish.
- Registry search as a requirement for local/GitHub workflows.
- Registry download as a requirement for local/GitHub workflows.
- `skit source <add/list/remove>`.
- Semver dependency solving.
- MVS.
- Automatic system package installation.
- Executing Skill scripts during install.
- Symlink support inside Skill directories.

---

## 14. First Implementation Step

Start with M1 and M2:

1. Create `go.mod`.
2. Create `cmd/skit/main.go`.
3. Create `internal/cli` with `version` and root help.
4. Create `internal/skill` and `internal/metadata`.
5. Add parser tests before wiring `add`.

The first useful checkpoint is:

```text
go test ./...
go run ./cmd/skit --help
go run ./cmd/skit version
```
