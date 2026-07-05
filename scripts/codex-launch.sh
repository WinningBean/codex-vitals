#!/usr/bin/env bash
#
# codex-vitals-panel: launch Codex with a live codex-vitals panel pane below it.
# Drop-in replacement for the fwyc0573 codex-hud wrapper.
#
#   CODEX_CLI_PATH           path to the codex binary (default: codex on PATH)
#   CODEX_VITALS_SIZE        panel size (xs|s|m|l|xl); also sets pane height
#   CODEX_VITALS_TMUX_HEIGHT override pane height
#
set -euo pipefail

CODEX_BIN="${CODEX_CLI_PATH:-codex}"

size="${CODEX_VITALS_SIZE:-m}"
case "$size" in
  xs | xsmall | s | small) dh=4 ;;
  l | large | xl | xlarge) dh=8 ;;
  *) dh=6 ;;
esac
height="${CODEX_VITALS_TMUX_HEIGHT:-$dh}"

# Pass -size explicitly so the panel honors the size regardless of how tmux
# propagates the environment to the split pane.
HUD_CMD="codex-vitals -context-mode current-hud -size $size -interval 1s"

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
  inner="CODEX_CLI_PATH=$(printf '%q' "$CODEX_BIN") CODEX_VITALS_SIZE=$(printf '%q' "$size") CODEX_VITALS_TMUX_HEIGHT=$(printf '%q' "$height") $(printf '%q ' "$0" "$@")"
  exec tmux new-session "$inner"
fi
