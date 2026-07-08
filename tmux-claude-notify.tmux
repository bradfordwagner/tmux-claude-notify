#!/usr/bin/env bash

PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY="$PLUGIN_DIR/bin/claude-notify"

# Compile if binary is missing or source is newer
needs_build() {
  [[ ! -f "$BINARY" ]] && return 0
  find "$PLUGIN_DIR/cmd" "$PLUGIN_DIR/internal" -name "*.go" -newer "$BINARY" 2>/dev/null | grep -q . && return 0
  return 1
}

if needs_build; then
  if ! (cd "$PLUGIN_DIR" && go build -o "$BINARY" ./cmd/claude-notify 2>&1); then
    tmux display-message -d 0 "tmux-claude-notify: build failed — check Go install"
    exit 1
  fi
fi

# Read configurable keybinding (default: C-M-p)
key=$(tmux show-option -gqv "@claude-notify-key")
key="${key:-C-M-p}"

# Use grimoire's custom_shpell if installed (check by path to avoid TPM load-order issues).
# Grimoire handles toggle by detecting _shpell-session and calling detach-client.
# Fall back to tmux popup if grimoire is not present.
GRIMOIRE_SHPELL="${HOME}/.tmux/plugins/tmux-grimoire/bin/custom_shpell"
if [[ -x "$GRIMOIRE_SHPELL" ]]; then
  tmux bind-key -n "$key" run-shell "$GRIMOIRE_SHPELL standard cn '$BINARY' --replay"
else
  tmux bind-key -n "$key" popup -E -w 80% -h 80% "$BINARY"
fi

# Quick-jump keybinding: navigates directly to the oldest waiting pane (no dashboard).
jump_key=$(tmux show-option -gqv "@claude-notify-jump-key")
jump_key="${jump_key:-C-M-\;}"
tmux bind-key -n "$jump_key" run-shell "'$BINARY' jump"

# Status-bar segment: appended by default; set @claude-notify-statusline 0 to disable.
statusline=$(tmux show-option -gqv "@claude-notify-statusline")
statusline="${statusline:-1}"
if [[ "$statusline" != "0" ]]; then
  tmux set-option -ga status-right " #('$BINARY' status)"
fi
