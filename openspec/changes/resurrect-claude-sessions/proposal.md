## Why

After a system reboot or tmux-resurrect restore, all tmux panes are dropped back to bare shells — any `claude` session that was running is lost. This plugin already tracks every active claude session (project path + session ID) in `sessions.jsonl`, making it the natural place to implement save/restore so users can get back into interrupted claude sessions in one command.

## What Changes

- New `claude-notify resurrect save` subcommand: snapshots all tmux panes currently running `claude*`, writing session ID + project path + positional key (session/window/pane index) to a JSON sidecar file (`~/.local/share/tmux-claude-notify/resurrect.json`).
- New `claude-notify resurrect restore` subcommand: reads the sidecar, matches saved entries to live panes by positional key, and sends `claude --resume <session-id>` into each matching pane.
- Integration hook: the save subcommand is wired into tmux-resurrect's `@resurrect-save-shell-history yes` hook or can be called directly by the user or by a pre-defined `@resurrect-save-hook`.
- TPM option `@claude-notify-resurrect-file` to override the default sidecar path.

## Capabilities

### New Capabilities

- `resurrect-save`: Scan all tmux panes for running `claude*` processes, persist positional key + session ID + project path to the resurrect sidecar.
- `resurrect-restore`: Load the sidecar, match entries to live panes by positional key, replay `claude --resume <session-id>` into each matched pane (skip missing panes silently).

### Modified Capabilities

- `session-index`: The `sessions.jsonl` index is the authoritative source for `session_id` and `project_path` during save — resurrect-save reads it rather than re-deriving those fields from scratch.

## Impact

- New file: `internal/resurrect/resurrect.go` (save/restore logic + sidecar types)
- New subcommands in `cmd/claude-notify/main.go`: `resurrect save` and `resurrect restore`
- `sessions.jsonl` read (read-only) during save
- New sidecar file at `~/.local/share/tmux-claude-notify/resurrect.json`
- No changes to existing notification, watcher, or TUI paths
- `README.md` gains a Resurrect section and a `@claude-notify-resurrect-file` entry in the Configuration table
