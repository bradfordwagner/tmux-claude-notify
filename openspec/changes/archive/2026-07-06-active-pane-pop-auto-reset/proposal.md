## Why

When a tmux window receives a pop (visual highlight + background color change) and the user is already viewing that pane, the notification is redundant — they're right there. Without auto-reset, the user must manually dismiss via the dashboard even when they never left the pane.

## What Changes

- The `notify` subcommand checks whether `$TMUX_PANE` is the currently active pane at the time it fires; if so, it starts a 15-second countdown and clears the pop automatically instead of persisting indefinitely.
- If the pane is not currently focused, the existing persistent-until-dismissed behavior is unchanged.
- A new `auto-reset-active-pane` capability is added to handle the timer + auto-clear logic.

## Capabilities

### New Capabilities

- `auto-reset-active-pane`: When the notified pane is already the active/focused pane, automatically reset window highlight and pane background pop after 15 seconds instead of waiting for manual dashboard dismissal.

### Modified Capabilities

- `window-highlight`: Auto-reset path clears highlight styles on the same timer schedule (15s) rather than requiring explicit user action.
- `notification-log`: The JSONL entry for an auto-reset notification must be marked cleared (not just updated to `running`) after the timer fires.

## Impact

- `cmd/claude-notify/main.go`: notify subcommand gains active-pane detection + timer logic
- `internal/store/store.go`: ClearPane called on auto-reset timer expiry
- `internal/tmux/tmux.go`: helpers to query current active pane and clear window styles
- No new dependencies; uses stdlib `time` package for the 15s timer
