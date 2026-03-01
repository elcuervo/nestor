#!/usr/bin/env sh
set -eu

TMP=""
cleanup() { [ -n "$TMP" ] && rm -f "$TMP"; }
trap cleanup EXIT INT TERM

case "$(uname -s)" in
  Linux)  OS="linux"  ;;
  Darwin) OS="darwin" ;;
  *)      echo "unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64)        ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)             echo "unsupported arch: $(uname -m)" >&2; exit 1 ;;
esac

TMP=$(mktemp)
curl -fsSL "https://github.com/elcuervo/nestor/releases/latest/download/nestor-${OS}-${ARCH}" -o "$TMP"
chmod +x "$TMP" && mv "$TMP" ./nestor && TMP=""
./nestor
