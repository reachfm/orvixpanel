#!/usr/bin/env bash
# OrvixPanel Bootstrap Installer v0.7.4
#
# One-line installation from GitHub:
#   curl -fsSL https://raw.githubusercontent.com/reachfm/orvixpanel/feature/v0.7.0-mail-hosting/scripts/install.sh | bash
#
# This bootstrap script downloads the release package from GitHub Releases
#, extracts it, and runs the full installer.
#
# Options:
#   --version <ver>   Install specific version (default: v0.7.4)
#   --channel <ch>    Release channel: stable (default), latest
set -euo pipefail

# -----------------------------------------------------------------------------
# Config
# -----------------------------------------------------------------------------
VERSION="${VERSION:-v0.7.4}"
CHANNEL="${CHANNEL:-stable}"
REPO="reachfm/orvixpanel"
RELEASE_URL="https://github.com/${REPO}/releases/download/${VERSION}/orvixpanel-installer-${VERSION}.tar.gz"

# Parse args
while [ $# -gt 0 ]; do
  case "$1" in
    --version) VERSION="$2"; shift 2 ;;
    --channel) CHANNEL="$2"; shift 2 ;;
    --help|-h)
      echo "OrvixPanel Bootstrap Installer"
      echo ""
      echo "Usage:"
      echo "  curl -fsSL https://raw.githubusercontent.com/reachfm/orvixpanel/feature/v0.7.0-mail-hosting/scripts/install.sh | bash"
      echo "  curl -fsSL .../install.sh | bash -s -- --version v0.7.4"
      echo ""
      echo "Options:"
      echo "  --version <ver>  Install specific version (default: v0.7.4)"
      echo "  --channel <ch>   Release channel: stable (default), latest"
      exit 0
      ;;
    *) shift ;;
  esac
done

# -----------------------------------------------------------------------------
# Helpers
# -----------------------------------------------------------------------------
red()   { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
blue()  { printf '\033[34m%s\033[0m\n' "$*"; }
yellow(){ printf '\033[33m%s\033[0m\n' "$*"; }

need_root() {
  if [ "$(id -u)" -ne 0 ]; then
    red "Error: This installer must be run as root (use sudo)"
    exit 1
  fi
}

# Detect OS
detect_os() {
  if [ -f /etc/os-release ]; then
    . /etc/os-release
    echo "$ID"
  else
    echo "unknown"
  fi
}

# Check for required tools
check_deps() {
  local missing=""
  for cmd in curl tar; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
      missing="$missing $cmd"
    fi
  done
  if [ -n "$missing" ]; then
    red "Error: missing required tools:$missing"
    exit 1
  fi
}

# -----------------------------------------------------------------------------
# Main
# -----------------------------------------------------------------------------
main() {
  blue ""
  blue "=============================================="
  blue "  OrvixPanel Bootstrap Installer"
  blue "  Version: $VERSION"
  blue "=============================================="
  echo ""

  # Check dependencies
  check_deps

  # Check for root
  need_root

  # Detect OS
  OS=$(detect_os)
  if [ "$OS" != "ubuntu" ] && [ "$OS" != "debian" ]; then
    yellow "Warning: This installer is tested on Ubuntu/Debian."
    yellow "Detected: $OS. Continuing anyway..."
  fi

  # Create temp directory
  blue "Preparing installation directory..."
  TEMP_DIR=$(mktemp -d)
  trap "rm -rf $TEMP_DIR" EXIT

  # Download release package
  blue "Downloading OrvixPanel ${VERSION}..."
  echo "  URL: $RELEASE_URL"

  if ! curl -fsSL "$RELEASE_URL" -o "${TEMP_DIR}/package.tar.gz"; then
    red ""
    red "Error: Failed to download release package."
    red "  URL: $RELEASE_URL"
    red ""
    red "Possible reasons:"
    red "  1. Version $VERSION does not exist"
    red "  2. Network connectivity issues"
    red "  3. GitHub rate limiting (try again later)"
    red ""
    red "To install a specific version:"
    red "  curl -fsSL .../install.sh | bash -s -- --version <version>"
    exit 1
  fi

  # Verify download
  if [ ! -s "${TEMP_DIR}/package.tar.gz" ]; then
    red "Error: Downloaded file is empty"
    exit 1
  fi

  # Extract package
  blue "Extracting package..."
  tar -xzf "${TEMP_DIR}/package.tar.gz" -C "$TEMP_DIR"

  # Strip leading 'v' from VERSION for path construction (tarball uses 'v0.7.4' not 'vv0.7.4')
  PKG_VERSION="${VERSION#v}"
  PKG_DIR="${TEMP_DIR}/release/orvixpanel-installer-${PKG_VERSION}"
  INSTALL_SCRIPT="${PKG_DIR}/scripts/install.sh"

  # Check for install.sh in extracted package
  if [ ! -f "$INSTALL_SCRIPT" ]; then
    red "Error: install.sh not found in package"
    red "Expected: $INSTALL_SCRIPT"
    red "Package contents:"
    find "$TEMP_DIR" -type f | head -20
    exit 1
  fi

  # Make install.sh executable
  chmod +x "$INSTALL_SCRIPT"

  # Run the actual installer
  green ""
  green "Bootstrap complete. Running full installer..."
  green ""

  # Pass through any additional arguments
  exec "$INSTALL_SCRIPT" "$@"
}

main "$@"
