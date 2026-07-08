## Why

The shpell window name `claude-notify` is long and clutters tmux window lists and capture-pane references. Shortening it to `cn` reduces visual noise and makes scripting against the window name more ergonomic.

## What Changes

- The shpell name argument passed to `custom_shpell` in `tmux-claude-notify.tmux` changes from `claude-notify` to `cn`
- Any tmux window named `claude-notify` (placeholder and dashboard) is now named `cn`
- Documentation updated to reflect the new window name in all examples

## Capabilities

### New Capabilities

- None

### Modified Capabilities

- `tpm-entry-point`: The shpell name argument in the keybinding command changes from `claude-notify` to `cn`, altering the tmux window name used for the dashboard popup.

## Impact

- `tmux-claude-notify.tmux`: one-line change to the `bind-key` call
- `CLAUDE.md`: update layout diagram and capture-pane examples that reference `claude-notify` window name
- `architecture.md`: update any diagram nodes referencing the window name
- No Go binary changes required — the window name is set by `custom_shpell` from the argument, not hard-coded in the binary
