#!/bin/bash
set -e

VERSION=${1:-"dev"}

if [ "$VERSION" = "dev" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 1.0.3"
    exit 1
fi

echo "Building Chief Summarizer v${VERSION} for all platforms..."
echo

# Create dist directory
mkdir -p dist
rm -f dist/*

# Build flags to reduce binary size
LDFLAGS="-s -w"

echo "→ Building Linux AMD64..."
GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o dist/chief-summarizer-linux-amd64 ./cmd/chief-summarizer

echo "→ Building Linux ARM64..."
GOOS=linux GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o dist/chief-summarizer-linux-arm64 ./cmd/chief-summarizer

echo "→ Building macOS AMD64..."
GOOS=darwin GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o dist/chief-summarizer-darwin-amd64 ./cmd/chief-summarizer

echo "→ Building macOS ARM64..."
GOOS=darwin GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o dist/chief-summarizer-darwin-arm64 ./cmd/chief-summarizer

echo "→ Building Windows AMD64..."
GOOS=windows GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o dist/chief-summarizer-windows-amd64.exe ./cmd/chief-summarizer

echo
echo "✓ All binaries built successfully in dist/"
echo
ls -lh dist/
echo

echo "Next steps:"
echo "1. Verify version constant in cmd/chief-summarizer/main.go is set to ${VERSION}"
echo "2. Commit and push changes"
echo "3. Create and push git tag: git tag -a v${VERSION} -m 'Release version ${VERSION}' && git push origin v${VERSION}"
echo "4. Create GitHub release and upload binaries from dist/ directory"
echo "   OR use: gh release create v${VERSION} --title 'v${VERSION}' --notes 'Release notes' dist/*"
