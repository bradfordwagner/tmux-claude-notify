# Spec: window-highlight

## Purpose

Visual indicator on the tmux window tab and pane background when claude is waiting for input. Persists until explicitly cleared via the dashboard, or — for a notification on the currently focused pane — until the auto-reset timer fires. Non-focused panes have no auto-clear; they require explicit dashboard dismissal.

## Requirements

### Requirement: Window tab is highlighted when claude is waiting
When `claude-notify notify` is invoked, the tmux window containing the notified pane SHALL have both `window-status-style` and `window-status-current-style` set to `fg=#AD8EE6,bold` so the tab is visually distinct whether or not the user is currently on that window.

#### Scenario: Highlight applied on notify subcommand
- **WHEN** `claude-notify notify` is invoked with a valid `$TMUX_PANE`
- **THEN** the window has both `window-status-style` and `window-status-current-style` set to `fg=#AD8EE6,bold`

#### Scenario: Highlight applies to the active (current) window tab
- **WHEN** the notified window is the currently selected window
- **THEN** `window-status-current-style fg=#AD8EE6,bold` is set, making the active tab visually distinct

The notification SHALL persist until either (a) the user selects the entry from the dashboard, or (b) the auto-reset timer fires for a focused-pane notification. The highlight MUST NOT auto-clear for non-focused panes.

#### Scenario: Highlight persists until explicitly cleared (non-focused pane)
- **WHEN** the highlight has been set
- **AND** the notified pane was not focused at notify time (or auto-reset is disabled)
- **AND** the user has not yet selected the notification from the dashboard
- **THEN** both window-status styles remain set across window switches

#### Scenario: Highlight cleared on dashboard selection
- **WHEN** the user selects the entry from the dashboard
- **THEN** both `window-status-style` and `window-status-current-style` are unset via `set-option -u`

#### Scenario: Highlight cleared by auto-reset timer
- **WHEN** the auto-reset subprocess fires after the configured delay
- **AND** the JSONL entry is still uncleared
- **THEN** both `window-status-style` and `window-status-current-style` are unset via `set-option -u`

#### Scenario: Idempotent on repeated notify calls
- **WHEN** `claude-notify notify` is called again for the same pane while an uncleared entry exists
- **THEN** the styles are re-applied but no new JSONL entry is created

#### Scenario: No tmux context — silent skip
- **WHEN** `$TMUX` is not set in the environment
- **THEN** `claude-notify notify` exits 0 without attempting any tmux commands

#### Scenario: Pane ID missing — silent skip
- **WHEN** `$TMUX_PANE` is not set in the environment
- **THEN** `claude-notify notify` exits 0 without attempting any tmux commands

### Requirement: Pane background pops when waiting
When `claude-notify notify` is invoked, the `window-active-style` SHALL be set to `bg=<color>` on the notified window. Color resolution order: `@claude-notify-pop-color` → `@tmux-pop-color` → `#1e1e2e` (Catppuccin Mocha base). The pop persists until cleared.

#### Scenario: Pop color resolved from @claude-notify-pop-color
- **WHEN** `@claude-notify-pop-color` is set in tmux global options
- **THEN** that color is used for the pane background

#### Scenario: Pop color falls back to @tmux-pop-color
- **WHEN** `@claude-notify-pop-color` is unset and `@tmux-pop-color` is set
- **THEN** `@tmux-pop-color` is used for the pane background

#### Scenario: Pop color defaults to #1e1e2e when both options unset or black
- **WHEN** neither pop color option is set, or the resolved value is `black` or `colour0`
- **THEN** the color defaults to `#1e1e2e` (Catppuccin Mocha base, visible on black terminal)

#### Scenario: Pop style cleared on dashboard selection
- **WHEN** the user selects the notification from the dashboard
- **THEN** `window-active-style` is unset via `set-option -u`

#### Scenario: Pop style error is non-fatal
- **WHEN** `set-option window-active-style` fails for any reason
- **THEN** `claude-notify notify` continues and returns success (cosmetic only)

### Requirement: Notify is idempotent across multiple hook firings
The Stop hook fires multiple times per claude turn. The notify subcommand SHALL check for an existing uncleared entry via `store.HasUnclearedPane` before appending a new record. If an uncleared entry already exists, styles are re-applied and the function returns without writing a new JSONL entry.

#### Scenario: Second notify call for same pane — no duplicate entry
- **WHEN** `claude-notify notify` is called for a pane that already has an uncleared JSONL entry
- **THEN** no new record is appended to the JSONL file
- **AND** `window-status-style`, `window-status-current-style`, and `window-active-style` are re-applied

#### Scenario: New entry created after previous cleared
- **WHEN** a pane's previous notification has been cleared (dashboard selection)
- **AND** `claude-notify notify` is called again for that pane
- **THEN** a new JSONL record is created and styles are set
