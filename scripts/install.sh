#!/usr/bin/env bash
# install.sh — installs the latest ssx release on Linux or macOS.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/highfredo/ssx/main/scripts/install.sh | bash
#
# Override the install directory:
#   INSTALL_DIR=~/.local/bin bash install.sh

set -euo pipefail

REPO="highfredo/ssx"
BINARY="ssx"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# ── helpers ──────────────────────────────────────────────────────────────────

info()  { printf '\033[1;34m→\033[0m  %s\n' "$*"; }
ok()    { printf '\033[1;32m✓\033[0m  %s\n' "$*"; }
die()   { printf '\033[1;31m✗\033[0m  %s\n' "$*" >&2; exit 1; }

require() { command -v "$1" &>/dev/null || die "Required tool not found: $1"; }

# ── detect OS ────────────────────────────────────────────────────────────────

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux)  ;;
  darwin) ;;
  *)      die "Unsupported OS: $OS" ;;
esac

# ── detect arch ──────────────────────────────────────────────────────────────

ARCH=$(uname -m)
case "$ARCH" in
  x86_64)          ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
  *)               die "Unsupported architecture: $ARCH" ;;
esac

# ── fetch latest version ─────────────────────────────────────────────────────

require curl

info "Fetching latest release…"
API_URL="https://api.github.com/repos/${REPO}/releases/latest"
TAG=$(curl -fsSL "$API_URL" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
[ -n "$TAG" ] || die "Could not determine latest version."

VERSION="${TAG#v}"   # strip leading 'v' for archive names
info "Latest version: ${TAG}"

# ── build URLs ───────────────────────────────────────────────────────────────

ARCHIVE="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"

# ── download ─────────────────────────────────────────────────────────────────

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

info "Downloading ${ARCHIVE}…"
curl -fsSL "${BASE_URL}/${ARCHIVE}"      -o "${TMP_DIR}/${ARCHIVE}"
curl -fsSL "${BASE_URL}/checksums.txt"   -o "${TMP_DIR}/checksums.txt"

# ── verify checksum ──────────────────────────────────────────────────────────

info "Verifying checksum…"
EXPECTED=$(grep " ${ARCHIVE}$" "${TMP_DIR}/checksums.txt" | awk '{print $1}')
[ -n "$EXPECTED" ] || die "Checksum not found for ${ARCHIVE} in checksums.txt"

if command -v sha256sum &>/dev/null; then
  ACTUAL=$(sha256sum "${TMP_DIR}/${ARCHIVE}" | awk '{print $1}')
elif command -v shasum &>/dev/null; then
  ACTUAL=$(shasum -a 256 "${TMP_DIR}/${ARCHIVE}" | awk '{print $1}')
else
  die "No sha256 tool found (sha256sum or shasum). Cannot verify checksum."
fi

[ "$ACTUAL" = "$EXPECTED" ] || die "Checksum mismatch!\n  expected: $EXPECTED\n  got:      $ACTUAL"
ok "Checksum OK"

# ── extract & install ────────────────────────────────────────────────────────

tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "${TMP_DIR}"

mkdir -p "$INSTALL_DIR"

if [ -w "$INSTALL_DIR" ]; then
  mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  info "Dir ${INSTALL_DIR} need elevate privileges…"
  sudo mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi
chmod +x "${INSTALL_DIR}/${BINARY}"

ok "ssx ${TAG} installed → ${INSTALL_DIR}/${BINARY}"

