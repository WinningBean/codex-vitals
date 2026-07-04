#!/usr/bin/env bash
#
# codex-vitals installer.
#
#   curl -fsSL https://raw.githubusercontent.com/WinningBean/codex-vitals/main/install.sh | bash
#
# Downloads the prebuilt binary for your OS/arch from the latest GitHub release
# and installs it to ~/.local/bin. If no matching release asset exists yet, it
# falls back to building from source with `go install`.
#
set -euo pipefail

REPO="WinningBean/codex-vitals"
BIN="codex-vitals"
INSTALL_DIR="${CODEX_VITALS_BIN_DIR:-$HOME/.local/bin}"

info() { printf '  %s\n' "$*"; }
err() { printf 'error: %s\n' "$*" >&2; exit 1; }

# --- detect OS / arch -------------------------------------------------------
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
  darwin) os="darwin" ;;
  linux)  os="linux" ;;
  *) err "unsupported OS: $os (macOS and Linux only)" ;;
esac

arch="$(uname -m)"
case "$arch" in
  arm64|aarch64) arch="arm64" ;;
  x86_64|amd64)  arch="amd64" ;;
  *) err "unsupported architecture: $arch" ;;
esac

asset="${BIN}-${os}-${arch}"
url="https://github.com/${REPO}/releases/latest/download/${asset}"

mkdir -p "$INSTALL_DIR"
target="${INSTALL_DIR}/${BIN}"

# --- try the prebuilt release binary ---------------------------------------
tmp="$(mktemp)"
info "Looking for a prebuilt binary (${asset})…"
if curl -fsSL "$url" -o "$tmp" 2>/dev/null && [ -s "$tmp" ]; then
  mv "$tmp" "$target"
  chmod +x "$target"
  # macOS: clear the quarantine flag so Gatekeeper doesn't block an unsigned binary.
  [ "$os" = "darwin" ] && xattr -d com.apple.quarantine "$target" 2>/dev/null || true
  info "Installed prebuilt binary → ${target}"
else
  rm -f "$tmp"
  # --- fall back to building from source ------------------------------------
  info "No prebuilt binary available; falling back to 'go install'."
  command -v go >/dev/null 2>&1 || err "Go is required for the source fallback. Install Go 1.22+ or wait for a release."
  info "Building from source with go install…"
  GOBIN="$INSTALL_DIR" go install "github.com/${REPO}/cmd/${BIN}@latest"
  info "Installed from source → ${target}"
fi

# --- PATH hint --------------------------------------------------------------
echo
info "✓ ${BIN} installed to ${INSTALL_DIR}"
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) : ;;
  *) info "Add it to your PATH:  export PATH=\"${INSTALL_DIR}:\$PATH\"" ;;
esac
info "Try it:  ${BIN} -once -style answer-footer"
