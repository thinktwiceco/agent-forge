# AGENT STEPS TO CREATE A PULL REQUEST

Analyze current changes and create a GitHub PR with appropriate description and labels.

## Actions

1. Check current branch name
2. Get git diff to analyze changes
3. Determine PR type from changes:
   - `type:feature` - New functionality
   - `type:bugfix` - Bug fixes
   - `type:breaking` - Breaking changes
   - `type:documentation` - Docs only
   - `type:refactor` - Code refactoring
   - `type:performance` - Performance improvements
   - `type:chore` - Maintenance
4. Determine area labels from changed files:
   - `area:agents`, `area:tools`, `area:llms`, `area:plugins`, `area:web`, `area:vector`, `area:git`, `area:os`, `area:coding`, `area:reasoning`, `area:core`
5. Generate PR description with:
   - Summary of changes
   - List of modified files
   - Breaking changes (if any)
   - Testing notes
6. Create PR using GitHub CLI or API:
   - Title: Follow conventional commit format
   - Body: Generated description
   - Labels: Type and area labels
   - Base branch: main
7. Output PR URL

## Rules

- PR title should follow conventional commit format
- Include clear description of what changed and why
- List breaking changes prominently if present
- Add appropriate labels for changelog generation
- Set base branch to `main`

