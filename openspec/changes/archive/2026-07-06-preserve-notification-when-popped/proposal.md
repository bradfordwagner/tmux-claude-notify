## Why

When the auto-reset timer fires while the grimoire shpell popup (claude-notify dashboard) is open, notifications disappear before the user can explicitly select and navigate to them. Opening the dashboard signals active intent to manage notifications; auto-clearing under the user's cursor is surprising and forces them to chase a notification that just vanished.

## What Changes

- The `claude-notify auto-reset` subcommand gains a guard: before clearing, it checks whether `_shpell-session` exists in tmux (indicating the popup is open). If the session is live, the auto-reset is skipped entirely — the notification persists until the user selects it from the dashboard.
- No new TPM option needed; the guard is unconditional when the popup is detected.

## Capabilities

### New Capabilities

- `auto-reset-popup-guard`: When the auto-reset subprocess wakes after its delay, it skips the clear if the claude-notify dashboard popup (`_shpell-session`) is currently open, deferring to explicit dashboard dismissal.

### Modified Capabilities

- `auto-reset-active-pane`: The clear-after-delay requirement gains a precondition — the popup must NOT be open for the auto-reset to execute.

## Impact

- `internal/tmux/tmux.go`: add `IsShpellOpen() bool` helper (checks `tmux has-session -t _shpell-session`)
- `cmd/claude-notify/main.go` (or wherever `auto-reset` subcommand runs): call `IsShpellOpen` before clearing
- No changes to the JSONL store schema, no new TPM options, no changes to the Stop hook path
