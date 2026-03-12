#!/bin/sh
set -e

REPO="jayl2kor/skillhub"
BINARY="skillhub"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS
OS="$(uname -s)"
case "$OS" in
    Linux*)  OS="linux" ;;
    Darwin*) OS="darwin" ;;
    *)       echo "Error: unsupported OS: $OS" >&2; exit 1 ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
    *)       echo "Error: unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Choose download tool
if command -v curl >/dev/null 2>&1; then
    DOWNLOAD="curl"
elif command -v wget >/dev/null 2>&1; then
    DOWNLOAD="wget"
else
    echo "Error: curl or wget is required" >&2
    exit 1
fi

fetch() {
    if [ "$DOWNLOAD" = "curl" ]; then
        curl -fsSL "$1"
    else
        wget -qO- "$1"
    fi
}

download() {
    if [ "$DOWNLOAD" = "curl" ]; then
        curl -fsSL -o "$2" "$1"
    else
        wget -qO "$2" "$1"
    fi
}

# Get latest release tag
echo "Detecting latest release..."
LATEST_URL="https://api.github.com/repos/${REPO}/releases/latest"
TAG="$(fetch "$LATEST_URL" | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"

if [ -z "$TAG" ]; then
    echo "Error: could not determine latest release" >&2
    exit 1
fi

echo "Latest release: $TAG"

# Build download URL
ASSET="${BINARY}-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${ASSET}"

# Download binary
TMPFILE="$(mktemp)"
trap 'rm -f "$TMPFILE"' EXIT

echo "Downloading ${ASSET}..."
download "$DOWNLOAD_URL" "$TMPFILE"

# Install
chmod +x "$TMPFILE"
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMPFILE" "${INSTALL_DIR}/${BINARY}"
else
    echo "Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo mv "$TMPFILE" "${INSTALL_DIR}/${BINARY}"
fi

echo "Installed ${BINARY} ${TAG} to ${INSTALL_DIR}/${BINARY}"
