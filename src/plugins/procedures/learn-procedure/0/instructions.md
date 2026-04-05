# Phase 0: Learning (trial and error)

Learn the goal through trial and error and record the experience in a document.

## 1. Ask: What is the goal?

Ask the user: **What is the goal?** Capture the goal clearly.

## 2. Break down the goal

If the main agent has **spawn_subagent** enabled, spawn a short-lived subagent with only the tools it needs. Pass a clear `prompt` and a minimal `tools` list. Ask it to break down the goal into:

- **Steps** — ordered list of what to do
- **Tools** — which tools are needed for each step
- **Verification** — for each step, describe what success looks like (how to verify the step succeeded)

## 3. Check tools

Ask the user: **Do you have all the tools?** Use `get_tools` (meta tool) to list available tools. Compare with the tools required by the breakdown. If any are missing, inform the user and discuss alternatives before proceeding.

## 4. Execute each step

For **each step** from the breakdown:

1. **Use the todo_handler** — create or update a todo list to track tasks for this step
2. **Execute** the step using the appropriate tools
3. **Record** in `learning-<name>.md` (use a slugified name from the goal):
   - What you did
   - Success or failure
   - What worked, what did not
   - Verification outcome
4. **Ask for human feedback** — after each step, ask the user for feedback before continuing

Keep `learning-<name>.md` in the working directory. Structure it with clear sections per step.

## 5. Advance to codify

When all steps are complete and documented, use the procedure tool with action `next_step` to move to Phase 1 (Codify).
