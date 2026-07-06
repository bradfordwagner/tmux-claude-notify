## Why

The "✓ hook configured" line permanently occupies the top of the dashboard TUI, cluttering the display with static status that is obvious once setup has been done. The notification is only meaningful the moment the hook is actually wired — after that it's noise.

## What Changes

- Remove the permanent "✓ hook configured" status line from the top of the TUI
- Add a transient toast that appears for 10 seconds when the hook is newly configured (auto-configured on first run, or configured after a settings-changed event)
- The "⚠ hook not configured" warning remains as a permanent line (still actionable)

## Capabilities

### New Capabilities
- `tui-toast`: Transient in-TUI toast notification that auto-dismisses after a configurable duration; used to surface one-time events without permanently occupying UI space.

### Modified Capabilities
- `hook-setup`: The TUI representation of the configured state changes — no longer a persistent badge, now a toast on transition.

## Impact

- `internal/ui/model.go`: add toast state, timer message type, render logic; remove permanent "✓ hook configured" from `renderSetupStatus`
- No changes to `internal/setup/`, `internal/store/`, or any tmux helpers
