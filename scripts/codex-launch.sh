#!/usr/bin/env bash
#
# codex-vitals-panel: launch Codex with a live codex-vitals panel pane below it.
#
#   CODEX_CLI_PATH            path to the codex binary (default: codex on PATH)
#   CODEX_VITALS_TMUX_HEIGHT  override the panel pane height
#   CODEX_VITALS_MARGIN       left indent (spaces) to align with Codex input (default 2)
#
# The panel size comes from ~/.config/codex-vitals/size (set it with
# `install.sh | bash -s -- <size>`); a running panel picks up changes live.
#
set -euo pipefail

CODEX_BIN="${CODEX_CLI_PATH:-codex}"
# No -size: the binary resolves the size from the config file every second,
# so changing the file updates the panel live.
HUD_CMD="codex-vitals -context-mode patched -margin ${CODEX_VITALS_MARGIN:-2} -interval 1s"

# Size the pane to the configured size (file > env > m).
size_file="${XDG_CONFIG_HOME:-$HOME/.config}/codex-vitals/size"
size="$(cat "$size_file" 2>/dev/null | tr -d '[:space:]' || true)"
size="${size:-${CODEX_VITALS_SIZE:-m}}"
case "$size" in
  xs | xsmall | s | small) dh=4 ;;
  l | large | xl | xlarge) dh=8 ;;
  *) dh=6 ;;
esac
height="${CODEX_VITALS_TMUX_HEIGHT:-$dh}"

command -v codex-vitals >/dev/null 2>&1 || { echo "codex-vitals-panel: codex-vitals not found on PATH" >&2; exit 1; }
command -v tmux >/dev/null 2>&1 || { echo "codex-vitals-panel: tmux is required" >&2; exit 1; }

if [ -n "${TMUX:-}" ]; then
  # Already in tmux: add the panel pane, run codex here, close the pane on exit.
  hud="$(tmux split-window -v -l "$height" -P -F '#{pane_id}' "$HUD_CMD")"
  tmux last-pane
  trap 'tmux kill-pane -t "$hud" 2>/dev/null || true' EXIT
  "$CODEX_BIN" "$@"
else
  # Not in tmux: spin up a session that re-runs this script inside tmux.
  inner="CODEX_CLI_PATH=$(printf '%q' "$CODEX_BIN") CODEX_VITALS_TMUX_HEIGHT=$(printf '%q' "$height") $(printf '%q ' "$0" "$@")"
  exec tmux new-session "$inner"
fi
