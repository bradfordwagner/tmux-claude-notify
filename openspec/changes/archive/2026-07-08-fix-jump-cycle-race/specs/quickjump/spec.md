## MODIFIED Requirements

### Requirement: jump subcommand navigates to oldest waiting pane
The `claude-notify jump` subcommand SHALL find the uncleared notification with the smallest `ts` (oldest), navigate to that pane, clear the notification, and immediately refresh the tmux status bar via `tmux refresh-client -S`. Navigation SHALL work regardless of which tmux session the user's current client is attached to. If no uncleared notifications exist, it SHALL exit 0 silently. Finding the oldest uncleared notification and clearing it SHALL be a single atomic operation so that a concurrent writer (the Stop hook, an auto-reset subprocess, or another `jump` invocation) cannot cause the same pane to be selected again on a subsequent `jump` after it was already cleared.

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

#### Scenario: Status bar refreshes even when the clear step fails
- **WHEN** `claude-notify jump` selects the oldest uncleared pane but the underlying clear operation returns an error
- **THEN** `tmux refresh-client -S` is still called before exit
- **AND** the badge reflects whatever the store actually contains rather than stale pre-jump state

#### Scenario: Stale session in JSONL
- **WHEN** `claude-notify jump` finds an uncleared notification whose recorded session no longer exists
- **THEN** the `switch-client` call fails silently (error is discarded)
- **AND** the notification clear and status-bar refresh still run
- **AND** exit code is 0

#### Scenario: Repeated jump presses advance through all waiting notifications
- **WHEN** three panes `%1`, `%2`, `%3` each have an uncleared entry, with a background auto-reset subprocess alive for each (per the existing auto-reset behavior)
- **AND** `claude-notify jump` is invoked three times in succession
- **THEN** each invocation navigates to and clears a distinct pane in oldest-first order
- **AND** no pane is selected or navigated to more than once across the three invocations, even though the auto-reset subprocesses for the other panes are concurrently polling and writing to the same store

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
- **AND** the status bar is immediately refreshed via `tmux refresh-client -S`
