## MODIFIED Requirements

### Requirement: Auto-reset subprocess clears notification after delay
The `claude-notify auto-reset` subcommand SHALL sleep for the specified delay, then check two conditions before clearing: (1) the JSONL entry for the given pane is still uncleared, and (2) the dashboard popup (`_shpell-session`) is NOT currently open. If either check fails, the subprocess exits without modifying the store or tmux styles. If both conditions are met, it SHALL call `ClearPane` on the store and unset `window-status-style`, `window-status-current-style`, and `window-active-style` on the window.

#### Scenario: Entry still uncleared and popup closed — cleared automatically
- **WHEN** the auto-reset subprocess wakes after N seconds
- **AND** the store still has an uncleared entry for the pane
- **AND** `tmux.IsShpellOpen()` returns `false`
- **THEN** `ClearPane` is called, removing the JSONL entry
- **AND** window tab styles and pane background pop are unset

#### Scenario: Entry still uncleared but popup is open — skipped
- **WHEN** the auto-reset subprocess wakes after N seconds
- **AND** the store still has an uncleared entry for the pane
- **AND** `tmux.IsShpellOpen()` returns `true`
- **THEN** the subprocess exits without touching the JSONL store or tmux styles

#### Scenario: Entry already cleared before delay — no-op
- **WHEN** the auto-reset subprocess wakes after N seconds
- **AND** the pane entry was already cleared (user dismissed via dashboard or transcript watcher)
- **THEN** the subprocess exits without touching tmux styles or the JSONL store

#### Scenario: Auto-reset subprocess is detached — no zombie processes
- **WHEN** the auto-reset subprocess is forked
- **THEN** it runs in a new session (`Setsid: true`) with stdin/stdout/stderr closed
- **AND** the parent notify process does not wait for it
