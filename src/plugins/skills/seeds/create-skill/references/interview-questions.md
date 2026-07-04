# Interview Questions

Ask in batches. Skip any question the user already answered clearly.

## Batch 1 — Purpose

1. What problem should this skill solve for the agent?
2. What should the agent do differently after loading this skill?
3. What is out of scope (things this skill should **not** handle)?

## Batch 2 — Triggers

1. When should the agent load this skill? List concrete user phrases or task types.
2. When should the agent **not** load it?
3. Does this skill depend on specific tools, plugins, or repo paths?

## Batch 3 — Workflow

1. What are the ordered steps the agent should follow?
2. Are there required inputs, checks, or validation steps?
3. What output format or template should the agent produce?

## Batch 4 — References

1. Is there detail that should stay out of `SKILL.md` (examples, edge cases, long checklists)?
2. If yes, what reference files should exist under `references/`?

## Batch 5 — Naming

1. Propose a slug for `name` (e.g. `release-notes`, `deploy-checklist`).
2. Confirm the slug with the user before writing files.

## After the interview

Summarize answers in a short outline, then draft frontmatter and body before asking for final approval.
