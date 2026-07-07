# Spec: navigate-to-clear

## Purpose

When a Claude notification fires on a pane the user is not currently watching, the auto-reset subprocess polls for window focus. Once the user navigates (switches tmux windows) to the notified pane, the notification auto-clears after a short configurable delay (`navClearSecs`, default 2 seconds). This delay is intentionally shorter than the already-focused grace period (`activeResetSecs`, default 15 seconds) because the act of navigating is a deliberate user action — the user is aware of the notification.

## Requirements

### Requirement: Navigate-to-clear uses a separate short delay
When the auto-reset subprocess's polling loop detects that the user has focused the popped window (navigated to it), it SHALL use `navClearSecs` as the grace period rather than `activeResetSecs`. `navClearSecs` is read from `@claude-notify-nav-clear-seconds` (default `2`). Setting this option to `0` disables the polling-detected-focus clear path only; the already-focused path (governed by `@claude-notify-active-reset-seconds`) is unaffected.

#### Scenario: User navigates to popped pane — short delay then clear
- **WHEN** the auto-reset subprocess is polling because the pane was not focused at notify time
- **AND** the polling loop detects both `window_active = 1` AND `pane_active = 1` for the notified pane
- **AND** `@claude-notify-nav-clear-seconds` is non-zero (default 2)
- **THEN** the subprocess sleeps `navClearSecs` seconds
- **AND** if the entry is still uncleared and the dashboard is not open, `ClearPane` and tmux style resets are applied

#### Scenario: Sibling pane in same window does not trigger nav-clear
- **WHEN** the notified pane has `window_active = 1` but `pane_active = 0`
- **THEN** `IsPaneFocused` returns false and the polling loop continues waiting

#### Scenario: Navigate-to-clear disabled via option
- **WHEN** `@claude-notify-nav-clear-seconds` is set to `0`
- **AND** the polling loop detects window focus
- **THEN** the subprocess exits without touching the store or tmux styles

#### Scenario: Navigate-to-clear and active-reset both non-zero — subprocess spawned
- **WHEN** `@claude-notify-active-reset-seconds` is non-zero
- **OR** `@claude-notify-nav-clear-seconds` is non-zero
- **THEN** the auto-reset subprocess is spawned on notify

#### Scenario: Both delays zero — subprocess not spawned
- **WHEN** `@claude-notify-active-reset-seconds` is `0`
- **AND** `@claude-notify-nav-clear-seconds` is `0`
- **THEN** no auto-reset subprocess is forked

### Requirement: nav-delay propagated via subprocess flag
The `claude-notify notify` subcommand SHALL read `@claude-notify-nav-clear-seconds`, resolve the default (2) if unset or empty, and pass it as `--nav-delay <N>` to the forked `claude-notify auto-reset` subprocess. The subprocess SHALL not re-read TPM options at runtime.

#### Scenario: --nav-delay flag parsed by auto-reset subcommand
- **WHEN** `claude-notify auto-reset --pane <id> --delay <N> --nav-delay <M>` is invoked
- **THEN** the subprocess uses `N` seconds for the already-focused path and `M` seconds for the poll-detected-focus path

#### Scenario: Default nav-delay is 2 when option unset
- **WHEN** `@claude-notify-nav-clear-seconds` is not set in tmux global options
- **THEN** `navClearSecs` resolves to `2`
