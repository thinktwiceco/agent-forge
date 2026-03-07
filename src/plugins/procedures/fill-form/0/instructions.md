# Fill Form

Generic procedure to fill any web form. Execute every step in order.

If `todo_handler` is available, create a todo list now with one item per step (Steps 1–8). Mark each item complete as you finish it.

---

## Step 1 — State the goal

Write one sentence describing the form-fill goal.

If this procedure was called from another procedure, the goal and vault key names are already defined. Mark Step 1 complete and continue to Step 2.

---

## Step 2 — Get the interactive element tree

```json
{ "action": "get_content", "type": "interactive_tree", "settle_ms": 3000 }
```

List every input and button found: label, placeholder, or visible text.

Mark Step 2 complete.

---

## Step 3 — Map each input to a vault key name or plain value

For each input from Step 2, assign one of:

- **Vault key name pre-specified by the calling procedure**: record the key name as-is for use in Step 6.
- **No vault key specified**: match the input label to the best vault key name in the vault. Confirm the key name with the user.
- **Non-secret value**: ask the user to supply the plain text value.

Wait for confirmation before proceeding. Every input must have a vault key name or a confirmed plain value.

Mark Step 3 complete.

---

## Step 4 — Save the full page HTML

```json
{ "action": "save_content", "type": "html", "settle_ms": 3000, "strip": false }
```

The response contains:
- `Filename`: the saved file name
- `Path`: the relative path ready to pass to `fs`

Read the file using the exact `Path` from the response:

```json
{ "action": "read", "path": "<Path from save_content response>" }
```

Mark Step 4 complete.

---

## Step 5 — Find a CSS selector for each input and the submit button

Search the HTML from Step 4. Use the first matching attribute found, in this priority order:

1. `name` attribute
2. `id` attribute
3. `aria-label` attribute
4. Single stable class combined with `role` or another attribute
5. **Full class chain** — when all above fail (see below)

Record each selector with its role and mapped vault key name or plain value.

When using ripgrep to search the HTML file, pass each flag and value as separate array elements — e.g. `["-B", "2", "-A", "1"]`, never combined like `["-B 2", "-A 1"]`.

### When simple selectors match multiple elements

If a selector like `div[role="button"]` or a single class matches more than one element, build a **full class chain**:

1. Search the HTML for all elements matching the broad selector to count how many exist.
2. Identify the target element and copy its full `class` attribute value.
3. Construct a selector by chaining **all** classes with dots:
   `div.class-a.class-b.class-c…`
4. Verify with ripgrep that this chained selector is unique in the HTML before using it.

> **Note:** Some frameworks (e.g. React Native Web, Instagram) generate many atomic utility classes per element. These make single-class selectors ambiguous but the full chain is usually unique.

Also check element state before clicking — a button with `aria-disabled="true"` will not respond until its required preconditions are met (e.g. all required fields filled).

If a selector is not found: wait 2 s, re-run Step 4, and search again. If still not found, stop and report:
- **Expected**: selectors searched for and which inputs they map to
- **Found**: elements visible in the interactive tree and in the HTML
- **File**: relative path to the saved HTML

Mark Step 5 complete.

---

## Step 6 — Fill each input

Execute in order, one input at a time.

**Vault secret:**
```json
{ "action": "fill_secret", "selector": "<selector>", "resolveSecretVaultKey": "<vault key name>" }
```

**Plain value:**
```json
{ "action": "fill", "selector": "<selector>", "value": "<plain value>" }
```

Mark Step 6 complete.

---

## Step 7 — Click the submit button

```json
{ "action": "click", "selector": "<submit button selector>" }
```

Wait for the page to settle:

```json
{ "action": "get_content", "type": "text", "settle_ms": 3000, "strip": false }
```

If the click fails:

1. Re-run Step 4 to get fresh HTML and verify the element is present.
2. Check `aria-disabled` — if `"true"`, the button is not yet active. Ensure all required inputs are filled first.
3. If the selector matched multiple elements, apply the **full class chain** strategy from Step 5.
4. If still failing, stop and report:
   - **Expected**: selector used for the submit button
   - **Found**: interactive elements visible at the time of the click, and number of elements matched by the selector

Mark Step 7 complete.

---

## Step 8 — Assess the outcome

Use **only** the page content already returned by Step 7. Do **not** refresh, navigate, or fetch the page again — that would reset the session.

Assess based on the URL, visible text, and error messages from Step 7:

- **Goal reached**: announce success. Return to the calling procedure or stop if standalone.
- **Next step visible** (e.g. a second form or a verification prompt): reset Steps 2–7 as pending and restart from Step 2.
- **Unexpected state**: stop and report:
  - **Expected**: the page or state anticipated after form submission
  - **Found**: actual URL, visible text, and error messages from Step 7

Mark Step 8 complete.

---

## Rules

- To get HTML: use `save_content`, then read the file with `fs`. The `get_content` action truncates and is insufficient for selector discovery.
- Use the `Path` from the `save_content` response verbatim. Saved files are always under `web/`.
- Pass vault key names to `resolveSecretVaultKey`. The system resolves the actual value internally.
- If `save_content` or `fs read` returns an error, stop immediately and apply the global procedure rule.
