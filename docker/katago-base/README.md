# KataGo Base Image

This directory contains the Dockerfile for building the pre-compiled KataGo base image.

## Purpose

The base image speeds up CI builds by pre-compiling KataGo. Instead of compiling KataGo from source in every CI run (which takes ~8-10 minutes), we can just copy the binary from this pre-built image (takes ~30 seconds).

## Building

### Automated (GitHub Actions)

The image is automatically built and pushed to GitHub Container Registry when:
1. Changes are pushed to `docker/katago-base/` on the main branch
2. The workflow is manually triggered via GitHub Actions UI

### Manual Build

To build locally:
```bash
cd docker/katago-base
./build.sh v1.14.1
```

To push to registry (requires authentication):
```bash
docker push ghcr.io/dmmcquay/katago-base:v1.14.1
docker push ghcr.io/dmmcquay/katago-base:latest
```

## Versions

- `v1.14.1` - KataGo v1.14.1 with Eigen backend (CPU only)
- `latest` - Points to the most recent stable version

## Usage

The image is used in `Dockerfile.e2e`:
```dockerfile
FROM ghcr.io/dmmcquay/katago-base:v1.14.1 AS katago-base
# ... copy binary from katago-base
```

## Updating KataGo Version

1. Update the version in `.github/workflows/build-katago-base.yml`
2. Update the version in `Dockerfile.e2e`
3. Trigger the workflow to build and push the new version
4. Update any documentation references