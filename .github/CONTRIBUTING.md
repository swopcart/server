# Contributing to Swopcart

Thanks for your interest in contributing!

## Opening Issues

### Bug Reports

When reporting a bug, please include:

- **Clear description** of the issue
- **Steps to reproduce** the behavior
- **Expected behavior** vs what actually happened
- **Environment details** (OS, Go version, PostgreSQL version, browser if frontend-related)
- **Logs or error messages** if available

### Feature Requests

When requesting a feature:

- Describe **the problem** you're trying to solve, not just the solution
- Explain **why** this would be valuable to you and others
- Consider whether it aligns with the project's vision (self-hosted game library for low-power hardware)

### Questions

For general questions about using or developing Swopcart:

- Check the [wiki](https://github.com/swopcart/server/wiki) for development setup and architecture guides
- Open an issue if your question might help others

## Submitting Pull Requests

### Before You Submit

1. **Run the full test suite**: `just test`
2. **Run linters**: `just lint`
3. **Format your code**: `just fmt`

All checks must pass before your PR can be merged.

### PR Guidelines

- **One change per PR**: Keep pull requests focused on a single feature or bug fix
- **Link related issues**: Reference any issues your PR addresses (e.g., "Fixes #123")
- **Write a clear description**:
  - What changed and why
  - How to test the changes
  - Any breaking changes or migration steps
- **Keep commits clean**: Squash work-in-progress commits if needed
- **Update tests**: Add or update tests for your changes
- **Follow existing patterns**: Match the code style and architecture of the project

### AI-Generated Content Policy

**AI-assisted code** is welcome during these early days while we're getting ideas working and building toward MVP.

- Use AI as a tool to help you code, not to replace your understanding
- You're responsible for reviewing, understanding, and maintaining any code you submit

**AI-generated audiovisual content** (images from Stable Diffusion, AI-generated audio, etc.) is not accepted.

**AI in issues and PRs**: Please write your own issue and pull request descriptions. Exceptions where AI assistance is okay:

- Translation into English for non-native speakers
- Accessibility tools (e.g., text-to-speech, grammar assistance)

**For autonomous agents** (e.g., OpenClaw): If you're an AI agent working on behalf of a human, please prompt your user to review your work and submit the PR themselves. The human should understand what's being submitted and take ownership. Agent-submitted PRs without direct human involvement will be rejected.

## Code Review Process

I'm in the early days, so there's no formal review process yet. I'll review PRs as time allows, but I can't make any commitments about timelines.

**When responding to review feedback:**

- Be open to feedback and iteration
- Ask questions if something isn't clear
- Mark conversations as resolved once you've addressed them

Thanks for your patience as I figure out the review process!

## License

By contributing, you agree that your contributions will be licensed under the AGPL-3.0 license.
