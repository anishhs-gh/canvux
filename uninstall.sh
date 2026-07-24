#!/bin/sh
# Canvux uninstaller — removes the canvux binary safely.
#
# Usage:
#   ./uninstall.sh              # find and remove the installed binary
#   ./uninstall.sh --dir ~/bin  # look in a specific directory
#   ./uninstall.sh --purge      # also remove ~/.config/canvux (asks first)
#   ./uninstall.sh --yes        # no questions (for curl | sh and scripts)
set -eu

BINARY=canvux
DIRS="/usr/local/bin $HOME/.local/bin $HOME/bin"
PURGE=no
ASSUME_YES=no

while [ $# -gt 0 ]; do
  case "$1" in
    --dir)   DIRS="$2"; shift 2 ;;
    --purge) PURGE=yes; shift ;;
    --yes|-y) ASSUME_YES=yes; shift ;;
    -h|--help) sed -n '2,9p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "unknown option: $1 (try --help)" >&2; exit 1 ;;
  esac
done

# ask <question>: yes/no that also works when the script is piped from
# curl (stdin is the pipe, so read from the terminal directly).
ask() {
  [ "$ASSUME_YES" = yes ] && return 0
  printf "%s [y/N] " "$1"
  if [ -t 0 ]; then
    read -r ans
  elif [ -r /dev/tty ]; then
    read -r ans < /dev/tty
  else
    echo ""
    echo "no terminal to confirm on — re-run with --yes" >&2
    exit 1
  fi
  case "$ans" in y|Y) return 0 ;; *) return 1 ;; esac
}

# Collect every installed copy across the search paths.
FOUND=""
for d in $DIRS; do
  if [ -f "$d/$BINARY" ]; then
    FOUND="$FOUND $d/$BINARY"
  fi
done
FOUND="${FOUND# }"

if [ -z "$FOUND" ]; then
  echo "canvux is not installed in: $DIRS"
  echo "(if you installed elsewhere, run: ./uninstall.sh --dir <path>)"
  exit 0
fi

for f in $FOUND; do
  # Safety: only delete something that actually behaves like canvux.
  if ! "$f" version 2>/dev/null | grep -q '^canvux '; then
    echo "refusing: $f does not look like the canvux binary" >&2
    exit 1
  fi
  echo "found $f ($("$f" version))"
done
ask "remove?" || { echo "cancelled"; exit 0; }

for f in $FOUND; do
  if [ -w "$(dirname "$f")" ]; then
    rm "$f"
  else
    sudo rm "$f"
  fi
  echo "removed $f"
done

CFG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/canvux"
if [ "$PURGE" = yes ] && [ -d "$CFG_DIR" ]; then
  echo "config found at $CFG_DIR:"
  ls "$CFG_DIR"
  if ask "delete it too?"; then
    rm -rf "$CFG_DIR"; echo "removed $CFG_DIR"
  else
    echo "kept $CFG_DIR"
  fi
elif [ -d "$CFG_DIR" ]; then
  echo "note: config kept at $CFG_DIR (use --purge to remove it)"
fi

echo "canvux uninstalled."
