## [0.4.13] - 2026-03-01
- fix(knowledge): fix "malformed JSON" error in traversal tools by adding json_valid check
- refactor(knowledge): store NULL instead of empty strings for node metadata

## [0.4.12] - 2026-03-01
- fix(knowledge): fix explore_fact returning empty results for titles/IDs

## [0.4.11] - 2026-03-01
- chore(release): prepare release v0.4.11

## [0.4.10] - 2026-02-28
- [f8c6661](http://github.com/thinktwiceco/agent-forge/commit/f8c6661acafa852628358efe39d16c5fbb2ce811) - feat(plugins): add knowledge plugin with graph and semantic search (#34)


## [Unreleased]
## [0.4.8] - 2026-02-27
- [cd3615f](http://github.com/thinktwiceco/agent-forge/commit/cd3615fb0c196bb44aeb2edf88c324dc0d16e0de) - chore(release): prepare release v0.4.8 (#30)
- [7249fc6](http://github.com/thinktwiceco/agent-forge/commit/7249fc6800bdd52c3158abe9275b7e42e4296027) - fix: replace WriteString+Sprintf with Fprintf for staticcheck QF1012

## [0.4.7] - 2026-02-20
- [a069354](http://github.com/thinktwiceco/agent-forge/commit/a0693548002141728da8fad30e131ca966c6511a) - chore: bump version to 0.4.7
- [a100130](http://github.com/thinktwiceco/agent-forge/commit/a10013015e914d866736be77a3351d1c247154e0) - feat(web): add push SSE endpoint and chat UI subscription
- [f0f41e4](http://github.com/thinktwiceco/agent-forge/commit/f0f41e4f0cc7d7cd62b68185a706432b77254d0d) - feat: add queue package and async inbox for agent
- [3313f94](http://github.com/thinktwiceco/agent-forge/commit/3313f946a99a5f0effb49e9f24bebf1bed1a06e5) - chore(release): prepare release v0.4.6 (#28)
- [2eb341a](http://github.com/thinktwiceco/agent-forge/commit/2eb341a8b00dbd7c7d1a4cd9985a4f3bc0facf49) - feat(history): add src/history package, fix gitignore to track it
- [67df18f](http://github.com/thinktwiceco/agent-forge/commit/67df18f2263d20e85d1762195bf7fce350204162) - chore: add commit history to v0.4.6 changelog

## [0.4.6] - 2026-02-18
### Fixed
- fix: stop gitignoring go.work (module workspace)
- [0fb186c](http://github.com/thinktwiceco/agent-forge/commit/0fb186cc949bbecd8d918391f17e97dbc24c2870) - fix(agents): use %w for error wrapping in executor (#25)
- [4aa8e2a](http://github.com/thinktwiceco/agent-forge/commit/4aa8e2a360cd271d0f00897f3f0913688ca1d07a) - Release/v0.4.5 (#27)
- [590a1e8](http://github.com/thinktwiceco/agent-forge/commit/590a1e8b73f45d4781fa2326b164e6ea0682eb27) - chore(release): prepare release v0.4.6 (#26)
- [9c9445a](http://github.com/thinktwiceco/agent-forge/commit/9c9445a293673527baa34a0ca123108d149fec10) - chore(release): prepare release v0.4.6 - fix gitignore for go.work module

## [0.4.4] - 2026-02-16
- [eec4bdf](http://github.com/thinktwiceco/agent-forge/commit/eec4bdf43b89b21df3777829ab748438f1f04e36) - chore(release): prepare release v0.4.4 (#22)

### Changed
- chore(release): prepare release v0.4.4 (patch version bump)

## [0.4.3] - 2026-02-10
- [8bb578c](http://github.com/thinktwiceco/agent-forge/commit/8bb578ce8ff132497ef6d3f16a4bc7f90c7b9320) - Merge pull request #20 from thinktwiceco/release/v0.4.3
- [8fcb52a](http://github.com/thinktwiceco/agent-forge/commit/8fcb52afddb4ceb8dcc0ef441d4ff4e3f414cae9) - chore(release): prepare release v0.4.3
- [f4769e3](http://github.com/thinktwiceco/agent-forge/commit/f4769e3d1eeebece931e8127d85efe9c6032a39b) - Merge pull request #19 from thinktwiceco/release/v0.4.2
- [d65f4d4](http://github.com/thinktwiceco/agent-forge/commit/d65f4d427d858d69317c05c6c325f5deaf460bbd) - Merge pull request #18 from thinktwiceco/changelog-v0.4.2
- [0824516](http://github.com/thinktwiceco/agent-forge/commit/08245161d3474a8c6ab381639938f23fd592c1b6) - chore(release): prepare release v0.4.2
- [b334c57](http://github.com/thinktwiceco/agent-forge/commit/b334c57b39f82392998b2974bbd6c201384af9b3) - chore: update changelog for v0.4.2 [skip ci]

### Added
- Add Kimi-K2.5 (moonshotai/Kimi-K2.5) to TogetherAI available models
- Tool call context in chat UI: show tool arguments summary, executing status, and success results (generic formatting, no tool-specific logic)

## [0.4.2] - 2026-02-09
- [6314e16](http://github.com/thinktwiceco/agent-forge/commit/6314e16da01692cccc984f72df4143cba7ec639d) - Merge pull request #17 from thinktwiceco/release/v0.4.2
- [02e083e](http://github.com/thinktwiceco/agent-forge/commit/02e083e0253e3acc273e426f0eccbbd88a8df2c3) - chore(release): prepare release v0.4.2
- [748e112](http://github.com/thinktwiceco/agent-forge/commit/748e1123386b140995fb11e3b59c07f2fc0dadcb) - Merge pull request #16 from thinktwiceco/changelog-v0.4.1
- [2c32d6d](http://github.com/thinktwiceco/agent-forge/commit/2c32d6dafe62336410a4a9f40f24434d36e76c7e) - chore: update changelog for v0.4.1 [skip ci]


## [0.4.1] - 2026-02-08
- [55e9ab9](http://github.com/thinktwiceco/agent-forge/commit/55e9ab9a0dc689971af4839da4f08b5d0f011ac7) - Merge pull request #15 from thinktwiceco/release/v0.4.1
- [7ff6c14](http://github.com/thinktwiceco/agent-forge/commit/7ff6c140c5d0ecb8952fac78dbce6353c6d3d671) - chore(release): prepare release v0.4.1
- [02974b1](http://github.com/thinktwiceco/agent-forge/commit/02974b11ec382f70a962cced5773f5a93fc4aa72) - Merge feat/api-tool into release/v0.4.1
- [84c2494](http://github.com/thinktwiceco/agent-forge/commit/84c2494e914a11bb8d046f0751e762cf68d658c2) - fix(tools): resolve linter errors in API tool
- [acca4fb](http://github.com/thinktwiceco/agent-forge/commit/acca4fb3eacb872a4ba4933b8cfaa655607b4b06) - feat: add API Tool for generic HTTP requests
- [8b27549](http://github.com/thinktwiceco/agent-forge/commit/8b27549e3ce49b1d028939fdf7ce28ad0ccd030b) - feat: add API Tool for generic HTTP requests
- [b58e8ac](http://github.com/thinktwiceco/agent-forge/commit/b58e8ac3413ca31a6b44273e37f6b7faea097c0f) - Merge pull request #13 from thinktwiceco/changelog-v0.4.0
- [6a4fa89](http://github.com/thinktwiceco/agent-forge/commit/6a4fa8947231d5e5e70f6348c3d6e805aa23b692) - chore: update changelog for v0.4.0 [skip ci]

## [0.4.0] - 2026-02-07
- [fe4d8f4](http://github.com/thinktwiceco/agent-forge/commit/fe4d8f4ca5fcc9d314277ed97ecfbd98d201ecc7) - Merge pull request #12 from thinktwiceco/release/v0.4.0
- [d3c8daa](http://github.com/thinktwiceco/agent-forge/commit/d3c8daaf751ab1e6fa8b1e4f072db013bf5b6d85) - ci: remove duplicate test workflow
- [e99d470](http://github.com/thinktwiceco/agent-forge/commit/e99d470b8e11f4210844bd1bf601cb63fcadf5bb) - chore(release): prepare release v0.4.0
- [c25b65f](http://github.com/thinktwiceco/agent-forge/commit/c25b65fd59a2485e04783220135a05a62f54f356) - Merge branch 'main' of github.com:thinktwiceco/agent-forge
- [7e09fca](http://github.com/thinktwiceco/agent-forge/commit/7e09fca3fe84ab5f3619acb03bf341b75363f0be) - Add builder module documentation with configuration guide
- [a236d59](http://github.com/thinktwiceco/agent-forge/commit/a236d59b3963ded4427f9b24d66f2d2dd3521b17) - Merge pull request #11 from thinktwiceco/changelog-v0.3.1
- [8a33339](http://github.com/thinktwiceco/agent-forge/commit/8a3333973f08c6979fb67344e4e6fe5f74ac66d9) - Merge branch 'main' into changelog-v0.3.1
- [366dc96](http://github.com/thinktwiceco/agent-forge/commit/366dc9639b6c72391f5baa04820df03443294d68) - Merge pull request #10 from thinktwiceco/release-0.3.1
- [f0421b9](http://github.com/thinktwiceco/agent-forge/commit/f0421b95250766419fb7de40754d57587b074d4c) - chore: update changelog for v0.3.1 [skip ci]

## [0.3.1] - 2026-01-29

### Fixed
- fix(module): correct module path from `github.com/thinktwice/agentForge` to `github.com/thinktwiceco/agent-forge` to match actual repository
- Updated all import statements across the codebase to use the correct module path
- Updated documentation to reference the correct module path

## [0.3.0] - 2026-01-28
- [6d62c5d](http://github.com/thinktwiceco/agent-forge/commit/6d62c5d7bc9b77f67083e5681c2e4af2bfab28c0) - ci: disable setup-go cache in CI workflow
- [4071bbc](http://github.com/thinktwiceco/agent-forge/commit/4071bbccdf41b286210548ec570f7a9b96b6dc03) - ci: fix changelog generation and disable setup-go cache to resolve tar errors
- [135995f](http://github.com/thinktwiceco/agent-forge/commit/135995fde5f77cb60989dcd8976590f9b3a76df7) - ci(release): remove redundant manual changelog update
- [faceda9](http://github.com/thinktwiceco/agent-forge/commit/faceda93f9ebe1b7a6dae4319a885f93ca23bdff) - ci: use commit-based changelog generation and fix triggers
- [6c8b2a0](http://github.com/thinktwiceco/agent-forge/commit/6c8b2a00c4068e8007f815e0c3396a8f1fe10ba6) - ci: prevent CI trigger on automated changelog PRs
- [38e0ce4](http://github.com/thinktwiceco/agent-forge/commit/38e0ce4316a825634c194577b7a3968daac0361a) - ci(release): fix changelog labels and add catch-all category
- [1c6ad85](http://github.com/thinktwiceco/agent-forge/commit/1c6ad85af7de1235d056a94876f79abd7b98a16f) - ci: remove redundant manual caching steps in favor of setup-go built-in cache
- [5bc9bef](http://github.com/thinktwiceco/agent-forge/commit/5bc9befc27988f9c4adce631a3366658545bba05) - ci(release): use GH_PAT for PR creation as fallback
- [e490e93](http://github.com/thinktwiceco/agent-forge/commit/e490e932a8b41f43e88a52d9e71c31a9ca7053e5) - ci(version-check): exclude current version from validation comparison
- [abf4c33](http://github.com/thinktwiceco/agent-forge/commit/abf4c330e8ad8c1102fe2cf1d6e9c5322a944f4b) - ci(release): grant write permissions for pull-requests
- [e7746da](http://github.com/thinktwiceco/agent-forge/commit/e7746da33e1ea6d64be29b5a78cca45afc2e7b36) - ci(release): create PR for changelog update to respect branch protection
- [5201dd0](http://github.com/thinktwiceco/agent-forge/commit/5201dd0e549b17e0df719969f49bbb90f6ea50e8) - ci(release): fix detached HEAD push failure in release workflow
- [7e827b7](http://github.com/thinktwiceco/agent-forge/commit/7e827b78ba3f48e910c9a80eafbf883d24beb01f) - fix(test): resolve linting errors across the test suite
- [a7b4f23](http://github.com/thinktwiceco/agent-forge/commit/a7b4f2310270219858ddaedeaec862739cfac93d) - test(core): improve test coverage and fix logger nil pointer
- [cbfd12a](http://github.com/thinktwiceco/agent-forge/commit/cbfd12ad04c27b818d13342e7613a5c9ed6debc4) - docs(project): update documentation and add process guides
- [e945916](http://github.com/thinktwiceco/agent-forge/commit/e945916bf91c85589c52667942391db3a1d4c0f3) - docs(readme): update documentation for multi-conversation support Add Conversation Persistence section explaining chatId usage Update code examples to reflect SubAgent.ChatStream signature change Add REST API examples for multi-conversation support
- [b2952ca](http://github.com/thinktwiceco/agent-forge/commit/b2952cae387bd58735b04eb354207b2f2b0b6b53) - fix(core): improve error channel handling and refactor tests Fix race condition in response channel error handling Refactor agent tests to use mocks Add integration build tag to reasoning tests Update test script to support integration tests
- [70d9e4e](http://github.com/thinktwiceco/agent-forge/commit/70d9e4e1c3fb7bbd771563348d9613a1eb58865c) - feat(core)!: implement multi-conversation support and directory persistence
- [5fa0818](http://github.com/thinktwiceco/agent-forge/commit/5fa08185f1ecf5b344c9233dddec1ea5f5a0f880) - fix(core): apply lint fixes and skip intentionally failing test
- [cc88546](http://github.com/thinktwiceco/agent-forge/commit/cc88546b06dc34033783f9cc53675f879825fc0e) - feat(core): complete agent forge rename and refine playground tracing
- [0257abe](http://github.com/thinktwiceco/agent-forge/commit/0257abe7f012afc94546621489a0c02327b2292e) - test(expand): update tests to match SubAgent interface
- [d95b05c](http://github.com/thinktwiceco/agent-forge/commit/d95b05cf9bea106f66a80e1b4ef94b9e5f7c4cf6) - refactor(core): use SubAgent interface directly and simplify server
- [e9cfe63](http://github.com/thinktwiceco/agent-forge/commit/e9cfe6354e13d6366816d52d30f91d924b7e8d4d) - fix(agents): fix configuration builder
- [b7dcd26](http://github.com/thinktwiceco/agent-forge/commit/b7dcd269e005c344599cbf9c5e93ce5cec8fc31e) - fix(agents): Fix agent builder and add support for vector database in config file
- [ac151fb](http://github.com/thinktwiceco/agent-forge/commit/ac151fb9dff1c0648ff91aee5c5514776fbaeeb9) - feat(core): support agent config files and add python sdk playground


# [0.1.0] - 2026-01-21
- no changes

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2024-01-XX

### Added
- Initial release of agentForge framework
- Agent creation and management system
- Tool system with validation
- Multi-agent collaboration support
- LLM integration (OpenAI, TogetherAI, DeepSeek)
- System agents (Reasoning, OS, Git, Coding, Web, Vector)
- Streaming responses
- Conversation persistence
- Plugin system
- Logger plugin
- Todo plugin

