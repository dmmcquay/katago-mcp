# Release Process

This document outlines the process for creating a new release of katago-mcp.

## Pre-release Checklist

- [ ] Update version numbers:
  - [ ] `cmd/katago-mcp/main.go` (version string)
  - [ ] `internal/config/config.go` (default version)
  - [ ] All example config files in `config/`
- [ ] Update `CHANGELOG.md` with release notes
- [ ] Update `LICENSE` copyright year if needed
- [ ] Run all tests: `make test`
- [ ] Run CI checks: `make ci`
- [ ] Test installation script: `./scripts/install.sh`
- [ ] Review and update documentation if needed

## Creating a Release

1. Create and push a release tag:
   ```bash
   git add .
   git commit -m "chore: prepare for v1.0.0 release"
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin main
   git push origin v1.0.0
   ```

2. The GitHub Actions workflow will automatically:
   - Build binaries for all platforms
   - Create a GitHub release
   - Upload release artifacts
   - Generate checksums
   - Build and push Docker images

3. After the release is created:
   - [ ] Verify release artifacts on GitHub
   - [ ] Test installation from release:
     ```bash
     curl -L https://raw.githubusercontent.com/dmmcquay/katago-mcp/main/scripts/install.sh | bash
     ```
   - [ ] Update any external documentation or announcements

## Post-release

1. Start development on next version:
   ```bash
   git checkout -b develop
   # Update version to next development version (e.g., 1.1.0-dev)
   ```

2. Update CHANGELOG.md with new "Unreleased" section

## Manual Release (if automation fails)

If the automated release fails, you can create a release manually:

```bash
# Install goreleaser
brew install goreleaser

# Create release locally
goreleaser release --clean --skip-publish

# The artifacts will be in ./dist/
# Upload them manually to the GitHub release
```

## Version Numbering

We follow Semantic Versioning (SemVer):
- MAJOR version for incompatible API changes
- MINOR version for backwards-compatible functionality additions
- PATCH version for backwards-compatible bug fixes

Examples:
- `1.0.0` - First stable release
- `1.1.0` - New features added
- `1.0.1` - Bug fixes only
- `2.0.0` - Breaking changes