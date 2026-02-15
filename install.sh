#!/usr/bin/env bash
set -e

REPO="alexanderritik/dbgraph" # Replace with actual repo if different

# Fetch latest version tag if no version specified
if [ -z "$1" ]; then
    VERSION=$(curl -s https://api.github.com/repos/$REPO/releases/latest \
      | grep tag_name \
      | cut -d '"' -f 4)
else
    VERSION=$1
fi

if [ -z "$VERSION" ]; then
    echo "Error: Could not determine latest version."
    exit 1
fi

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Map architecture names
if [ "$ARCH" = "x86_64" ]; then
  ARCH="amd64"
elif [ "$ARCH" = "arm64" ] || [ "$ARCH" = "aarch64" ]; then
  ARCH="arm64"
fi

BINARY="dbgraph-${OS}-${ARCH}"
URL="https://github.com/$REPO/releases/download/$VERSION/$BINARY"

echo "Downloading $BINARY $VERSION..."
curl -L --fail $URL -o dbgraph
chmod +x dbgraph

# Move to path
echo "Installing to /usr/local/bin..."
sudo mv dbgraph /usr/local/bin/

echo "Installed successfully! Run 'dbgraph --version' to verify."
