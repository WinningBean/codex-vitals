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
# CODEX_VITALS_SINCE = panel launch time, so the panel binds to the session
# started with it and ignores older sessions in the same directory.
SINCE="${CODEX_VITALS_SINCE:-$(date +%s)}"
# No -size: the binary resolves the size from the config file every second,
# so changing the file updates the panel live.
HUD_CMD="CODEX_VITALS_SINCE=$SINCE codex-vitals -context-mode codex -margin ${CODEX_VITALS_MARGIN:-2} -interval 1s"

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

# mouse off면 휠 스크롤이 화살표 키로 변환돼 대화 스크롤백을 볼 수 없으므로 안내만 출력한다.
# (사용자 설정을 강제로 바꾸지 않는다 — 텍스트 선택 동작이 달라질 수 있어 opt-in으로 남긴다.)
if [ "$(tmux show -gv mouse 2>/dev/null)" = "off" ]; then
  echo "codex-vitals-panel: tmux 'mouse' is off — add 'set -g mouse on' to ~/.tmux.conf so the wheel scrolls the conversation" >&2
fi

if [ -n "${TMUX:-}" ]; then
  # Already in tmux: add the panel pane, run codex here, close the pane on exit.
  hud="$(tmux split-window -v -l "$height" -P -F '#{pane_id}' "$HUD_CMD")"
  tmux last-pane
  trap 'tmux kill-pane -t "$hud" 2>/dev/null || true' EXIT
  "$CODEX_BIN" "$@"
else
  # Not in tmux: spin up a session that re-runs this script inside tmux.
  inner="CODEX_CLI_PATH=$(printf '%q' "$CODEX_BIN") CODEX_VITALS_TMUX_HEIGHT=$(printf '%q' "$height") CODEX_VITALS_SINCE=$(printf '%q' "$SINCE") $(printf '%q ' "$0" "$@")"
  exec tmux new-session "$inner"
fi
