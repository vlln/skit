# skit v1 Specification

> **Status**: Draft
> **Date**: 2026-04-24
> **Scope**: Skill metadata, source resolution, store, lock, compatibility imports, CLI behavior

This document turns `design.md` into implementation-oriented contracts. v1 keeps skit focused on Skill management only: parse, fetch, validate, store, lock, diagnose.

---

## 1. Goals / Non-goals

### Goals

- Parse and validate standard `SKILL.md`.
- Read skit metadata from `metadata.skit`.
- Keep the metadata protocol compatible with a future `skill.yaml` standard.
- Resolve local and GitHub sources into stable content identities.
- Resolve GitHub, GitLab, and generic git sources into stable content identities.
- Store fetched Skills in a skit-managed store.
- Write reproducible project/global lock files.
- Diagnose metadata, source, lock, hash, and environment issues.
- Import existing `skills` and `clawhub` lock data where practical.
- Preserve compatibility with current `skills` and ClawHub frontmatter conventions without adopting their full runtime installation models.

### Non-goals

- Manage Agent target directories.
- Record Agent names, target paths, copy/symlink mode, or sync state in lock files.
- Record install timestamps in lock files.
- Automatically install system packages.
- Execute Skill scripts during install.
- Implement MVS or semver dependency solving.
- Export lock files back into `skills` or `clawhub` formats.
- Require `metadata.skit` or `skill.yaml` for existing Skills.
- Activate Skills into Agent target directories. v1 records and stores Skills only; it does not make Codex, Claude Code, or other agents discover them automatically.

---

## 2. Skill Format

A Skill is a directory containing `SKILL.md`.

`SKILL.md` must contain YAML frontmatter followed by Markdown instructions.

Compatibility rule:

- `SKILL.md` is the canonical filename.
- `skill.md` is accepted for ecosystem compatibility, but skit must emit a warning.
- If both `SKILL.md` and `skill.md` exist in the same Skill directory, skit uses `SKILL.md` and warns that `skill.md` was ignored.

Required fields:

- `name`
- `description`

Optional standard fields:

- `license`
- `compatibility`
- `metadata`
- `allowed-tools`

The runtime Skill identity is `SKILL.md` frontmatter `name`.

Rules:

- `name` must be 1-64 characters.
- `name` must be lowercase letters, numbers, and hyphens.
- `name` must not start or end with a hyphen.
- `name` must not contain consecutive hyphens.
- `name` is the runtime Skill identity.
- If `name` differs from the Skill root directory basename after source resolution, skit accepts the Skill and emits a warning. This matches common ecosystem practice in `skills` and `clawhub`, where metadata identity and source path are not strictly coupled.
- `description` must be 1-1024 characters.
- `compatibility`, when present, must be 1-500 characters.
- `metadata`, when present, must be a YAML mapping.
- `allowed-tools`, when present, must be a string.

---

## 3. skit Metadata Protocol

The canonical v1 extension location is `metadata.skit` inside `SKILL.md`.

Example:

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
        - qpdf
      env:
        - PDF_API_KEY
    platforms:
      os:
        - linux
        - darwin
    keywords:
      - pdf
      - document
---
```

### Fields

`version`: optional string. v1 treats this as inspect-only display/publish metadata. It is not shown by default in `list` and has no dependency resolution semantics. Reproducibility is based on source ref, resolved ref, and content hash.

`dependencies`: optional array.

Each dependency:

- `source`: required string.
- `ref`: optional string.
- `skill`: optional runtime Skill name.
- `optional`: optional boolean, default false.

Dependency rules:

- `source` plus `skill` identifies the dependency edge before resolution.
- `ref`, when present, is the requested ref for that dependency source.
- Required dependencies are installed before the parent lock entry is finalized.
- Optional dependencies are attempted; failures become warnings and do not block the parent install.
- Circular dependencies are validation failures.
- Duplicate dependencies with the same `source`, `ref`, and `skill` are deduplicated.
- Duplicate dependencies with the same `source` and `skill` but different `ref` values are validation failures in v1. Semver constraint merging and MVS are deferred.

`requires`: optional object.

Fields:

- `bins`: string array. All binaries should exist.
- `anyBins`: string array. At least one binary should exist.
- `env`: string array. Environment variables expected by the Skill.
- `config`: string array. Config file paths expected by the Skill.

`platforms`: optional object.

Fields:

- `os`: string array. Values: `linux`, `darwin`, `windows`.
- `arch`: string array. Values: `amd64`, `arm64`.

`keywords`: optional string array.

`registry`: optional object.

Fields:

- `slug`: optional string.
- `homepage`: optional string.

---

## 4. Ecosystem Compatibility Metadata

Existing ecosystem tools use `SKILL.md` frontmatter differently:

- `skills` reads the standard frontmatter and exposes `metadata` as an arbitrary mapping. It uses `metadata.internal: true` as a discovery/install filter, but it does not define a package-management schema under `metadata.skills`.
- ClawHub stores registry/runtime declarations as incremental metadata, not as a copy of the full YAML frontmatter. Its preferred namespace is `metadata.clawdbot`; `metadata.clawdis` and `metadata.openclaw` are accepted aliases. Some legacy skills also use top-level `clawdis` or top-level runtime fields such as `requires`, `primaryEnv`, `homepage`, `env`, `dependencies`, `author`, and `links`.
- `metadata.clawhub` is not a current compatibility namespace. v1 must not special-case it unless a future ClawHub spec introduces it.

skit v1 reads compatibility metadata only for diagnosis, inspection, and conservative install planning. It must not execute install declarations from compatibility metadata.

### ClawHub / OpenClaw Mapping

Source priority:

1. `metadata.clawdbot`
2. `metadata.clawdis`
3. `metadata.openclaw`
4. top-level `clawdis`
5. top-level fallback fields listed below

If multiple compatibility blocks are present, the first non-empty block by priority wins. skit may warn about lower-priority blocks being ignored.

Normalized fields:

- `requires.bins`, `requires.anyBins`, `requires.env`, `requires.config` map to skit `requires`.
- `primaryEnv` is preserved for `inspect` output and doctor messaging. It may also be treated as an expected env var if not already listed under `requires.env`, but this should be reported as compatibility-derived.
- `os` maps to skit `platforms.os`. Values are normalized as: `macos` -> `darwin`, `linux` -> `linux`, `windows`/`win32` -> `windows`. Unknown values are preserved in compatibility metadata and reported as warnings, not hard validation failures.
- `homepage` maps to `registry.homepage` only when `metadata.skit.registry.homepage` is absent.
- `install`, `nix`, `config`, `cliHelp`, `always`, `skillKey`, `emoji`, `envVars`, `dependencies`, `author`, and `links` are preserved as compatibility metadata for `inspect` and safety review. They do not create skit Skill dependencies in v1.
- Top-level `requires`, `primaryEnv`, and `homepage` are accepted as fallback compatibility fields when no `metadata.clawdbot` / `metadata.clawdis` / `metadata.openclaw` / top-level `clawdis` block exists.
- Top-level `env`, `dependencies`, `author`, and `links` are preserved as compatibility metadata only. They do not map to skit `requires` or `dependencies` unless a future spec explicitly defines that conversion.

Conflict behavior:

- Explicit `metadata.skit` fields win over compatibility-derived fields.
- Compatibility metadata must not conflict-check with `metadata.skit`; lower-priority compatibility values are recorded as provenance/warnings rather than fatal errors.
- `skill.yaml` conflicts only with `metadata.skit`, not with compatibility metadata.

---

## 5. skill.yaml Future Standard

`skill.yaml` is the desired future standard manifest name.

v1 must parse it if present, but must not generate it by default. The protocol must be identical to `metadata.skit`.

Rules:

- `schema`, when present, must be `skit.skill/v1`. It is a manifest format marker and is not part of `metadata.skit`.
- `skill.yaml` must not redefine `name` or `description`; those remain in `SKILL.md`.
- `skill.yaml` fields are the same fields under `metadata.skit`.
- `license` belongs to standard `SKILL.md` frontmatter in v1. `skill.yaml` must not define it.
- `metadata.skit` and `skill.yaml` are mutually exclusive v1 carriers. If both exist, skit fails with a clear conflict error and does not merge them.
- `metadata.skit` is the practical v1 carrier because it preserves compatibility with current Skill clients.
- `skit init` and migration commands generate `metadata.skit` by default in v1.

---

## 6. Source Identity And Resolution

### Recognized Inputs

```text
owner/repo
owner/repo@skill-name
github:owner/repo
github:owner/repo@skill-name
https://github.com/owner/repo
https://github.com/owner/repo/tree/ref/path/to/skill
https://gitlab.com/group/repo
https://gitlab.com/group/repo/-/tree/ref/path/to/skill
git@github.com:owner/repo.git
./local-skill
/absolute/local-skill
source#ref
source#ref@skill-name
gitlab:group/subgroup/repo
registry:slug
clawhub:slug
```

v1 closed-loop install support is required for `local` and `github` sources. Generic `git` and `gitlab` forms may be parsed in v1 to produce clear "provider not implemented yet" errors, but their fetch/restore closed loop is v1.x work.

`owner/repo@skill-name`, `github:owner/repo@skill-name`, and `source#ref@skill-name` are compatibility shortcuts. New scripts should prefer `--skill <name>` because git refs, SSH locators, URLs, and local paths may legally contain `@`.

### Parsing Rules

Parsing must be deterministic before network resolution.

General rules:

- `#` separates a source locator from a ref selector. Use the first `#`; additional `#` characters are invalid.
- `@skill-name` inside the ref selector is recognized only as a compatibility shortcut for interactive CLI input.
- `owner/repo@skill-name` and `github:owner/repo@skill-name` are recognized as compatibility shortcuts when the suffix after the last `@` matches the standard Skill name pattern.
- Non-interactive use should pass `--skill <name>` instead of encoding the Skill selector in the source string.
- If `@skill-name` is present and `--skill` is also present, `--skill` wins and skit should warn that the inline selector was ignored.
- If the suffix after the last `@` matches the standard Skill name pattern, it may be treated as a Skill selector; otherwise the whole selector is treated as `ref`.
- Git SSH user separators, such as `git@github.com:owner/repo.git`, are part of the source locator and must not be interpreted as Skill selectors.
- URLs, generic git URLs, and local paths treat `@` as part of the locator unless a `#ref@skill-name` selector is present. Use `--skill` for these source forms.
- Branch and tag refs may contain `/` and `@`. Use `--skill` when a ref contains `@` to avoid ambiguity.
- Local paths may contain `@` and `#` only when passed through the platform path parser as an existing path. Otherwise `#` keeps its source/ref meaning.
- Ambiguous non-interactive input must fail with a usage error that suggests `--skill <name>`.

GitHub tree URL rules:

- `https://github.com/owner/repo/tree/<selector>/<subpath>` is parsed as a GitHub source with a tree selector and optional subpath.
- Because refs may contain `/`, the parser should first keep the full post-`tree/` path as an unresolved tree path.
- During resolution, try candidate refs from longest to shortest path prefix against the remote refs. The first candidate that resolves to a branch, tag, or commit becomes `ref`; the remaining path becomes `subpath`.
- If network resolution is unavailable, use the first path segment as `ref` and the remainder as `subpath`, and mark the source as unresolved.

GitLab tree URL rules:

- `https://gitlab.com/group/repo/-/tree/<selector>/<subpath>` follows the same tree selector rules as GitHub.
- The repository locator is everything before `/-/tree/`, excluding the host.
- `gitlab:group/subgroup/repo` is accepted as GitLab shorthand. The full path after `gitlab:` is treated as the repository locator to preserve subgroup support.

Generic git URL rules:

- SSH git locators such as `git@github.com:owner/repo.git` and HTTP(S) URLs ending in `.git` are accepted as generic git sources.
- Generic git sources support `#ref` and `#ref@skill-name` selectors.
- Generic git URLs do not support provider-specific tree URL path parsing; use a repository URL plus `#ref`, or select a Skill by `--skill`.

Well-known and registry source rules:

- HTTP(S) URLs that are not GitHub, GitLab, custom GitLab tree URLs, or `.git` URLs are recognized as `well-known` sources.
- `registry:<slug>` and `clawhub:<slug>` are recognized as registry source locators.
- v1 recognizes these source kinds for clear diagnostics and future compatibility, but does not fetch or install them by default.

Skill selection rules:

- `--skill <name>` overrides any `@skill-name` selector in the source string.
- Lock files must store the resolved Skill selector in `source.skill`; they must not preserve ambiguous inline `@skill-name` syntax.
- If neither source selector nor `--skill` is provided and the source contains multiple Skills, interactive mode prompts and non-interactive mode fails unless `--all` is set.
- If the source resolves to exactly one Skill, that Skill is selected automatically.

### Skill Discovery

Source discovery must be bounded and deterministic in v1:

- If the source root directly contains `SKILL.md` or `skill.md`, the root is one Skill.
- Otherwise, scan direct child directories of the source root and direct child directories under known Skill roots such as `skills/`, `.agents/skills`, `.codex/skills`, `.claude/skills`, `.cline/skills`, `.opencode/skills`, `.goose/skills`, `.windsurf/skills`, and related Agent-specific directories.
- If no Skills are found by bounded priority discovery, fall back to a depth-limited recursive search.
- `--full-depth` forces depth-limited recursive discovery even when the root itself is a Skill or priority discovery finds Skills.
- Skip `.git`, `.skit`, `node_modules`, hidden directories, and non-directory entries during discovery.
- `SKILL.md` is preferred over `skill.md` in the same directory; lowercase `skill.md` produces a warning.
- The parsed `name` is the runtime Skill identity.
- If the parsed `name` differs from the Skill directory basename, the mismatch is a warning, not a validation error.
- Duplicate runtime `name` values discovered in one source are validation errors unless the user selects a single unambiguous Skill by subpath.
- Skills with `metadata.internal: true` are skipped by default during source discovery and produce a warning. Explicit `--skill <name>` selection includes internal Skills.

### Source Object

```json
{
  "type": "github",
  "locator": "owner/repo",
  "url": "https://github.com/owner/repo",
  "ref": "main",
  "resolvedRef": "commit-sha",
  "subpath": "skills/foo",
  "skill": "foo"
}
```

### Safety Rules

Source parsing rejects:

- `..` path traversal in subpaths.
- Absolute subpaths inside repository URLs.
- Empty owner or repo.
- Unsupported URL schemes.

Local paths are resolved to absolute paths before lock writing.

---

## 7. Store And Lock

skit does not record Agent targets in lock files.

### Store

Project store:

```text
.skit/store/
```

Global store:

```text
$XDG_DATA_HOME/skit/store/
```

Fallback:

```text
~/.local/share/skit/store/
```

The store should be content-addressed when possible.

v1 store layout:

```text
.skit/store/<hashes.tree>/<skill-name>/
```

Global store uses the same layout under `$XDG_DATA_HOME/skit/store/`.

Store write rules:

- Fetch and validate into `.skit/tmp/<random>/` for project installs or an equivalent temporary directory beside the global store.
- Compute `hashes.tree` before moving into the final store path.
- Atomically rename the validated Skill into the content-addressed path.
- If the final path already exists with the same tree hash and Skill name, reuse it.
- If the final path already exists but the content differs, fail instead of overwriting.
- `skit remove` removes the lock entry by default. It may remove an unreferenced store directory only when this can be proven locally.
- Future `skit store prune` may remove unreferenced content-addressed directories; v1 does not require it.

### Canonical Tree Hash

`hashes.tree` is a deterministic SHA-256 digest over the Skill directory contents.

The hash algorithm favors reproducibility over mirroring every filesystem detail. It intentionally ignores ambient metadata such as owner, group, directory modes, mtimes, ctimes, xattrs, and ACLs.

Algorithm:

1. Walk the resolved Skill root recursively.
2. Exclude `.git/` and skit-managed metadata directories: `.skit/`, `.clawhub/`, `.clawdhub/`.
3. Reject entries whose relative path is empty, absolute, contains `..`, or uses platform-specific path separators after normalization.
4. Normalize all relative paths to UTF-8 with `/` separators.
5. Sort entries lexicographically by normalized relative path.
6. Include regular files only in v1. Symlinks, device files, sockets, FIFOs, and other non-regular entries are rejected and are never followed.
7. For each file, append this record to the hash stream:

```text
file <relative-path>\n
mode <octal-file-mode>\n
size <decimal-byte-size>\n
sha256 <hex-file-sha256>\n
\n
```

8. The final tree hash is encoded as `sha256-<base64url-no-padding-digest>`.

File hash rules:

- Hash raw bytes exactly as stored; do not normalize line endings.
- `hashes.skillMd` is the same encoded SHA-256 format over raw `SKILL.md` bytes.
- File mode is normalized before hashing:
  - executable regular files are recorded as `0755`;
  - non-executable regular files are recorded as `0644`.
- Directory modes are not hashed.
- `.gitignore`, `.skitignore`, and `.clawhubignore` do not affect lock hashing in v1. They may affect future publish/archive packaging, but `hashes.tree` always reflects the actual installed Skill directory after the fixed exclusions above.

### Project Lock

Path:

```text
.skit/lock.json
```

Purpose:

- Safe to commit.
- Deterministic output.
- No machine-specific timestamps.
- Describes Skill source and content integrity only.

Schema:

```json
{
  "schema": "skit.lock/v1",
  "skills": {
    "frontend-design": {
      "name": "frontend-design",
      "description": "Design frontend interfaces...",
      "source": {
        "type": "github",
        "locator": "vercel-labs/agent-skills",
        "url": "https://github.com/vercel-labs/agent-skills",
        "ref": "main",
        "resolvedRef": "4f2c...",
        "subpath": "skills/frontend-design",
        "skill": "frontend-design"
      },
      "registry": {
        "name": "clawhub",
        "url": "https://clawhub.ai",
        "slug": "frontend-design",
        "version": "1.2.3",
        "digest": "sha256-..."
      },
      "download": {
        "url": "https://clawhub.ai/api/v1/skills/frontend-design/versions/1.2.3/download",
        "sha256": "sha256-..."
      },
      "hashes": {
        "tree": "sha256-...",
        "skillMd": "sha256-..."
      },
      "dependencies": [
        {
          "name": "pdf-core",
          "source": {
            "type": "github",
            "locator": "example/pdf-core",
            "url": "https://github.com/example/pdf-core",
            "ref": "v1.2.0",
            "resolvedRef": "9b1d...",
            "skill": "pdf-core"
          },
          "optional": false
        }
      ]
    }
  }
}
```

Field rules:

- `source.type`: required. Values currently fetched by v1 are `local`, `github`, `gitlab`, and `git`. `registry` and `well-known` are recognized for diagnostics and future provider support but are not fetched by v1.
- `source.locator`: required stable provider locator, such as `owner/repo` for GitHub or an absolute local path for local sources.
- `source.url`: optional canonical URL when one exists.
- `source.ref`: optional requested ref, branch, tag, commit, or registry tag.
- `source.resolvedRef`: optional immutable resolved identity, such as a git commit SHA. It is required for mutable git refs after network resolution.
- `source.subpath`: optional slash-normalized path to the Skill root within the source.
- `source.skill`: optional final runtime Skill selector. It must be present when the original source contained multiple Skills or an inline `@skill` shortcut was used.
- `registry`: optional. Present only when the Skill came from a registry provider or a compatibility import can identify registry origin.
- `registry.name`: optional configured source name, such as `clawhub`.
- `registry.url`: optional registry base URL.
- `registry.slug`: optional registry public identifier.
- `registry.version`: optional registry version or tag.
- `registry.digest`: optional stable digest for registry version metadata or package metadata.
- `download`: optional. Present only when installation used a downloaded package/archive.
- `download.url`: optional canonical download URL.
- `download.sha256`: optional SHA-256 digest of the raw downloaded bytes, encoded as `sha256-<base64url-no-padding-digest>`.
- `dependencies`: optional array of dependency edges resolved during install. Each edge records the dependency runtime `name`, resolved dependency `source`, and `optional` flag. The full dependency Skill must also have its own top-level entry in `skills`.
- `incomplete`: optional boolean. Present and true only for lossy compatibility imports that cannot yet provide enough source/hash data for reproducible restore.
- `warnings`: optional string array. Used for lossy imports and compatibility decisions such as lowercase `skill.md`.

Lock key and name conflict rules:

- The `skills` object key is the runtime Skill `name`.
- v1 does not support installing two different Skills with the same runtime `name` into the same project or global lock.
- Re-adding the same `name` with the same source identity and tree hash is a no-op/update of equivalent metadata.
- Adding the same `name` with a different source identity or tree hash fails with a conflict error.
- Alias, namespace, or multi-origin same-name installs are deferred.

Write rules:

- Sort `skills` keys alphabetically.
- Sort dependencies by `source`, then `skill`.
- End file with newline.
- Do not include timestamps.
- Do not include Agent targets.

### Global Lock

Path:

```text
$XDG_STATE_HOME/skit/lock.json
```

Fallback:

```text
~/.local/state/skit/lock.json
```

Purpose:

- Track globally installed Skills.
- Same schema shape as project lock.
- No install timestamps in v1.

---

## 8. Commands

### `skit add <source>`

Installs a Skill into the project or global skit store.

Options:

- `--global`
- `--project`
- `--skill <name>`
- `--all`
- `--yes`
- `--ignore-deps`
- `--full-depth`

Behavior:

1. Parse source.
2. List available Skills.
3. Select Skill.
4. Resolve source to stable ref/hash.
5. Fetch into a temporary directory.
6. Validate `SKILL.md`.
7. Parse metadata.
8. Hash the resolved Skill directory.
9. Atomically place the Skill under the content-addressed store path.
10. Install dependencies unless ignored.
11. Update project or global lock.

`skit add --global` never modifies project `.skit/lock.json`.

`skit add` in v1 installs into the skit-managed store and writes the relevant lock file. It does not copy or symlink the Skill into Agent target directories. Successful output must make this explicit and should mention that Agent activation is deferred to future `skit sync` or manual copy/symlink.

Required dependency failures block installation. Optional dependency failures warn and continue.

Flag rules:

- `--global` and `--project` are mutually exclusive.
- `--all` and `--skill` are mutually exclusive.
- When a source contains multiple Skills and neither `--all` nor `--skill` is set, interactive mode prompts and non-interactive mode fails with a usage error.
- `--yes` disables confirmation prompts but does not imply `--all`.
- `--ignore-deps` skips dependency installation and records a warning in the result output; it must not create fake dependency lock entries.
- `--full-depth` enables depth-limited recursive source discovery for compatibility with repositories that store Skills outside common priority paths.

### `skit search <query>`

Searches for Skills using a skills.sh-compatible search API.

Environment:

- `SKIT_SEARCH_API_URL`: optional search API base URL.
- `SKILLS_API_URL`: compatibility fallback used when `SKIT_SEARCH_API_URL` is unset.

Behavior:

- Sends `GET /api/search?q=<query>&limit=<limit>` to the configured API.
- Parses `skills[]` items with `id`, `name`, `source`, and `installs`.
- Sorts results by install count descending.
- Prints installable hints in the form `skit add <source> --skill <name>`.
- Supports `--json`.
- Does not install or mutate lock/store state.

### `skit install`

Restores project Skills from `.skit/lock.json`.

If `.skit/lock.json` is absent, skit may read compatible existing lock files and suggest `skit import-lock`.

Entries with `incomplete: true` are not restored automatically. `skit install` must report them and suggest re-adding or inspecting the Skill.

### `skit list`

Lists Skills in project/global skit lock files.

Default output should show runtime Skill name, source, and resolved ref/hash. Declared metadata version is not shown by default; use `skit inspect` for that.

Options:

- `--global`
- `--project`
- `--json`

`--json` returns lock-derived Skill entries.

### `skit remove <name>`

Removes a Skill from the project or global lock. v1 primarily removes the lock entry; content-addressed store garbage collection is conservative and must not remove content still referenced by another lock entry.

Options:

- `--global`
- `--project`
- `--yes`

### `skit init [name]`

Creates a `SKILL.md` template.

v1 behavior:

- With `name`, create `<name>/SKILL.md` under the current working directory.
- Without `name`, create `SKILL.md` in the current working directory and use the current directory basename as the Skill name.
- The generated file must use canonical uppercase `SKILL.md`.
- The generated file must include standard frontmatter and `metadata.skit`.
- Existing `SKILL.md` must not be overwritten.

### `skit update [name]`

Updates mutable sources.

v1 behavior:

- Local sources are re-read, re-hashed, copied into the store, and written back to the lock.
- Commit-pinned git sources do not change unless `--ref` is provided.
- Branch refs update to latest commit.
- Dependencies are refreshed using the same dependency rules as `skit add`; `--ignore-deps` skips dependency refresh and records a warning.
- Registry latest updates are future provider behavior. They are not required for the git/local v1 provider set.

### `skit doctor`

Checks:

- `SKILL.md` validity.
- Metadata conflicts.
- Lock parse errors.
- Hash mismatch.
- Missing binaries.
- Missing environment variables.
- Missing config paths.
- Unsupported platform.

No automatic package installation in v1.

Options:

- `--global`
- `--project`
- `--json`

`--json` returns checks grouped by severity. If any error check exists, the command exits non-zero even when JSON output succeeds.

### `skit inspect <source-or-name>`

Displays:

- Runtime Skill identity.
- Source identity.
- Hashes.
- Dependencies.
- Required bins/env/config.
- File list.
- Warnings.

Options:

- `--global`
- `--project`
- `--skill`
- `--json`

`--json` returns source, metadata-derived requirements, hashes, files, and warnings.

### `skit import-lock <kind>`

Supported kinds:

- `skills`
- `clawhub`

No `export-lock` in v1.

---

## 9. Compatibility Imports

### `skills-lock.json`

Map:

- `source` -> `source.locator`
- `sourceType` -> `source.type`
- `ref` -> `source.ref`
- `computedHash` is preserved as a diagnostic warning in v1 imports. It is not mapped to `hashes.tree` because `skills` uses a different hash algorithm.

### `.clawhub/lock.json`

Map:

- lock key -> registry slug
- `version` -> registry version
- lock key -> `source.skill` when the runtime Skill name can be confirmed from files or origin metadata

If `<skill>/.clawhub/origin.json` exists:

- `registry` -> registry URL
- `slug` -> registry slug
- `installedVersion` -> registry version

Imported ClawHub entries should populate the best available source locator from the lock key and optional `origin.json`. They may leave `source.resolvedRef`, `download.sha256`, and `hashes.tree` empty until the corresponding files can be fetched or inspected.

Imports may be lossy and must report fields that cannot be represented. Lossy entries must set `incomplete: true` and include a warning explaining why they cannot be restored reproducibly yet.

---

## 10. Safety Requirements

Install must never:

- Execute Skill scripts.
- Execute install hooks.
- Run package managers.
- Write outside skit store paths.
- Accept unsafe archive paths.
- Accept path traversal in source subpaths.

Install should warn on:

- `curl | sh`
- `wget | sh`
- base64 decode followed by shell execution
- obfuscated shell payloads
- executable files in Skill directories

---

## 11. Test Matrix

### Metadata

- Parses minimal valid `SKILL.md`.
- Parses lowercase `skill.md` with a warning.
- Uses `SKILL.md` and warns when both `SKILL.md` and `skill.md` exist.
- Rejects missing `name`.
- Rejects missing `description`.
- Rejects invalid `name`.
- Reads `metadata.skit`.
- Reads `skill.yaml` only when `metadata.skit` is absent.
- Fails when both `metadata.skit` and `skill.yaml` exist.
- Maps ClawHub/OpenClaw compatibility `requires`.
- Normalizes compatibility `os: macos` to `platforms.os: darwin`.
- Preserves compatibility `install`, `nix`, `config`, and `cliHelp` without executing them.

### Source Parser

- Parses `owner/repo`.
- Parses GitHub URL.
- Parses GitHub tree URL.
- Parses GitLab URL.
- Parses GitLab shorthand.
- Parses generic git URL.
- Parses well-known HTTP(S) URL.
- Parses registry shorthand.
- Parses SSH git URL.
- Parses local relative path.
- Parses local absolute path.
- Parses `#ref`.
- Parses `#ref@skill`.
- Rejects path traversal.

### Lock

- Writes sorted `.skit/lock.json`.
- Does not write timestamps into project lock.
- Does not write Agent targets.
- Records `source.skill` when a Skill selector is needed.
- Preserves optional `registry` and `download` identity fields.
- Records resolved dependency edges.
- Marks lossy imports as `incomplete: true`.
- Imports `skills-lock.json`.
- Imports `.clawhub/lock.json`.
- Detects hash mismatch.

### Commands

- `add` local Skill.
- `search` Skills.
- `install` from lock.
- `list` project/global.
- `remove` project/global.
- `doctor` reports missing bins/env.
- `inspect` prints source and metadata.

---

## 12. Milestones

### M1: CLI Skeleton

- Go module.
- CLI entrypoint.
- `skit --help`.
- `skit version`.

### M2: Skill Parser

- `SKILL.md` parser.
- Standard field validation.
- `metadata.skit` parser.
- `skill.yaml` parser with mutual exclusion.
- ClawHub/OpenClaw compatibility mapper.

### M3: Source Parser

- Recognized source syntax and v1 provider diagnostics.
- Local path normalization.
- Unsafe subpath rejection.
- Clear not-implemented diagnostics for recognized v1.x source forms.

### M4: Store And Lock

- Store path resolution.
- Project lock read/write.
- Global lock read/write.
- Existing lock imports.

### M5: Local Closed Loop

- `skit add ./skill`.
- `skit list`.
- `skit remove`.
- `skit install`.

### M6: Git Closed Loop

- Fetch GitHub source.
- Fetch GitLab source.
- Fetch generic git source.
- Resolve branch/tag to commit.
- Hash Skill directory.
- Registry search/download remains optional after this milestone and is not required for the v1 MVP.

### M7: Doctor And Inspect

- Environment diagnostics.
- Metadata/source inspection.
- Initial JSON output for `list`, `inspect`, and `doctor`.
- `skit init` template generation.

### M8 / v1.x: Registry Providers

- Registry provider closed loop.
- Well-known registry discovery.

---

## 13. Decisions

- Lock files contain no timestamps in v1. A separate state file is deferred.
- `metadata.skit.version` is inspect-only in v1.
- v1 parses `skill.yaml` only when `metadata.skit` is absent and does not generate it by default.
- Lowercase `skill.md` is accepted with a warning for ecosystem compatibility.
- Lossy lock imports set `incomplete: true` and are not restored automatically.
- Required dependency failure blocks installation. Optional dependency failure warns and continues.
- `metadata.clawhub` is not a v1 compatibility namespace.
- v1 provider scope is `local`, `github`, `gitlab`, and generic `git`. Registry and well-known providers are recognized but remain future work.
- `owner/repo@skill-name`, `github:owner/repo@skill-name`, and `source#ref@skill-name` are compatibility shortcuts; `--skill <name>` is the preferred non-ambiguous selector.
- v1 does not activate Skills into Agent target directories after `add`; Agent sync is deferred.
- v1 store paths use `.skit/store/<hashes.tree>/<skill-name>/`.
- v1 rejects same-name different-origin lock conflicts.
- v1 Skill discovery scans source root, direct children, common Skill roots, and falls back to depth-limited recursive discovery when needed. `--full-depth` forces recursive discovery.
