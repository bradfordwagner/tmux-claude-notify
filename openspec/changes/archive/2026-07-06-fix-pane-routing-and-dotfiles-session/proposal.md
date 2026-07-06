## Why

Two bugs undermine notification reliability: (1) when multiple panes share a window, clearing one pane's notification prematurely removes the window highlight even if sibling panes still need attention; (2) the dashboard silently hides notifications whenever the source pane is no longer live, which causes the ~/dotfiles session (and any session whose pane was recreated or whose dashboard opened fresh) to appear notification-free even though the user hasn't responded.

## What Changes

- `runClear` checks whether other uncleared notifications for the same window remain before clearing window-level styles — window highlight and pop style are only cleared when the last uncleared entry for that window is removed
- `store` gains a helper to query uncleared records by window ID, used by `runClear` to gate style teardown
- `loadEntries` removes the `liveSet[r.Pane]` guard — uncleared notifications are displayed regardless of pane liveness; panes that no longer exist are shown with a "gone" indicator instead of being hidden

## Capabilities

### New Capabilities

*(none — these are bug fixes to existing behavior)*

### Modified Capabilities

- `window-highlight`: clearing logic must check remaining window notifications before calling `ClearWindowStyle`/`ClearPopStyle`; pane ID is not the right scope for window-style teardown when the window has multiple notified panes
- `notification-log`: add `UnclearedForWindow(windowID string) ([]Record, error)` to support the multi-pane clearing check; remove or relax the live-pane filter in the UI's `loadEntries` to show notifications whose pane no longer exists

## Impact

- `internal/store/store.go`: new `UnclearedForWindow` function
- `cmd/claude-notify/main.go`: `runClear` calls `store.UnclearedForWindow` before clearing window styles
- `internal/ui/model.go`: `loadEntries` no longer filters by `liveSet`; entries whose pane is gone get a visual indicator (e.g. dimmed or tagged "gone")
- `internal/tmux/tmux.go`: no change needed for this fix
