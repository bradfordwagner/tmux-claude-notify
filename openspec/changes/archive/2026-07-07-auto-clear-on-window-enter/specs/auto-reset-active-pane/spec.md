## MODIFIED Requirements

### Requirement: Auto-reset subprocess is always spawned when a notification fires
When `claude-notify notify` is invoked and at least one of `@claude-notify-active-reset-seconds` or `@claude-notify-nav-clear-seconds` is non-zero, the binary SHALL always fork a detached subprocess (`claude-notify auto-reset --pane <id> --delay <N> --nav-delay <M>`), regardless of whether the notified pane's window is currently active. The subprocess behavior differs based on whether the window is focused at notify time (see below). If both options are `0`, auto-reset is disabled entirely.

#### Scenario: Always spawn subprocess when either delay is non-zero
- **WHEN** `claude-notify notify` is invoked
- **AND** `@claude-notify-active-reset-seconds` is non-zero **OR** `@claude-notify-nav-clear-seconds` is non-zero
- **THEN** a detached `claude-notify auto-reset --pane <id> --delay <N> --nav-delay <M>` subprocess is started unconditionally
- **AND** the notify command exits immediately without blocking

#### Scenario: Active window at notify time — grace-period then clear
- **WHEN** `claude-notify notify` is invoked
- **AND** `tmux display-message -t $TMUX_PANE -p "#{window_active}"` returns `1`
- **AND** `@claude-notify-active-reset-seconds` is non-zero
- **THEN** the subprocess sleeps `activeResetSecs` seconds (grace period), then clears the notification if still uncleared and popup is not open

#### Scenario: Active window at notify time but active-reset disabled — no clear from already-focused path
- **WHEN** `claude-notify notify` is invoked
- **AND** the window is the user's current window (`window_active = 1`)
- **AND** `@claude-notify-active-reset-seconds` is `0`
- **THEN** the subprocess exits the already-focused path without clearing (nav-clear may still handle it if user leaves and returns)

#### Scenario: Inactive window at notify time — poll for focus, then nav-delay, then clear
- **WHEN** `claude-notify notify` is invoked
- **AND** the window is not the user's current window (`window_active = 0`)
- **AND** `@claude-notify-nav-clear-seconds` is non-zero
- **THEN** the subprocess polls `window_active` every 2 seconds (up to 4 hours)
- **AND** when `window_active=1` is first detected, sleeps `navClearSecs` seconds (not `activeResetSecs`), then clears the notification (if still uncleared and popup not open)

#### Scenario: User navigates to popped pane via tmux — nav-delay then auto-clears
- **WHEN** a notification is set on a pane in an inactive window
- **AND** the user switches to that window using tmux (not the dashboard)
- **THEN** the auto-reset subprocess detects focus, waits `navClearSecs` seconds, then clears the pop

#### Scenario: Auto-reset disabled via both options — no subprocess
- **WHEN** `@claude-notify-active-reset-seconds` is set to `0`
- **AND** `@claude-notify-nav-clear-seconds` is set to `0`
- **THEN** no auto-reset subprocess is spawned regardless of pane focus state

### Requirement: Auto-reset delay is configurable via TPM options
The active-reset delay SHALL be read from `@claude-notify-active-reset-seconds` (default `15`). The navigate-to-clear delay SHALL be read from `@claude-notify-nav-clear-seconds` (default `2`). Both are passed to the subprocess as `--delay` and `--nav-delay` respectively. Setting either to `0` disables its respective clear path.

#### Scenario: active-reset option unset — default 15 seconds used
- **WHEN** `@claude-notify-active-reset-seconds` is not set
- **THEN** the active-reset delay is 15 seconds

#### Scenario: nav-clear option unset — default 2 seconds used
- **WHEN** `@claude-notify-nav-clear-seconds` is not set
- **THEN** the nav-clear delay is 2 seconds

#### Scenario: Option set to custom value
- **WHEN** `set -g @claude-notify-active-reset-seconds 30` is in `tmux.conf`
- **THEN** the active-reset delay is 30 seconds

#### Scenario: nav-clear option set to 0 — navigate-to-clear disabled
- **WHEN** `set -g @claude-notify-nav-clear-seconds 0` is in `tmux.conf`
- **THEN** the polling-detected-focus path does not clear the notification
- **AND** the already-focused path is unaffected
