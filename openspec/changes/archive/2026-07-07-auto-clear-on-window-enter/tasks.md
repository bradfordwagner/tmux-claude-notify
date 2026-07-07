## 1. TMux Option Reader

- [x] 1.1 Add `GetNavClearSeconds() int` to `internal/tmux/tmux.go` — reads `@claude-notify-nav-clear-seconds`, defaults to `2`, returns `0` if set to `"0"`

## 2. Auto-Reset Subprocess: Add --nav-delay Flag

- [x] 2.1 In `cmd/claude-notify/main.go` `auto-reset` subcommand block, add `--nav-delay` flag parsing alongside the existing `--delay` flag
- [x] 2.2 Pass `navDelaySecs` through to `runAutoReset(paneID, delaySecs, navDelaySecs int)`

## 3. Fork Logic: Spawn When Either Delay > 0

- [x] 3.1 Update `forkAutoReset` signature to `forkAutoReset(paneID string, delaySecs, navDelaySecs int)`
- [x] 3.2 Append `--nav-delay <navDelaySecs>` to the subprocess args slice
- [x] 3.3 In `runNotify`, change the spawn condition from `if delaySecs > 0` to `if delaySecs > 0 || navDelaySecs > 0`
- [x] 3.4 In `runNotify`, call `GetNavClearSeconds()` and pass it to `forkAutoReset`

## 4. runAutoReset: Split Delay by Path

- [x] 4.1 Update `runAutoReset` signature to `runAutoReset(paneID string, delaySecs, navDelaySecs int)`
- [x] 4.2 In the already-focused branch: guard on `delaySecs > 0` before calling `clearAfterGracePeriod`; if `delaySecs == 0`, return without clearing
- [x] 4.3 In the polling loop, when focus is detected: call `clearAfterGracePeriod(paneID, navDelaySecs)` only if `navDelaySecs > 0`; if `navDelaySecs == 0`, return without clearing

## 5. Documentation

- [x] 5.1 Add `@claude-notify-nav-clear-seconds` row to the Configuration table in `README.md` (default `2`, description: seconds to wait before clearing a notification when you navigate to its window; `0` disables)

## 6. Fix: Pane-level Focus Detection

- [x] 6.1 Update `IsPaneFocused` in `internal/tmux/tmux.go` to check `#{window_active}#{pane_active}` (both must be `1`) so split-pane navigation within the same window is detected correctly — previously only `window_active` was checked, causing sibling panes to appear "focused"
