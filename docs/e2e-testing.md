# End-to-End Testing

This document describes the end-to-end (e2e) testing setup for the KataGo MCP server.

## Overview

E2E tests verify that the MCP server can successfully communicate with a real KataGo instance to provide Go analysis capabilities. These tests cover:

- **Position Analysis** - Analyzing Go positions and getting move suggestions
- **Game Review** - Finding mistakes in game sequences  
- **Territory Estimation** - Calculating ownership maps and territory counts
- **Move Explanation** - Getting strategic explanations for moves

## Requirements

E2E tests require:

1. **KataGo binary** - The actual KataGo executable
2. **Neural network model** - A trained model file (`.bin.gz`)
3. **Configuration file** - KataGo settings for analysis mode

## Running E2E Tests

### Option 1: Automated Setup Script (Recommended)

The easiest way to run e2e tests is using the provided script:

```bash
./scripts/run-e2e-tests.sh
```

This script will:
- Detect existing KataGo installations
- Auto-configure using KaTrain if available
- Download a test model if needed
- Run the tests with appropriate timeouts

Script options:
```bash
./scripts/run-e2e-tests.sh --help           # Show help
./scripts/run-e2e-tests.sh -v               # Verbose output
./scripts/run-e2e-tests.sh -t 120s          # Custom timeout
./scripts/run-e2e-tests.sh -r TestAnalyze   # Run specific test
```

### Option 2: Manual Setup

1. **Install KataGo**:
   ```bash
   # macOS
   brew install katago
   
   # Linux - download from GitHub releases
   wget https://github.com/lightvector/KataGo/releases/download/v1.15.1/katago-v1.15.1-linux-x64.tar.gz
   tar -xzf katago-v1.15.1-linux-x64.tar.gz
   sudo mv katago /usr/local/bin/
   ```

2. **Set up test environment**:
   ```bash
   export KATAGO_TEST_MODEL=/path/to/model.bin.gz
   export KATAGO_TEST_CONFIG=/path/to/config.cfg
   ```

3. **Run tests**:
   ```bash
   go test -v -tags=e2e ./e2e -timeout 60s
   ```

### Option 3: Using KaTrain

If you have KaTrain installed, the tests will automatically detect and use its KataGo installation:

```bash
pip install katrain
go test -v -tags=e2e ./e2e
```

## CI/CD Integration

### GitHub Actions

The project includes automated e2e testing in CI with two approaches:

1. **Ubuntu Runner** - Downloads KataGo and a test model
2. **macOS Runner** - Uses Homebrew for better compatibility (main branch only)

The CI workflow:
- Attempts to install KataGo and download a neural network model
- Runs e2e tests if installation succeeds
- Gracefully skips tests if KataGo cannot be installed
- Provides clear instructions for local testing

### Configuration

CI uses optimized settings for faster execution:
- Smaller neural network model (15-block vs 40-block)
- Reduced visit counts (50 vs 1000)
- Shorter time limits (1s vs 10s)
- Single-threaded analysis

## Test Structure

### Test Files

- `e2e/e2e_test.go` - Main test implementations
- `e2e/testdata/` - SGF files for testing
  - `simple_opening.sgf` - Basic opening position
  - `game_with_mistakes.sgf` - Game with intentional errors
  - `9x9_endgame.sgf` - Small board endgame position

### Test Cases

1. **TestAnalyzePositionE2E**
   - Tests position analysis with different board sizes
   - Verifies move suggestions and win rates
   - Covers empty board, opening, and 9x9 positions

2. **TestFindMistakesE2E**
   - Tests game review functionality
   - Identifies mistakes and categorizes them
   - Calculates accuracy percentages

3. **TestEvaluateTerritoryE2E**
   - Tests territory estimation
   - Verifies ownership maps and territory counts
   - Includes visual territory representation

4. **TestMCPServerE2E**
   - Tests full MCP integration
   - Verifies tool handlers work correctly
   - Tests all MCP tools end-to-end

## Troubleshooting

### Common Issues

1. **KataGo not found**
   ```
   Solution: Install KataGo or add it to PATH
   ```

2. **Model download fails**
   ```
   Solution: Check internet connection or manually download model
   ```

3. **Tests timeout**
   ```
   Solution: Increase timeout or reduce analysis parameters
   ```

4. **Analysis errors**
   ```
   Solution: Verify model and config file compatibility
   ```

### Performance Considerations

E2E tests can be slow because they involve:
- Neural network initialization (~1-2 seconds)
- Real analysis computations (~0.5-3 seconds per position)
- Model loading overhead

For CI, we use:
- Smaller models for faster loading
- Reduced analysis depth
- Parallel test execution where possible

### Skipping E2E Tests

To run only unit tests (skip e2e):
```bash
go test ./...  # Normal tests only
```

E2E tests are tagged with `e2e` build tag, so they won't run unless explicitly requested.

## Model Information

### Test Models Used

1. **CI Environment**: `g170-b15c192` (15-block network, ~50MB)
   - Smaller and faster for CI
   - Good enough for testing functionality
   - Downloads quickly

2. **Local Development**: KaTrain models (if available)
   - `kata1-b18c384nbt` (18-block network, ~90MB)
   - Better analysis quality
   - Used in real applications

### Model Sources

Models are downloaded from the official KataGo training data:
- https://media.katagotraining.org/g170/neuralnets/

These are the same models used by KataGo and other Go applications.

## Security Considerations

- Models are downloaded from official sources only
- SHA256 checksums should be verified (future enhancement)
- No user-provided models are executed in CI
- Tests run in isolated environments

## CI Performance Optimization

### Pre-built Base Image

To speed up CI builds, we use a pre-built Docker image with KataGo already compiled:

- **Image**: `ghcr.io/dmmcquay/katago-base:v1.14.1`
- **Build time**: Reduced from ~10 minutes to ~2 minutes
- **Workflow**: `.github/workflows/build-katago-base.yml`

The base image is rebuilt only when:
1. KataGo version changes
2. Base image Dockerfile is modified
3. Manually triggered via GitHub Actions

## Future Improvements

1. **Model Caching** - Cache downloaded models between CI runs
2. **Parallel Testing** - Run multiple tests concurrently
3. **Mock Mode** - Optional mock KataGo for faster testing
4. **Model Verification** - Verify model checksums
5. **Performance Benchmarks** - Track analysis speed over time