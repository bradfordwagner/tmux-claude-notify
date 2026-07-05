# tmux-claude-notify

A [TPM](https://github.com/tmux-plugins/tpm) plugin that provides persistent visual notifications in tmux when an interactive `claude` session is waiting for user input.

When Claude Code finishes a response, your tmux window tab lights up in a distinctive color. Press the configured keybinding to open a dashboard showing all pending notifications and jump to any waiting session.

## Features

- Window tab highlight (`#AD8EE6`) persists until you return to the pane
- Optional desktop notification via `notify-send`
- Interactive dashboard (bubbletea TUI) listing all pending notifications sorted by recency
- Configurable keybinding via TPM option
- No external runtime dependencies beyond `tmux` (and optionally `notify-send`)

## Requirements

- tmux
- Go (build-time only — for compiling the binary)
- `notify-send` (optional, for desktop notifications)

## Installation

### 1. Add to TPM in `tmux.conf`

```tmux
set -g @plugin 'bradfordwagner/tmux-claude-notify'

# Optional: override the default keybinding (default is M-p / C-M-p)
# set -g @claude-notify-key 'M-p'
```

### 2. Wire the Stop hook in `~/.claude/settings.json`

```json
{
  "hooks": {
    "Stop": [
      {
        "type": "command",
        "command": "~/.tmux/plugins/tmux-claude-notify/bin/claude-notify notify"
      }
    ]
  }
}
```

### 3. Reload TPM

Press `<prefix> + I` to install and compile the plugin.

---

## Local Development Installation

To use a local checkout instead of the TPM-installed version:

```bash
task setup   # links repo into ~/.tmux/plugins/tmux-claude-notify and verifies deps
task build   # compile bin/claude-notify
```

`task setup` creates a symlink at `~/.tmux/plugins/tmux-claude-notify` pointing at your working directory. TPM will use it as-is, and `task build` recompiles in place. No need to reinstall via TPM after code changes — just rebuild.

Add to `tmux.conf` using the local path directly if you want to bypass TPM entirely:

```tmux
run-shell "~/.tmux/plugins/tmux-claude-notify/tmux-claude-notify.tmux"
```

## Usage

| Action | What happens |
|---|---|
| Claude finishes a response | Window tab highlights in `#AD8EE6`; optional desktop notification fires |
| Focus the notified pane | Highlight clears automatically |
| Press `C-M-p` (or your configured key) | Opens the notification dashboard |
| Select an entry in the dashboard | Switches to that window and clears the notification |

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for implementation order and status.
See [architecture.md](architecture.md) for the full architecture diagram and design decisions.

### Quick start

```bash
task setup   # verify Go and tmux are on PATH
task build   # compile bin/claude-notify
task test    # run tests
```
