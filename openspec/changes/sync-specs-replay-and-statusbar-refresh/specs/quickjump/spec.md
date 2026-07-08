## MODIFIED Requirements

### Requirement: jump subcommand navigates to oldest waiting pane
The `claude-notify jump` subcommand SHALL find the uncleared notification with the smallest `ts` (oldest), navigate to that pane, clear the notification, and immediately refresh the tmux status bar via `tmux refresh-client -S`. If no uncleared notifications exist, it SHALL exit 0 silently.

#### Scenario: Single notification waiting
- **WHEN** `claude-notify jump` is run and one uncleared entry exists
- **THEN** the notified pane is focused via `SelectPane` + `SwitchToWindow`
- **AND** the notification is cleared (JSONL marked cleared, window styles unset if no siblings remain)
- **AND** the status bar is immediately refreshed so the badge disappears without waiting for `status-interval`
- **AND** exit code is 0

#### Scenario: Multiple notifications waiting — oldest selected
- **WHEN** `claude-notify jump` is run and multiple uncleared entries exist
- **THEN** the entry with the smallest `ts` value is selected
- **AND** that pane is focused and cleared

#### Scenario: No notifications — silent exit
- **WHEN** `claude-notify jump` is run and no uncleared entries exist
- **THEN** no tmux commands are issued
- **AND** exit code is 0

#### Scenario: Status bar refreshes immediately after jump
- **WHEN** `claude-notify jump` clears the last uncleared notification
- **THEN** `tmux refresh-client -S` is called before exit
- **AND** the status bar badge disappears immediately regardless of `status-interval`

### Requirement: Jump clears notification identically to dashboard enter
- **WHEN** `claude-notify jump` clears a notification
- **THEN** the JSONL entry is marked cleared
- **AND** window-status-style is unset if no siblings remain (same behavior as dashboard enter handler)
- **AND** pane pop (`window-style`) is unset
- **AND** the status bar is immediately refreshed via `tmux refresh-client -S`
