# KataGo Artifacts Directory

This directory contains downloaded KataGo binaries and neural network models used for Docker builds.

## Files

The following files are downloaded by `scripts/download-katago-artifacts.sh`:

- `katago-v1.16.3-eigen-linux-x64.zip` - KataGo binary for Linux x64
- `test-model.bin.gz` - Neural network model for KataGo
- `test-config.cfg` - Configuration file for KataGo (tracked in git)

## Important Notes

- The `.zip` and `.bin.gz` files are in `.gitignore` and not tracked by git
- These files must be downloaded before building the Docker image
- The download script will be run automatically by CI
- For local development, run `./scripts/download-katago-artifacts.sh` before building

## Download

To download the required artifacts:

```bash
./scripts/download-katago-artifacts.sh
```

This will download approximately 130MB of data:
- KataGo binary: ~35MB
- Neural network model: ~95MB