---
name: make-skill
description: Create or revise Agent Skills repositories with one or more skills under skills/, precise SKILL.md frontmatter, concise instructions, validation checks, and skit-friendly metadata.
license: MIT
compatibility: Agent Skills SKILL.md format; optimized for skit-managed skill repositories.
metadata:
  skit:
    version: 0.1.0
    requires:
      bins:
        - skit
    keywords:
      - skill-authoring
      - agent-skills
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

Start from this template and remove fields that do not apply:

```yaml
---
name: skill-name
description: [One concise sentence describing the skill's specific capability and the situations where it is useful.]
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
      skills:
        - github:owner/repo@required-skill
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
- Write `description` as a concise capability summary with enough context for
  an agent to decide when the skill is relevant.
- Include `license` when the skill is intended to be shared.
- Include `compatibility` only for real requirements.
- Use `metadata.skit.requires` only for requirements the agent can diagnose.
- Use `metadata.skit.requires.skills` for required Skills, written as complete
  skit install targets such as `github:owner/repo@skill-name`.
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

Do not duplicate large README template content inside this skill. If the
repository README needs install or quick-start wording changes, update the
generated README directly or the init template in skit. This skill should
describe the workflow and validation rules, not carry a second copy of the
template.

Do not add a README inside an individual skill directory unless the user asks
for one or the skill needs human-facing package documentation.

## Validation Checklist

Before finishing, check:

- Every skill directory name matches its `name`.
- Every `SKILL.md` frontmatter parses as YAML.
- Every `description` clearly identifies the skill's purpose and relevant use
  cases without boilerplate phrasing.
- Required tools/env/platforms are real and minimal.
- Each skill body is procedural, concise, and domain-specific.
- Agent-facing files are high-density, low-noise, and non-repetitive.
- Any examples are runnable or clearly marked as templates.
- The root README is human-facing and lists all public skills.
- No generic filler remains.
- `skit install ./<repo-name> --all` and `skit check` succeed in a disposable
  test environment.
