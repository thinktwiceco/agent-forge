# Step 0: Procedure name and description

Ask the user for the **name** and **description** of the procedure they want to create.

Then:
1. Create a new folder **inside** `procedures/` with a slugified name (e.g. `procedures/my-procedure`). Procedures MUST be created under the `procedures/` folder — never at the working dir root or elsewhere.
2. Create `manifest.yaml` inside that folder with the `name` and `description` fields filled from the user's input

When done, use the procedure tool with action `next_step` to continue.
