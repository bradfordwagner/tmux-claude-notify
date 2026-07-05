# CLAUDE.md

## What this repo is

A TPM (tmux plugin manager) plugin that provides persistent visual notifications in tmux when an interactive `claude` session is waiting for user input. The indicator stays until the user responds — no timeout.

> **Non-goal**: background agent / fleet-view integration (`claude agents`). The plugin targets foreground `claude` sessions only.

## Plugin structure (TPM convention)

```
tmux-claude-notify.tmux        # main plugin entry point (sourced by TPM, sets keybindings + hooks)
scripts/
  notify.sh                    # called by Claude Code Stop hook; highlights window + notify-send
  clear.sh                     # clears the waiting indicator (called on pane focus-in)
openspec/                      # spec-driven development artifacts
```

A Go binary is an option if the shell logic grows complex or performance matters.

## How it works

When `claude` finishes a response and returns to the prompt, the Claude Code `Stop` hook fires.
The hook script:
1. Reads `$TMUX_PANE` (inherited env from the shell that launched `claude`) to identify the window
2. Sets a persistent window-tab highlight in the tmux status bar
3. Fires `notify-send` for a desktop notification
4. Registers a one-shot `pane-focus-in` hook to clear the indicator when the user returns to that pane

### Claude Code hook setup (in ~/.claude/settings.json)

```json
{
  "hooks": {
    "Stop": [{"type": "command", "command": "/path/to/scripts/notify.sh"}]
  }
}
```

### Key env vars available in hook scripts

```bash
$TMUX_PANE    # e.g. %34 — inherited from shell that launched claude; identifies the window
$TMUX         # socket path — confirms we're inside a tmux session
```

## Dotfiles integration

Used in bradfordwagner's devtainer dotfiles:
- TPM entry in `dots/tmux/tmux.conf`: `set -g @plugin 'bradfordwagner/tmux-claude-notify'`
- Stop hook wired in `~/.claude/settings.json` (Ansible-managed)
- Keybinding `C-M-p` to jump to the waiting window (free in existing `C-M-*` space; mnemonic: "prompt")

### Existing tmux context

- Status bar `@status_bg`: `#000000` (black)
- Grimoire accent color `#AD8EE6` — use for the waiting indicator to match the scheme
- `status-right` currently: `#[fg=#cdd6f4 bg=#{@status_bg}][#{session_id}:#{session_name}]`
- tmux-pop installed — can trigger a flash via pane switch on arrival
- Prefix: `ctrl+space`

### Taken `C-M-*` bindings (do not reuse)

`a c d f g h i j k l n o s u w y Space` — all taken. Free right-hand options: `p ; m , . /`

`C-M-p` is the chosen binding.

## Conventions

- Shell scripts: `#!/usr/bin/env bash`; hook scripts must not exit non-zero noisily (tmux logs errors)
- `notify-send` for Linux/WSL; guard with `command -v notify-send` for portability
- TPM plugin entry point stays thin — logic lives in `scripts/`
- No timeouts on the visual indicator; persists until user acknowledges
