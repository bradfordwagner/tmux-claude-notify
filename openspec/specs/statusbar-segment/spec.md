# Spec: statusbar-segment

## Purpose

The `claude-notify status` subcommand and the `@claude-notify-statusline` TPM option together provide a passive notification count in the tmux status bar.

## Requirements

### Requirement: status subcommand outputs notification count
The `claude-notify status` subcommand SHALL print a formatted count of uncleared notifications to stdout and exit 0. When no uncleared notifications exist, it SHALL print nothing (empty output).

#### Scenario: Notifications pending
- **WHEN** `claude-notify status` is run and the JSONL store contains 2 uncleared entries
- **THEN** stdout is `⚡ 2`
- **AND** exit code is 0

#### Scenario: No notifications pending
- **WHEN** `claude-notify status` is run and all entries are cleared
- **THEN** stdout is empty
- **AND** exit code is 0

#### Scenario: JSONL file missing
- **WHEN** `claude-notify status` is run and `notifications.jsonl` does not exist
- **THEN** stdout is empty
- **AND** exit code is 0

### Requirement: @claude-notify-statusline TPM option controls status-right segment
The `status-right` segment is appended by default. `tmux-claude-notify.tmux` SHALL append `#(<binary-path> status)` to `status-right` unless `@claude-notify-statusline` is explicitly set to `0`.

#### Scenario: Option absent — segment appended (default on)
- **WHEN** `@claude-notify-statusline` is not set in `tmux.conf`
- **THEN** `tmux-claude-notify.tmux` appends ` #(/path/to/bin/claude-notify status)` to `status-right`
- **AND** tmux polls it on every status-right refresh

#### Scenario: Option set to 0 — segment suppressed
- **WHEN** `set -g @claude-notify-statusline 0` is in `tmux.conf`
- **THEN** `tmux-claude-notify.tmux` does NOT modify `status-right`

#### Scenario: Segment disappears when no notifications
- **WHEN** `claude-notify status` prints empty output
- **THEN** the segment contributes nothing visible to the status bar
