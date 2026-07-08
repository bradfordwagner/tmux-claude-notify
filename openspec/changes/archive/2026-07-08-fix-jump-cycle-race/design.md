## Context

`internal/store/store.go` persists notifications as a JSONL file with no database and no locking. Three kinds of processes mutate it independently and concurrently:

- The Stop hook (`runNotify`), which appends a new record when a `claude` session finishes.
- A per-notification auto-reset subprocess (`forkAutoReset` → `runAutoReset`/`clearAfterGracePeriod`), forked at notify time, that polls every 2s and eventually calls `runClear` → `store.ClearPane` on its own.
- `claude-notify jump` (`runJump`), which reads `store.OldestUncleared()` and then separately calls `runClear` → `store.ClearPane`.

`ClearPane` and `UpdateStatus` both follow the same pattern: read the entire JSONL file into memory, mutate the in-memory slice, then atomically replace the file via a temp-file + `os.Rename`. The rename itself is atomic, but the *read* that precedes it is not synchronized with any other writer's rename. With one waiting notification this never surfaces. With several notifications waiting at once — which is exactly the scenario in the bug report — there are multiple auto-reset subprocesses alive simultaneously, each on its own timer, each capable of doing a full read-modify-write of the same file. A comment already in `ClearPane` ("Multiple uncleared records can accumulate from a race between the Stop hook subprocess and the watcher") shows the maintainers are aware concurrent writers already collide here.

The failure mode: writer P1 reads the file at state S0 (A, B, C all uncleared) and marks A cleared in memory. Writer P2 (e.g. an auto-reset subprocess for a different pane, or another `jump` invocation) reads the file at the same S0, before P1's rename lands, and marks its own pane cleared in memory. Whichever of P1/P2 renames last silently overwrites the other's clear — A can revert from cleared back to uncleared without any process observing an error. Since `OldestUncleared()` picks the smallest timestamp among uncleared records, a reverted A (the oldest) is what `jump` keeps landing on, producing the "stuck on the first notification" symptom.

## Goals / Non-Goals

**Goals:**
- Make every read-modify-write cycle against the notifications JSONL file mutually exclusive across processes, so a clear performed by one process can never be silently reverted by another.
- Make `jump`'s "pick the oldest uncleared pane and clear it" a single atomic operation, removing the TOCTOU window between reading `OldestUncleared()` and calling `ClearPane()`.
- Keep the fix process-safe (works across independently-forked OS processes, not just goroutines within one binary), since the real writers here are separate `os.StartProcess` invocations and separate `run-shell` invocations.
- Ensure the status bar is refreshed after every `jump` attempt regardless of whether the clear step errors.

**Non-Goals:**
- Changing the observable navigation semantics of `jump` (still: oldest uncleared pane, cleared immediately, per the existing `quickjump` spec). This change only makes that existing behavior race-free.
- Introducing a database, daemon, or IPC mechanism. The fix stays within the existing "flat JSONL file, no daemon" architecture (see `architecture.md`).
- Deduplicating or compacting the JSONL log (unbounded growth is out of scope for this change).

## Decisions

**Use `syscall.Flock` on a sidecar lock file, not a new dependency.** `golang.org/x/sys` is already present only as an indirect dependency; adding a direct flock library (e.g. `gofrs/flock`) would be a new dependency for one syscall. `syscall.Flock` is available in the stdlib on both Linux and Darwin (this plugin's only supported platforms — WSL2/Linux and macOS; no Windows target per `CLAUDE.md`). The lock file lives at `notifications.jsonl.lock` next to the log, opened/created on demand, and held only for the duration of the read-modify-write cycle (`LOCK_EX`, blocking).

**Add one new atomic store function, `store.ClearOldestUncleared()`, instead of adding a generic "transaction" abstraction.** `runJump` currently calls `OldestUncleared()` then `ClearPane(pane)` as two separate operations. Collapsing "read all, find oldest, mark it cleared, write" into one function under one lock acquisition removes the TOCTOU gap without inventing a general transaction API that nothing else needs yet. `OldestUncleared()` stays as-is (still used for read-only inspection, e.g. potential future dashboard use) but is no longer used by `runJump`.

**Wrap `ClearPane` and `UpdateStatus` bodies in the same lock helper** (`withStoreLock(func() error) error`), rather than only fixing the `jump` path. `runNotify`'s idempotency check (`HasUnclearedPane` before `Append`) and the auto-reset subprocesses' `ClearPane` calls are the other read-modify-write writers that can race with `jump` or each other; leaving them unlocked would still allow the same lost-update failure mode to happen via those paths (e.g. two auto-reset subprocesses for different panes clearing at the same moment). `Append` itself is left unlocked: it's a pure `O_APPEND` write of one small JSON line with no preceding read, so it can't lose another writer's update (worst case with true POSIX atomicity limits is a torn line, deemed acceptable — existing `ReadAll` already skips unparseable lines).

**Always call `RefreshStatusBar()` in `runJump`, even when the clear step errors.** Currently `runJump` returns early on a `runClear` error, skipping the refresh. Since the lock acquisition is now part of the clear path, a lock timeout or I/O error becomes marginally more visible; refreshing regardless keeps the badge honest with whatever the store actually contains, matching the proposal's "force a re-render" ask.

## Risks / Trade-offs

- **[Risk] `Flock` blocks the calling process while another writer holds the lock.** Every writer in this system does a small, fast file read/rewrite (JSONL files here are tiny — tens of entries at most), so worst-case blocking is on the order of milliseconds. → Mitigation: no timeout is added initially; if contention ever becomes visible, a bounded `LOCK_EX | LOCK_NB` retry loop can be added later without changing the public store API.
- **[Risk] A crashed process could theoretically hold the lock forever.** `flock` locks are released automatically by the kernel when the holding file descriptor is closed, including on process exit/crash, so this is not a real risk in practice. → Mitigation: none needed; documented here for future readers.
- **[Trade-off] This only fixes JSONL-store races; it does not address any equivalent race in tmux option writes** (`SetWindowStyle`/`ClearWindowStyle` are separate `tmux set-option` calls, not file writes, and tmux itself serializes those). Out of scope — no evidence they contribute to the reported symptom.

## Migration Plan

No data migration needed — the JSONL schema is unchanged. Roll out is a normal binary rebuild (`tmux-claude-notify.tmux` already recompiles `bin/claude-notify` from source on TPM load/update). Rollback is reverting the commit; the lock file is harmless to leave behind if the binary is downgraded (older binaries simply ignore it).

## Open Questions

None — the fix is scoped to the store package and `runJump`; no external decisions are pending.
