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
# -s: Strip symbol table
# -w: Strip DWARF debug info
# Combined with -trimpath and CGO_ENABLED=0 for minimal binary size
BUILD_FLAGS="-s -w -X 'github.com/gittower/git-flow-next/version.BuildTime=${BUILD_TIME}' -X 'github.com/gittower/git-flow-next/version.GitCommit=${GIT_COMMIT}'"

# Create build directory if it doesn't exist
mkdir -p $BUILD_DIR

# Build for each platform/architecture
echo "Building $PACKAGE_NAME version $VERSION..."

# macOS (both Intel and Apple Silicon)
echo "Building darwin/amd64..."
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/${BINARY_NAME}-${VERSION}-darwin-amd64" main.go
echo "Building darwin/arm64..."
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/${BINARY_NAME}-${VERSION}-darwin-arm64" main.go

# Linux
echo "Building linux/amd64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/${BINARY_NAME}-${VERSION}-linux-amd64" main.go
echo "Building linux/arm64..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/${BINARY_NAME}-${VERSION}-linux-arm64" main.go
echo "Building linux/386..."
CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -trimpath -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/${BINARY_NAME}-${VERSION}-linux-386" main.go

# Windows
echo "Building windows/amd64..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/${BINARY_NAME}-${VERSION}-windows-amd64.exe" main.go
echo "Building windows/386..."
CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -trimpath -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/${BINARY_NAME}-${VERSION}-windows-386.exe" main.go
echo "Building windows/arm64..."
CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/${BINARY_NAME}-${VERSION}-windows-arm64.exe" main.go

# Verify all binaries were created
echo "Verifying binaries..."
MISSING_BINARIES=()

for binary in "${BINARY_NAME}-${VERSION}-darwin-amd64" "${BINARY_NAME}-${VERSION}-darwin-arm64" \
              "${BINARY_NAME}-${VERSION}-linux-amd64" "${BINARY_NAME}-${VERSION}-linux-arm64" "${BINARY_NAME}-${VERSION}-linux-386" \
              "${BINARY_NAME}-${VERSION}-windows-amd64.exe" "${BINARY_NAME}-${VERSION}-windows-386.exe" "${BINARY_NAME}-${VERSION}-windows-arm64.exe"; do
    if [[ ! -f "$BUILD_DIR/$binary" ]]; then
        MISSING_BINARIES+=("$binary")
    fi
done

if [[ ${#MISSING_BINARIES[@]} -gt 0 ]]; then
    echo "Error: The following binaries were not created:"
    for binary in "${MISSING_BINARIES[@]}"; do
        echo "  - $binary"
    done
    exit 1
fi

# Create archives for each binary
echo "Creating archives..."

# macOS
echo "Creating darwin archives..."
tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-darwin-amd64.tar.gz" -C "$BUILD_DIR" "${BINARY_NAME}-${VERSION}-darwin-amd64"
tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-darwin-arm64.tar.gz" -C "$BUILD_DIR" "${BINARY_NAME}-${VERSION}-darwin-arm64"

# Linux
echo "Creating linux archives..."
tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-linux-amd64.tar.gz" -C "$BUILD_DIR" "${BINARY_NAME}-${VERSION}-linux-amd64"
tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-linux-arm64.tar.gz" -C "$BUILD_DIR" "${BINARY_NAME}-${VERSION}-linux-arm64"
tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-linux-386.tar.gz" -C "$BUILD_DIR" "${BINARY_NAME}-${VERSION}-linux-386"

# Windows (using zip instead of tar.gz)
echo "Creating windows archives..."
if command -v zip >/dev/null 2>&1; then
    (cd "$BUILD_DIR" && zip "${PACKAGE_NAME}-${VERSION}-windows-amd64.zip" "${BINARY_NAME}-${VERSION}-windows-amd64.exe")
    (cd "$BUILD_DIR" && zip "${PACKAGE_NAME}-${VERSION}-windows-386.zip" "${BINARY_NAME}-${VERSION}-windows-386.exe")
    (cd "$BUILD_DIR" && zip "${PACKAGE_NAME}-${VERSION}-windows-arm64.zip" "${BINARY_NAME}-${VERSION}-windows-arm64.exe")
else
    echo "Warning: zip command not found, falling back to tar.gz for Windows"
    tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-windows-amd64.tar.gz" -C "$BUILD_DIR" "${BINARY_NAME}-${VERSION}-windows-amd64.exe"
    tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-windows-386.tar.gz" -C "$BUILD_DIR" "${BINARY_NAME}-${VERSION}-windows-386.exe"
    tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-windows-arm64.tar.gz" -C "$BUILD_DIR" "${BINARY_NAME}-${VERSION}-windows-arm64.exe"
fi

# Generate checksums
echo "Generating checksums..."
if command -v shasum >/dev/null 2>&1; then
    (cd "$BUILD_DIR" && shasum -a 256 * > "${PACKAGE_NAME}-${VERSION}-checksums.txt")
elif command -v sha256sum >/dev/null 2>&1; then
    (cd "$BUILD_DIR" && sha256sum * > "${PACKAGE_NAME}-${VERSION}-checksums.txt")
else
    echo "Warning: Neither shasum nor sha256sum found, skipping checksums"
fi

echo "Build complete! Artifacts are in the $BUILD_DIR directory" 