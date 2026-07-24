#!/bin/sh
# Canvux installer — installs a prebuilt binary when one is available, so
# end users do NOT need Go. Go is only used as a last-resort fallback to
# build from source.
#
# Usage:
#   ./install.sh                 # auto: prebuilt binary > release download > go build
#   ./install.sh --dir ~/bin     # install somewhere specific
#   ./install.sh --local         # force ~/.local/bin (never sudo)
#   ./install.sh --version 1.0.0 # install a specific release (or 1.1.0-beta.1)
#
# Environment:
#   CANVUX_REPO      GitHub repo for release downloads (default: anishhs-gh/canvux)
#   CANVUX_VERSION   same as --version
set -eu

REPO="${CANVUX_REPO:-anishhs-gh/canvux}"
BINARY=canvux
INSTALL_DIR=""
FORCE_LOCAL=no
VERSION="${CANVUX_VERSION:-}"

while [ $# -gt 0 ]; do
  case "$1" in
    --dir)     INSTALL_DIR="$2"; shift 2 ;;
    --local)   FORCE_LOCAL=yes; shift ;;
    --version) VERSION="$2"; shift 2 ;;
    -h|--help)
      sed -n '2,14p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "unknown option: $1 (try --help)" >&2; exit 1 ;;
  esac
done
VERSION="${VERSION#v}" # accept both 1.0.0 and v1.0.0

# --- detect platform ---------------------------------------------------------
OS=$(uname -s)
case "$OS" in
  Linux)  OS=linux ;;
  Darwin) OS=darwin ;;
  *) echo "error: unsupported OS '$OS' — on Windows, use dist/canvux-windows-amd64.exe" >&2; exit 1 ;;
esac
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64)  ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) echo "error: unsupported architecture '$ARCH'" >&2; exit 1 ;;
esac

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
SRC=""

# --- 1. binary already built next to this script -----------------------------
# (skipped when a specific --version is requested: that means "download it")
if [ -z "$VERSION" ] && [ -x "$SCRIPT_DIR/$BINARY" ] && "$SCRIPT_DIR/$BINARY" version >/dev/null 2>&1; then
  SRC="$SCRIPT_DIR/$BINARY"
  echo "using existing binary: $SRC"
fi

# --- 2. cross-compiled binary in dist/ ---------------------------------------
if [ -z "$VERSION" ] && [ -z "$SRC" ] && [ -f "$SCRIPT_DIR/dist/$BINARY-$OS-$ARCH" ]; then
  SRC="$SCRIPT_DIR/dist/$BINARY-$OS-$ARCH"
  echo "using prebuilt binary: $SRC"
fi

# --- 3. download from GitHub releases (no Go required) -----------------------
if [ -z "$SRC" ]; then
  if [ -n "$VERSION" ]; then
    URL="https://github.com/$REPO/releases/download/v$VERSION/$BINARY-$OS-$ARCH"
  else
    URL="https://github.com/$REPO/releases/latest/download/$BINARY-$OS-$ARCH"
  fi
  echo "trying release download: $URL"
  fetch() { # fetch <url> <out>; returns nonzero on failure
    if command -v curl >/dev/null 2>&1; then curl -fsSL -o "$2" "$1" 2>/dev/null
    elif command -v wget >/dev/null 2>&1; then wget -q -O "$2" "$1" 2>/dev/null
    else return 1; fi
  }
  if fetch "$URL" "$TMP/$BINARY"; then
    SRC="$TMP/$BINARY"
  elif fetch "$URL.gz" "$TMP/$BINARY.gz" && gzip -d "$TMP/$BINARY.gz"; then
    SRC="$TMP/$BINARY"   # compressed release asset (about 3x smaller)
  else
    if [ -n "$VERSION" ]; then
      echo "error: release v$VERSION not found for $OS/$ARCH" >&2
      exit 1
    fi
    echo "no release available (repo not published yet, or offline)"
  fi
fi

# --- 4. last resort: build from source (requires Go) -------------------------
if [ -z "$SRC" ]; then
  if command -v go >/dev/null 2>&1 && [ -f "$SCRIPT_DIR/go.mod" ]; then
    echo "building from source with $(go version | cut -d' ' -f3)…"
    (cd "$SCRIPT_DIR" && go build -ldflags="-s -w" -o "$TMP/$BINARY" ./cmd/canvux)
    SRC="$TMP/$BINARY"
  else
    echo "error: no prebuilt binary found and Go is not installed." >&2
    echo "  Either install Go (https://go.dev/dl/) and re-run, or download a" >&2
    echo "  binary for $OS/$ARCH and put it on your PATH yourself." >&2
    exit 1
  fi
fi

# --- pick install dir --------------------------------------------------------
if [ -z "$INSTALL_DIR" ]; then
  if [ "$FORCE_LOCAL" = yes ]; then
    INSTALL_DIR="$HOME/.local/bin"
  elif [ -w /usr/local/bin ]; then
    INSTALL_DIR=/usr/local/bin
  elif command -v sudo >/dev/null 2>&1; then
    INSTALL_DIR=/usr/local/bin
  else
    INSTALL_DIR="$HOME/.local/bin"
  fi
fi

# --- install -----------------------------------------------------------------
mkdir -p "$INSTALL_DIR" 2>/dev/null || true
if [ -w "$INSTALL_DIR" ]; then
  install -m 0755 "$SRC" "$INSTALL_DIR/$BINARY"
else
  echo "need sudo to write to $INSTALL_DIR"
  sudo install -m 0755 "$SRC" "$INSTALL_DIR/$BINARY"
fi

echo "installed: $INSTALL_DIR/$BINARY"
"$INSTALL_DIR/$BINARY" version || true

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo ""
    echo "NOTE: $INSTALL_DIR is not on your PATH. Add this to ~/.zshrc or ~/.bashrc:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    ;;
esac
