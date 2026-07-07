# Spec: auto-reset-active-pane

## Purpose

When a Claude notification fires on the currently focused pane (the user is already looking at it), the visual pop auto-clears after a configurable delay rather than persisting indefinitely. This avoids stale highlights for panes the user is actively watching. Non-focused panes are unaffected and continue to require explicit dashboard dismissal.

## Requirements

### Requirement: Auto-reset subprocess is always spawned when a notification fires
When `claude-notify notify` is invoked and `@claude-notify-active-reset-seconds` is non-zero, the binary SHALL always fork a detached subprocess (`claude-notify auto-reset --pane <id> --delay <N>`), regardless of whether the notified pane's window is currently active. The subprocess behavior differs based on whether the window is focused at notify time (see below). If `@claude-notify-active-reset-seconds` is `0`, the auto-reset is disabled entirely.

#### Scenario: Always spawn subprocess when auto-reset is enabled
- **WHEN** `claude-notify notify` is invoked
- **AND** `@claude-notify-active-reset-seconds` is non-zero
- **THEN** a detached `claude-notify auto-reset --pane <id> --delay <N>` subprocess is started unconditionally
- **AND** the notify command exits immediately without blocking

#### Scenario: Active window at notify time — grace-period then clear
- **WHEN** `claude-notify notify` is invoked
- **AND** `tmux display-message -t $TMUX_PANE -p "#{window_active}"` returns `1`
- **THEN** the subprocess sleeps N seconds (grace period), then clears the notification if still uncleared and popup is not open

#### Scenario: Inactive window at notify time — poll for focus then clear
- **WHEN** `claude-notify notify` is invoked
- **AND** the window is not the user's current window (`window_active = 0`)
- **THEN** the subprocess polls `window_active` every 2 seconds (up to 4 hours)
- **AND** when `window_active=1` is first detected, clears the notification immediately (if still uncleared and popup not open)

#### Scenario: User navigates to popped pane via tmux — auto-clears
- **WHEN** a notification is set on a pane in an inactive window
- **AND** the user switches to that window using tmux (not the dashboard)
- **THEN** the auto-reset subprocess detects focus and clears the pop within 2 seconds

#### Scenario: Auto-reset disabled via option — no subprocess
- **WHEN** `@claude-notify-active-reset-seconds` is set to `0`
- **THEN** no auto-reset subprocess is spawned regardless of pane focus state

### Requirement: Auto-reset subprocess clears notification after delay
The `claude-notify auto-reset` subcommand SHALL sleep for the specified delay, then check two conditions before clearing: (1) the JSONL entry for the given pane is still uncleared, and (2) the dashboard popup (`_shpell-session`) is NOT currently open. If either check fails, the subprocess exits without modifying the store or tmux styles. If both conditions are met, it SHALL call `ClearPane` on the store, then call `store.UnclearedForWindow(windowID)` — only if that returns empty SHALL it unset `window-status-style` and `window-status-current-style` on the window. The pane background pop SHALL always be cleared per-pane via `set-option -t <paneID> -p -u window-style` regardless of sibling notifications.

#### Scenario: Entry still uncleared and popup closed — cleared automatically
- **WHEN** the auto-reset subprocess wakes after N seconds
- **AND** the store still has an uncleared entry for the pane
- **AND** `tmux.IsShpellOpen()` returns `false`
- **THEN** `ClearPane` is called, removing the JSONL entry
- **AND** `set-option -t <paneID> -p -u window-style` is called to clear the pane background pop
- **AND** if no other uncleared entries remain for the same window, `window-status-style` and `window-status-current-style` are unset via `set-option -u`
- **AND** if sibling panes in the same window still have uncleared entries, window tab styles are NOT cleared

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
