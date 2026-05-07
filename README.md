<p align="center">
  <img src="docs/assets/skill-kit-mark.svg" width="112" alt="Skill Kit logo">
</p>

<h1 align="center">Skill Kit</h1>

<p align="center">
  <strong>Agent-first Skill inventory for local AI coding agents.</strong>
</p>

<p align="center">
  <code>skit</code> manages <code>SKILL.md</code> directories in one local store, then exposes them through simple CLI commands and symlinks.
</p>

<p align="center">
  <a href="LICENSE"><img alt="License: MIT" src="https://img.shields.io/badge/license-MIT-111827?style=flat-square"></a>
  <img alt="Go 1.24+" src="https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat-square">
  <img alt="CLI: skit" src="https://img.shields.io/badge/CLI-skit-2F80ED?style=flat-square">
  <img alt="Model: manifest" src="https://img.shields.io/badge/model-manifest-27AE60?style=flat-square">
  <img alt="No skill execution during install" src="https://img.shields.io/badge/install-no%20skill%20execution-F2C94C?style=flat-square">
  <img alt="Platforms" src="https://img.shields.io/badge/platforms-macOS%20%7C%20Linux%20%7C%20Windows-101820?style=flat-square">
</p>

---

Skill Kit is a small manager for [Agent Skills](https://agentskills.io/). It is
designed for an AI agent to operate directly: search for a capability, install
the matching Skill, list what is available, and repair local state without
needing an interactive package-manager session.

The problem it solves is local fragmentation. Codex reads `~/.codex/skills`,
Claude Code reads its own skills directory, Cursor has another location, and
agents end up with separate copies of the same capability. `skit` keeps one
machine-level Skill inventory and exposes it through the standard active skills
directory:

```text
${XDG_DATA_HOME:-~/.local/share}/skit/
├── manifest.json
└── skills/
    └── code-review/
        └── SKILL.md

~/.agents/skills/code-review -> ~/.local/share/skit/skills/code-review
```

Agent-specific directories are compatibility targets, not the core model. The
core model is the store.

## Design

`skit` deliberately avoids acting like a heavyweight package manager. A Skill is
usually a directory of instructions and supporting files, not a compiled
artifact with a meaningful update protocol.

That leads to a simple design:

- install Skills by copying them into the local skit store;
- record source, name, description, and active links in `manifest.json`;
- activate Skills with symlinks;
- export the manifest as `skit.json` when the setup should be shared;
- update by reinstalling from recorded sources.

The store is intentionally plain. `skit` records where a Skill came from and
reinstalls from that source when asked; it does not build a second package
system around hashes, locks, or garbage collection.

Text output is compact on purpose. It is meant for an active AI agent: high
signal, low boilerplate, no repeated command essays. More detail is available
through `help`, `check`, `list --all`, and `--json`.

## Install

Install macOS/Linux with the project installer:

```sh
curl -fsSL https://raw.githubusercontent.com/vlln/skit/main/install.sh | sh
```

Package-manager distribution can layer on the same release artifacts:

```sh
brew install <tap>/skit
```

From a checkout:

```sh
go build -o skit ./cmd/skit
```

Requirements are intentionally small: `git` for remote sources, and Go only
when building from source.

## Use

Install a Skill source:

```sh
skit install github:owner/repo@code-review
```

List the local skit inventory:

```sh
skit list
```

Search when the agent does not know the source yet:

```sh
skit search "code review"
skit search review --source github:owner/skills
```

Create a Skill repository:

```sh
skit init review
```

This creates:

```text
review-skill/
├── README.md
└── skills/
    └── review/
        └── SKILL.md
```

Export the current inventory to a repository-friendly file:

```sh
skit export
```

Another agent can apply that setup from a checkout containing `skit.json`:

```sh
skit install
```

Preview first:

```sh
skit install --dry-run
```

## Output Shape

Default output is short and stable enough for an LLM to read without wasting
context.

```text
$ skit search review --source ./skills
./skills/review    Review code changes before committing.

$ skit install ./skills/review
installed review
active ~/.agents/skills/review

$ skit list
review    active    Review code changes before committing.
```

`list --all` is the escape hatch for discovering Skills that already exist in
agent directories but are not managed by skit:

```text
managed
review    active    Review code changes before committing.

external
legacy-review    ~/.codex/skills/legacy-review    Legacy review workflow.
```

## Sharing

The local database lives at:

```text
${XDG_DATA_HOME:-~/.local/share}/skit/manifest.json
```

`skit export` writes the same manifest format to `./skit.json`. Commit that file
when a repository or team wants a shared Skill setup. Local Skills created with
`skit init` should usually be committed as normal repository files under
`skills/`; externally installed Skills belong in `skit.json`.

Example manifest:

```json
{
  "schema": "skit.manifest/v1",
  "skills": {
    "review": {
      "name": "review",
      "description": "Review code changes before committing.",
      "source": {
        "type": "github",
        "locator": "owner/skills",
        "url": "https://github.com/owner/skills.git",
        "ref": "main",
        "subpath": "skills/review",
        "skill": "review"
      },
      "path": "skills/review",
      "agents": ["universal"]
    }
  }
}
```

## Sources

Common source forms:

```text
./skill
owner/repo
owner/repo@skill-name
owner/repo/path/to/skill
github:owner/repo#main@skill-name
https://github.com/owner/repo/tree/main/skills/review
https://skills.sh/owner/repo/skill-name
git@github.com:owner/repo.git
```

Use `--skill <name>` when an inline selector would be ambiguous. Use `--all`
when the source contains multiple Skills and all of them should be installed.

GitHub private repositories are token-friendly. Credentials are resolved in
this order:

1. `SKIT_GITHUB_TOKEN`
2. `GITHUB_TOKEN`
3. `GH_TOKEN`
4. `gh auth token`

Git prompts are disabled during fetches, so agent workflows fail instead of
hanging for credentials.

## Commands

The command surface is intentionally small.

| Command | Purpose |
|---------|---------|
| `skit search <query>` | Find installable Skills. |
| `skit install <source>` | Install a source into the local store. |
| `skit install` | Apply `./skit.json` if present. |
| `skit list` | Show the managed inventory. |
| `skit list --all` | Include unmanaged external Skills found on disk. |
| `skit update [name]` | Reinstall from recorded sources. |
| `skit remove <name>` | Remove from manifest, active link, and local store. |
| `skit check` | Validate manifest entries and active links. |
| `skit export [path]` | Write a shareable manifest, default `./skit.json`. |
| `skit init <name>` | Create a Skill repository skeleton. |

Useful flags:

```text
--skill <names...> Select Skills from one source
--name <name>      Install one Skill under a local name
--all              Install all Skills from a source; with list, scan external dirs
--full-depth       Search recursively during source discovery
--force            Replace an existing non-symlink active path
--keep             Remove manifest/link state but keep the local copy
--dry-run          Preview manifest installation
--json             Print structured output for scripts
```

`--agent <name>` exists for compatibility when an agent requires a private
skills directory. It is not the default workflow.

## Safety

`skit install`, `search`, `list`, `update`, and `check` do not execute Skill
code. Install copies files, validates basic Skill shape, records metadata, and
creates symlinks.

The CLI rejects unsafe source subpaths, rejects non-regular files while copying,
normalizes executable modes, and warns about suspicious content such as piping a
network download into a shell.

## Bundled Skills

This repository includes Skills for working with Skill repositories:

| Skill | Description |
|-------|-------------|
| [`search-skills`](skills/search-skills) | Find, evaluate, and install agent skills with `skit`. |
| [`make-skill`](skills/make-skill) | Create or revise Agent Skills repositories. |

Install a bundled Skill from a published repository:

```sh
skit install owner/repo/skills/make-skill
```

## Development

```sh
go test ./...
go build -o skit ./cmd/skit
```

Manual smoke:

```sh
go build -o ./skit-e2e ./cmd/skit
./skit-e2e init demo
./skit-e2e install ./demo-skill
./skit-e2e list
./skit-e2e check
./skit-e2e remove demo
```

## License

MIT.
