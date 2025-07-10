# Contributing to KataGo MCP

Thank you for your interest in contributing to KataGo MCP! This document provides guidelines for contributing to the project.

## Development Process

We use GitHub flow for development. All changes should be made through pull requests.

### 1. Fork and Clone

```bash
# Fork the repository on GitHub, then:
git clone https://github.com/YOUR_USERNAME/katago-mcp.git
cd katago-mcp
git remote add upstream https://github.com/dmmcquay/katago-mcp.git
```

### 2. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/your-bug-fix
```

### 3. Make Your Changes

- Write clean, idiomatic Go code
- Follow the existing code style
- Add tests for new functionality
- Update documentation as needed

### 4. Run Local Checks

Before committing, run:

```bash
make pre-commit
```

This will:
- Format your code
- Run the linter
- Run all tests

### 5. Commit Your Changes

```bash
git add .
git commit -m "feat: add new analysis feature"
```

Follow conventional commit format:
- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation changes
- `test:` for test additions/changes
- `refactor:` for code refactoring
- `chore:` for maintenance tasks

### 6. Push and Create PR

```bash
git push origin feature/your-feature-name
```

Then create a pull request on GitHub.

## Pull Request Guidelines

### ⚠️ Required Process
**ALL changes must go through pull requests. Direct pushes to `main` are prohibited.**

Every PR must satisfy these requirements before merging:

#### 🤖 Automated Checks (ALL must pass)
- ✅ **Linting**: Code formatting and style checks
- ✅ **Tests**: Unit and integration tests across multiple Go versions  
- ✅ **Build**: Successful compilation and binary generation
- ✅ **Security**: Vulnerability scanning with no critical/high issues

#### 👤 Manual Review (Required)
- ✅ **Maintainer Approval**: At least one approval from a repository maintainer
- ✅ **Code Review**: Thorough review of logic, architecture, and security

### PR Title
Use the same conventional commit format as commits.

### PR Description
Include:
- What changes were made
- Why the changes were made
- How to test the changes
- Any breaking changes

### PR Checklist
- [ ] Tests pass (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] Code is formatted (`make fmt`)
- [ ] Documentation updated (if needed)
- [ ] No breaking changes (or clearly documented)
- [ ] All CI checks are green
- [ ] PR has maintainer approval

## Code Style

### Go Code
- Follow standard Go conventions
- Use `gofmt` for formatting
- Use meaningful variable and function names
- Add comments for exported functions
- Keep functions small and focused

### Testing
- Write unit tests for new code
- Aim for >80% code coverage
- Include edge cases in tests
- Use table-driven tests when appropriate
- Add integration tests for complex features

### Error Handling
- Always check errors
- Wrap errors with context
- Use structured logging for errors
- Return meaningful error messages

## Running CI Locally

To run all CI checks locally:

```bash
make ci
```

This runs the same checks as GitHub Actions.

## Testing with KataGo

### Setup
1. Install KataGo (see README.md)
2. Download a neural network
3. Generate config file

### Integration Tests
When adding KataGo-specific features:
- Mock the engine for unit tests
- Add integration tests that use real KataGo
- Document any special test requirements

## Getting Help

- Check existing issues and PRs
- Ask questions in issues
- Read the architecture documentation
- Review the implementation plan

## Code of Conduct

- Be respectful and inclusive
- Welcome newcomers
- Focus on constructive feedback
- Assume positive intent

## License

By contributing, you agree that your contributions will be licensed under the MIT License.