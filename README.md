# tmux-claude-notify

A [TPM](https://github.com/tmux-plugins/tpm) plugin that provides persistent visual notifications in tmux when an interactive `claude` (Claude Code) session needs your attention.

The indicator stays until you acknowledge it — no timeout, no auto-dismiss.

## What it does

- **Tab highlight** — the tmux window tab turns purple (`#AD8EE6`) when claude is waiting for input
- **Pane background pop** — the pane background changes color to draw your eye (configurable)
- **Dashboard** — a bubbletea TUI with two views: a Notifications view (pending notifications) and a Sessions view (all discovered Claude sessions, browseable and resumable)
- **Fuzzy search** — press `/` in the dashboard to filter any view in real time
- **Session browser** — two-level drill-in (projects → sessions) with pin, filter, and open/resume via `w`/`h`/`v`
- **Desktop notification** — fires `notify-send` if available (Linux/WSL)
- **Transcript watcher** — reads Claude Code's own JSONL session files to derive richer state (`running`, `waiting`, `stale`) in near-realtime while the dashboard is open
- **Auto-clear on navigation** — navigating to a popped pane clears it automatically after `@claude-notify-nav-clear-seconds` (default 2s); no dashboard interaction needed
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

No manual action required. On first dashboard open, `claude-notify` automatically adds the `Stop` hook to `~/.claude/settings.json` and shows a brief toast confirmation.

For dotfile-managed setups (Ansible, chezmoi, etc.), the hook entry looks like:

```json
{
  "hooks": {
    "Stop": [{"matcher": "", "hooks": [{"type": "command", "command": "/path/to/bin/claude-notify notify"}]}]
  }
}
```

Replace `/path/to/bin/claude-notify` with the actual path (e.g. `~/.tmux/plugins/tmux-claude-notify/bin/claude-notify`).

## Usage

Press `C-M-p` (default) to open the notification dashboard.

### Notifications view (default)

| Action | Result |
|---|---|
| Claude finishes a response | Tab highlights; pane background changes |
| Press `C-M-p` | Dashboard opens showing all pending notifications |
| `enter` on an entry | Switches to that window, clears the notification, closes popup |
| `q` / `esc` | Close dashboard without clearing |
| `/` | Enter search mode — fuzzy-filter the list in real time |
| `Tab` (in search) | Toggle keyboard focus between the filter input and the table |

### Sessions view

Press `Tab` to switch between the Notifications view and the Sessions view. The Sessions view is a two-level browser: Level-1 lists projects, Level-2 lists individual sessions within a project.

| Action | Result |
|---|---|
| `Tab` | Toggle between Notifications and Sessions views |
| `enter` (L1) | Drill into a project's session list (Level-2) |
| `esc` (L2) | Return to Level-1 |
| `esc` (L1) / `q` | Close dashboard |
| `p` (L2) | Toggle pin on the selected session |
| `f` | Toggle filter: show only sessions with an active tmux pane |
| `w` (L1) | Open a new `claude` session in the selected project (new window) |
| `h` / `v` (L1) | Open a new `claude` session in a horizontal/vertical split |
| `w` (L2, closed session) | Resume the session with `claude --resume <id>` in a new window |
| `h` / `v` (L2, closed session) | Resume in a horizontal/vertical split |
| `w` (L2, active session) | Focus the session's existing pane and close the popup |
| `/` | Fuzzy-filter the session list |

Pinned sessions (`p`) float to the top of the Sessions view and also appear in the Notifications view even when their pane is not active.

## Configuration

| Option | Default | Description |
|---|---|---|
| `@claude-notify-key` | `C-M-p` | Keybinding to open the dashboard |
| `@claude-notify-pop-color` | `#1e1e2e` | Pane background color when waiting (falls back to `@tmux-pop-color`, then `#1e1e2e`) |
| `@claude-notify-stale-minutes` | `5` | Minutes of transcript inactivity before marking a session stale |
| `@claude-notify-transcript-age-days` | `14` | Maximum age in days of transcript files considered when discovering sessions. Applies to both the dashboard watcher and `resurrect save`. |
| `@claude-notify-active-reset-seconds` | `15` | Seconds before auto-clearing a notification when the notified pane is already focused. Set to `0` to disable. |
| `@claude-notify-nav-clear-seconds` | `2` | Seconds before auto-clearing a notification when you navigate to its window via tmux. Set to `0` to disable. |

## Resurrect integration

After a reboot or tmux-resurrect restore, panes are dropped back to bare shells. The resurrect commands let you resume all interrupted claude sessions in one step.

```bash
# Save a snapshot of all currently running claude sessions
~/.tmux/plugins/tmux-claude-notify/bin/claude-notify resurrect save

# Replay "claude --resume <id>" into each matching pane after tmux-resurrect restores
~/.tmux/plugins/tmux-claude-notify/bin/claude-notify resurrect restore
```

### Wiring into tmux-resurrect hooks

Add these lines to your `tmux.conf` (append to any existing hook values):

```tmux
set -g @resurrect-hook-pre-save '~/.tmux/plugins/tmux-claude-notify/bin/claude-notify resurrect save'
set -g @resurrect-hook-post-restore-all '~/.tmux/plugins/tmux-claude-notify/bin/claude-notify resurrect restore'
```

If you already have other commands in those hooks, chain them with `;`:

```tmux
set -g @resurrect-hook-pre-save 'other-save-cmd; ~/.tmux/plugins/tmux-claude-notify/bin/claude-notify resurrect save'
set -g @resurrect-hook-post-restore-all '~/.tmux/plugins/tmux-claude-notify/bin/claude-notify resurrect restore'
```

The sidecar file (`resurrect.json`) is gitignored and lives at `~/.local/share/tmux-claude-notify/resurrect.json`.

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
