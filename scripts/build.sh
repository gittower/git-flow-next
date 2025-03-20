#!/bin/bash

# Get version from command line or use "dev" as default
VERSION=${1:-dev}
BINARY_NAME="git-flow"
PACKAGE_NAME="git-flow-next"

# Build directory
BUILD_DIR="dist"

# Get build information
GIT_COMMIT=$(git rev-parse --short HEAD)
BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S')

# Build flags
BUILD_FLAGS="-X github.com/gittower/git-flow-next/version.BuildTime='${BUILD_TIME}' -X github.com/gittower/git-flow-next/version.GitCommit=${GIT_COMMIT}"

# Create build directory if it doesn't exist
mkdir -p $BUILD_DIR

# Build for each platform/architecture
echo "Building $PACKAGE_NAME version $VERSION..."

# macOS (both Intel and Apple Silicon)
GOOS=darwin GOARCH=amd64 go build -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/${BINARY_NAME}-${VERSION}-darwin-amd64" main.go
GOOS=darwin GOARCH=arm64 go build -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/${BINARY_NAME}-${VERSION}-darwin-arm64" main.go

# Linux
GOOS=linux GOARCH=amd64 go build -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/${BINARY_NAME}-${VERSION}-linux-amd64" main.go
GOOS=linux GOARCH=arm64 go build -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/${BINARY_NAME}-${VERSION}-linux-arm64" main.go
GOOS=linux GOARCH=386 go build -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/${BINARY_NAME}-${VERSION}-linux-386" main.go

# Windows
GOOS=windows GOARCH=amd64 go build -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/${BINARY_NAME}-${VERSION}-windows-amd64.exe" main.go
GOOS=windows GOARCH=386 go build -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/${BINARY_NAME}-${VERSION}-windows-386.exe" main.go

# Create archives for each binary
echo "Creating archives..."

# macOS
tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-darwin-amd64.tar.gz" -C "$BUILD_DIR" "${BINARY_NAME}-${VERSION}-darwin-amd64"
tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-darwin-arm64.tar.gz" -C "$BUILD_DIR" "${BINARY_NAME}-${VERSION}-darwin-arm64"

# Linux
tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-linux-amd64.tar.gz" -C "$BUILD_DIR" "${BINARY_NAME}-${VERSION}-linux-amd64"
tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-linux-arm64.tar.gz" -C "$BUILD_DIR" "${BINARY_NAME}-${VERSION}-linux-arm64"
tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-linux-386.tar.gz" -C "$BUILD_DIR" "${BINARY_NAME}-${VERSION}-linux-386"

# Windows (using zip instead of tar.gz)
if command -v zip >/dev/null 2>&1; then
    (cd "$BUILD_DIR" && zip "${PACKAGE_NAME}-${VERSION}-windows-amd64.zip" "${BINARY_NAME}-${VERSION}-windows-amd64.exe")
    (cd "$BUILD_DIR" && zip "${PACKAGE_NAME}-${VERSION}-windows-386.zip" "${BINARY_NAME}-${VERSION}-windows-386.exe")
else
    echo "Warning: zip command not found, skipping Windows archives"
fi

# Generate checksums
echo "Generating checksums..."
(cd "$BUILD_DIR" && shasum -a 256 * > "${PACKAGE_NAME}-${VERSION}-checksums.txt")

echo "Build complete! Artifacts are in the $BUILD_DIR directory" 