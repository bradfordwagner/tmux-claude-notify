## Context

`internal/resurrect/resurrect.go` contains the save/restore logic for resuming claude sessions across tmux-resurrect cycles. The current implementation uses `(session, window_index, pane_index)` as the identity key for matching. Window indices in tmux are assigned by insertion order and can shift whenever any window is created, deleted, or moved — including the transient "cn" grimoire placeholder window, which can occupy any index slot between sessions. When the index occupied by the "cn" window at restore time happens to match a previously-saved claude pane's index, the restore sends `claude --resume` to the wrong pane. The `cd <path> && claude --resume` fallback compounded the problem by allowing any positionally-matched pane regardless of its working directory.

## Goals / Non-Goals

**Goals:**
- Eliminate wrong-pane targeting caused by window index drift
- Make restore safe against the "cn" placeholder window and any other transiently-inserted windows
- Keep sidecar forward-compatible (v1 sidecars inert; v2 sidecars correct)

**Non-Goals:**
- Handling renamed windows (user changes window name between save and restore)
- Fallback restore when window name no longer exists (user manually killed that window)
- Multi-pane disambiguation within the same window (pane index still used as secondary key)

## Decisions

### Decision: Match by window_name instead of window_index

Window names are set intentionally by users (or by the shell prompt via `automatic-rename`) and persist across tmux server restarts. They are far more stable than window indices, which are implicitly assigned by insertion position. Matching by `(session, window_name, pane_index)` is both more robust and more intuitive.

**Alternatives considered:**
- *pane_id*: Not viable — pane IDs (`%N`) are assigned by the running server and reset on restart.
- *project_path alone*: Could work but ambiguous when multiple panes share the same directory; also doesn't account for sessions.
- *window_index + path validation*: Reduces blast radius but doesn't fix the root cause.

### Decision: Require exact path match; drop the cd fallback

The `cd <path> && claude --resume` fallback was added to handle panes that resurrect placed in a slightly different directory. In practice, tmux-resurrect restores working directories accurately. The fallback silently targets wrong panes (any pane at the right position regardless of context). Removing it means a pane not already in the correct directory is simply skipped — the user can manually resume if needed. This is safer than the alternative.

### Decision: Bump sidecar version to 2; skip v1 on restore

A v1 sidecar has `window_index` (int) but no `window_name`. Attempting to restore from it would always fail to match (empty string key). Explicitly checking `version < 2` and returning early is clearer than silent no-op behavior and prevents any accidental partial match. The next save (triggered by the pre-save hook) produces a v2 sidecar.

## Risks / Trade-offs

- **Renamed windows between save and restore** → restore silently skips the pane. The user must manually run `claude --resume <id>`. Acceptable given that window names rarely change between a save (shutdown) and restore (startup).
- **automatic-rename enabled** → if tmux renames the window to the running process name at save time, the name at restore time (shell name) may differ. Mitigation: save captures the name at save time; if automatic-rename changes it during the session, the mismatch is the same as today's index mismatch, which is no regression.
- **Multiple panes in same window** → pane_index still used as secondary key, so this case is handled the same as before.

## Migration Plan

1. Deploy updated binary (compiled via TPM or `task build`).
2. On first tmux-resurrect save after deploy, the pre-save hook rewrites the sidecar as v2.
3. Until then, any v1 sidecar is skipped on restore (no wrong-pane targeting, no resume either).
4. No manual file migration needed; no rollback steps — reverting the binary restores v1 behavior.
