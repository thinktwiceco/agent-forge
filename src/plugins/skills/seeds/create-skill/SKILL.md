---
name: create-skill
description: Guide the user through authoring a new SKILL.md package with interview questions and a validated write at the end. Use when the user asks to create, add, author, or install a custom skill.
version: 1.0.0
---

# Create Skill

## When to Use

Use this skill when the user wants a new agent skill: a reusable `skills/<slug>/SKILL.md` package with clear triggers, a concise body, and optional `references/`.

Do not use this skill to edit unrelated code or to install an existing remote skill without authoring new content.

## Workflow

1. **Confirm scope.** Restate what the skill should help the agent do. If the user already gave enough detail, skip redundant questions.
2. **Interview.** Load `references/interview-questions.md`. Ask in small batches (2–4 questions). Wait for answers before the next batch.
3. **Propose slug.** Suggest `name` (lowercase letters, numbers, hyphens only, max 64 chars). Confirm with the user before writing files.
4. **Draft metadata.** Write `description` (what + when, max 1024 chars) and a short `## When to Use` summary.
5. **Draft body.** Load `references/skill-template.md`. Keep `SKILL.md` short; push long examples or edge cases into `references/`.
6. **Review.** Show the draft paths and frontmatter. Ask for approval or edits.
7. **Write.** Create `skills/<name>/SKILL.md` and any `references/*.md` files in the agent working directory.
8. **Verify.** Run `skill` with `list_skills`, then `load_skill` for the new name. Fix validation errors before finishing.

## Rules

- Never write invalid frontmatter; invalid skills are skipped at discovery.
- Prefer one default workflow in the skill body, not many equal options.
- Do not add markdown docs outside `skills/<name>/` unless the user explicitly asks.
- If the skill folder already exists, ask whether to overwrite or pick a new name.

## References

- `references/interview-questions.md` — question batches for the user interview.
- `references/skill-template.md` — output shape and frontmatter checklist.
