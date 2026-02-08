---
description: AGENT STEPS TO CREATE A NEW RELEASE
---

Create a new release following the project's release process and conventions.

## Prerequisites

1. Ensure all changes are committed and pushed to a release branch (`release/v{version}`)
2. Verify the `VERSION` file has been updated to the target release version
3. Confirm the release branch is based on `main` and up to date

## Phase 1: Pre-Release Validation

### Step 1: Check Current State
```bash
git status
git branch --show-current
cat VERSION
```

### Step 2: Run Linting
Execute linting to ensure code quality:
```bash
./scripts/lint.sh
```

**Reference**: See `.github/workflows/ci.yml` for linting configuration

**Fix any linting errors**:
- Format code: `go fmt ./...`
- Fix errcheck issues: Handle all error return values
- Fix staticcheck issues: Follow Go best practices
- Remove unused code

### Step 3: Run Tests
Execute unit tests to ensure everything passes:
```bash
./scripts/test.sh --unit
```

**Reference**: See `scripts/test.sh` for test execution details

**Critical**: All tests must pass before proceeding

### Step 4: Verify Commit Message Format
Check that commits follow Conventional Commits specification.

**Reference**: See `CONTRIBUTING.md` (lines 5-52) for commit message conventions

**Commit Message Format**:
```
<type>(<scope>): <subject>

[optional body]

[optional footer(s)]
```

**Valid Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`, `ci`, `build`

**Valid Scopes**: `agents`, `tools`, `llms`, `plugins`, `web`, `vector`, `git`, `os`, `coding`, `reasoning`, `core`, `config`, `docs`, `python-sdk`

**Rules**:
- Use lowercase for type and scope
- Subject should be imperative mood, no period at end
- Maximum 72 characters for subject line
- Maximum 100 characters per line in commit body
- Add `!` after scope for breaking changes

**Reference**: See `.agent/workflows/commit.md` for detailed commit guidelines

## Phase 2: Create Release Commit

### Step 5: Stage All Changes
```bash
git add VERSION
git add cmd/ src/ playground/ go.mod go.sum
# Include any new test files or documentation
git add src/**/*_test.go src/**/testing_helpers.go
```

**Note**: Exclude build artifacts, binaries, and database files:
- `thinktwice-agent` (binary)
- `*.db`, `*.db-shm`, `*.db-wal` (database files)
- `test-report.md` (if temporary)

### Step 6: Create Release Commit
Create commit with conventional commit format:
```bash
git commit -m "chore(release): prepare release v{VERSION}"
```

**Example**: `chore(release): prepare release v0.4.0`

**Reference**: See `CONTRIBUTING.md` for commit conventions

## Phase 3: Push and Create Pull Request

### Step 7: Push Release Branch
```bash
git push origin release/v{VERSION}
```

### Step 8: Create Pull Request
Create PR from `release/v{VERSION}` to `main` branch.

**PR Title**: `chore(release): prepare release v{VERSION}`

**PR Description Template**:
```markdown
## Release v{VERSION}

This PR prepares the release of version {VERSION}.

### Changes Summary
- **Version bump**: Updated VERSION file to {VERSION}
- [List major changes/features]

### Testing
- ✅ All unit tests passing
- ✅ Linting and formatting checks passing
- ✅ Code coverage maintained

### Files Changed
[Number] files changed, [insertions] insertions(+), [deletions] deletions(-)

### Next Steps
After merge:
1. Tag the release: `git tag v{VERSION}`
2. Push tag: `git push origin v{VERSION}`
3. GitHub Actions will automatically create the release
```

**PR Labels** (add via GitHub UI):
- Primary: `type:chore`
- Status: `status:ready`
- Optional area labels: `area:agents`, `area:tools`, etc.

**Reference**: See `CONTRIBUTING.md` (lines 54-95) for PR guidelines

## Phase 4: Tag and Release

### Step 9: Wait for PR Merge
Wait for the release PR to be reviewed and merged into `main`.

### Step 10: Switch to Main Branch
```bash
git checkout main
git pull origin main
```

### Step 11: Verify Version File
```bash
cat VERSION
# Should match the release version
```

### Step 12: Create Release Tag
```bash
git tag -a v{VERSION} -m "Release v{VERSION}"
```

**Example**: `git tag -a v0.4.0 -m "Release v0.4.0"`

### Step 13: Push Tag
```bash
git push origin v{VERSION}
```

**Critical**: Pushing the tag triggers the automated release workflow

## Phase 5: Automated Release Workflow

### Step 14: Monitor Release Workflow
The `.github/workflows/release.yml` workflow will automatically:

1. **Validate Version**: Check that `VERSION` file matches the tag
2. **Run Tests**: Execute `go test -v -race ./src/... ./cmd/...`
3. **Generate Changelog**: Parse commits since last tag using `metcalfc/changelog-generator@v4`
4. **Update CHANGELOG.md**: Add new version section with generated changelog
5. **Create Changelog PR**: Automatically create PR `changelog-v{VERSION}` for changelog update
6. **Create GitHub Release**: Create release with changelog content

**Reference**: See `.github/workflows/release.yml` for workflow details
**Reference**: See `docs/RELEASE_PIPELINE.md` (lines 233-240) for automated process

### Step 15: Review and Merge Changelog PR
- Review the automatically created changelog PR
- Merge it to complete the release process

## Verification Checklist

Before starting release process, verify:
- [ ] All tests pass (`./scripts/test.sh --unit`)
- [ ] Linting passes (`./scripts/lint.sh`)
- [ ] Code is formatted (`go fmt ./...`)
- [ ] VERSION file updated correctly
- [ ] Commit messages follow Conventional Commits
- [ ] Release branch is up to date with main

After creating tag, verify:
- [ ] Tag pushed successfully
- [ ] Release workflow triggered (check GitHub Actions)
- [ ] Changelog PR created automatically
- [ ] GitHub Release created with correct content

## Documentation References

- **Commit Conventions**: `CONTRIBUTING.md` (lines 5-52)
- **PR Guidelines**: `CONTRIBUTING.md` (lines 54-95)
- **Release Process**: `docs/RELEASE_PIPELINE.md`
- **Commit Workflow**: `.agent/workflows/commit.md`
- **CI Configuration**: `.github/workflows/ci.yml`
- **Release Workflow**: `.github/workflows/release.yml`
- **Commitlint Config**: `.commitlintrc.json`

## Common Issues and Solutions

### Linting Errors
- **Formatting**: Run `go fmt ./...`
- **Errcheck**: Handle all error return values (use `_ = func()` if intentionally ignored)
- **Staticcheck**: Follow Go best practices, fix error string formatting

### Test Failures
- Fix failing tests before proceeding
- Ensure test coverage is maintained
- Check for race conditions with `-race` flag

### Commit Message Validation
- Use `wagoid/commitlint-github-action@v5` (configured in CI)
- Reference `.commitlintrc.json` for validation rules
- Ensure subject ≤ 72 characters, body lines ≤ 100 characters

### Release Workflow Issues
- Check GitHub Actions logs for errors
- Verify `GH_PAT` secret is configured
- Ensure version tag format matches `v*.*.*` pattern

## CRITICAL REMINDERS

- **DO NOT** commit binaries, database files, or temporary files
- **DO NOT** exceed commit message character limits
- **DO NOT** skip linting or testing steps
- **ALWAYS** verify VERSION file matches tag before pushing tag
- **ALWAYS** wait for PR merge before creating tag
- **ALWAYS** push tag from `main` branch after merge
