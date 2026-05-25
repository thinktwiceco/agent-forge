# Navigation Basics

Use this default loop for most browser tasks:

1. `open_session`
2. `navigate`
3. `get_snapshot`
4. If needed, `get_content` with `type: "interactive_tree"`
5. Interact with `click`, `fill`, `fill_secret`, or `upload_file`
6. Inspect again after each major UI change
7. `close_session` when done

Prefer named sessions such as `login`, `checkout`, or `research` so cookies and history stay isolated and reusable.

Use `get_snapshot` when you need the semantic structure of the current page. It returns the Chrome accessibility tree as JSON and is the best first inspection step on React, Next.js, Vue, Angular, and other dynamic pages.

Use `get_content` with `type: "interactive_tree"` when you need a compact list of visible controls. Use `type: "text"` only when readable page text matters more than structure.

Increase `settle_ms` on slow or JS-heavy pages. A range of `1500` to `3000` is often better than the default when content appears after initial load.

Example:

```json
{"action":"open_session","session":"main"}
{"action":"navigate","session":"main","url":"https://example.com","settle_ms":1500}
{"action":"get_snapshot","session":"main","settle_ms":1500}
{"action":"get_content","session":"main","type":"interactive_tree"}
```
