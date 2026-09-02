#!/usr/bin/env bash
set -euo pipefail

REPO="ju4n97/hclapi"
INSTALL_DIR="/usr/local/bin"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

case "$OS" in
  linux|darwin|freebsd) ;;
  *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

LATEST_TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
TAG_NO_V="${LATEST_TAG#v}"
FILENAME="hclapi_${TAG_NO_V}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${FILENAME}"

echo "Downloading hclapi ${LATEST_TAG} for ${OS}/${ARCH}..."
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

curl -fsSL "$DOWNLOAD_URL" | tar -xz -C "$TMP_DIR"

if [[ -w "$INSTALL_DIR" ]]; then
  install -m 0755 "$TMP_DIR/hclapi" "$INSTALL_DIR/hclapi"
else
  echo "Elevated permissions required to install to $INSTALL_DIR:"
  sudo install -m 0755 "$TMP_DIR/hclapi" "$INSTALL_DIR/hclapi"
fi

echo "hclapi was installed successfully to $INSTALL_DIR/hclapi"