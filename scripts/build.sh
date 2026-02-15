#!/usr/bin/env bash

# Exit immediately if a command exits with a non-zero status
set -e

APP_NAME=dbgraph
VERSION=${1:-dev}

echo "Building $APP_NAME version $VERSION..."

mkdir -p dist

# Helper function to build
build() {
    local os=$1
    local arch=$2
    local output="dist/$APP_NAME-$os-$arch"
    
    echo "Building for $os/$arch..."
    GOOS=$os GOARCH=$arch go build -ldflags "-s -w -X main.version=$VERSION" -o "$output"
    
    # Compress (optional, user didn't explicitly ask but it's good practice. User asked for binaries directly though)
    # Keeping raw binaries as per user request example: "v0.1.0 ... yourtool-darwin-amd64"
}

# Mac Intel
build darwin amd64

# Mac Apple Silicon
build darwin arm64

# Linux AMD64
build linux amd64

# Linux ARM64
build linux arm64

echo "Build complete! Artifacts in dist/"
ls -lh dist/
