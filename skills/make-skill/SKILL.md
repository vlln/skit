---
name: make-skill
description: Create or revise Agent Skills with precise SKILL.md frontmatter, concise task-specific instructions, validation checks, and skit-friendly metadata. Use when the user asks to make, draft, improve, package, review, or standardize a skill.
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

Use this skill to create or improve an Agent Skill. Produce a small, concrete
skill package, not a generic tutorial.

## Naming

Prefer short command-like names:

- lowercase letters, numbers, and hyphens only
- 1-64 characters
- no leading, trailing, or repeated hyphens
- directory name must match `name`

Use `make-skill` for this skill. It is short, imperative, and Unix-like.

## Workflow

1. Identify the recurring task the skill should help with.
2. Ask only for missing domain-specific facts that would change the skill.
3. Choose a narrow scope. If the request contains multiple domains, split them.
4. Create `skills/<skill-name>/SKILL.md` for a repo-level skill collection, or
   `<skill-name>/SKILL.md` for a standalone skill directory.
5. Add `scripts/`, `references/`, or `assets/` only when they remove real
   repetition or keep `SKILL.md` concise.
6. For a repo-level skill collection, create or update the root `README.md`
   from `assets/README.template.md`.
7. Validate frontmatter, trigger description, body constraints, and examples.

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

Keep `SKILL.md` focused on what the agent would not reliably know.

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

The README is repository-level documentation. Do not add a README inside an
individual skill directory unless the user asks for one or the skill needs
human-facing package documentation.

## Validation Checklist

Before finishing, check:

- Directory name matches `name`.
- Frontmatter parses as YAML.
- `description` states when to use the skill and when not to.
- Required tools/env/platforms are real and minimal.
- The body is procedural, concise, and domain-specific.
- Any examples are runnable or clearly marked as templates.
- No generic filler remains.
- If the skill is in a skit repo, `skit inspect ./skills/<skill-name>` succeeds
  for repo-level collections, or `skit inspect ./<skill-name>` succeeds for a
  standalone skill directory.
