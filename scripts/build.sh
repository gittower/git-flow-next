#!/bin/bash

# Abort on the first error so a failed build or archive step never reaches the
# staging-directory cleanup below, which would otherwise delete the binaries
# needed to retry or diagnose the failure.
set -e

# Get version from command line or use "dev" as default
VERSION=${1:-dev}
BINARY_NAME="git-flow"
PACKAGE_NAME="git-flow-next"

# Build directory
BUILD_DIR="dist"

# Per-platform staging directories. Each binary is built with the plain name
# ($BINARY_NAME / $BINARY_NAME.exe) inside its own subdir so the archives can
# ship the plain name without the binaries colliding in $BUILD_DIR.
PLATFORM_DIRS=(darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 linux-386 windows-amd64 windows-386 windows-arm64)

# Get build information
GIT_COMMIT=$(git rev-parse --short HEAD)
BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S')

# Build flags
# -s: Strip symbol table
# -w: Strip DWARF debug info
# Combined with -trimpath and CGO_ENABLED=0 for minimal binary size
BUILD_FLAGS="-s -w -X 'github.com/gittower/git-flow-next/version.BuildTime=${BUILD_TIME}' -X 'github.com/gittower/git-flow-next/version.GitCommit=${GIT_COMMIT}'"

# Start from a clean build directory so stale archives from an earlier run
# never leak into the checksums and re-archiving can't retain an obsolete
# binary entry. Then create the per-platform staging directories.
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"
for dir in "${PLATFORM_DIRS[@]}"; do
    mkdir -p "$BUILD_DIR/$dir"
done

# Build for each platform/architecture
echo "Building $PACKAGE_NAME version $VERSION..."

# macOS (both Intel and Apple Silicon)
echo "Building darwin/amd64..."
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/darwin-amd64/${BINARY_NAME}" main.go
echo "Building darwin/arm64..."
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/darwin-arm64/${BINARY_NAME}" main.go

# Linux
echo "Building linux/amd64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/linux-amd64/${BINARY_NAME}" main.go
echo "Building linux/arm64..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/linux-arm64/${BINARY_NAME}" main.go
echo "Building linux/386..."
CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -trimpath -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/linux-386/${BINARY_NAME}" main.go

# Windows
echo "Building windows/amd64..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/windows-amd64/${BINARY_NAME}.exe" main.go
echo "Building windows/386..."
CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -trimpath -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/windows-386/${BINARY_NAME}.exe" main.go
echo "Building windows/arm64..."
CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath -ldflags "${BUILD_FLAGS}" -o "$BUILD_DIR/windows-arm64/${BINARY_NAME}.exe" main.go

# Verify all binaries were created
echo "Verifying binaries..."
MISSING_BINARIES=()

for dir in "${PLATFORM_DIRS[@]}"; do
    case "$dir" in
        windows-*) binary="$dir/${BINARY_NAME}.exe" ;;
        *)         binary="$dir/${BINARY_NAME}" ;;
    esac
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

# Create archives for each binary. Archive filenames keep the version/platform
# info; the binary inside each archive uses the plain name.
echo "Creating archives..."

# macOS
echo "Creating darwin archives..."
tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-darwin-amd64.tar.gz" -C "$BUILD_DIR/darwin-amd64" "${BINARY_NAME}"
tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-darwin-arm64.tar.gz" -C "$BUILD_DIR/darwin-arm64" "${BINARY_NAME}"

# Linux
echo "Creating linux archives..."
tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-linux-amd64.tar.gz" -C "$BUILD_DIR/linux-amd64" "${BINARY_NAME}"
tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-linux-arm64.tar.gz" -C "$BUILD_DIR/linux-arm64" "${BINARY_NAME}"
tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-linux-386.tar.gz" -C "$BUILD_DIR/linux-386" "${BINARY_NAME}"

# Windows (using zip instead of tar.gz)
echo "Creating windows archives..."
if command -v zip >/dev/null 2>&1; then
    (cd "$BUILD_DIR/windows-amd64" && zip "../${PACKAGE_NAME}-${VERSION}-windows-amd64.zip" "${BINARY_NAME}.exe")
    (cd "$BUILD_DIR/windows-386" && zip "../${PACKAGE_NAME}-${VERSION}-windows-386.zip" "${BINARY_NAME}.exe")
    (cd "$BUILD_DIR/windows-arm64" && zip "../${PACKAGE_NAME}-${VERSION}-windows-arm64.zip" "${BINARY_NAME}.exe")
else
    echo "Warning: zip command not found, falling back to tar.gz for Windows"
    tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-windows-amd64.tar.gz" -C "$BUILD_DIR/windows-amd64" "${BINARY_NAME}.exe"
    tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-windows-386.tar.gz" -C "$BUILD_DIR/windows-386" "${BINARY_NAME}.exe"
    tar czf "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-windows-arm64.tar.gz" -C "$BUILD_DIR/windows-arm64" "${BINARY_NAME}.exe"
fi

# Remove per-platform staging directories, leaving only the archives in
# $BUILD_DIR so the checksum step hashes exactly the release artifacts.
for dir in "${PLATFORM_DIRS[@]}"; do
    rm -rf "$BUILD_DIR/$dir"
done

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
