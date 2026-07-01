---
name: make-skill
description: Create or revise Agent Skills repositories with one or more skills under skills/, precise SKILL.md frontmatter, concise instructions, validation checks, and skit-friendly metadata.
license: MIT
metadata:
  author: vlln
  version: "0.1.0"
requires:
  bins:
    - skit
---
# Make Skill

Use this skill to create or improve an Agent Skills repository. Produce a small,
concrete repository with one or more focused skills under `skills/`, not a
generic tutorial.

## Naming

Prefer short command-like skill names:

- lowercase letters, numbers, and hyphens only
- 1-64 characters
- no leading, trailing, or repeated hyphens
- each skill directory name should match its `name`

Use `make-skill` for this skill. It is short, imperative, and Unix-like.

## Workflow

1. Identify the repository's domain and the recurring tasks it should cover.
2. Ask only for missing domain-specific facts that would change the repository
   structure or public skill list.
3. For a new repository, run `skit init <name>` first. This creates the
   canonical `<name>-skill/README.md` and
   `<name>-skill/skills/<name>/SKILL.md` skeleton.
4. Work inside the generated repository and revise the initial skill instead of
   recreating the directory layout by hand.
5. Split the domain into one or more focused skills. Do not force everything
   into a single skill when separate activation triggers would be clearer.
6. For additional public skills, create `skills/<skill-name>/SKILL.md` using
   the same frontmatter and body rules.
7. Add per-skill `scripts/`, `references/`, or `assets/` only when they remove
   real repetition or keep that skill's `SKILL.md` concise. Treat these files
   as agent-facing when an agent is expected to read or run them.
8. Update the generated repository root `README.md` so it remains human-facing
   and lists every public skill.
9. Validate every skill's frontmatter, description, body constraints, and
   examples.

## Frontmatter Template

Start from this template and remove fields that do not apply. For a minimal
skill, only `name` and `description` are needed — everything else is optional.

Full template:

```yaml
---
# Required ──────────────────────────────────────────
name: skill-name          # 1-64 lowercase letters/digits/hyphens, no leading/trailing/consecutive hyphen
description: >            # recommended: one concise sentence describing capability and when to use
  TODO: describe the skill's purpose and situations where an agent should use it.

# Optional: discovery & activation ─────────────────
# when-to-use: >          # additional trigger context (appended to description, shared 1536-char limit)
#   TODO: extra situations.
# allowed-tools:           # tools pre-approved while skill is active (YAML list or space-separated string)
#   - Bash(git:*)
#   - Read
# disallowed-tools:        # tools removed from the pool while skill is active
#   - AskUserQuestion
# argument-hint: <file> [format]   # auto-complete hint for expected arguments
# arguments:               # named positional args for $name substitution in body
#   - file
#   - format
# disable-model-invocation: false   # set true to prevent automatic loading (manual /name only)
# user-invocable: true     # set false to hide from the / menu

# Optional: metadata ────────────────────────────────
license: MIT              # SPDX identifier or reference to bundled license file
metadata:                 # arbitrary string→string annotations
  author: your-name
  version: "0.1.0"

# Optional: skit requirements ───────────────────────
# requires:               # structured runtime requirements for automated diagnostics
#   bins:                 # all must be present
#     - tool-name
#   any-bins:             # at least one must be present
#     - pdftotext
#     - mutool
#   env:                  # environment variables that must be set
#     - REQUIRED_ENV_VAR
#   config:               # config file paths that must exist
#     - ~/.config/tool
#   skills:               # dependent skills (skit install targets)
#     - github:owner/repo@required-skill
#   platforms:            # supported platforms
#     os:
#       - linux
#       - darwin
#     arch:
#       - amd64
#       - arm64
---
```

Rules:

- `name` is recommended but not required; defaults to directory basename.
- `description` is recommended; keep it under 1024 characters.
- `when-to-use` + `description` are combined and truncated to 1536 characters.
- `allowed-tools` / `disallowed-tools` accept a YAML list or space-separated string.
- Include `license` when the skill is intended to be shared.
- Use `requires` only for requirements the agent can diagnose.
- Use `requires.skills` for dependent skills, written as complete skit install
  targets such as `github:owner/repo@skill-name`.
- Do not invent dependencies, tools, environment variables, or platforms.

## Body Structure

Keep each `SKILL.md` focused on what the agent would not reliably know.

Recommended sections:

```markdown
# Skill Title

## When To Use

[One short paragraph or bullets describing task boundaries.]

## Workflow

1. [Concrete first step]
2. [Concrete second step]
3. [Validation or handoff step]

## Rules

- [Non-obvious constraint]
- [Project/domain-specific convention]
- [Failure mode to avoid]

## Output

[Expected artifact, command, patch, or response shape.]
```

## Audience Separation

A skill repository has three audiences with distinct needs. Keep them strictly
separated — never leak information from one audience's file into another's.

| Audience | File | Purpose |
|----------|------|---------|
| Human user | `README.md` | Decide whether to install. Install instructions, skill list, value proposition. |
| Agent using the skill | `SKILL.md` | When to activate, what to do, boundaries, how to validate. |
| Skill developer | `references/`, commit messages | Design decisions, architecture rationale, why things are the way they are. |

Rules:

- `SKILL.md` must not contain development history ("we discussed X so we chose
  Y"), design rationale, or why a particular approach was taken.
- `SKILL.md` must not contain references to the repository structure beyond what
  the agent needs to navigate (e.g. `references/api.md` is fine; "we put this
  in scripts/ to keep the skill clean" is not).
- `README.md` must not describe how the agent should use the skill internally.
- When writing a skill, you are the developer. The skill you write is for the
  agent. Do not confuse the two roles — the agent that reads the skill does not
  need to know what you were thinking.

## Abstraction vs Implementation

`SKILL.md` declares **intent and capability**, not implementation. The agent
reading the skill needs to know what it can do and when, not how it is done
under the hood.

- **Declaration** (put in SKILL.md): "Extract text and tables from PDF files."
- **Implementation** (put in `scripts/` or `references/`): "Run `scripts/parse.py --format json`."

Do not write the implementation as the primary description. If the skill body
says "use `scripts/foo.py` to do X", the skill breaks when the script is
renamed or replaced. Instead say "do X" and let the agent discover the script
through the file system or a `references/` index.

Implementation details belong in:
- `scripts/` — executable helpers the agent runs
- `references/` — detailed specs, API docs, schemas
- `assets/` — templates or static resources

`SKILL.md` should remain readable even if every implementation file is swapped
out.

## Content Constraints

These constraints apply to every agent-facing artifact: `SKILL.md`, helper
scripts, command output examples, reference snippets, assets intended for agent
inspection, and any generated prompts. The root `README.md` is the exception:
it is human-facing repository documentation.

Write for an active AI agent, not for a passive human reader and not primarily
for a strict parser. The agent can inspect files, run `help`, execute commands,
and ask for missing facts.

Do include:

- domain-specific procedures
- exact commands or APIs when they are known
- project conventions and file locations
- compact examples that prevent likely mistakes
- validation checks the agent should actually run
- decision criteria, failure modes, and stop conditions that are not obvious
- high-density instructions with low repetition

Do not include:

- generic advice such as "handle errors", "write clean code", or "follow best
  practices"
- broad background the model already knows
- long explanations of common concepts
- unrelated setup, changelog, README, or marketing text
- multiple equal menus of tools when one default should be chosen
- verbose next-step blocks when a short command or rule is enough
- machine-oriented JSON schemas unless the skill specifically requires
  structured data output
- repeated command prefixes or boilerplate that the agent can infer
- **development context**: discussions, decisions, trade-offs, or rationale
  behind why the skill is structured a certain way
- **implementation as description**: using a script path as the primary
  description of what the skill does
- **repository self-reference**: "this repository", "we use", "the skill
  author decided" — the agent does not need this meta-information

If detailed reference material is needed, put it in `references/` and tell the
agent exactly when to read it.

## Repository README

For a new repository, use `skit init <name>` to create the root `README.md`.
Then replace placeholders, keep only install methods that apply, and list every
public skill in the `Skills` table.

The README is human-facing repository documentation for people deciding whether
to use the skills and how to install them. Do not describe internal skill
mechanics there, such as when the agent should read `references/`, how helper
scripts are invoked, or detailed implementation rules. Put that operational
guidance in the relevant `SKILL.md`.

Do not add a README inside an individual skill directory unless the user asks
for one or the skill needs human-facing package documentation.

Use `assets/README.template.md` as the README template. Replace all
`<placeholders>` with real values, remove sections that do not apply, and
keep only the install methods that actually work for this repository.

## Validation Checklist

Before finishing, check:

- Every skill directory name matches its `name`.
- Every `SKILL.md` frontmatter parses as YAML.
- Every `description` clearly identifies the skill's purpose and relevant use
  cases without boilerplate phrasing.
- Required tools/env/platforms are real and minimal.
- Each skill body is procedural, concise, and domain-specific.
- No development context, design rationale, or repository self-reference has
  leaked into any `SKILL.md`.
- `SKILL.md` describes intent and capability, not implementation paths.
  Script paths appear only in concrete steps, never as the primary description
  of what the skill does.
- `README.md` is human-facing and does not describe agent-internal mechanics.
- Agent-facing files are high-density, low-noise, and non-repetitive.
- Any examples are runnable or clearly marked as templates.
- The root README is human-facing and lists all public skills.
- No generic filler remains.
- `skit install ./<repo-name> --all` and `skit check` succeed in a disposable
  test environment.