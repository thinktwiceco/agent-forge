# Failures And Recovery

If the page did not settle or content looks partial:

- retry with a larger `settle_ms`
- use `refresh`
- inspect again with `get_snapshot`

If a selector is not found:

- confirm you are in the correct session
- inspect again with `get_snapshot`
- use `get_content` with `type: "interactive_tree"`
- switch to a more semantic selector

If an element exists but is not actionable:

- keep `wait_visible: true`
- open the modal, tab, or accordion that contains it
- raise `timeout` for slow pages

If the browser is in the wrong auth state:

- confirm the session name
- close the session and reopen it fresh

If the flow reaches MFA, captcha, or a verification challenge:

- stop treating the task as a normal login completion
- inspect the new state carefully
- identify whether a separate step or manual input is required

Example reset:

```json
{"action":"close_session","session":"login"}
{"action":"open_session","session":"login","headless":false}
```
