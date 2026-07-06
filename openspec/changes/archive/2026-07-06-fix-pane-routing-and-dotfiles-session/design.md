## Context

Two separate bugs degrade notification reliability:

1. **Multi-pane window routing**: `runClear` calls `ClearWindowStyle(windowID)` and `ClearPopStyle(windowID)` unconditionally whenever a pane's notification is dismissed. If two panes share a window, clearing pane A wipes the window highlight for pane B even though pane B's notification is still uncleared. Root: styles are scoped to the window, but the "are we done?" check only looks at the current pane (`HasUnclearedPane`).

2. **Dotfiles / live-pane filter**: `loadEntries` in `model.go` guards every JSONL entry with `liveSet[r.Pane]`, where `liveSet` is built from `tmux list-panes -a`. Any pane that no longer appears in that list (e.g. the session was restarted since the Stop hook fired, or the pane was in a different context) causes the entry to be silently dropped from the dashboard — the notification exists on disk but is invisible to the user.

## Goals / Non-Goals

**Goals:**
- Window highlight and pop style are only torn down when the LAST uncleared notification for that window is cleared
- Dashboard shows all uncleared JSONL entries regardless of pane liveness; entries whose pane is gone are displayed with a "gone" marker
- `store.ClearPane` still works even when the pane is gone (it only writes JSONL, no tmux calls)
- `ClearWindowStyle`/`ClearPopStyle` fail gracefully when window ID no longer exists (already the case)

**Non-Goals:**
- Correlating a "gone" pane entry back to a running process — once the pane is gone, clearing the entry from the dashboard is the only action
- Cross-server / nested tmux support (separate problem)

## Decisions

### 1. Add `store.UnclearedForWindow(windowID string) ([]Record, error)`

Gate window-style teardown in `runClear` on this call: after `store.ClearPane(paneID)` writes the JSONL, call `UnclearedForWindow(windowID)`. Only if the result is empty (no remaining uncleared entries for this window) do we call `ClearWindowStyle` and `ClearPopStyle`.

**Alternative considered**: count uncleared entries in-memory before writing. Rejected because it adds a read-modify-write race with the Stop hook's concurrent writes. Delegating to the store makes the check happen after the write with file-level serialization.

### 2. Remove `liveSet[r.Pane]` guard from `loadEntries`; add "gone" indicator

Remove the live-pane filter entirely. Build the live set only for the purpose of rendering a visual marker (e.g. `(gone)` in the PATH column) when the pane no longer exists. All uncleared entries are eligible for display.

**Alternative considered**: keep the filter but add a separate "stale notifications" section. Rejected because it adds UI complexity and the user just needs to see the entry to dismiss it.

**Alternative considered**: auto-clear entries for gone panes on dashboard open. Rejected because the notification represents real work that finished — silently discarding it would cause the user to miss the signal.

### 3. `PanePath` failure is already non-fatal

When the pane is gone, `tmux display-message -t paneID` fails. The existing code already ignores this error (`p, _ := tmuxclient.PanePath(r.Pane)`), so `p` is empty and `trimPath("", home)` returns `""`. We use this empty path as the signal to append `(gone)` in the row display.

## Risks / Trade-offs

- **Window-style not cleared on last pane dismissed**: If `UnclearedForWindow` has a timing gap (Stop hook appends between `ClearPane` and `UnclearedForWindow`), window styles remain set when they should clear. Mitigation: acceptable — the hook would then immediately re-set them on the next notify call anyway; no double-clear risk.

- **Gone-pane entries accumulate if never dismissed**: If the user never opens the dashboard, old gone-pane entries persist in the JSONL forever. Mitigation: out of scope for this change; a future garbage-collection pass can prune gone entries older than N days.
