## MODIFIED Requirements

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
When `claude-notify notify` is invoked, `select-pane -t <paneID> -P bg=<color>` SHALL be called to highlight the specific pane's background. Using `select-pane -P` instead of `window-active-style` ensures only the notified pane is highlighted, not whichever pane happens to be focused when the hook fires. Color resolution order: `@claude-notify-pop-color` → `@tmux-pop-color` → `#1e1e2e` (Catppuccin Mocha base). The pop is pane-scoped and is cleared for each pane individually when its notification is dismissed — it is NOT gated on being the last notified pane in the window.

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
- **THEN** `select-pane -t <paneID> -P ""` is called to reset that pane's background
- **AND** sibling panes' backgrounds are unaffected

#### Scenario: Pop error is non-fatal
- **WHEN** `select-pane -P` fails for any reason
- **THEN** `claude-notify notify` continues and returns success (cosmetic only)
