# Web Tool How-To

Repeatable steps for using `web_browser` to navigate pages, find login forms, fill credentials safely, and recover from common failures.

## 1. Before You Start

Make sure the agent has:

- the `web` tool enabled
- a valid `working_dir`
- the `vault` plugin enabled if passwords or tokens are involved

Recommended config:

```yaml
agent:
  working_dir: "${AGENT_WORKING_DIR}"
  tools:
    - name: web
  plugins:
    - "vault"
```

Common pitfalls:

- `fill_secret` fails because the `vault` plugin is not enabled.
- `save_content` has nowhere to write because `working_dir` is missing.
- Browser behavior is confusing because the tool is running headless by default.

Issue resolution:

- Enable `vault` before attempting any password flow.
- Ensure `working_dir` is set in config.
- If visual debugging is needed, start a session with `headless: false`.

## 2. Default Navigation Loop

Use this sequence for most websites:

1. `open_session`
2. `navigate`
3. inspect with `get_snapshot`
4. if needed, inspect with `get_content` and `type: "interactive_tree"`
5. interact with `click`, `fill`, `fill_secret`, or `upload_file`
6. inspect again after each important UI change
7. `close_session` when done

Example:

```json
{"action":"open_session","session":"main"}
{"action":"navigate","session":"main","url":"https://example.com"}
{"action":"get_snapshot","session":"main"}
```

Why this works:

- `navigate` waits for the page to finish loading
- the tool also waits for network idle using `settle_ms`
- `get_snapshot` is the most reliable first inspection method on modern SPAs

Common pitfalls:

- Clicking before the page finishes rendering.
- Mixing up browser sessions and losing cookies or state.
- Reading plain text too early on a JS-heavy app.

Issue resolution:

- Increase `settle_ms` to `1500` to `3000` on slow pages.
- Reuse one session name consistently for the same task.
- Prefer `get_snapshot` before `get_content` on dynamic apps.

## 3. Open And Reuse Sessions

Open a named session first:

```json
{"action":"open_session","session":"login","headless":false}
```

Then reuse that same `session` value on every later action:

```json
{"action":"navigate","session":"login","url":"https://service.example.com"}
```

Use named sessions when:

- one workflow needs its own cookies and auth state
- you want to keep a login flow separate from general browsing
- you need to resume work later in the same conversation

Common pitfalls:

- Omitting `session` later and accidentally using the default session.
- Assuming a session is still alive after several idle minutes.

Issue resolution:

- Pick stable names like `login`, `checkout`, or `research`.
- Use `list_sessions` if you need to see active sessions.
- If a session has gone idle, call `open_session` again with the same name.

## 4. Inspect The Page Properly

The tool has four main inspection modes:

- `get_snapshot`
  Best for structure, controls, and SPA pages.
- `get_content` with `type: "interactive_tree"`
  Best for a compact list of visible buttons, inputs, and links.
- `get_content` with `type: "text"`
  Best for readable page text.
- `save_content`
  Best when the result is long and should be written to disk for later review.

Examples:

```json
{"action":"get_snapshot","session":"login"}
{"action":"get_content","session":"login","type":"interactive_tree"}
{"action":"get_content","session":"login","type":"text","strip":false}
```

Use `strip: false` when navigation elements, menus, tabs, or buttons matter.

Common pitfalls:

- Using stripped text when trying to discover buttons.
- Using plain text on a SPA and missing the important UI.
- Forgetting that `save_content` uses the current page only.

Issue resolution:

- For navigation tasks, start with `get_snapshot`.
- Use `interactive_tree` when you need fast control discovery.
- Call `navigate` first, then `save_content`.

## 5. Find Login Forms

Use this repeatable sequence:

1. navigate to the login page or homepage
2. inspect with `get_snapshot`
3. inspect with `get_content` and `type: "interactive_tree"`
4. identify likely selectors for:
   - username or email field
   - password field
   - submit button
   - login modal trigger if the form is hidden

Useful selector patterns:

- `input[type='email']`
- `input[name='email']`
- `input[name='username']`
- `input[autocomplete='username']`
- `input[type='password']`
- `input[autocomplete='current-password']`
- `button[type='submit']`

Common login button text:

- `Sign in`
- `Log in`
- `Continue`
- `Next`
- `Submit`

Example:

```json
{"action":"navigate","session":"login","url":"https://service.example.com/login","settle_ms":1500}
{"action":"get_snapshot","session":"login","settle_ms":1500}
{"action":"get_content","session":"login","type":"interactive_tree"}
```

Common pitfalls:

- The page only shows a login button, not the actual form.
- The flow is multi-step and only reveals the password field later.
- The form is inside a modal, shadow DOM, or same-origin iframe.

Issue resolution:

- If the form is hidden, click the login trigger first and inspect again.
- If only email is visible, assume a two-step login and continue to the next page.
- Re-run `get_snapshot` after every major click so the current UI is clear.

## 6. Fill Username Or Email

Use `fill` for non-secret values such as username, email, search text, or codes.

Example:

```json
{"action":"fill","session":"login","selector":"input[type='email']","value":"user@example.com"}
```

Prefer semantic selectors over generated classes:

- `type`
- `name`
- `placeholder`
- `autocomplete`
- `aria-label`

Examples:

```json
{"action":"fill","session":"login","selector":"input[name='email']","value":"user@example.com"}
{"action":"fill","session":"login","selector":"input[autocomplete='username']","value":"user@example.com"}
```

Common pitfalls:

- Using unstable CSS classes from a minified frontend build.
- Filling the wrong field when there are multiple similar inputs.
- Trying to fill an input that is present but still hidden.

Issue resolution:

- Inspect first, then choose the most semantic selector available.
- Keep `wait_visible: true` unless you know the element is safely actionable.
- If the field still behaves oddly, inspect again to confirm the selector matches the intended input.

## 7. Store And Use Passwords With Vault

Never put plaintext passwords into normal tool arguments when `vault` is available.

Recommended sequence:

1. store the secret with `saveSecret`
2. verify the key with `listSecrets`
3. use `fill_secret` with the vault key name

Example secret workflow:

```json
{"tool":"saveSecret","key":"service-password","value":"..."}
{"tool":"listSecrets"}
```

Then fill the password field:

```json
{"action":"fill_secret","session":"login","selector":"input[type='password']","resolveSecretVaultKey":"service-password"}
```

Important:

- `resolveSecretVaultKey` is the vault key name
- it is not the password itself

Wrong:

```json
{"action":"fill_secret","selector":"input[type='password']","resolveSecretVaultKey":"hunter2"}
```

Right:

```json
{"action":"fill_secret","selector":"input[type='password']","resolveSecretVaultKey":"service-password"}
```

Common pitfalls:

- Passing plaintext into `resolveSecretVaultKey`.
- Forgetting to enable the `vault` plugin.
- Using a vault key that does not exist.

Issue resolution:

- Run `listSecrets` before filling if there is any doubt.
- If the key is missing, store it first with `saveSecret`.
- If `fill_secret` says no vault exists, enable the `vault` plugin and restart the agent.

## 8. Submit The Login Form

After filling the required fields:

1. inspect the page again if needed
2. click the most stable submit selector
3. inspect the page immediately after submit

Example:

```json
{"action":"click","session":"login","selector":"button[type='submit']","timeout":60}
{"action":"get_snapshot","session":"login","settle_ms":1500}
```

If the login is multi-step:

1. fill email
2. click `Next` or `Continue`
3. inspect again
4. fill password
5. click final submit

Common pitfalls:

- Clicking the wrong button when multiple actions are visible.
- The submit button stays disabled due to validation.
- The page asks for MFA or a captcha after submission.

Issue resolution:

- Inspect before and after the click so the current state is explicit.
- If the button is disabled, look for missing fields or required checkboxes.
- If MFA appears, stop treating it as a normal login completion and handle that new step explicitly.

## 9. Verify Login Success Or Failure

Do not assume login succeeded just because the page changed.

Check for:

- a dashboard URL
- a profile avatar or account menu
- a logout button
- an inline error message
- an MFA prompt
- a captcha or verification interstitial

Useful commands:

```json
{"action":"get_snapshot","session":"login","settle_ms":2000}
{"action":"get_content","session":"login","type":"interactive_tree"}
{"action":"save_content","session":"login","type":"text"}
```

Common failure text:

- `invalid password`
- `incorrect code`
- `try again`
- `verify your identity`
- `too many attempts`

Common pitfalls:

- Only checking the page title.
- Missing inline error text after submit.
- Failing to save the page when the result is ambiguous.

Issue resolution:

- Always inspect after submit.
- If the result is unclear, save the content and review the saved file.
- Check both URL changes and visible UI signals.

## 10. Use `get_snapshot` vs `get_content` Correctly

Use `get_snapshot` when:

- the app is React, Vue, Next.js, Angular, or similarly dynamic
- you need a semantic tree of controls
- body text looks incomplete

Use `get_content` when:

- you need readable text
- you want HTML
- you want a compact interactive list with `type: "interactive_tree"`

Use `save_content` when:

- the content is long
- you want a stable file artifact under `working_dir/web`

Common pitfalls:

- Expecting plain text extraction to describe the full UI on a SPA.
- Saving content before navigation finishes.
- Using HTML extraction when only visible structure is needed.

Issue resolution:

- Start with `get_snapshot`, then move to other modes only if needed.
- Increase `settle_ms` on dynamic sites.
- Use `interactive_tree` when the next action depends on visible controls.

## 11. Recover From Common Failures

### Page did not settle

Symptoms:

- `navigate` or `get_snapshot` times out
- content looks partial

Resolution:

- retry with larger `settle_ms`
- use `refresh`
- confirm the page is not continuously polling forever

Example:

```json
{"action":"navigate","session":"main","url":"https://example.com","settle_ms":2500}
```

### Selector not found

Symptoms:

- `click` or `fill` says the element is missing

Resolution:

- inspect again with `get_snapshot`
- use `interactive_tree`
- switch to a more semantic selector
- confirm the current session is the right one

### Element not visible

Symptoms:

- the selector exists but interaction still fails

Resolution:

- keep `wait_visible: true`
- trigger the enclosing modal, tab, or accordion first
- increase `timeout`

### Wrong auth state

Symptoms:

- page behaves as if already logged in
- site shows a different account than expected

Resolution:

- confirm which named session you are using
- close the session and reopen a fresh one

Example:

```json
{"action":"close_session","session":"login"}
{"action":"open_session","session":"login"}
```

### Unexpected MFA, captcha, or challenge screen

Symptoms:

- password submit does not reach the app
- page shows verification or challenge UI

Resolution:

- stop assuming normal login is complete
- inspect the new state carefully
- identify whether manual user input or a separate automation step is required

## 12. Static Pages vs Real Browser Pages

Use `fetch` for:

- static documentation
- simple HTML pages
- JSON or XML APIs

Use browser actions for:

- login flows
- SPAs
- anything requiring clicks, typing, cookies, or client-side rendering

Examples:

```json
{"action":"fetch","url":"https://example.com/docs"}
{"action":"navigate","session":"main","url":"https://app.example.com"}
```

Common pitfalls:

- Trying to use `fetch` for authenticated or interactive pages.
- Assuming `fetch` sees the same content as a browser session.

Issue resolution:

- If the page needs JavaScript, state, or interaction, use `navigate`.
- Reserve `fetch` for lightweight retrieval only.

## 13. Recommended Login Playbook

This is the most reusable login recipe:

1. `open_session`
2. `navigate` to the login page
3. `get_snapshot`
4. `get_content` with `type: "interactive_tree"`
5. `fill` username or email
6. if needed, `click` next or continue
7. `get_snapshot` again
8. `fill_secret` password
9. `click` submit
10. `get_snapshot` again
11. verify success, failure, or MFA

Example:

```json
{"action":"open_session","session":"login","headless":false}
{"action":"navigate","session":"login","url":"https://service.example.com/login","settle_ms":1500}
{"action":"get_snapshot","session":"login"}
{"action":"fill","session":"login","selector":"input[type='email']","value":"user@example.com"}
{"action":"click","session":"login","selector":"button[type='submit']"}
{"action":"get_snapshot","session":"login","settle_ms":1500}
{"action":"fill_secret","session":"login","selector":"input[type='password']","resolveSecretVaultKey":"service-password"}
{"action":"click","session":"login","selector":"button[type='submit']","timeout":60}
{"action":"get_snapshot","session":"login","settle_ms":2000}
```

Common pitfalls:

- Treating login as a single click instead of a stateful flow.
- Filling password before the correct password field is visible.
- Failing to inspect after each transition.

Issue resolution:

- Treat every login as a sequence of smaller checkpoints.
- Re-inspect after each click.
- Prefer semantic selectors and vault-backed secrets.

## 14. Cleanup

When the workflow is complete:

- close sessions you no longer need
- keep saved content paths if they were produced

Example:

```json
{"action":"close_session","session":"login"}
```

Common pitfalls:

- Leaving many sessions open during long conversations.
- Forgetting which saved file corresponds to which page state.

Issue resolution:

- Close sessions when they are no longer needed.
- Report or record returned save paths immediately.
