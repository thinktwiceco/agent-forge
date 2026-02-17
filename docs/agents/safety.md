# Safety Boundaries & Permissions

## ✅ Agents CAN Do Without Asking

- Read any file in the repository
- Run tests on specific files: `go test ./path/to/file_test.go`
- Run linter on specific files
- Format code: `go fmt ./...`
- Search codebase (grep, ripgrep, find)
- Analyze code structure and dependencies

## ⚠️ Agents MUST Ask Before

- Installing new dependencies (`go get`, updating `go.mod`)
- Making git commits or pushes
- Deleting files or directories
- Running full project builds (expensive/slow)
- Modifying `.env` or config files
- Running database migrations
- Making breaking changes to public APIs

## ❌ NEVER Do

- Force push to main/master
- Skip git hooks (--no-verify)
- Modify `VERSION` file (release process only)
- Delete test files without permission
- Remove validation or security checks
- Use `panic()` in library code (return errors)
- Ignore linter errors without reason
- Commit code that doesn't pass tests
