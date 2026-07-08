## MODIFIED Requirements

### Requirement: jump subcommand navigates to oldest waiting pane
The `claude-notify jump` subcommand SHALL find the uncleared notification with the smallest `ts` (oldest), navigate to that pane, clear the notification, and immediately refresh the tmux status bar via `tmux refresh-client -S`. Navigation SHALL work regardless of which tmux session the user's current client is attached to. If no uncleared notifications exist, it SHALL exit 0 silently.

#### Scenario: Single notification waiting — same session
- **WHEN** `claude-notify jump` is run and one uncleared entry exists in the current session
- **THEN** the notified pane is focused via `SelectPane` + `SwitchClientToSessionWindow`
- **AND** the notification is cleared (JSONL marked cleared, window styles unset if no siblings remain)
- **AND** the status bar is immediately refreshed so the badge disappears without waiting for `status-interval`
- **AND** exit code is 0

#### Scenario: Single notification waiting — different session
- **WHEN** `claude-notify jump` is run from session `k8s` and the uncleared entry lives in session `edit`
- **THEN** the current tmux client switches to session `edit` and activates the notified window
- **AND** the notified pane is selected within that window
- **AND** the notification is cleared and the status bar is refreshed
- **AND** exit code is 0

#### Scenario: Multiple notifications waiting — oldest selected
- **WHEN** `claude-notify jump` is run and multiple uncleared entries exist across sessions
- **THEN** the entry with the smallest `ts` value is selected
- **AND** that pane is focused (crossing sessions if necessary) and cleared

#### Scenario: No notifications — silent exit
- **WHEN** `claude-notify jump` is run and no uncleared entries exist
- **THEN** no tmux commands are issued
- **AND** exit code is 0

#### Scenario: Status bar refreshes immediately after jump
- **WHEN** `claude-notify jump` clears a notification
- **THEN** `tmux refresh-client -S` is called before exit
- **AND** the status bar badge disappears immediately regardless of `status-interval`

#### Scenario: Stale session in JSONL
- **WHEN** `claude-notify jump` finds an uncleared notification whose recorded session no longer exists
- **THEN** the `switch-client` call fails silently (error is discarded)
- **AND** the notification clear and status-bar refresh still run
- **AND** exit code is 0
