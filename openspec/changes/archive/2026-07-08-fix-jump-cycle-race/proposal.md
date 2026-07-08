## Why

`claude-notify jump` is supposed to clear the notification it navigates to so a repeated hotkey press advances to the next-oldest waiting pane. In practice, when several notifications are waiting, repeated presses can land on the same pane over and over instead of cycling through the badge. The root cause is that the JSONL notification store (`internal/store/store.go`) has no locking around its read-modify-write cycles (`ClearPane`, `UpdateStatus`). Every notification spawns a background auto-reset subprocess that independently reads the whole file, decides to clear its own pane, and rewrites the whole file — and `runJump` itself performs "find oldest" and "clear it" as two separate, unsynchronized file operations. When two of these full-file rewrites overlap (which happens routinely, since 2-15s auto-reset timers are running concurrently for every outstanding notification), the rewrite that lands last wins and silently reverts clears made by the other, so a pane that was already jumped-to can reappear as uncleared and get re-selected as "oldest" on the next press.

## What Changes

- Add file locking around all read-modify-write operations on the notifications JSONL store so concurrent writers (Stop hook, auto-reset subprocesses, `jump`) can no longer clobber each other's updates.
- Replace `runJump`'s two-step "find oldest, then clear" sequence with a single atomic store operation that finds and clears the oldest uncleared record under one lock, closing the gap where a concurrent writer could act on stale data in between.
- Always force a status-bar refresh (`tmux refresh-client -S`) after a jump attempt, including when the clear step reports an error, so the badge never lags behind the true store state.
- Add a regression test that exercises concurrent clears/updates against the store and asserts no clear is ever lost, plus a test that repeated `jump`-style calls against N waiting panes visit N distinct panes with no repeats.

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
- `quickjump`: `claude-notify jump` gains an explicit atomicity/concurrency-safety guarantee — clearing the oldest uncleared pane must be race-free with respect to other concurrent writers (Stop hook, auto-reset subprocesses, other `jump` invocations), and the status bar must be refreshed even when the clear step fails.

## Impact

- `internal/store/store.go`: add a file-lock helper; wrap `ClearPane`/`UpdateStatus` in it; add a new atomic "find oldest and clear" function.
- `cmd/claude-notify/main.go`: `runJump` uses the new atomic store function; status bar refresh moves outside the error-short-circuit.
- No new external dependencies (uses stdlib `syscall.Flock`, Unix-only — consistent with this plugin's existing Linux/WSL/macOS-only scope).
- No behavior change visible to the user in the common (uncontended) case — this is a correctness fix for a race that only manifests under concurrent JSONL writes.
