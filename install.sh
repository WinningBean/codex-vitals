#!/usr/bin/env bash
#
# codex-vitals installer.
#
#   curl -fsSL https://raw.githubusercontent.com/WinningBean/codex-vitals/main/install.sh | bash
#
# Pick a default panel size in one shot (xs, s, m, l, xl):
#
#   curl -fsSL https://raw.githubusercontent.com/WinningBean/codex-vitals/main/install.sh | bash -s -- xl
#
# Downloads the prebuilt binary for your OS/arch from the latest GitHub release
# and installs it to ~/.local/bin. If no matching release asset exists yet, it
# falls back to building from source with `go install`. When a size is given,
# it writes ~/.config/codex-vitals/size, which the panel reads every second —
# so a running panel switches size live, no restart or shell reload needed.
# Re-run with a different size any time to change it.
#
set -euo pipefail

REPO="WinningBean/codex-vitals"
BIN="codex-vitals"
INSTALL_DIR="${CODEX_VITALS_BIN_DIR:-$HOME/.local/bin}"
SIZE="${1:-}"

info() { printf '  %s\n' "$*"; }
err() { printf 'error: %s\n' "$*" >&2; exit 1; }

case "$SIZE" in
  ""|xs|xsmall|s|small|m|medium|l|large|xl|xlarge) : ;;
  *) err "invalid size: $SIZE (use xs, s, m, l, or xl)" ;;
esac

# Persist the panel size to the config file the panel reads every second, so a
# running panel picks up the change live (no shell reload, no restart).
persist_size() {
  local size="$1" dir="${XDG_CONFIG_HOME:-$HOME/.config}/codex-vitals"
  mkdir -p "$dir"
  printf '%s\n' "$size" > "$dir/size"
  info "Set panel size '${size}' (${dir/#$HOME/~}/size) — running panels update live"
}

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

# --- persist chosen size ----------------------------------------------------
if [ -n "$SIZE" ]; then
  persist_size "$SIZE"
fi

# --- PATH hint --------------------------------------------------------------
echo
info "✓ ${BIN} installed to ${INSTALL_DIR}"
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) : ;;
  *) info "Add it to your PATH:  export PATH=\"${INSTALL_DIR}:\$PATH\"" ;;
esac
if [ -n "$SIZE" ]; then
  info "Try it:  ${BIN} -once    (change size any time: … | bash -s -- <xs|s|m|l|xl>)"
else
  info "Try it:  ${BIN} -once -size l    (or set a default: … | bash -s -- xl)"
fi
