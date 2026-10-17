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

# Dashboard launcher: a display-popup running the binary directly. CN_CLIENT (the
# invoking client) lets it switch the right terminal. display-popup does not
# expand formats in its command, run-shell does: it bakes the client name in.
tmux bind-key -n "$key" run-shell \
  "tmux display-popup -c '#{client_name}' -E -w 80% -h 80% -b rounded -T ' claude-notify ' \"CN_CLIENT='#{client_name}' '$BINARY'\""

# Quick-jump keybinding: navigates directly to the oldest waiting pane (no dashboard).
jump_key=$(tmux show-option -gqv "@claude-notify-jump-key")
jump_key="${jump_key:-C-M-\;}"
tmux bind-key -n "$jump_key" run-shell "'$BINARY' jump"

# Status-bar segment: appended by default; set @claude-notify-statusline 0 to disable.
statusline=$(tmux show-option -gqv "@claude-notify-statusline")
statusline="${statusline:-1}"
# Strip any existing copies, then append exactly one: re-sourcing tmux.conf
# re-runs this script, and a blind -ga stacked duplicates ("⚡ 5 ⚡ 5").
segment=" #('$BINARY' status)"
status_right="$(tmux show-option -gqv status-right)"
status_right="${status_right//"$segment"/}"
if [[ "$statusline" != "0" ]]; then
  status_right="$status_right$segment"
fi
tmux set-option -g status-right "$status_right"
