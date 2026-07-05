# CLAUDE.md

## What this repo is

A TPM (tmux plugin manager) plugin that provides persistent visual notifications in tmux when an interactive `claude` session is waiting for user input. The indicator stays until the user responds — no timeout.

> **Non-goal**: background agent / fleet-view integration (`claude agents`). The plugin targets foreground `claude` sessions only.

## Plugin structure (TPM convention)

```
tmux-claude-notify.tmux        # TPM entry point: compiles binary, registers keybinding
bin/claude-notify              # compiled Go binary (gitignored)
cmd/claude-notify/main.go      # binary entry point + subcommand routing
internal/
  store/store.go               # JSONL notification log (Append, ReadAll, ClearPane)
  tmux/tmux.go                 # tmux command helpers
  setup/setup.go               # ~/.claude/settings.json hook check
  ui/model.go                  # bubbletea dashboard TUI
Taskfile.yml                   # build / dev / test / lint / setup tasks
architecture.md                # canonical architecture diagram (update with every change)
DEVELOPMENT.md                 # ordered development items and status
openspec/                      # spec-driven development artifacts
```

The binary owns the full hook path — there are no bash scripts in the notification or clear flow.

## How it works

When `claude` finishes a response and returns to the prompt, the Claude Code `Stop` hook fires.
`bin/claude-notify notify` (invoked by the Stop hook):
1. Reads `$TMUX_PANE` (inherited env from the shell that launched `claude`) to identify the window
2. Sets a persistent window-tab highlight (`#AD8EE6,bold`) in the tmux status bar
3. Fires `notify-send` for a desktop notification (if available)
4. Registers a one-shot `pane-focus-in` hook calling `claude-notify clear --pane <id>` to clear the indicator
5. Appends a record to `~/.local/share/tmux-claude-notify/notifications.jsonl`

### Claude Code hook setup (in ~/.claude/settings.json)

The binary auto-configures this on first run, but the correct schema is:
```json
{
  "hooks": {
    "Stop": [{"matcher": "", "hooks": [{"type": "command", "command": "/path/to/bin/claude-notify notify"}]}]
  }
}
```

### Key env vars available in hook scripts

```bash
$TMUX_PANE    # e.g. %34 — inherited from shell that launched claude; identifies the window
$TMUX         # socket path — confirms we're inside a tmux session
```

## Grimoire shpell layout and buffer capture

`C-M-p` launches the dashboard inside a grimoire shpell. Understanding the layout matters for scripting against it.

```
Your session (e.g. "main")
├── window 0: vim                  ← normal work
└── window: claude-notify          ← placeholder window (swapped in while popup is open)

_shpell-session                    ← temporary session, lives only while popup is open
└── window: claude-notify          ← where bin/claude-notify actually runs
    └── pane                       ← bubbletea TUI renders here

[display-popup overlay]            ← what you see: attaches to _shpell-session
```

The window is swapped between `_shpell-session` and your session via `tmux swap-window`. When you toggle off (`C-M-p` again), a hook swaps it back and kills `_shpell-session`. The placeholder window preserves pane history.

### Capturing the shpell buffer

```bash
# While shpell popup is open — reads the live TUI output:
tmux capture-pane -t "_shpell-session:claude-notify" -p

# From your own session when popup is closed — reads the placeholder window:
tmux capture-pane -t "$(tmux list-windows -F '#{session_name}:#{window_name}' | grep ':claude-notify$')" -p

# To identify which pane the dashboard is showing as selected/active:
# Parse the captured buffer — the selected entry is prefixed with "> " by the TUI
tmux capture-pane -t "_shpell-session:claude-notify" -p | grep "^> "
```

The `> ` prefix identifies the currently highlighted entry. Use this to script "jump to active claude pane" without user interaction.

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
- **All changes must update architecture diagrams** — any modification to data flow, component boundaries, or hook wiring must be reflected in the relevant diagram(s) before the change is considered complete
