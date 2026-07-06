## ADDED Requirements

### Requirement: Auto-reset fires when notified pane is currently focused
When `claude-notify notify` is invoked and the notified pane is both (a) the active pane in its window and (b) in the currently active window, the binary SHALL fork a detached subprocess (`claude-notify auto-reset --pane <id> --delay <N>`) immediately after applying the visual pop, then exit. The subprocess sleeps N seconds (from `@claude-notify-active-reset-seconds`, default `15`) and then clears the notification. If `@claude-notify-active-reset-seconds` is `0`, the auto-reset is disabled and the notification persists as normal.

#### Scenario: Active pane and active window — auto-reset subprocess spawned
- **WHEN** `claude-notify notify` is invoked
- **AND** `tmux display-message -t $TMUX_PANE -p "#{pane_active}#{window_active}"` returns `11`
- **THEN** a detached `claude-notify auto-reset --pane <id> --delay <N>` subprocess is started
- **AND** the notify command exits immediately without blocking

#### Scenario: Active pane but inactive window — no auto-reset
- **WHEN** `claude-notify notify` is invoked
- **AND** the pane is active in its window but the window is not the current window (`window_active = 0`)
- **THEN** no auto-reset subprocess is spawned
- **AND** the notification persists until manually dismissed

#### Scenario: Inactive pane — no auto-reset
- **WHEN** `claude-notify notify` is invoked
- **AND** `pane_active` is `0`
- **THEN** no auto-reset subprocess is spawned

#### Scenario: Auto-reset disabled via option — no subprocess
- **WHEN** `@claude-notify-active-reset-seconds` is set to `0`
- **THEN** no auto-reset subprocess is spawned regardless of pane focus state

### Requirement: Auto-reset subprocess clears notification after delay
The `claude-notify auto-reset` subcommand SHALL sleep for the specified delay, then check whether the JSONL entry for the given pane is still uncleared. If uncleared, it SHALL call `ClearPane` on the store and unset `window-status-style`, `window-status-current-style`, and `window-active-style` on the window.

#### Scenario: Entry still uncleared after delay — cleared automatically
- **WHEN** the auto-reset subprocess wakes after N seconds
- **AND** the store still has an uncleared entry for the pane
- **THEN** `ClearPane` is called, removing the JSONL entry
- **AND** window tab styles and pane background pop are unset

#### Scenario: Entry already cleared before delay — no-op
- **WHEN** the auto-reset subprocess wakes after N seconds
- **AND** the pane entry was already cleared (user dismissed via dashboard or transcript watcher)
- **THEN** the subprocess exits without touching tmux styles or the JSONL store

#### Scenario: Auto-reset subprocess is detached — no zombie processes
- **WHEN** the auto-reset subprocess is forked
- **THEN** it runs in a new session (`Setsid: true`) with stdin/stdout/stderr closed
- **AND** the parent notify process does not wait for it

### Requirement: Auto-reset delay is configurable via TPM option
The auto-reset delay SHALL be read from `@claude-notify-active-reset-seconds` using `tmux show-option -gqv`. The default SHALL be `15` seconds when the option is unset or empty. A value of `0` SHALL disable auto-reset entirely.

#### Scenario: Option unset — default 15 seconds used
- **WHEN** `@claude-notify-active-reset-seconds` is not set
- **THEN** the auto-reset delay is 15 seconds

#### Scenario: Option set to custom value
- **WHEN** `set -g @claude-notify-active-reset-seconds 30` is in `tmux.conf`
- **THEN** the auto-reset delay is 30 seconds

#### Scenario: Option set to 0 — auto-reset disabled
- **WHEN** `set -g @claude-notify-active-reset-seconds 0` is in `tmux.conf`
- **THEN** no auto-reset subprocess is spawned for any notification
