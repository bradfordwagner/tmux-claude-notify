## Why

There is no working plugin yet — only a CLAUDE.md describing intent. Before writing any scripts we need to understand whether the standard Claude Code `Stop` hook environment actually provides `$TMUX_PANE`, how tmux window highlighting works reliably across sessions, and where edge cases (non-tmux terminals, multiple simultaneous claude panes) will bite us.

## What Changes

- Introduce `tmux-claude-notify.tmux` — TPM entry point that registers keybindings and the pane-focus-in clear hook
- Introduce `scripts/notify.sh` — fired by the Claude Code `Stop` hook; sets window-tab highlight and calls `notify-send`
- Introduce `scripts/clear.sh` — registered as a one-shot `pane-focus-in` hook; clears the highlight when the user returns to the pane
- Introduce architecture diagrams documenting component boundaries, hook wiring, and data flow

## Capabilities

### New Capabilities

- `window-highlight`: Persist a visual accent (`#AD8EE6`) on the tmux window tab when claude is waiting for input; clear on pane focus
- `desktop-notification`: Fire `notify-send` from the Stop hook with a configurable message; guard with `command -v` for portability
- `jump-to-waiting`: `C-M-p` keybinding that switches to the most-recently-notified window
- `tpm-entry-point`: TPM-compliant `.tmux` entry point that wires all hooks and keybindings on plugin load

### Modified Capabilities

<!-- none — this is a greenfield plugin -->

## Impact

- New files: `tmux-claude-notify.tmux`, `cmd/claude-notify/main.go` (Go binary source), `architecture.md`, `DEVELOPMENT.md`, `README.md`, `Taskfile.yml`
- Consumer: `~/.claude/settings.json` Stop hook must point at the compiled `bin/claude-notify notify`
- Consumer: `dots/tmux/tmux.conf` must load the plugin via TPM
- External dependencies: `go` (build-time), `tmux`, optionally `notify-send`
