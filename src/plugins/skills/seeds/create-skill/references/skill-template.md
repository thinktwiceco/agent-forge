# Skill Template

Use this shape when writing `skills/<name>/SKILL.md`.

## Frontmatter checklist

- `name`: required, matches folder slug, lowercase letters/numbers/hyphens only, max 64 chars
- `description`: required, states what + when, max 1024 chars
- `version`: optional (e.g. `1.0.0`)
- Body must not be empty

## Template

````md
---
name: your-skill-name
description: Specific purpose plus clear trigger conditions for when the skill should be used.
version: 1.0.0
---

# Your Skill Title

## When to Use

Use this skill when:
- trigger 1
- trigger 2

Do not use this skill when:
- non-goal 1

## Workflow

1. First step.
2. Second step.
3. Third step.

## Output Format

Use this structure when reporting results:

```text
...
```

## References

- Read `references/examples.md` when examples are needed.
````

## Optional references layout

```text
skills/<name>/
  SKILL.md
  references/
    examples.md
    edge-cases.md
```

Keep reference paths flat when possible. The `skill` tool reads paths relative to `references/` only.
