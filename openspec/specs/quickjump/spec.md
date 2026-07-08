# Spec: quickjump

## Purpose

The `claude-notify jump` subcommand provides a non-interactive path to navigate directly to the oldest waiting pane and clear its notification, without opening the dashboard.

## Requirements

### Requirement: jump subcommand navigates to oldest waiting pane
The `claude-notify jump` subcommand SHALL find the uncleared notification with the smallest `ts` (oldest), navigate to that pane, and clear the notification. If no uncleared notifications exist, it SHALL exit 0 silently.

#### Scenario: Single notification waiting
- **WHEN** `claude-notify jump` is run and one uncleared entry exists
- **THEN** the notified pane is focused via `SelectPane` + `SwitchToWindow`
- **AND** the notification is cleared (JSONL marked cleared, window styles unset if no siblings remain)
- **AND** exit code is 0

#### Scenario: Multiple notifications waiting — oldest selected
- **WHEN** `claude-notify jump` is run and multiple uncleared entries exist
- **THEN** the entry with the smallest `ts` value is selected
- **AND** that pane is focused and cleared

#### Scenario: No notifications — silent exit
- **WHEN** `claude-notify jump` is run and no uncleared entries exist
- **THEN** no tmux commands are issued
- **AND** exit code is 0

### Requirement: @claude-notify-jump-key TPM option binds the jump command
The TPM option `@claude-notify-jump-key` SHALL control the keybinding for `claude-notify jump`, defaulting to `C-M-\;`. The binding SHALL always use `run-shell` (not grimoire shpell) since jump is non-interactive.

#### Scenario: Default keybinding
- **WHEN** `@claude-notify-jump-key` is not set in `tmux.conf`
- **THEN** `C-M-\;` is bound to `run-shell '<binary> jump'`

#### Scenario: Custom keybinding
- **WHEN** `set -g @claude-notify-jump-key 'C-M-n'` is in `tmux.conf`
- **THEN** `C-M-n` is bound to `run-shell '<binary> jump'`

#### Scenario: Jump clears notification identically to dashboard enter
- **WHEN** `claude-notify jump` clears a notification
- **THEN** the JSONL entry is marked cleared
- **AND** window-status-style is unset if no siblings remain (same behavior as dashboard enter handler)
- **AND** pane pop (`window-style`) is unset
