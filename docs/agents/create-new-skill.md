# How To Create A New Skill

This guide is agent-first: optimize for discoverability, low token cost, and safe reuse by the `skills` plugin.

## Goal

Create a skill that an agent can discover from `skills/<slug>/SKILL.md`, decide to use from its metadata, and load on demand without wasting context.

## What The Runtime Expects

The `skills` plugin loads installed skills from:

```text
skills/<slug>/SKILL.md
```

Optional deeper material can live in:

```text
skills/<slug>/references/
```

The `skill` tool supports:

- `list_skills`
- `load_skill`
- `list_skill_references`
- `load_skill_reference`
- `list_installable`
- `install_skill`
- `delete_skill`

## Recommended Layout

Use this structure:

```text
skills/
  your-skill-name/
    SKILL.md
    references/
      examples.md
      edge-cases.md
```

Keep the main instructions in `SKILL.md`. Put optional detail in `references/` so the agent can load it only when needed.

## Frontmatter Rules

`SKILL.md` must start with YAML frontmatter:

```md
---
name: your-skill-name
description: What the skill does and when to use it.
version: 1.0.0
---
```

Rules enforced by the code:

- `name` is required
- `name` must use lowercase letters, numbers, and hyphens only
- `name` must be at most 64 characters
- `description` is required
- `description` must be at most 1024 characters
- the markdown body must not be empty
- `version` is optional

## Write For Discovery First

The most important field is `description`. It is what helps the agent decide whether the skill is relevant before loading the full body.

Write the description so it answers both:

1. What does this skill help with?
2. When should the agent use it?

Good:

```yaml
description: Create and review GitHub release notes from recent commits. Use when the user asks for release notes, changelogs, or version summaries.
```

Weak:

```yaml
description: Helps with GitHub.
```

## Keep `SKILL.md` Short

Assume the agent already knows general programming concepts. Only include repo-specific process, constraints, decision rules, and output formats.

Good skill bodies usually contain:

- a short purpose statement
- a `## When to Use` section
- a concise workflow
- any required output template
- references to optional files under `references/`

Avoid:

- long tutorials
- repeated background explanation
- multiple equally weighted approaches without a default
- legacy instructions unless the user explicitly asked for backward-compatible behavior

## Suggested `SKILL.md` Shape

Use this template:

````md
---
name: your-skill-name
description: Specific purpose plus clear trigger conditions for when the skill should be used.
version: 1.0.0
---

# Your Skill Name

## When to Use

Use this skill when:
- trigger 1
- trigger 2

Do not use this skill when:
- non-goal 1

## Workflow

1. Inspect the relevant inputs.
2. Apply the project-specific rules.
3. Produce the result in the expected format.
4. Validate the result before finishing.

## Output Format

Use this structure:

```text
...
```

## References

- If examples are needed, read `references/examples.md`.
- If edge cases matter, read `references/edge-cases.md`.
````

## Use `## When to Use` Intentionally

The runtime extracts a short usage summary from:

- `## When to Use`
- or `## Usage`

If neither exists, it falls back to the first paragraph of the body.

Because of that, make `## When to Use` a short, high-signal section. It should be readable as a standalone summary in `list_skills`.

## References Best Practices

Use `references/` for content that is useful but not always necessary:

- examples
- checklists
- edge cases
- repo-specific conventions
- detailed API or format notes

Keep references flat and readable. The tool reads them by relative path under `references/`, and path traversal is intentionally blocked.

## Authoring Checklist

Before considering a skill done, verify:

- the folder is `skills/<slug>/`
- `SKILL.md` exists
- frontmatter is valid
- `name` matches the folder intent
- `description` clearly states what and when
- the body is concise
- `## When to Use` exists and is high signal
- optional detail is pushed into `references/`
- terminology is consistent throughout

## Example

```text
skills/release-notes/SKILL.md
skills/release-notes/references/examples.md
```

Example `SKILL.md`:

````md
---
name: release-notes
description: Draft release notes from recent commits and notable changes. Use when the user asks for changelogs, release summaries, or version notes.
version: 1.0.0
---

# Release Notes

## When to Use

Use this skill when the task is to summarize shipped changes into user-facing release notes or a changelog entry.

## Workflow

1. Gather the relevant change set.
2. Group changes by user-facing outcome.
3. Omit low-signal internal churn unless the user asked for it.
4. Keep the writing concise and scannable.

## Output Format

```text
## Summary
- item

## Fixes
- item
```

## References

- Read `references/examples.md` when tone or structure examples are needed.
````

## Installing And Testing A Skill

For local authoring:

1. Create `skills/<slug>/SKILL.md` directly in the workspace, or author it elsewhere.
2. If authored elsewhere, install it with `install_skill`.
3. Verify discovery with `list_skills`.
4. Verify the full body with `load_skill`.
5. Verify optional files with `list_skill_references` and `load_skill_reference`.

For remote distribution:

- publish the skill under `repository/skills/<slug>/`
- ensure `SKILL.md` is valid
- verify it appears in `list_installable`

## Default Mindset

Write the smallest skill that reliably changes agent behavior in the right direction. Prefer precise triggers, explicit non-goals, and progressive disclosure over long instructions.
