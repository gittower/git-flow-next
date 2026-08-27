#!/bin/bash

# Abort on the first error so a failed archive step never reaches the
# staging-directory cleanup below, which would otherwise delete the binaries
# needed to retry or diagnose the failure.
set -e

# Get version from command line or use "dev" as default
VERSION=${1:-dev}
BINARY_NAME="git-flow"
PACKAGE_NAME="git-flow-next"

# Build directory
BUILD_DIR="dist"

# Per-platform staging directories, shared with build.sh.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/platforms.sh"

echo "Packaging $PACKAGE_NAME version $VERSION..."

# Verify every binary is present before archiving. build.sh checks this too,
# but package.sh also runs on its own in the release workflow, against a
# staging tree restored from a build artifact, so it cannot assume the
# binaries survived that round trip.
echo "Verifying binaries..."
MISSING_BINARIES=()

for dir in "${PLATFORM_DIRS[@]}"; do
    binary="$(platform_binary "$dir" "$BINARY_NAME")"
    if [[ ! -f "$BUILD_DIR/$binary" ]]; then
        MISSING_BINARIES+=("$binary")
    fi
done

if [[ ${#MISSING_BINARIES[@]} -gt 0 ]]; then
    echo "Error: The following binaries were not found:"
    for binary in "${MISSING_BINARIES[@]}"; do
        echo "  - $binary"
    done
    exit 1
fi

# Drop archives left by an earlier run of this version so re-running package.sh
# cannot retain an obsolete archive or fold a stale checksums file into the new
# one. build.sh wipes $BUILD_DIR wholesale; package.sh only owns its outputs.
rm -f "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-"*.tar.gz \
      "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-"*.zip \
      "$BUILD_DIR/${PACKAGE_NAME}-${VERSION}-checksums.txt"

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

echo "Packaging complete! Artifacts are in the $BUILD_DIR directory"
