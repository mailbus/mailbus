#!/bin/bash
# MailBus Automatic Installation Script
# This script detects your platform and downloads the appropriate MailBus binary

set -e

VERSION=${1:-latest}
INSTALL_DIR=${INSTALL_DIR:-/usr/local/bin}

echo "MailBus Installation Script"
echo "==========================="
echo "Version: $VERSION"
echo "Install Directory: $INSTALL_DIR"
echo ""

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
echo "Detected OS: $OS"

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
  x86_64|amd64)
    ARCH="amd64"
    ;;
  aarch64|arm64)
    ARCH="arm64"
    ;;
  armv7l)
    ARCH="armv7"
    ;;
  i386|i686)
    ARCH="386"
    ;;
  *)
    echo "Error: Unsupported architecture: $ARCH"
    echo "Supported architectures: amd64, arm64, armv7, 386"
    exit 1
    ;;
esac
echo "Detected Architecture: $ARCH"

# Determine binary name
BINARY_NAME="mailbus-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
  BINARY_NAME="${BINARY_NAME}.exe"
fi
echo "Binary: $BINARY_NAME"

# Download URL
DOWNLOAD_URL="https://github.com/mailbus/mailbus/releases/${VERSION}/download/${BINARY_NAME}"
echo "Download URL: $DOWNLOAD_URL"
echo ""

# Create temp directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

echo "Downloading MailBus..."
if command -v wget &> /dev/null; then
  wget -q --show-progress "$DOWNLOAD_URL" -O "$TEMP_DIR/mailbus${BINARY_NAME##mailbus}" || {
    echo "Error: Download failed"
    echo "Please check your internet connection and try again"
    exit 1
  }
elif command -v curl &> /dev/null; then
  curl -L -o "$TEMP_DIR/mailbus${BINARY_NAME##mailbus}" "$DOWNLOAD_URL" || {
    echo "Error: Download failed"
    echo "Please check your internet connection and try again"
    exit 1
  }
else
  echo "Error: Neither wget nor curl is available"
  exit 1
fi

# Download checksum if available
CHECKSUM_URL="https://github.com/mailbus/mailbus/releases/${VERSION}/download/${BINARY_NAME}.sha256"
if wget -q --spider "$CHECKSUM_URL" 2>/dev/null || curl -s --head "$CHECKSUM_URL" | head -n 1 | grep "200" > /dev/null; then
  echo "Downloading checksum..."
  if command -v wget &> /dev/null; then
    wget -q "$CHECKSUM_URL" -O "$TEMP_DIR/mailbus.sha256"
  else
    curl -sL "$CHECKSUM_URL" -o "$TEMP_DIR/mailbus.sha256"
  fi

  # Verify checksum
  echo "Verifying checksum..."
  cd "$TEMP_DIR"
  if ! sha256sum -c mailbus.sha256 2>/dev/null; then
    echo "Warning: Checksum verification failed"
    echo "The downloaded file may be corrupted"
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      exit 1
    fi
  else
    echo "Checksum verified successfully"
  fi
  cd - > /dev/null
fi

# Make binary executable
chmod +x "$TEMP_DIR/mailbus${BINARY_NAME##mailbus}"

# Install
echo ""
echo "Installing MailBus to $INSTALL_DIR..."
if [ -w "$INSTALL_DIR" ]; then
  cp "$TEMP_DIR/mailbus${BINARY_NAME##mailbus}" "$INSTALL_DIR/mailbus${BINARY_NAME##mailbus}"
else
  echo "Sudo required for installation to $INSTALL_DIR"
  sudo cp "$TEMP_DIR/mailbus${BINARY_NAME##mailbus}" "$INSTALL_DIR/mailbus${BINARY_NAME##mailbus}"
fi

echo ""
echo "MailBus installed successfully!"
echo ""
echo "To verify installation:"
echo "  mailbus version"
echo ""
echo "To get started:"
echo "  mailbus config init"
echo ""
echo "For more information:"
echo "  https://github.com/mailbus/mailbus"
echo "  https://github.com/mailbus/mailbus/blob/main/AGENT_INSTALLATION.md"
