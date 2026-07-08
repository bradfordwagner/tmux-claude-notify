## Why

`claude-notify jump` fails silently when the user is in a different tmux session than the one holding the waiting claude pane — `switch-client -t <windowID>` does not reliably activate the correct window when crossing session boundaries. Users with multiple sessions (e.g., a `k8s` session and an `edit` session) cannot use the jump keybinding unless they happen to already be in the session that contains the notification.

## What Changes

- Replace `SwitchToWindow(r.Window)` in `runJump()` with a cross-session-aware call that passes both session name and window ID (`switch-client -t <session>:<windowID>`)
- Add a `SwitchClientToSessionWindow(session, windowID string)` helper to the tmux package (or reuse an existing one) so the jump path matches the pattern already used in the dashboard enter handler
- Ensure `SelectPane` still fires before the switch so the user lands on the right pane within the window

## Capabilities

### New Capabilities
<!-- none: this is a bug fix, not a new capability -->

### Modified Capabilities
- `quickjump`: The `jump` subcommand requirement "navigates to oldest waiting pane" is extended to work across tmux sessions, not just within the current session.

## Impact

- `cmd/claude-notify/main.go` — `runJump()` function: one-line change to use session-qualified target
- `internal/tmux/tmux.go` — add or reuse a helper that runs `switch-client -t <session>:<window>`
- No schema, config, or user-visible behavior changes beyond the fix; existing unit tests for jump remain valid
