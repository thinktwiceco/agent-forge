# Release Pipeline Plan

This document outlines the release pipeline strategy for agentForge, including semantic versioning, commit conventions, PR management, changelog generation, and automated release workflows.

## 1. Semantic Versioning

We follow [Semantic Versioning 2.0.0](https://semver.org/) with the format: `MAJOR.MINOR.PATCH`

- **MAJOR** (X.0.0): Breaking changes that require users to modify their code
- **MINOR** (0.X.0): New features added in a backward-compatible manner
- **PATCH** (0.0.X): Backward-compatible bug fixes

### Version Management

- Current version is tracked in `VERSION` file at repository root
- Version is also embedded in Go code via build flags (optional)
- Git tags follow the format: `v{MAJOR}.{MINOR}.{PATCH}` (e.g., `v1.2.3`)

## 2. Commit Message Conventions

We use [Conventional Commits](https://www.conventionalcommits.org/) specification:

### Format
```
<type>(<scope>): <subject>

[optional body]

[optional footer(s)]
```

### Types

- **feat**: New feature (triggers MINOR version bump)
- **fix**: Bug fix (triggers PATCH version bump)
- **docs**: Documentation changes only
- **style**: Code style changes (formatting, missing semicolons, etc.)
- **refactor**: Code refactoring without feature changes or bug fixes
- **perf**: Performance improvements
- **test**: Adding or updating tests
- **chore**: Maintenance tasks (dependencies, build config, etc.)
- **ci**: CI/CD pipeline changes
- **build**: Build system or external dependencies changes

### Breaking Changes

- Use `!` after type/scope: `feat(api)!: change method signature`
- Or include `BREAKING CHANGE:` in footer:
  ```
  feat(api): add new parameter
  
  BREAKING CHANGE: Method signature changed, old code will break
  ```

### Examples

```
feat(agents): add support for custom system prompts
fix(web): resolve browser session cleanup issue
docs(readme): update installation instructions
refactor(llms): simplify LLM factory pattern
feat(tools)!: change tool validation API
```

## 3. Pull Request Labeling System

PRs must be labeled to categorize changes for changelog generation:

### Primary Labels (Required)

- `type:feature` - New features
- `type:bugfix` - Bug fixes
- `type:breaking` - Breaking changes
- `type:documentation` - Documentation updates
- `type:refactor` - Code refactoring
- `type:performance` - Performance improvements
- `type:chore` - Maintenance tasks

### Secondary Labels (Optional)

- `area:agents` - Changes to agent system
- `area:tools` - Changes to tool system
- `area:llms` - Changes to LLM integration
- `area:plugins` - Changes to plugin system
- `area:web` - Changes to web agent
- `area:vector` - Changes to vector agent
- `area:git` - Changes to git agent
- `area:os` - Changes to OS agent
- `area:coding` - Changes to coding agent
- `area:reasoning` - Changes to reasoning agent

### Status Labels

- `status:ready` - PR is ready for review
- `status:blocked` - PR is blocked
- `status:wip` - Work in progress

## 4. Branch Strategy

### Main Branches

- `main` - Production-ready code (protected)
- `develop` - Integration branch for features (optional, can use main directly)

### Feature Branches

- `feature/<name>` - New features
- `fix/<name>` - Bug fixes
- `docs/<name>` - Documentation updates
- `refactor/<name>` - Refactoring

### Release Branches

- `release/v{version}` - Release preparation branch
  - Created from `main` when ready to release
  - Only bug fixes and release-related changes
  - Merged back to `main` and tagged

## 5. Changelog Generation

### Automated Changelog

Changelog is auto-generated from:
1. Conventional commit messages since last release
2. PR labels and titles
3. PR descriptions (for detailed notes)

### Changelog Format

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- New features not yet released

### Changed
- Changes in existing functionality

### Deprecated
- Soon-to-be removed features

### Removed
- Removed features

### Fixed
- Bug fixes

### Security
- Security fixes

## [1.2.3] - 2024-01-15

### Added
- feat(agents): support for custom system prompts
- feat(tools): new validation framework

### Changed
- refactor(llms): simplified factory pattern

### Fixed
- fix(web): browser session cleanup issue

### Breaking Changes
- feat(tools)!: tool validation API changed
```

### Changelog Sections

- **Added**: New features
- **Changed**: Changes in existing functionality
- **Deprecated**: Features that will be removed
- **Removed**: Removed features
- **Fixed**: Bug fixes
- **Security**: Security vulnerabilities fixed

## 6. Release Workflow

### Manual Release Process

1. **Prepare Release Branch**
   ```bash
   git checkout main
   git pull origin main
   git checkout -b release/v1.2.3
   ```

2. **Update Version**
   - Update `VERSION` file
   - Update any version references in code/docs

3. **Generate Changelog**
   ```bash
   # Automated tool will:
   # - Parse commits since last tag
   # - Parse merged PRs
   # - Generate CHANGELOG.md entries
   ```

4. **Final Checks**
   - Run tests: `go test ./...`
   - Run linters
   - Review changelog
   - Update documentation if needed

5. **Create Release PR**
   - PR from `release/v1.2.3` to `main`
   - Title: `Release v1.2.3`
   - Include changelog in PR description
   - Label with `type:chore` and `status:ready`

6. **Merge and Tag**
   ```bash
   git checkout main
   git merge release/v1.2.3
   git tag -a v1.2.3 -m "Release v1.2.3"
   git push origin main
   git push origin v1.2.3
   ```

7. **Create GitHub Release**
   - Use GitHub Releases UI or API
   - Title: `v1.2.3`
   - Description: Copy from CHANGELOG.md for this version
   - Attach release notes

### Automated Release Process (Recommended)

Triggered by:
- Pushing a tag matching `v*.*.*`
- Creating a GitHub Release via UI
- Manual workflow dispatch

Steps:
1. Validate version format
2. Check that version matches `VERSION` file
3. Run full test suite
4. Generate changelog from commits/PRs
5. Create/update CHANGELOG.md
6. Build artifacts (if applicable)
7. Create GitHub Release with changelog
8. Publish to package registry (if applicable)

## 7. GitHub Actions Workflow

### Workflow Files Structure

```
.github/
├── workflows/
│   ├── ci.yml              # Continuous Integration
│   ├── release.yml         # Release automation
│   ├── changelog.yml       # Changelog generation
│   └── version-check.yml   # Version validation
```

### CI Workflow (`ci.yml`)

- Runs on: push to `main`, PRs
- Steps:
  - Checkout code
  - Setup Go
  - Run tests
  - Run linters (golangci-lint)
  - Check commit message format (on PRs)

### Changelog Workflow (`changelog.yml`)

- Runs on: PR merge to `main`
- Steps:
  - Generate changelog entry for merged PR
  - Update `CHANGELOG.md` (prepend to Unreleased section)
  - Create PR with changelog update (optional)

### Release Workflow (`release.yml`)

- Triggers:
  - Push tag matching `v*.*.*`
  - Manual workflow dispatch
  - GitHub Release creation
- Steps:
  1. Validate tag format
  2. Extract version from tag
  3. Verify `VERSION` file matches tag
  4. Run full test suite
  5. Generate complete changelog
  6. Update `CHANGELOG.md` with version section
  7. Commit changelog update
  8. Create GitHub Release with:
     - Release notes from changelog
     - Attached artifacts (if any)

### Version Check Workflow (`version-check.yml`)

- Runs on: PRs
- Steps:
  - Check if `VERSION` file was modified
  - If modified, verify format matches semantic versioning
  - Check if version was incremented appropriately based on PR labels

## 8. Tools and Dependencies

### Recommended Tools

- **goreleaser** (optional): For building and releasing Go binaries
- **git-chglog**: For generating changelogs from git history
- **semantic-release** (or custom script): For automated versioning
- **conventional-changelog**: For changelog generation
- **golangci-lint**: For code quality checks

### Alternative: Custom Scripts

If preferred, we can create custom Go scripts:
- `scripts/changelog/main.go` - Changelog generator
- `scripts/release/main.go` - Release automation
- `scripts/version/main.go` - Version management

## 9. Release Checklist

### Pre-Release

- [ ] All tests passing
- [ ] No critical bugs in `main`
- [ ] Documentation updated
- [ ] Changelog reviewed
- [ ] Version number updated
- [ ] Release branch created (if using)

### Release

- [ ] Release PR created and reviewed
- [ ] Changelog finalized
- [ ] Tag created and pushed
- [ ] GitHub Release created
- [ ] Release notes published

### Post-Release

- [ ] Verify release appears on GitHub
- [ ] Update any external documentation
- [ ] Announce release (if applicable)
- [ ] Monitor for issues

## 10. Version Bump Rules

### Automatic Version Bumping

Based on commit types and PR labels:

- **MAJOR**: 
  - Commit with `!` or `BREAKING CHANGE:`
  - PR labeled `type:breaking`
  
- **MINOR**:
  - Commit type `feat`
  - PR labeled `type:feature`
  
- **PATCH**:
  - Commit type `fix`
  - PR labeled `type:bugfix`

### Manual Override

Maintainers can manually specify version in release PR or tag.

## 11. Consumer Version Selection Guide

This section explains how consumers of the `agentForge` package discover, evaluate, and select which version to use.

### How Go Modules Work with Versions

Go modules use semantic versioning tags from the Git repository. When you tag a release with `v1.2.3`, Go automatically makes it available to consumers.

### Discovering Available Versions

#### 1. **GitHub Releases Page**
The primary source for consumers to discover versions:
- URL: `https://github.com/thinktwice/agentForge/releases`
- Shows all tagged releases with release notes
- Each release includes:
  - Version number (e.g., `v1.2.3`)
  - Release date
  - Changelog excerpt
  - Installation command

#### 2. **Go Module Proxy**
Go automatically queries the module proxy:
```bash
# List available versions
go list -m -versions github.com/thinktwice/agentForge
```

#### 3. **Git Tags**
Direct inspection of Git tags:
```bash
git ls-remote --tags https://github.com/thinktwice/agentForge.git
```

### Version Selection Strategies

#### Strategy 1: Latest Stable (Recommended for New Projects)

**Use Case**: Starting a new project or want the latest features

```bash
# Get latest version
go get github.com/thinktwice/agentForge@latest

# Or in go.mod
require github.com/thinktwice/agentForge latest
```

**Decision Process**:
1. Check GitHub Releases for the latest version
2. Review CHANGELOG.md for recent changes
3. Verify no breaking changes if upgrading from existing version
4. Check release notes for known issues or deprecations

#### Strategy 2: Specific Version (Recommended for Production)

**Use Case**: Production applications requiring stability

```bash
# Get specific version
go get github.com/thinktwice/agentForge@v1.2.3

# Or in go.mod
require github.com/thinktwice/agentForge v1.2.3
```

**Decision Process**:
1. Review CHANGELOG.md to understand what's in each version
2. Check GitHub Release notes for detailed changes
3. Look for:
   - **Breaking Changes** section (MAJOR version changes)
   - **Added** section (new features you might need)
   - **Fixed** section (bug fixes relevant to your use case)
   - **Security** section (critical security patches)

#### Strategy 3: Version Range (Flexible Updates)

**Use Case**: Allow patch and minor updates, but lock major versions

```go
// In go.mod
require (
    github.com/thinktwice/agentForge v1.2.0
)

// Go will automatically use latest v1.x.x
// But won't upgrade to v2.x.x
```

**Decision Process**:
1. Choose a base version (e.g., `v1.2.0`)
2. Go will automatically use latest `v1.x.x` (e.g., `v1.2.5`)
3. Review changelog before major version upgrades
4. Test after `go mod tidy` to ensure compatibility

#### Strategy 4: Pre-release Versions (Early Access)

**Use Case**: Testing new features or providing feedback

```bash
# Get pre-release version
go get github.com/thinktwice/agentForge@v2.0.0-beta.1
go get github.com/thinktwice/agentForge@v2.0.0-rc.1
```

**Decision Process**:
1. Check if pre-release addresses specific needs
2. Understand that pre-releases may have breaking changes
3. Be prepared to update frequently
4. Report issues to help improve final release

### Using the Changelog for Decision Making

The `CHANGELOG.md` file is the primary resource for understanding what changed:

#### Reading the Changelog

1. **Start with Latest Version**
   - Check the most recent release entry
   - Look at the date to understand recency

2. **Review Sections**:
   - **Added**: New features you might want
   - **Changed**: Modifications that might affect your code
   - **Deprecated**: Features being phased out (plan migration)
   - **Removed**: Features no longer available (check if you use them)
   - **Fixed**: Bug fixes that might resolve your issues
   - **Security**: Critical security patches (upgrade immediately)

3. **Breaking Changes**:
   - Always read this section carefully
   - Indicates MAJOR version bump
   - May require code changes

#### Example Decision Flow

**Scenario**: You're using `v1.1.0` and see `v1.2.0` is available

1. **Check CHANGELOG.md for v1.2.0**:
   ```
   ## [1.2.0] - 2024-01-15
   
   ### Added
   - feat(agents): support for custom system prompts
   - feat(tools): new validation framework
   
   ### Fixed
   - fix(web): browser session cleanup issue
   ```

2. **Evaluate**:
   - ✅ New features you might use? → Consider upgrading
   - ✅ Bug fixes that affect you? → Upgrade recommended
   - ⚠️ Breaking changes? → Review migration guide
   - ❌ No relevant changes? → Can stay on current version

3. **Decision**:
   - If you use web agent and had session issues → **Upgrade to v1.2.0**
   - If you don't need new features → **Stay on v1.1.0** (or upgrade for bug fixes)

### Version Compatibility Matrix

Understanding semantic versioning helps predict compatibility:

| Your Version | Safe to Upgrade To | Requires Code Changes |
|-------------|-------------------|----------------------|
| `v1.2.0`     | `v1.2.x` (any patch) | ❌ No |
| `v1.2.0`     | `v1.x.x` (any minor) | ⚠️ Maybe (new features) |
| `v1.2.0`     | `v2.0.0` (major) | ✅ Yes (breaking changes) |

**Rules**:
- **PATCH** (`v1.2.0` → `v1.2.1`): Safe, backward-compatible bug fixes
- **MINOR** (`v1.2.0` → `v1.3.0`): Safe, new features added
- **MAJOR** (`v1.2.0` → `v2.0.0`): Breaking changes, review migration guide

### Practical Examples

#### Example 1: New Project Setup

```bash
# 1. Check latest version on GitHub Releases
# 2. Review CHANGELOG.md for recent changes
# 3. Install latest stable
go get github.com/thinktwice/agentForge@latest

# Or specify in go.mod
go mod init myproject
# Edit go.mod:
require github.com/thinktwice/agentForge latest
go mod tidy
```

#### Example 2: Upgrading Existing Project

```bash
# 1. Check current version
go list -m github.com/thinktwice/agentForge

# 2. Review CHANGELOG.md for changes since your version
# 3. Check for breaking changes
# 4. Upgrade to latest patch (safe)
go get github.com/thinktwice/agentForge@latest
go mod tidy
go test ./...  # Verify everything still works
```

#### Example 3: Staying on Specific Version

```go
// go.mod
require (
    github.com/thinktwice/agentForge v1.2.3
)

// This locks to exact version
// Use when you need stability and don't want automatic updates
```

#### Example 4: Following a Major Version

```go
// go.mod
require (
    github.com/thinktwice/agentForge v2.0.0
)

// After major version upgrade:
// 1. Read migration guide in release notes
// 2. Update your code for breaking changes
// 3. Run full test suite
// 4. Update documentation
```

### Best Practices for Consumers

1. **Read the Changelog Before Upgrading**
   - Always review `CHANGELOG.md` or GitHub Release notes
   - Pay special attention to "Breaking Changes" section

2. **Test After Upgrading**
   - Run your test suite after version changes
   - Test critical paths manually

3. **Use Version Constraints Wisely**
   - Production: Pin to specific version (`v1.2.3`)
   - Development: Allow minor updates (`v1.2.0`)
   - Libraries: Allow patch updates (`>=v1.2.0, <v2.0.0`)

4. **Monitor Security Advisories**
   - Upgrade immediately for security patches
   - Check "Security" section in changelog

5. **Plan for Major Upgrades**
   - Major versions (`v1.x.x` → `v2.x.x`) require planning
   - Review migration guides
   - Allocate time for code updates
   - Test thoroughly before deploying

6. **Use Go's Module Commands**
   ```bash
   # Check what version you're using
   go list -m github.com/thinktwice/agentForge
   
   # See available updates
   go list -m -u github.com/thinktwice/agentForge
   
   # Update to latest
   go get -u github.com/thinktwice/agentForge
   ```

### Version Information in Code

Consumers can also check version at runtime (if implemented):

```go
import "github.com/thinktwice/agentForge/src/agents"

// If version is exposed via build info
// (requires build-time version injection)
```

### Summary: Consumer Decision Tree

```
Start: Need to choose/upgrade version
│
├─ New project?
│  └─ Use @latest → Check CHANGELOG → Install
│
├─ Production app?
│  └─ Pin to specific version → Review changelog → Test → Deploy
│
├─ Want latest features?
│  └─ Check CHANGELOG "Added" section → Upgrade if useful
│
├─ Experiencing bugs?
│  └─ Check CHANGELOG "Fixed" section → Upgrade if fixed
│
├─ Security concern?
│  └─ Check CHANGELOG "Security" section → Upgrade immediately
│
└─ Major version available?
   └─ Read migration guide → Plan upgrade → Test → Migrate
```

## 12. Implementation Phases

### Phase 1: Foundation
- [ ] Create `VERSION` file
- [ ] Set up commit message linting
- [ ] Create PR label templates
- [ ] Document conventions in CONTRIBUTING.md

### Phase 2: Automation
- [ ] Set up GitHub Actions CI workflow
- [ ] Create changelog generation script/workflow
- [ ] Set up version validation

### Phase 3: Release Automation
- [ ] Create release workflow
- [ ] Set up automated tagging
- [ ] Configure GitHub Releases automation

### Phase 4: Polish
- [ ] Add release checklist automation
- [ ] Set up release notifications
- [ ] Create release templates

## 13. Example Workflow

### Developer Workflow

1. Create feature branch: `git checkout -b feature/new-tool`
2. Make changes and commit:
   ```bash
   git commit -m "feat(tools): add new validation tool"
   ```
3. Push and create PR
4. Label PR: `type:feature`, `area:tools`
5. After review, merge PR
6. Changelog auto-updates with new entry

### Release Manager Workflow

1. Review `CHANGELOG.md` Unreleased section
2. Determine next version (e.g., v1.2.3)
3. Create release branch: `git checkout -b release/v1.2.3`
4. Update `VERSION` file
5. Finalize changelog
6. Create release PR
7. After merge, tag: `git tag v1.2.3`
8. Push tag to trigger release workflow
9. GitHub Release auto-created with changelog

## 14. Migration Plan

For existing repository:

1. **Initial Setup**
   - Create `VERSION` file with current version (e.g., `0.1.0`)
   - Create initial `CHANGELOG.md`
   - Add `.github/workflows/` directory structure

2. **Gradual Adoption**
   - Start using conventional commits immediately
   - Begin labeling PRs
   - Set up CI workflow first

3. **First Release**
   - Manually create v0.1.0 release
   - Establish baseline
   - Document current state in changelog

4. **Full Automation**
   - Enable all workflows
   - Train team on conventions
   - Refine process based on feedback

