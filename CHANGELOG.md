## [Unreleased]

## [0.3.0] - 2026-01-28
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
- [a7e0ada](http://github.com/thinktwiceco/agent-forge/commit/a7e0ada5909550349aafce53d16e286bd8658860) - Merge pull request #5 from thinktwiceco/release-0.2.0
- [8e190fc](http://github.com/thinktwiceco/agent-forge/commit/8e190fca80084bd083de01bfd178a6029da0f21e) - chore: update version
- [5ad0e43](http://github.com/thinktwiceco/agent-forge/commit/5ad0e4315d3264669e3a99b6f1bb071390a334b6) - chore(lint): upgrade golangci-lint to v2 and remove unused code
- [5b3c7ea](http://github.com/thinktwiceco/agent-forge/commit/5b3c7ea5a094e92ae7966ee77ab66ebbf2d66269) - fix(git,llms): handle empty repositories and correct deepseek model
- [a05c70c](http://github.com/thinktwiceco/agent-forge/commit/a05c70c631373f1661c074f6699a1da15afd7588) - feat(apis): add list agents endpoint
- [6b75abc](http://github.com/thinktwiceco/agent-forge/commit/6b75abca2ffa7158372fd9067f491e5f6fd65c44) - feat(server): add HTTP API server for agent chat
- [4a87d3c](http://github.com/thinktwiceco/agent-forge/commit/4a87d3ceac6690670a0fb329391fc6de36d5eb52) - refactor(web!): simplify web tool to 4 core actions
- [fff4035](http://github.com/thinktwiceco/agent-forge/commit/fff4035bc7dd707ad6033e0ea02c8c5915f43625) - fix(agents): make subagents silent and result-focused
- [9217f45](http://github.com/thinktwiceco/agent-forge/commit/9217f455819fba26687feb14ab9c33db5eaa4f0a) - refactor(core): introduce agent builder pattern
- [0e1b4e9](http://github.com/thinktwiceco/agent-forge/commit/0e1b4e908ce796adedce88bd45f5e6b4d6ada8af) - chore: update changelog for v0.1.0
- [7c1fd11](http://github.com/thinktwiceco/agent-forge/commit/7c1fd11ea2bb55d6ae19b999855afade148e3915) - type(fix): fix tag detection
- [58936e5](http://github.com/thinktwiceco/agent-forge/commit/58936e5cf6b880dab3157096abb27b84a3c8103d) - fix(ci): replace changelog generator action in release workflow
- [008b5b5](http://github.com/thinktwiceco/agent-forge/commit/008b5b54be08f50d4b2c5c1e2fa0de49586b61f0) - Merge pull request #4 from thinktwiceco/release/v0.1.0
- [619af16](http://github.com/thinktwiceco/agent-forge/commit/619af16d679b60a9d9ab324f9d3b474a1d32cefd) - Merge pull request #3 from thinktwiceco/feat/web-tool-enhancements-and-docs
- [ed741c8](http://github.com/thinktwiceco/agent-forge/commit/ed741c8e7d8b6c7192ac185a7ffc52ba8440ff6e) - Merge pull request #2 from thinktwiceco/feat/web-tool-enhancements-and-docs


## [Unreleased]
## [0.1.0] - 2026-01-21
- no changes

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
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

