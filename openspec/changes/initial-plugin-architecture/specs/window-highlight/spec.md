## ADDED Requirements

### Requirement: Window tab is highlighted when claude is waiting
When `claude-notify notify` is invoked, the tmux window containing the notified pane SHALL have its `window-status-style` set to `fg=#AD8EE6,bold` so it is visually distinct in the status bar.

#### Scenario: Highlight applied on notify subcommand
- **WHEN** `claude-notify notify` is invoked with a valid `$TMUX_PANE`
- **THEN** the window containing that pane has `window-status-style` set to `fg=#AD8EE6,bold`

#### Scenario: Highlight persists until user focuses the pane
- **WHEN** the highlight has been set
- **AND** the user has not yet focused the notified pane
- **THEN** the window tab remains highlighted across other window switches

#### Scenario: Highlight cleared on pane focus
- **WHEN** the user focuses the notified pane
- **THEN** `claude-notify clear --pane <id>` runs and unsets `window-status-style` via `set-option -u`

#### Scenario: No tmux context — silent skip
- **WHEN** `$TMUX` is not set in the environment
- **THEN** `claude-notify notify` exits 0 without attempting any tmux commands

#### Scenario: Pane ID missing — silent skip
- **WHEN** `$TMUX_PANE` is not set in the environment
- **THEN** `claude-notify notify` exits 0 without attempting any tmux commands

### Requirement: One-shot clear hook is registered per pane
After setting the highlight, `claude-notify notify` SHALL register a `pane-focus-in` hook scoped to the notified pane using its numeric ID as the array index. The hook SHALL call `claude-notify clear --pane <id>` and deregister itself after firing.

#### Scenario: Clear hook registered with pane-scoped index
- **WHEN** `claude-notify notify` sets the window highlight
- **THEN** a `pane-focus-in` hook is registered at index equal to the pane's numeric ID

#### Scenario: Clear hook fires exactly once
- **WHEN** the user focuses the notified pane
- **THEN** `claude-notify clear` runs, clears the highlight, and unregisters the hook
- **AND** subsequent focus events on that pane do not trigger the hook

#### Scenario: Window closed before clear
- **WHEN** the notified window is closed before the user focuses it
- **THEN** `claude-notify clear` handles the missing window gracefully and exits 0

### Requirement: Pane background pops when waiting
When `claude-notify notify` is invoked, the active pane in the notified window SHALL have its `window-active-style` set to `bg=<@tmux-pop-color>` so the pane background is visually distinct. The pop persists until cleared — there is no timer.

#### Scenario: Pop style applied on notify
- **WHEN** `claude-notify notify` is invoked with a valid `$TMUX_PANE`
- **THEN** `window-active-style bg=<@tmux-pop-color>` is set on that window
- **AND** if `@tmux-pop-color` is unset, the value defaults to `black`

#### Scenario: Pop style persists until cleared
- **WHEN** the pop style has been set
- **AND** the user has not yet focused or selected the notified pane
- **THEN** `window-active-style` remains set across other window switches

#### Scenario: Pop style cleared on pane focus
- **WHEN** the user focuses the notified pane (pane-focus-in hook fires)
- **THEN** `claude-notify clear` unsets `window-active-style` via `set-option -u`

#### Scenario: Pop style cleared on dashboard selection
- **WHEN** the user selects the notification from the dashboard
- **THEN** `claude-notify clear` unsets `window-active-style` via `set-option -u`

#### Scenario: Pop style error is non-fatal
- **WHEN** `set-option window-active-style` fails for any reason
- **THEN** `claude-notify notify` continues and returns success (cosmetic only)
