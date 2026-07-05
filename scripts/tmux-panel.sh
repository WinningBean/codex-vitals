#!/usr/bin/env bash
#
# Add a codex-vitals panel pane to the bottom of the current tmux window.
#
#   scripts/tmux-hud.sh                 # size m (default), refreshing every 1s
#   scripts/tmux-hud.sh -size l
#   scripts/tmux-hud.sh -size xs
#   CODEX_VITALS_TMUX_HEIGHT=8 scripts/tmux-hud.sh -size xl
#
# Run it from inside a tmux session (e.g. the one where Codex is running).
# Focus returns to your original pane; Ctrl+C in the panel pane closes it.
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

if [ "$#" -gt 0 ]; then
  set -- "$@"
else
  set -- -interval 1s
fi

# Derive the requested size so the pane can be sized to fit its line count.
# Falls back to CODEX_VITALS_SIZE (what the binary itself defaults to).
size_file="${XDG_CONFIG_HOME:-$HOME/.config}/codex-vitals/size"
size="$(cat "$size_file" 2>/dev/null | tr -d '[:space:]' || true)"
size="${size:-${CODEX_VITALS_SIZE:-m}}"
prev=""
for arg in "$@"; do
  case "$arg" in
    -size=*) size="${arg#-size=}" ;;
  esac
  if [ "$prev" = "-size" ]; then
    size="$arg"
  fi
  prev="$arg"
done

case "$size" in
  xs | xsmall | s | small) default_height=4 ;;
  l | large | xl | xlarge) default_height=7 ;;
  *) default_height=6 ;;
esac
height="${CODEX_VITALS_TMUX_HEIGHT:-$default_height}"

# Split a bottom pane running the panel, then return focus to the original pane.
# mouse off면 휠 스크롤이 화살표 키로 변환돼 대화 스크롤백을 볼 수 없으므로 안내만 출력한다.
# (사용자 설정을 강제로 바꾸지 않는다 — 텍스트 선택 동작이 달라질 수 있어 opt-in으로 남긴다.)
if [ "$(tmux show -gv mouse 2>/dev/null)" = "off" ]; then
  echo "hint: tmux 'mouse' is off — add 'set -g mouse on' to ~/.tmux.conf so the wheel scrolls the conversation" >&2
fi

tmux split-window -v -l "$height" "$bin $*"
tmux last-pane
