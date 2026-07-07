## Why

When a user navigates to a tmux window that has a pending claude notification, the visual pop persists for the full auto-reset grace period (15 seconds default) even though arriving at the window is itself an acknowledgment. Navigation should feel immediate — enter the window, and the pop clears within a second or two.

## What Changes

- Add `@claude-notify-nav-clear-seconds` TPM option (default `2`) controlling how long to wait before clearing a notification when the user *navigates to* a popped window via tmux (e.g. `next-window`, `previous-window`, `select-window`).
- Existing `@claude-notify-active-reset-seconds` (default `15`) is unchanged and continues to govern the case where the pane was *already focused* when the notification fired.
- The auto-reset polling loop now passes `navClearSecs` instead of `delaySecs` when it is the polling branch that detects focus — distinguishing "navigated in" from "was already here".
- Dashboard TUI reflects the cleared state on its next store poll (already works; no new code needed, but verify in testing).

## Capabilities

### New Capabilities
- `navigate-to-clear`: When the auto-reset subprocess's poll loop detects that the user has navigated to the popped window, it uses a short delay (`@claude-notify-nav-clear-seconds`) rather than the full grace period, producing an immediate-feeling clear.

### Modified Capabilities
- `auto-reset-active-pane`: The polling-detected-focus path now uses `navClearSecs` (short) instead of `delaySecs` (long). The already-focused path keeps `delaySecs`. The option `@claude-notify-nav-clear-seconds` is added.

## Impact

- `cmd/claude-notify/main.go` — `runAutoReset` and `forkAutoReset` gain a second delay parameter (`navClearSecs`); `auto-reset` subcommand gains `--nav-delay` flag.
- `internal/tmux/tmux.go` — add `GetNavClearSeconds()` to read the new TPM option.
- `openspec/specs/auto-reset-active-pane/spec.md` — update polling-focus scenario to reflect `navClearSecs`.
- `README.md` — add `@claude-notify-nav-clear-seconds` row to Configuration table.
- No breaking changes; new option defaults to `2` so existing installs behave noticeably better without any config change.
