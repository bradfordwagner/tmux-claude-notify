# Spec: window-highlight

## Purpose

Visual indicator on the tmux window tab and pane background when claude is waiting for input. Persists until explicitly cleared via the dashboard, or — for a notification on the currently focused pane — until the auto-reset timer fires. Non-focused panes have no auto-clear; they require explicit dashboard dismissal.

## Requirements

### Requirement: Window tab is highlighted when claude is waiting
When `claude-notify notify` is invoked, the tmux window containing the notified pane SHALL have both `window-status-style` and `window-status-current-style` set to `fg=<color>,bold` where `<color>` is resolved from `@claude-notify-highlight-color` (defaulting to `#a6e3a1`, Catppuccin Mocha green). The tab SHALL be visually distinct whether or not the user is currently on that window.

#### Scenario: Highlight applied on notify subcommand
- **WHEN** `claude-notify notify` is invoked with a valid `$TMUX_PANE`
- **THEN** the window has both `window-status-style` and `window-status-current-style` set to `fg=<highlight-color>,bold`

#### Scenario: Highlight applies to the active (current) window tab
- **WHEN** the notified window is the currently selected window
- **THEN** `window-status-current-style fg=<highlight-color>,bold` is set, making the active tab visually distinct

#### Scenario: Highlight color configurable via @claude-notify-highlight-color
- **WHEN** `set -g @claude-notify-highlight-color '#ff0000'` is in `tmux.conf`
- **THEN** the window tab style uses `fg=#ff0000,bold` instead of the default

#### Scenario: Default color is Catppuccin Mocha green
- **WHEN** `@claude-notify-highlight-color` is not set
- **THEN** the window tab style uses `fg=#a6e3a1,bold`

The notification SHALL persist until either (a) the user selects the entry from the dashboard, or (b) the auto-reset timer fires for a focused-pane notification. The highlight MUST NOT auto-clear for non-focused panes.

#### Scenario: Highlight persists until explicitly cleared (non-focused pane)
- **WHEN** the highlight has been set
- **AND** the notified pane was not focused at notify time (or auto-reset is disabled)
- **AND** the user has not yet selected the notification from the dashboard
- **THEN** both window-status styles remain set across window switches

#### Scenario: Highlight cleared on dashboard selection when last notified pane in window
- **WHEN** the user selects the entry from the dashboard
- **AND** no other uncleared entries exist for the same window
- **THEN** both `window-status-style` and `window-status-current-style` are unset via `set-option -u`

#### Scenario: Highlight preserved when sibling pane still has uncleared notification
- **WHEN** the user selects one entry from a window that has multiple notified panes
- **AND** at least one other pane in the same window still has an uncleared JSONL entry
- **THEN** the window-status styles are NOT cleared
- **AND** the cleared pane's JSONL entry is removed from the store

#### Scenario: Highlight cleared by auto-reset timer when last notified pane in window
- **WHEN** the auto-reset subprocess fires after the configured delay
- **AND** the JSONL entry is still uncleared
- **AND** no other uncleared entries exist for the same window
- **THEN** both `window-status-style` and `window-status-current-style` are unset via `set-option -u`

#### Scenario: Highlight preserved when auto-reset fires but sibling pane still notified
- **WHEN** the auto-reset subprocess fires for one pane in a multi-pane window
- **AND** another pane in the same window still has an uncleared JSONL entry
- **THEN** the window-status styles are NOT cleared

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
When `claude-notify notify` is invoked, `set-option -t <paneID> -p window-style bg=<color>` SHALL be called to highlight the specific pane's background. Using `set-option -p window-style` instead of `select-pane -P` ensures the pane style is set without selecting the pane (which would move the user's cursor focus). Color resolution order: `@claude-notify-pop-color` → `@tmux-pop-color` → `#1e1e2e` (Catppuccin Mocha base). The pop is pane-scoped and is cleared for each pane individually when its notification is dismissed — it is NOT gated on being the last notified pane in the window.

#### Scenario: Pop applies to the specific notified pane
- **WHEN** `claude-notify notify` fires for pane `%5` while the user is focused on pane `%6` in the same window
- **THEN** pane `%5` gets the background highlight, not pane `%6`

#### Scenario: Pop color resolved from @claude-notify-pop-color
- **WHEN** `@claude-notify-pop-color` is set in tmux global options
- **THEN** that color is used for the pane background

#### Scenario: Pop color falls back to @tmux-pop-color
- **WHEN** `@claude-notify-pop-color` is unset and `@tmux-pop-color` is set
- **THEN** `@tmux-pop-color` is used for the pane background

#### Scenario: Pop color defaults to #1e1e2e when both options unset or black
- **WHEN** neither pop color option is set, or the resolved value is `black` or `colour0`
- **THEN** the color defaults to `#1e1e2e` (Catppuccin Mocha base, visible on black terminal)

#### Scenario: Pop cleared per-pane on dashboard selection
- **WHEN** the user selects a notification from the dashboard
- **THEN** `set-option -t <paneID> -p -u window-style` is called to reset that pane's background
- **AND** sibling panes' backgrounds are unaffected

#### Scenario: Pop error is non-fatal
- **WHEN** `set-option -p window-style` fails for any reason
- **THEN** `claude-notify notify` continues and returns success (cosmetic only)

### Requirement: Notify is idempotent across multiple hook firings
The Stop hook fires multiple times per claude turn. The notify subcommand SHALL check for an existing uncleared entry via `store.HasUnclearedPane` before appending a new record. If an uncleared entry already exists, styles are re-applied and the function returns without writing a new JSONL entry.

#### Scenario: Second notify call for same pane — no duplicate entry
- **WHEN** `claude-notify notify` is called for a pane that already has an uncleared JSONL entry
- **THEN** no new record is appended to the JSONL file
- **AND** `window-status-style`, `window-status-current-style`, and the pane pop (`set-option -p window-style`) are re-applied

#### Scenario: New entry created after previous cleared
- **WHEN** a pane's previous notification has been cleared (dashboard selection)
- **AND** `claude-notify notify` is called again for that pane
- **THEN** a new JSONL record is created and styles are set
