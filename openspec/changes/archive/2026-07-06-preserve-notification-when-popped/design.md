## Context

The `auto-reset-active-pane` feature spawns a detached subprocess (`claude-notify auto-reset`) when a notification fires on the currently focused pane. After a configurable delay (default 15s), the subprocess wakes and clears the notification automatically.

The subprocess currently has one precondition: the JSONL entry must still be uncleared. It has no awareness of whether the user is actively viewing the claude-notify dashboard.

The grimoire shpell popup runs in `_shpell-session`, a tmux session that exists only while the dashboard is open. This session name is a reliable sentinel: `tmux has-session -t _shpell-session` succeeds iff the popup is live.

## Goals / Non-Goals

**Goals:**
- Prevent the auto-reset subprocess from clearing a notification while the claude-notify dashboard popup is open
- Keep the guard simple and side-effect-free — no new state, no timers, no retry loops

**Non-Goals:**
- Pausing the auto-reset timer and resuming it when the popup closes (too complex, marginal benefit)
- Suppressing the auto-reset subprocess spawn at notify time (popup may not be open then)
- Any change to the Stop hook path or JSONL schema

## Decisions

**Decision: Skip-on-popup, not delay-and-retry**

When the auto-reset subprocess wakes and detects the popup is open, it exits without clearing. The notification then persists indefinitely until the user selects it from the dashboard.

Alternatives considered:
- *Poll until popup closes, then clear*: adds complexity and keeps a process alive unexpectedly; not worth it — the user is already in the dashboard and can dismiss manually.
- *Re-spawn with a new timer*: same complexity concern; still may race with user action.

Skip-and-exit is the simplest behavior that satisfies the requirement and avoids surprise.

**Decision: `tmux has-session -t _shpell-session` as the detection mechanism**

`_shpell-session` is the well-known session name used by grimoire for its popup. Its existence is a reliable indicator that the dashboard popup is open. No new TPM option or IPC channel needed.

**Decision: New `IsShpellOpen() bool` helper in `internal/tmux/tmux.go`**

Keeps the detection logic in the tmux package alongside `DetachIfShpell`, which already references `_shpell-session`. Single source of truth for the session name constant.

## Risks / Trade-offs

- [Risk] User opens popup after auto-reset already fired → Mitigation: none needed; the notification was legitimately cleared before the popup opened. The user will see an empty dashboard.
- [Risk] `_shpell-session` name changes in a future grimoire version → Mitigation: low probability; the name is a stable grimoire convention. If it changes, `DetachIfShpell` would also break, making the issue obvious.
- [Trade-off] Skipping (not delaying) means a notification that auto-reset would have cleared now persists until dashboard dismissal. This is the intended behavior — the user is in the dashboard, so explicit dismissal is correct.

## Migration Plan

No migration needed. The change is additive: existing auto-reset behavior is unchanged except when the popup is open, which was previously undefined/unguarded behavior.

Deploy: rebuild binary via `task build`, TPM will pick up the new binary on next `tmux source-file`.
