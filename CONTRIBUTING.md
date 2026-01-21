# Contributing to agentForge

Thank you for your interest in contributing to agentForge! This document provides guidelines and conventions for contributing to the project.

## Commit Message Conventions

We use [Conventional Commits](https://www.conventionalcommits.org/) specification for all commit messages.

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

To indicate a breaking change (MAJOR version bump), use one of these methods:

1. Add `!` after type/scope: `feat(api)!: change method signature`
2. Include `BREAKING CHANGE:` in footer:
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

## Pull Request Guidelines

### PR Labels

All PRs should be labeled to categorize changes for changelog generation:

#### Primary Labels (Required)

- `type:feature` - New features
- `type:bugfix` - Bug fixes
- `type:breaking` - Breaking changes
- `type:documentation` - Documentation updates
- `type:refactor` - Code refactoring
- `type:performance` - Performance improvements
- `type:chore` - Maintenance tasks

#### Secondary Labels (Optional)

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

#### Status Labels

- `status:ready` - PR is ready for review
- `status:blocked` - PR is blocked
- `status:wip` - Work in progress

### PR Requirements

- PR title should follow conventional commit format
- PR description should explain the changes and motivation
- All tests must pass
- Code should follow existing style and patterns
- Documentation should be updated if needed

## Branch Naming

- `feature/<name>` - New features
- `fix/<name>` - Bug fixes
- `docs/<name>` - Documentation updates
- `refactor/<name>` - Refactoring
- `release/v{version}` - Release preparation

## Release Process

For detailed information about the release process, see [docs/RELEASE_PIPELINE.md](docs/RELEASE_PIPELINE.md).

### Quick Overview

1. Changes are merged to `main` branch
2. Release manager creates a release branch: `release/v{version}`
3. Version is updated in `VERSION` file
4. Changelog is generated and updated
5. Release PR is created and reviewed
6. After merge, tag is created: `git tag v{version}`
7. Tag push triggers automated release workflow
8. GitHub Release is created with changelog

## Development Setup

1. Fork the repository
2. Clone your fork: `git clone https://github.com/your-username/agentForge.git`
3. Create a branch: `git checkout -b feature/your-feature`
4. Make your changes
5. Commit using conventional commit format
6. Push to your fork: `git push origin feature/your-feature`
7. Create a Pull Request

## Testing

- Run tests: `go test ./...`
- Run tests with race detection: `go test -race ./...`
- Run tests with coverage: `go test -cover ./...`

## Code Style

- Follow Go standard formatting: `go fmt ./...`
- Use `golangci-lint` for linting (configured in CI)
- Write clear, self-documenting code
- Add comments for exported functions and types

## Questions?

If you have questions about contributing, please open an issue or reach out to the maintainers.

