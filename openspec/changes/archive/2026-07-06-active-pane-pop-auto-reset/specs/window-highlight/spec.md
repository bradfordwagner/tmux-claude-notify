## MODIFIED Requirements

### Requirement: Highlight persists until explicitly cleared
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
