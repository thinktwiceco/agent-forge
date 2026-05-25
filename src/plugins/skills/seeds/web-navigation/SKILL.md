---
name: web-navigation
description: Guide browser-based navigation with the web tool. Use before navigating pages, inspecting structure, clicking controls, filling forms, logging in, uploading files, or recovering from browser-state issues.
version: 1.0.0
---

# Web Navigation

## When to Use

Use this skill before any task that requires browser navigation or interaction with the `web` tool, especially for dynamic pages, login flows, form filling, uploads, and page inspection.

Do not use this skill for plain static retrievals that only need `fetch`.

## Workflow

1. Load `references/navigation-basics.md` before the first browser action.
2. If the task includes forms, auth, or file inputs, also load `references/forms-and-auth.md`.
3. If the page is unstable or the flow breaks, load `references/failures-and-recovery.md`.
4. Follow the referenced playbooks instead of improvising ad hoc browser steps.

## References

- `references/navigation-basics.md` for sessions, inspection, `get_snapshot`, and the default navigation loop.
- `references/forms-and-auth.md` for selectors, `fill`, `fill_secret`, submit flows, and `upload_file`.
- `references/failures-and-recovery.md` for settle issues, hidden elements, wrong auth state, MFA, and captcha recovery.
