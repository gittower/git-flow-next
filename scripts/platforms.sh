#!/bin/bash

# Shared release target list, sourced by build.sh and package.sh.
#
# build.sh and package.sh run in separate CI jobs (binaries are signed in
# between), so the target list has to agree across both. Keeping it here means
# a new platform is added once; adding a target still needs a matching
# `go build` line and an archive line in the respective script.
#
# Each binary is built with the plain name inside its own subdirectory so the
# archives can ship the plain name without the binaries colliding in $BUILD_DIR.
PLATFORM_DIRS=(darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 linux-386 windows-amd64 windows-386 windows-arm64)

# Echo the $BUILD_DIR-relative path of the binary for a platform directory.
# Usage: platform_binary <platform-dir> <binary-name>
platform_binary() {
    case "$1" in
        windows-*) echo "$1/$2.exe" ;;
        *)         echo "$1/$2" ;;
    esac
}
