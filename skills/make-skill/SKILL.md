---
name: make-skill
description: Create or revise Agent Skills repositories with one or more skills under skills/, precise SKILL.md frontmatter, concise instructions, validation checks, and skit-friendly metadata.
license: MIT
metadata:
  author: vlln
  version: "0.1.0"
---
# Make Skill

Use this skill to create or improve an Agent Skills repository. Produce a small,
concrete repository with one or more focused skills under `skills/`, not a
generic tutorial.

## Principles

### Audience Separation

A skill repository has three audiences. Keep them strictly separated — never
leak information from one audience's file into another's.

| Audience | File | Purpose |
|----------|------|---------|
| Human user | `README.md` | Decide whether to install. |
| Agent using the skill | `SKILL.md` | When to activate, what to do, how to validate. |
| Skill developer | All files outside `skills/` | Design rationale, architecture decisions. |

The `skills/` directory is the product. Everything outside it is development
context. Do not let development context leak into `SKILL.md`.

### Abstraction over Implementation

`SKILL.md` declares **intent and capability**, not how standard operations are
implemented. The agent already knows how to write YAML, structure markdown, or
split concerns across files — don't explain those.

**Proprietary scripts and tools are different.** Scripts bundled in the skill
are not in the LLM's pretraining data. The agent cannot discover them by
reasoning from general knowledge. For these, describe what they do, their
arguments, and when to use them:

- Standard operation: "Create a SKILL.md with proper frontmatter and body."
- Proprietary script: "Run `skit init <name>` to scaffold a new skill
  repository with the correct directory layout and template files."

### Write Only What the LLM Does Not Know

The agent is driven by an LLM with broad general knowledge. Do not explain
standard tools, common concepts, or widely known workflows. Focus exclusively
on domain-specific business logic, project conventions, non-obvious constraints,
proprietary APIs, and org-specific processes. If the agent could infer it from
context or already knows it, leave it out.

### Calibrating Control

Not every part of a skill needs the same level of prescriptiveness. Match the
specificity to the fragility of the task.

- **Give freedom** when multiple approaches are valid and the task tolerates
  variation. Explain *why* rather than giving rigid directives — an agent that
  understands the purpose makes better context-dependent decisions.
- **Be prescriptive** when operations are fragile, consistency matters, or a
  specific sequence must be followed. Pin exact commands, forbid deviation.

Provide **defaults, not menus**. When multiple tools could work, pick one and
mention alternatives briefly — don't present them as equal options.

Favor **procedures over declarations**. Teach the agent *how to approach* a
class of problems, not *what to produce* for a specific instance.

### Progressive Disclosure

Keep `SKILL.md` under **500 lines**. It should give a clear macro understanding
and enable direct work after a single read.

For complex skills, use progressive disclosure: `SKILL.md` describes the macro
mechanism and workflow; `references/` holds detailed specs, API docs, schemas.
Tell the agent exactly when to read each reference file.

Do not over-decouple: if the content fits in one `SKILL.md`, keep it there.
Split to `references/` only when the skill would exceed 500 lines or when the
detail is genuinely secondary.

## Path Referencing Convention

When referring to files inside the skill directory (e.g., scripts, assets, references), use the placeholder `$_S` to denote the absolute path of the skill root directory.

- `$_S` is not a shell variable that exists by default. It is a convention used in SKILL.md to unambiguously indicate "the skill’s own directory".
- In actual execution, the agent must replace `$_S` with the real absolute path before running commands. For example:
  ```bash
  export _S="/absolute/path/to/skill"; $_S/scripts/run.sh
  ```
  Here, `_S` (without `$`) is the actual environment variable name; `$_S` in documentation is the placeholder notation.
- Do not use bare relative paths such as `scripts/run.sh` or `./assets/config.yaml`, because the agent may resolve them relative to its current working directory, not the skill directory.
- When a script has many parameters, describe only its core function in SKILL.md and direct the agent to `--help` for the full usage. Do not exhaustively list every flag.

Every SKILL.md that uses `$_S` must include a one-line definition near the top of its body, so the consuming agent knows what it means:

> `$_S` — placeholder for this skill's root directory. Set `_S` to the actual path before running commands.

## Frontmatter

Start from this template and remove fields that do not apply. For a minimal
skill, only `name` and `description` are needed.

```yaml
---
name: skill-name          # 1-64 lowercase letters/digits/hyphens, no leading/trailing/consecutive hyphen
description: >            # what the skill does and when the agent should activate it
  TODO: describe the skill's capability and the situations where it should be used.

# Optional: discovery & activation
# when-to-use: >          # additional trigger context (combined with description, 1536-char limit)
#   TODO: extra situations.
# argument-hint: <file> [format]
# arguments:
#   - file
#   - format
# disable-model-invocation: false
# user-invocable: true

# Optional: metadata
license: MIT
metadata:
  author: your-name
  version: "0.1.0"

# Optional: skit requirements
# requires:
#   bins:
#     - tool-name
#   env:
#     - REQUIRED_ENV_VAR
#   config:
#     - ~/.config/tool
#   skills:
#     - github:owner/repo@required-skill
#   platforms:
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
- `description` is recommended; keep it under 1024 characters. Use imperative
  phrasing: "Use this skill when..." rather than "This skill does...". Focus on
  user intent — describe what the user is trying to achieve, not the skill's
  internal mechanics. Err on being pushy: list contexts explicitly, including
  cases where the user doesn't name the domain directly.
- `when-to-use` + `description` are combined and truncated to 1536 characters.
- Include `license` when the skill is intended to be shared.
- Use `requires` only for requirements the agent can diagnose.
- Use `requires.skills` for dependent skills, written as skit install targets
  such as `github:owner/repo@skill-name`.
- Do not invent dependencies, tools, environment variables, or platforms.

## Body Structure

Choose the pattern that matches the skill's primary task. Most real skills
combine multiple patterns — Pipeline is the common orchestrator.

### 1. Tool Wrapper

Wrap an external tool/API as an on-demand expert. Trigger keywords → load
`references/` dynamically.

Distinguishing structure: `## Trigger Keywords` + `## Capabilities` + pointers
to `references/` files with usage conditions.

Use when: wrapping CLI tools, APIs, domain-specific reference material.

### 2. Generator

Use `assets/` templates and `references/` style guides to enforce consistent
output. The skill coordinates the generation pipeline.

Distinguishing structure: `## Workflow` that reads a style guide → renders a
template → validates against a checklist. Key files: `assets/` templates,
`references/` style guides.

Use when: generating standardized artifacts (API docs, reports, configs).

### 3. Reviewer

Separate "what to check" (swappable checklist file) from "how to check" (fixed
review workflow).

Distinguishing structure: `## Workflow` that reads `references/<type>-checklist.md`,
iterates each item, and outputs structured findings with severity.

Use when: code review, security audit, accessibility check, compliance scan.

### 4. Inversion

Gate the agent with a three-phase interview (goal → constraints → execution)
to prevent guessing. The user drives the specification.

Distinguishing structure: `## Phase 1: Goal`, `## Phase 2: Constraints`,
`## Phase 3: Execution`. Each phase ends with a confirmation checkpoint.
Rule: never guess the user's intent.

Use when: project planning, requirements gathering — any task where guessing
is more expensive than asking.

### 5. Pipeline

Multi-step workflow with mandatory checkpoints. Each stage produces a verified
intermediate result before the next stage begins.

Distinguishing structure: `## Pipeline` with `### Stage N: [Name]` subsections.
Each stage has steps and a **Checkpoint** before proceeding. Rules: never skip
a checkpoint; fix failures before continuing.

Use when: multi-step processes where order matters and intermediate validation
is critical.

### Choosing and Combining

| Task needs | Pattern |
|------------|---------|
| External tool/API | 1. Tool Wrapper |
| Standardized output | 2. Generator |
| Quality/compliance check | 3. Reviewer |
| User intent is ambiguous | 4. Inversion |
| Ordered steps with checkpoints | 5. Pipeline |

Pipeline is the common orchestrator — it strings together stages, each using
a different pattern:

```
Pipeline:
  1. [Inversion]     Interview the user to confirm the goal.
  2. [Tool Wrapper]  Call the API to fetch data.
  3. [Generator]     Render the output using templates.
  4. [Reviewer]      Validate the result against the checklist.
```

### Writing Techniques

**Gotchas section.** The highest-value content in many skills is a list of
gotchas — domain-specific facts that defy reasonable assumptions. When an agent
makes a mistake you have to correct, add the correction to a `## Gotchas`
section in `SKILL.md`. Keep the agent from repeating the error next time.

**Output templates.** When the agent must produce output in a specific format,
provide a concrete template. Short templates inline in `SKILL.md`; longer ones
in `assets/` and reference them.

**Checklists for multi-step workflows.** An explicit `- [ ]` checklist helps
the agent track progress and avoid skipping steps, especially when steps have
dependencies or validation gates.

## Workflow

1. Identify the repository's domain and the recurring tasks it should cover.
2. Ask only for missing domain-specific facts that would change the repository
   structure or public skill list.
3. Create the repository skeleton. Prefer `skit init <name>` if available.
   If `skit` is not installed, create manually:
   - $_S/README.md from `assets/README.template.md`
   - $_S/skills/<name>/SKILL.md from the frontmatter template above
4. Work inside the repository and revise the initial skill instead of
   recreating the directory layout by hand.
5. Split the domain into focused skills. Do not force everything into a single
   skill when separate activation triggers would be clearer.
6. For additional skills, create `skills/<skill-name>/SKILL.md` using the same
   frontmatter and body rules.
7. Add per-skill `scripts/`, `references/`, or `assets/` only when they remove
   real repetition or keep `SKILL.md` concise.
8. Create the root `README.md` from `assets/README.template.md`. Replace all
   `placeholders`, keep only install methods that apply, and list every public
   skill in the `Skills` table. The README is human-facing — do not describe
   internal skill mechanics or agent workflows there.
9. Validate (see checklist below).

## Validation Checklist

Before finishing, check:

- Every skill directory name matches its `name`.
- Every `SKILL.md` frontmatter parses as YAML.
- Every `description` clearly identifies the skill's purpose and when to use it.
- Required tools/env/platforms are real and minimal.
- Every `SKILL.md` is under 500 lines.
- Any examples are runnable or clearly marked as templates.
- If `skit` is available: `skit install ./<repo-name> --all` and `skit check`
  succeed in a disposable test environment.