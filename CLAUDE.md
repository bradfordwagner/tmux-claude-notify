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
  store/store.go               # JSONL notification log (Append, ReadAll, ClearPane, HasUnclearedPane, UpdateStatus)
  tmux/tmux.go                 # tmux command helpers
  setup/setup.go               # ~/.claude/settings.json hook check + auto-configure
  watcher/watcher.go           # transcript file watcher: state derivation, pane correlation
  ui/model.go                  # bubbletea dashboard TUI (embeds transcript watcher)
Taskfile.yml                   # build / dev / test / lint / setup tasks
architecture.md                # canonical architecture diagram (update with every change)
DEVELOPMENT.md                 # ordered development items and status
openspec/                      # spec-driven development artifacts
```

The binary owns the full hook path — there are no bash scripts in the notification or clear flow.

## How it works

Notifications flow from two sources: the Stop hook (always active) and the transcript watcher (active while the dashboard is open).

### Stop hook (fallback)

When `claude` finishes a response and returns to the prompt, the Claude Code `Stop` hook fires.
`bin/claude-notify notify` (invoked by the Stop hook):
1. Reads `$TMUX_PANE` (inherited env from the shell that launched `claude`) to identify the window
2. Checks `~/.local/share/tmux-claude-notify/notifications.jsonl` for an existing uncleared entry for this pane — if one exists, re-applies styles and returns (idempotent; prevents duplicate entries from rapid hook firings)
3. Sets `window-status-style` and `window-status-current-style` to `fg=#AD8EE6,bold` on the window
4. Sets `window-active-style bg=<color>` (pane background pop)
5. Fires `notify-send` for a desktop notification (if available)
6. Appends a record to `~/.local/share/tmux-claude-notify/notifications.jsonl` with `status:"waiting"`

### Transcript watcher (while dashboard is open)

When the dashboard TUI starts, it launches a transcript watcher that reads Claude Code's own JSONL session files at `~/.claude/projects/<encoded-path>/<session-id>.jsonl`. The watcher:
- Discovers panes running `claude*` via `tmux list-panes`
- Encodes each pane's `pane_current_path` (replace `/` and `.` with `-`) to locate the project dir
- Watches the latest transcript JSONL via fsnotify
- Derives agent state from the last 20 transcript events:
  - `assistant` + `tool_use` content → `running`
  - `assistant` + `stop_reason:"end_turn"` → `waiting`
  - `user` + `tool_result` content → `running` (tool output returning to Claude)
  - `user` + text content → user responded → **clear notification**
  - No activity for `@claude-notify-stale-minutes` (default 5) → `stale`
- On `waiting`: highlights window tab; creates or updates JSONL entry
- On `running`/`stale`: updates JSONL status only (no window highlight)
- On user response: clears JSONL entry and window styles

### Reconciliation on dashboard open

On startup, `watcher.Reconcile()` scans all active transcripts and corrects any JSONL entries that changed while the dashboard was closed (e.g., clears entries where the user already responded via the Stop-hook period).

Notifications are cleared in three ways: (1) selecting them in the dashboard, (2) the auto-reset subprocess clearing after `@claude-notify-active-reset-seconds` (default 15s) when the notified pane was already focused at notify time, or (3) the same subprocess clearing after `@claude-notify-nav-clear-seconds` (default 2s) when the user navigates to the notified pane. The subprocess polls `#{window_active}#{pane_active}` every 2s to detect pane-level focus — pane-focus-in hooks are not used (broken in WSL2; after-select-window fires on any tmux command).

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
└── window: cn                     ← placeholder window (swapped in while popup is open)

_shpell-session                    ← temporary session, lives only while popup is open
└── window: cn                     ← where bin/claude-notify actually runs
    └── pane                       ← bubbletea TUI renders here

[display-popup overlay]            ← what you see: attaches to _shpell-session
```

The window is swapped between `_shpell-session` and your session via `tmux swap-window`. When you toggle off (`C-M-p` again), a hook swaps it back and kills `_shpell-session`. The placeholder window preserves pane history.

### Capturing the shpell buffer

```bash
# While shpell popup is open — reads the live TUI output:
tmux capture-pane -t "_shpell-session:cn" -p

# From your own session when popup is closed — reads the placeholder window:
tmux capture-pane -t "$(tmux list-windows -F '#{session_name}:#{window_name}' | grep ':cn$')" -p

# To identify which pane the dashboard is showing as selected/active:
# Parse the captured buffer — the selected entry is prefixed with "> " by the TUI
tmux capture-pane -t "_shpell-session:cn" -p | grep "^> "
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
- TPM plugin entry point stays thin — all logic lives in the Go binary
- No timeouts on the visual indicator; persists until user acknowledges
- **All changes must update architecture diagrams** — any modification to data flow, component boundaries, or hook wiring must be reflected in the relevant diagram(s) before the change is considered complete
- **New or modified `@claude-notify-*` TPM options must be documented in `README.md`** — add or update the row in the Configuration table with the option name, default value, and description
