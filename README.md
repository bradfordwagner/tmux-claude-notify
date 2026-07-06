# tmux-claude-notify

A [TPM](https://github.com/tmux-plugins/tpm) plugin that provides persistent visual notifications in tmux when an interactive `claude` (Claude Code) session needs your attention.

The indicator stays until you acknowledge it — no timeout, no auto-dismiss.

## What it does

- **Tab highlight** — the tmux window tab turns purple (`#AD8EE6`) when claude is waiting for input
- **Pane background pop** — the pane background changes color to draw your eye (configurable)
- **Dashboard** — a bubbletea TUI listing all pending notifications with session, window, status, and age
- **Desktop notification** — fires `notify-send` if available (Linux/WSL)
- **Transcript watcher** — reads Claude Code's own JSONL session files to derive richer state (`running`, `waiting`, `stale`) in near-realtime while the dashboard is open
- **Reconciliation** — on dashboard open, corrects any stale notifications from while the dashboard was closed
- **Auto-configures** — wires Claude Code hooks into `~/.claude/settings.json` on first launch

Notifications come from two sources: the `Stop` hook (always active, fires at end of each turn) and the transcript watcher (active while dashboard is open, provides live state).

## Requirements

- tmux
- Go (build-time only — compiled automatically by the TPM entry point)
- Claude Code CLI

## Installation

### TPM

Add to your `tmux.conf`:

```tmux
set -g @plugin 'bradfordwagner/tmux-claude-notify'
```

Then press `prefix + I` to install.

### Manual

```bash
git clone https://github.com/bradfordwagner/tmux-claude-notify \
  ~/.tmux/plugins/tmux-claude-notify
~/.tmux/plugins/tmux-claude-notify/tmux-claude-notify.tmux
```

## Hook setup

The plugin auto-configures `~/.claude/settings.json` on first dashboard open. To configure manually:

```json
{
  "hooks": {
    "Stop": [{"matcher": "", "hooks": [{"type": "command", "command": "/path/to/bin/claude-notify notify"}]}]
  }
}
```

Replace `/path/to/bin/claude-notify` with the actual path (e.g. `~/.tmux/plugins/tmux-claude-notify/bin/claude-notify`).

## Usage

Press `C-M-p` (default) to open the notification dashboard. Select an entry with enter to jump to that window and clear the notification.

| Action | Result |
|---|---|
| Claude finishes a response | Tab highlights; pane background changes |
| Press `C-M-p` | Dashboard opens showing all pending notifications |
| Select an entry (enter) | Switches to that window, clears the notification, closes popup |
| `q` / `esc` | Close dashboard without clearing |

## Configuration

| Option | Default | Description |
|---|---|---|
| `@claude-notify-key` | `C-M-p` | Keybinding to open the dashboard |
| `@claude-notify-pop-color` | `#1e1e2e` | Pane background color when waiting (falls back to `@tmux-pop-color`, then `#1e1e2e`) |
| `@claude-notify-stale-minutes` | `5` | Minutes of transcript inactivity before marking a session stale |
| `@claude-notify-active-reset-seconds` | `15` | Seconds before auto-clearing a notification when the notified pane is already focused. Set to `0` to disable auto-reset. |

## Grimoire integration

If [tmux-grimoire](https://github.com/bradfordwagner/tmux-grimoire) is installed, the dashboard opens as a toggleable shpell popup — press `C-M-p` again to close it. Without grimoire, a standard `display-popup` is used.

## Local development

```bash
task build   # compile bin/claude-notify
task r       # build, clear screen, and run the dashboard
task test    # run tests
```

## Reference

- [architecture.md](architecture.md) — component diagram, data flow, design decisions
- [DEVELOPMENT.md](DEVELOPMENT.md) — implementation status and task list
