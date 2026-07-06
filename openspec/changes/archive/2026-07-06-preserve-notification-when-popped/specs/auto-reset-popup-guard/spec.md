## ADDED Requirements

### Requirement: Auto-reset is suppressed when the dashboard popup is open
When the `claude-notify auto-reset` subprocess wakes after its delay, it SHALL check whether the grimoire shpell session (`_shpell-session`) is currently active before clearing the notification. If the session exists, the subprocess SHALL exit without touching the JSONL store or tmux styles, leaving the notification for explicit dashboard dismissal.

#### Scenario: Popup open at wake time — auto-reset skipped
- **WHEN** the auto-reset subprocess wakes after the configured delay
- **AND** `tmux has-session -t _shpell-session` exits 0 (popup is live)
- **THEN** the subprocess exits without calling `ClearPane`, `ClearWindowStyle`, or `ClearPopStyle`
- **AND** the JSONL entry remains uncleared
- **AND** the window tab highlight and pane background pop remain set

#### Scenario: Popup closed at wake time — auto-reset proceeds normally
- **WHEN** the auto-reset subprocess wakes after the configured delay
- **AND** `tmux has-session -t _shpell-session` exits non-zero (no popup)
- **AND** the JSONL entry is still uncleared
- **THEN** `ClearPane` is called, removing the JSONL entry
- **AND** window tab styles and pane background pop are unset

#### Scenario: IsShpellOpen helper used for detection
- **WHEN** the auto-reset subprocess checks for the popup
- **THEN** it calls `tmux.IsShpellOpen()` which runs `tmux has-session -t _shpell-session`
- **AND** returns `true` only when that command exits 0
