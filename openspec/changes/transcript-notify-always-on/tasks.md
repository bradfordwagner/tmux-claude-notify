## 1. Transcript Format Research

- [ ] 1.1 Read real Claude Code transcript files in `~/.claude/projects/` to confirm JSONL event schema (field names: `type`, `role`, `content`, tool event structure)
- [ ] 1.2 Document confirmed event types and state derivation mapping in design.md Open Questions section

## 2. Store Schema Extension

- [ ] 2.1 Add `Status string` field to `store.Record` (values: `running`, `waiting`, `idle`, `stale`); default to `waiting` when field absent on read
- [ ] 2.2 Implement `store.UpdateStatus(paneID, status string) error` — atomic rewrite of most recent uncleared record's status field
- [ ] 2.3 Update `store.Append` to accept and write the `Status` field

## 3. Transcript Watcher Package

- [ ] 3.1 Create `internal/watcher/watcher.go` with `Watcher` struct; fsnotify watch on `~/.claude/projects/` directory
- [ ] 3.2 Implement transcript file discovery: scan `~/.claude/projects/**/*.jsonl`, skip files modified more than 24 hours ago
- [ ] 3.3 Implement encoded-path decoder: `-home-bw-foo-bar` → `/home/bw/foo/bar`
- [ ] 3.4 Implement state derivation from tail of transcript file: `tool_use` → `running`, `assistant` message + silence ≥2s → `waiting`, `user` message → clear
- [ ] 3.5 Implement stale detection: no transcript activity for >5 minutes → `stale`
- [ ] 3.6 Implement pane correlation: match decoded project dir against `pane_current_path` of live tmux panes where `pane_current_command` matches prefix `claude*`
- [ ] 3.7 On state transition to `waiting`: call `store.Append` (if no uncleared entry) or `store.UpdateStatus`; call `tmuxclient.SetWindowStyle`
- [ ] 3.8 On state transition to `running`/`stale`: call `store.UpdateStatus` only (no window style change)
- [ ] 3.9 On `user` message detected (user responded): call `store.ClearPane` and `tmuxclient.ClearWindowStyle`
- [ ] 3.10 Expose `Reconcile() []StateChange` method: scans active transcripts, returns current state per pane for use at dashboard open

## 4. Dashboard Integration

- [ ] 4.1 Update `ui.newModel()` to create and start a `watcher.Watcher` alongside the existing fsnotify watcher
- [ ] 4.2 Wire watcher state-change events into bubbletea `Cmd` so dashboard re-renders on transcript updates
- [ ] 4.3 Call `watcher.Reconcile()` during `newModel()` after loading JSONL entries; apply corrections (ClearPane / UpdateStatus) before first render
- [ ] 4.4 Close watcher in TUI exit path (`q`/`esc`/`ctrl+c`)
- [ ] 4.5 Update entry rendering to show `status` field per entry with distinct styles: `waiting` → accent, `running` → warn, `stale` → dim

## 5. Documentation

- [ ] 5.1 Update `architecture.md` with transcript watcher data flow and reconciliation on open
- [ ] 5.2 Update `CLAUDE.md` "How it works" section
- [ ] 5.3 Update `README.md` usage table and description
