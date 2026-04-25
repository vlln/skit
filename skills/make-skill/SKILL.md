---
name: make-skill
description: Create or revise an Agent Skills repository with one or more skills under skills/, precise SKILL.md frontmatter, concise task-specific instructions, validation checks, and skit-friendly metadata. Use when the user asks to make, draft, improve, package, review, or standardize a skill repo or skill collection.
license: MIT
compatibility: Agent Skills SKILL.md format; optimized for skit-managed skill repositories.
metadata:
  skit:
    version: 0.1.0
    keywords:
      - skill-authoring
      - agent-skills
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
- each skill directory name must match its `name`

Use `make-skill` for this skill. It is short, imperative, and Unix-like.

## Workflow

1. Identify the repository's domain and the recurring tasks it should cover.
2. Ask only for missing domain-specific facts that would change the repository
   structure or public skill list.
3. Split the domain into one or more focused skills. Do not force everything
   into a single skill when separate activation triggers would be clearer.
4. Create `skills/<skill-name>/SKILL.md` for each public skill.
5. Add per-skill `scripts/`, `references/`, or `assets/` only when they remove
   real repetition or keep that skill's `SKILL.md` concise.
6. Create or update the repository root `README.md` from
   `assets/README.template.md`.
7. Validate every skill's frontmatter, trigger description, body constraints,
   and examples.

## Frontmatter Template

Start from this template and remove fields that do not apply:

```yaml
---
name: skill-name
description: Use this skill when the user needs [specific task], including [trigger phrases or adjacent intents]. It helps the agent [concrete capability] without [important boundary].
license: MIT
compatibility: Requires [specific tools, product, network, platform, or runtime]. Omit this field if there are no special requirements.
metadata:
  skit:
    version: 0.1.0
    requires:
      bins:
        - tool-name
      env:
        - REQUIRED_ENV_VAR
      platforms:
        os:
          - linux
          - darwin
    keywords:
      - keyword
      - domain
---
```

Rules:

- `name` and `description` are required.
- Keep `description` under 1024 characters.
- Write `description` as activation guidance: "Use this skill when..."
- Include `license` when the skill is intended to be shared.
- Include `compatibility` only for real requirements.
- Use `metadata.skit.requires` only for requirements the agent can diagnose.
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

## Content Constraints

Do include:

- domain-specific procedures
- exact commands or APIs when they are known
- project conventions and file locations
- compact examples that prevent likely mistakes
- validation checks the agent should actually run

Do not include:

- generic advice such as "handle errors", "write clean code", or "follow best
  practices"
- broad background the model already knows
- long explanations of common concepts
- unrelated setup, changelog, README, or marketing text
- multiple equal menus of tools when one default should be chosen

If detailed reference material is needed, put it in `references/` and tell the
agent exactly when to read it.

## Repository README

When creating a repository that stores skills under `skills/`, use
`assets/README.template.md` as the root `README.md` starting point. Replace all
placeholders, keep only install methods that apply, and list every public skill
in the `Skills` table.

The README is human-facing repository documentation for people deciding whether
to use the skills and how to install them. Do not describe internal skill
mechanics there, such as when the agent should read `references/`, how helper
scripts are invoked, or detailed implementation rules. Put that operational
guidance in the relevant `SKILL.md`.

Do not add a README inside an individual skill directory unless the user asks
for one or the skill needs human-facing package documentation.

## Validation Checklist

Before finishing, check:

- Every skill directory name matches its `name`.
- Every `SKILL.md` frontmatter parses as YAML.
- Every `description` states when to use that skill and when not to.
- Required tools/env/platforms are real and minimal.
- Each skill body is procedural, concise, and domain-specific.
- Any examples are runnable or clearly marked as templates.
- The root README is human-facing and lists all public skills.
- No generic filler remains.
- `skit inspect ./skills/<skill-name>` succeeds for every public skill.
