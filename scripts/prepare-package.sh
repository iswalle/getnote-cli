#!/usr/bin/env bash
# Build every supported binary into the npm package's prebuilt/ directory.
# Uses npm naming convention (x64 instead of amd64) for consistency.
set -euo pipefail

VERSION="${1:-$(node -p "require('./package.json').version")}"
VERSION="${VERSION% }"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "==> Preparing package for version ${VERSION}"
echo "==> Cleaning prebuilt directory..."
rm -rf prebuilt

echo "==> Creating platform directories..."
mkdir -p prebuilt/darwin-amd64 prebuilt/darwin-arm64 prebuilt/linux-amd64 prebuilt/linux-arm64 prebuilt/windows-amd64 prebuilt/windows-arm64

LDFLAGS="-s -w -X github.com/iswalle/getnote-cli/internal/version.Version=${VERSION}"

echo "==> Building binaries..."
echo "  - macOS x64..."
GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o prebuilt/darwin-amd64/getnote .

echo "  - macOS ARM64..."
GOOS=darwin GOARCH=arm64 go build -ldflags "$LDFLAGS" -o prebuilt/darwin-arm64/getnote .

echo "  - Linux x64..."
GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o prebuilt/linux-amd64/getnote .

echo "  - Linux ARM64..."
GOOS=linux GOARCH=arm64 go build -ldflags "$LDFLAGS" -o prebuilt/linux-arm64/getnote .

echo "  - Windows x64..."
GOOS=windows GOARCH=amd64 go build -ldflags "$LDFLAGS" -o prebuilt/windows-amd64/getnote.exe .

echo "  - Windows ARM64..."
GOOS=windows GOARCH=arm64 go build -ldflags "$LDFLAGS" -o prebuilt/windows-arm64/getnote.exe .

echo "==> Verifying prebuilt binaries..."
find prebuilt -type f -exec ls -lh {} \; | awk '{print $5, $9}'

TOTAL_SIZE=$(du -sh prebuilt | awk '{print $1}')
echo "==> Total prebuilt size: ${TOTAL_SIZE}"

echo "==> Package preparation complete!"
echo ""
echo "Next steps:"
echo "  1. Test with: npm pack --dry-run"
echo "  2. Verify with: tar -tzf getnote-cli-${VERSION}.tgz | grep prebuilt"
echo "  3. Test install: npm install -g ./getnote-cli-${VERSION}.tgz"
echo "  4. Publish with: npm publish"
