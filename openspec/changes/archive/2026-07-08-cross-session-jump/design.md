## Context

The `claude-notify jump` subcommand lets a user navigate directly to the oldest waiting claude pane without opening the dashboard. It is invoked via a tmux keybinding (`run-shell '<binary> jump'`).

Current implementation in `cmd/claude-notify/main.go:runJump()`:
```go
_ = tmuxclient.SelectPane(r.Pane)      // select-pane -t %34
_ = tmuxclient.DetachIfShpell()        // no-op for jump
_ = tmuxclient.SwitchToWindow(r.Window) // switch-client -t @5
```

`SwitchToWindow` issues `tmux switch-client -t @5` (window ID only). When the current session differs from the session that owns `@5`, tmux may switch to the correct session but leave the previously-active window selected — the user does not land on the right window.

The dashboard enter-handler already solves this correctly with:
```go
_ = tmuxclient.SelectPane(paneID)
_ = tmuxclient.SelectWindow(session, windowID) // select-window -t session:@5
_ = tmuxclient.DetachIfShpell()
```

`select-window` only affects the target session's active window but does not move the client across sessions. The dashboard works because it closes the popup (DetachIfShpell) and the outer client was already in the right session. The jump path has no popup to close, so it must issue a `switch-client` that also specifies the window.

## Goals / Non-Goals

**Goals:**
- Jump navigates to the correct window regardless of which tmux session the user is currently in
- Minimal diff: change only what is needed to fix cross-session navigation
- No new user-visible config or keybinding changes

**Non-Goals:**
- Changing jump behavior when already in the same session as the notification
- Modifying how the dashboard enter-handler navigates (it already works correctly)
- Adding multi-hop or nested-session support

## Decisions

### Decision: Use `switch-client -t <session>:<window>` instead of `switch-client -t <window>`

`tmux switch-client -t session:@windowID` switches the current client to `session` AND activates window `@windowID` in one command. This is the standard tmux idiom for cross-session window navigation.

Alternative considered: `switch-client -t %paneID` — tmux resolves the session from the pane ID and can select the containing window, but behavior is version-dependent and less explicit.

Alternative considered: two commands (`switch-client -t session` + `select-window -t session:window`) — introduces a visible flash as the client briefly shows the wrong window.

### Decision: Add `SwitchClientToSessionWindow(session, windowID string)` to `internal/tmux`

Keeps the tmux command logic in one place, consistent with the existing pattern of one function per tmux primitive. `runJump()` calls this instead of `SwitchToWindow`.

`SwitchToWindow` is kept for any callers that genuinely don't need a session (there are currently none outside jump, but removing it is out of scope).

## Risks / Trade-offs

- [Risk] `switch-client -t session:@windowID` requires the session to still exist between when the notification was recorded and when jump fires → Mitigation: silently ignore errors (same as existing behavior — all tmux calls in jump already discard errors with `_ =`).
- [Risk] If `r.Session` in the JSONL is stale (session renamed or destroyed) the switch will fail silently and jump exits 0 without navigating → Mitigation: acceptable; same failure mode as the current single-session case.
