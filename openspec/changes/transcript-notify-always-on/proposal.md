## Why

The Stop hook only fires at the end of a complete claude turn, which means there is a window where claude is actively doing work (running tools, waiting for permission approval) but no notification exists yet. Transcript-based detection reads Claude Code's own JSONL session files to derive richer agent state in near-realtime.

The dashboard must also handle the case where it is not running — the Stop hook and transcript watcher both write to the JSONL store so state accumulates whether or not the TUI is open. When the dashboard opens, it reads current transcript state to catch up on anything that happened while it was closed.

## What Changes

- The dashboard TUI (`claude-notify` with no args) starts a transcript watcher internally when it launches; no separate daemon subcommand or background process
- Add a transcript watcher that tails `~/.claude/projects/**/*.jsonl` to detect agent state changes (`running`, `waiting`, `idle`, `stale`) without relying solely on the Stop hook
- Keep the Stop hook as a secondary signal (belt-and-suspenders) — writes to JSONL store even when the dashboard is not open
- On dashboard open: reconcile JSONL store state with current transcript state so stale entries are corrected
- The existing toggle mechanism (`C-M-p` / grimoire shpell) is unchanged — the popup model is preserved
- Notifications are derived from transcript state in addition to hook invocations

## Capabilities

### New Capabilities

- `transcript-watcher`: Watches Claude Code JSONL transcript files to detect agent state (`running`, `waiting`, `idle`, `stale`). Fires notify/clear events based on transcript activity. Runs embedded in the dashboard TUI process.

### Modified Capabilities

- `hook-setup`: Stop hook becomes supplementary — primary detection when dashboard is open is transcript-based; Stop hook writes to JSONL store as fallback for when dashboard is closed.
- `window-highlight`: Highlight lifecycle driven by transcript state transitions (when dashboard open) or Stop hook (when closed).
- `notification-log`: Store extended with `status` field (`running`/`waiting`/`idle`/`stale`) and a new `UpdateStatus` method. On dashboard open, store is reconciled against live transcript state.
- `dashboard`: TUI starts transcript watcher on init; reconciles store state on open.

## Impact

- `internal/watcher/` — new package for fsnotify-based transcript file watching and state derivation
- `internal/store/store.go` — record schema extended with `status` field; new `UpdateStatus` method
- `internal/ui/model.go` — starts watcher on init, reconciles state on open, renders `status` per entry
- `cmd/claude-notify/main.go` — no new subcommands; default invocation unchanged
- `tmux-claude-notify.tmux` — no changes needed (TPM entry point unchanged)
- New dependency: none beyond existing fsnotify (already used for settings watch in UI)
