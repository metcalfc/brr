# Release Process

## Creating a Release

To create a new release:

```bash
# Create and push a version tag
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

This will trigger the GitHub Actions workflow which:

1. **Builds brr (TUI)** for:
   - Linux: amd64, arm64
   - macOS: amd64 (Intel), arm64 (Apple Silicon)

2. **Builds grr (GUI)** for:
   - Linux: amd64, arm64
   - macOS: amd64 (Intel), arm64 (Apple Silicon)

3. **Creates GitHub Release** with:
   - All platform binaries as `.tar.gz` archives
   - SHA256 checksums
   - Auto-generated release notes

## Build Details

- **brr**: Pure Go, cross-compiles easily (CGO_ENABLED=0)
- **grr**: Fyne GUI, requires native builds per platform (CGO_ENABLED=1)
  - Linux builds on Ubuntu with OpenGL/X11 dependencies
  - macOS builds on macOS runner with native toolchain

## Artifacts

Each release produces 8 archives:
- `brr_VERSION_linux_amd64.tar.gz`
- `brr_VERSION_linux_arm64.tar.gz`
- `brr_VERSION_darwin_amd64.tar.gz`
- `brr_VERSION_darwin_arm64.tar.gz`
- `grr_VERSION_linux_amd64.tar.gz`
- `grr_VERSION_linux_arm64.tar.gz`
- `grr_VERSION_darwin_amd64.tar.gz`
- `grr_VERSION_darwin_arm64.tar.gz`
- `checksums.txt`

## Homebrew Distribution

After release, update the Homebrew tap (future):
- Formula downloads from GitHub release
- Uses checksums for verification
- Supports both Intel and Apple Silicon Macs
