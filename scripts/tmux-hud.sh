#!/usr/bin/env bash
#
# Add a codex-vitals HUD pane to the bottom of the current tmux window.
#
#   scripts/tmux-hud.sh                 # answer-footer, refreshing every 1s
#   scripts/tmux-hud.sh -style current-hud
#   CODEX_VITALS_TMUX_HEIGHT=7 scripts/tmux-hud.sh
#
# Run it from inside a tmux session (e.g. the one where Codex is running).
# Focus returns to your original pane; Ctrl+C in the HUD pane closes it.
#
set -euo pipefail

if [ -z "${TMUX:-}" ]; then
  echo "error: run this inside a tmux session" >&2
  exit 1
fi

if command -v codex-vitals >/dev/null 2>&1; then
  bin="codex-vitals"
elif [ -x "./codex-vitals" ]; then
  bin="$(pwd)/codex-vitals"
else
  echo "error: codex-vitals not found on PATH (install it first)" >&2
  exit 1
fi

height="${CODEX_VITALS_TMUX_HEIGHT:-6}"

if [ "$#" -gt 0 ]; then
  args="$*"
else
  args="-style answer-footer -interval 1s"
fi

# Split a bottom pane running the HUD, then return focus to the original pane.
tmux split-window -v -l "$height" "$bin $args"
tmux last-pane
