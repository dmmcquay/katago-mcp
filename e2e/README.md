# E2E Testing with Docker

This directory contains end-to-end tests that run against a real KataGo instance.

## Docker Architecture

The e2e tests use Docker with locally cached KataGo artifacts:

1. **Artifact Management**:
   - KataGo binary and neural network model are downloaded on-demand
   - Artifacts are cached locally in `docker/katago-artifacts/`
   - Not stored in git to keep repository size small
   - Downloaded automatically by CI before building

2. **Test Image** (`Dockerfile.e2e`):
   - Based on `golang:1.23-bookworm`
   - Copies pre-downloaded KataGo artifacts
   - Runs the e2e tests against real KataGo instance

## Running E2E Tests

### Using Docker (Recommended)

The easiest way to run e2e tests is using Docker:

```bash
# Run e2e tests in Docker
./scripts/run-e2e-docker.sh

# Or use docker-compose
docker-compose -f docker-compose.e2e.yml up --build

# Use a specific base image
BASE_IMAGE=ghcr.io/dmmcquay/katago-base:1.15.3-cpu docker-compose -f docker-compose.e2e.yml up --build
```

### Manual Setup

If you prefer to run tests without Docker:

1. Install KataGo:
   ```bash
   # macOS
   brew install katago
   
   # Ubuntu/Debian
   sudo apt install katago
   ```

2. Set up test model and config:
   ```bash
   ./scripts/setup-katago-test.sh
   ```

3. Run tests:
   ```bash
   go test -v -tags=e2e ./e2e -timeout 300s
   ```

## CI/CD

The e2e tests run automatically in GitHub Actions using Docker. This ensures:
- Consistent test environment
- No dependency on external model downloads during CI
- Faster and more reliable test execution

## Test Structure

- `e2e_test.go` - Main test file with test helpers
- `testdata/` - SGF files used for testing
- `Dockerfile.e2e` - Docker image with KataGo pre-installed
- `docker-compose.e2e.yml` - Docker Compose configuration for development

## Adding New Tests

1. Add test SGF files to `testdata/`
2. Create test functions in `e2e_test.go`
3. Use the `SetupTestEnvironment()` helper to initialize KataGo
4. Tests will automatically run in both local and Docker environments