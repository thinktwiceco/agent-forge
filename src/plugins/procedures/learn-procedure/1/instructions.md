# Phase 1: Codify (transform document into procedure)

Review the learning document and transform it into a reusable procedure.

## 1. Review the learning document

Read `learning-<name>.md` (the document created in Phase 0). Extract:

- The goal and context
- Each step that succeeded (what worked)
- Verification criteria per step
- Any failure patterns and remedies

## 2. Create the procedure

Create a new procedure **inside** `procedures/` with a slugified name (e.g. `procedures/my-learned-task`). Procedures MUST be created under the `procedures/` folder.

1. **manifest.yaml** — `name` and `description` derived from the goal
2. **Phase folders** — one folder per step: `0/`, `1/`, `2/`, …
3. **instructions.md** in each folder — machine-first instructions:
   - Clear, unambiguous instructions an agent can execute
   - Include verification (what success looks like)
   - Include remedies for known failure modes from the learning doc

Use the same structure as create-procedure: `procedures/<slug>/<step-number>/instructions.md`.

## 3. Done

The procedure is now available. Use `start_procedure` with the new procedure name to run it.
