# AGENT STEPS TO COMMIT CHANGES

Commit the current changes following Conventional Commits specification.

## Actions

1. Check git status to see what files have changed
2. Analyze the changes to determine:
   - Type: feat, fix, docs, style, refactor, perf, test, chore, ci, build
   - Scope: agents, tools, llms, plugins, web, vector, git, os, coding, reasoning, core, config, docs, python-sdk
   - Subject: concise description of the change
3. Check if changes contain breaking changes (API changes, signature changes, etc.)
4. Generate commit message in format: `<type>(<scope>): <subject>`
5. Add `!` after scope if breaking change detected
6. Include body if change needs explanation (ensure each body line is ≤100 characters)
7. Execute: `git add -A`
8. Execute: `git commit -m "<message>"`

## Rules

- Use lowercase for type and scope
- Subject should be imperative mood, no period at end
- Maximum 72 characters for subject line
- Maximum 100 characters per line in commit body (wrap long lines with proper indentation)
- If breaking change, add `!` after scope or include `BREAKING CHANGE:` in footer
- Scope is optional but recommended

## Test and linting

- Before committing execute tests and lint using:
   - scripts/lint.sh
   - scripts/test.sh

## CRITICAL

- DO NOT EXCEED THE CHARACTER LIMIT