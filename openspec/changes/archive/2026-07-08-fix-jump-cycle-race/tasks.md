## 1. Store locking

- [x] 1.1 Add a `withStoreLock(func() error) error` helper in `internal/store/store.go` that opens/creates `notifications.jsonl.lock` next to `LogPath()` and holds `syscall.Flock(fd, syscall.LOCK_EX)` for the duration of the callback, unlocking and closing on return.
- [x] 1.2 Wrap `ClearPane`'s existing read-modify-write body in `withStoreLock`.
- [x] 1.3 Wrap `UpdateStatus`'s existing read-modify-write body in `withStoreLock`.

## 2. Atomic clear-oldest

- [x] 2.1 Add `store.ClearOldestUncleared() (*Record, error)` that, under one `withStoreLock` acquisition, reads all records, finds the uncleared record with the smallest `ts`, marks it cleared, writes the file back, and returns the (now-cleared) record — mirroring the read/rewrite pattern already used by `ClearPane`.
- [x] 2.2 Update `runJump` in `cmd/claude-notify/main.go` to call `store.ClearOldestUncleared()` in place of the current `store.OldestUncleared()` + `runClear(r.Pane)` pair, keeping the rest of `runClear`'s non-store side effects (pop style, clear hook unregister, window style teardown) invoked explicitly around the new call so window/pane tmux state is still cleaned up.
- [x] 2.3 Move `tmuxclient.RefreshStatusBar()` so it always runs before `runJump` returns, even when the clear step returns an error.

## 3. Tests

- [x] 3.1 Add `internal/store/store_test.go` with a concurrency test: launch N goroutines each calling `ClearPane` for a distinct pane against a shared temp JSONL file, then assert all N panes end up marked cleared (no lost updates).
- [x] 3.2 Add a test in `internal/store/store_test.go` for `ClearOldestUncleared`: given three uncleared records with distinct `ts`, three sequential calls return the three panes in oldest-first order with no repeats, and a fourth call on an empty store returns `nil, nil`.
- [x] 3.3 Add a test (in `cmd/claude-notify` or `internal/store`, whichever avoids import cycles) that simulates the reported bug scenario: three panes with uncleared entries plus concurrent background goroutines calling `ClearPane` on other panes (simulating auto-reset subprocesses), while three sequential `ClearOldestUncleared` calls run — assert each of the three original panes is returned exactly once across the three calls.

## 4. Docs

- [x] 4.1 Update `architecture.md` to note the JSONL store's file-lock guarantee for concurrent writers (Stop hook, auto-reset subprocesses, jump).
- [x] 4.2 Run `openspec archive fix-jump-cycle-race` once implementation and tests pass, per this repo's spec-driven workflow.
