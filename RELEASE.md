# Release Process Quick Reference

This guide explains how to create a new release with binaries for Chief Summarizer.

## Prerequisites

- Go 1.21+ installed
- Git repository access with push permissions
- GitHub CLI (`gh`) installed (optional, but recommended)
  - Install: https://cli.github.com/

## Step-by-Step Release Process

### 1. Update Version Number

Edit `cmd/chief-summarizer/main.go`:
```go
const version = "1.0.2"  // Change this to your new version
```

### 2. Commit and Push Version Change

```bash
git add cmd/chief-summarizer/main.go
git commit -m "Bump version to 1.0.2"
git push
```

### 3. Build All Binaries

Use the provided build script:
```bash
./scripts/build-release.sh 1.0.2
```

This creates binaries in the `dist/` directory for:
- Linux (AMD64 and ARM64)
- macOS (Intel and Apple Silicon)
- Windows (AMD64)

### 4. Create Git Tag

```bash
git tag -a v1.0.2 -m "Release version 1.0.2"
git push origin v1.0.2
```

### 5. Create GitHub Release

#### Option A: Using GitHub CLI (Recommended)

```bash
gh release create v1.0.2 \
  --title "v1.0.2" \
  --notes "Release notes:
- Feature: Description
- Fix: Description
- Change: Description" \
  dist/*
```

#### Option B: Using GitHub Web Interface

1. Go to: https://github.com/danst0/Chief-Summarizer/releases/new
2. Select tag: `v1.0.2`
3. Release title: `v1.0.2`
4. Add release notes
5. Drag and drop all files from `dist/` to the assets section
6. Click "Publish release"

### 6. Verify Auto-Update Works

Users with older versions will automatically update:
```bash
chief-summarizer -version
# Output shows update detection and installation
```

## Build Script Details

The `scripts/build-release.sh` script:
- Takes version as parameter
- Builds for all supported platforms
- Uses `-ldflags="-s -w"` to strip debug info and reduce size
- Outputs binaries to `dist/` directory
- Shows next steps after build

## Manual Cross-Compilation

If you prefer manual control:

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o chief-summarizer-linux-amd64 ./cmd/chief-summarizer

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o chief-summarizer-linux-arm64 ./cmd/chief-summarizer

# macOS AMD64
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o chief-summarizer-darwin-amd64 ./cmd/chief-summarizer

# macOS ARM64
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o chief-summarizer-darwin-arm64 ./cmd/chief-summarizer

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o chief-summarizer-windows-amd64.exe ./cmd/chief-summarizer
```

## Environment Variables Explained

- `GOOS`: Target operating system (linux, darwin, windows)
- `GOARCH`: Target architecture (amd64, arm64)
- `-ldflags="-s -w"`: Linker flags to strip debug info and symbol table (reduces binary size)

## Troubleshooting

**Problem**: Build fails with missing dependencies
```bash
go mod tidy
go mod download
```

**Problem**: Binary is too large
- The `-ldflags="-s -w"` flag reduces size significantly
- Current sizes: ~6-7 MB per binary

**Problem**: Auto-update not working
- Ensure binary names follow the pattern: `chief-summarizer-{GOOS}-{GOARCH}[.exe]`
- Verify all binaries are uploaded to the GitHub release
- Check that the tag starts with `v` (e.g., `v1.0.2`)

## Complete Example Workflow

```bash
# 1. Update version in code
vim cmd/chief-summarizer/main.go  # Set version = "1.0.2"

# 2. Commit and push
git add cmd/chief-summarizer/main.go
git commit -m "Bump version to 1.0.2"
git push

# 3. Build binaries
./scripts/build-release.sh 1.0.2

# 4. Create and push tag
git tag -a v1.0.2 -m "Release version 1.0.2"
git push origin v1.0.2

# 5. Create GitHub release with binaries
gh release create v1.0.2 --title "v1.0.2" --notes "Release notes" dist/*

# 6. Clean up
rm -rf dist/
```

## Important Notes

- Always test the build script on a clean checkout before release
- Use semantic versioning (MAJOR.MINOR.PATCH)
- Document breaking changes in release notes
- The self-update mechanism requires binaries to be attached to GitHub releases
- Binary naming is critical - don't rename the files
