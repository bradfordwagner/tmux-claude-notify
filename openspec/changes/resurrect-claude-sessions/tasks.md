## 1. Resurrect Package

- [x] 1.1 Create `internal/resurrect/resurrect.go`: define `ResurrectPane` struct (`TmuxSession`, `WindowIndex`, `PaneIndex`, `PaneID`, `SessionID`, `ProjectPath`) and `ResurrectState` struct (`Version int`, `Panes []ResurrectPane`)
- [x] 1.2 Add `Load(dataDir string) (ResurrectState, error)` — reads sidecar JSON; returns empty state if file absent
- [x] 1.3 Add `(s ResurrectState) Write(dataDir string) error` — writes sidecar atomically via temp file + rename
- [x] 1.4 Add `ResurrectFilePath(dataDir string) string` — resolves path from `@claude-notify-resurrect-file` tmux option, falling back to `dataDir/resurrect.json`

## 2. Save Logic

- [x] 2.1 Add `Save(dataDir string) error` in `internal/resurrect/resurrect.go`: call `watcher.ListClaudePanes()` (export or inline) to enumerate active `claude*` panes
- [x] 2.2 Load `sessions.jsonl` via `sessions.ReadAll()`; build a `paneID → SessionRecord` map for fast lookup
- [x] 2.3 For each active pane: look up `session_id` and `project_path` from the sessions map by `pane_id`; if absent, derive `session_id` from latest transcript filename in `~/.claude/projects/<encoded-path>/`
- [x] 2.4 Skip panes where no `session_id` can be derived; build `ResurrectState` and call `Write()`

## 3. Restore Logic

- [x] 3.1 Add `Restore(dataDir string) error` in `internal/resurrect/resurrect.go`: call `Load()` and `watcher.ListAllPanes()` (positional list of all panes with `session/window_index/pane_index/pane_id/current_command/current_path`)
- [x] 3.2 Build a `(session, windowIdx, paneIdx) → TmuxPane` map from live panes
- [x] 3.3 For each saved entry: skip if pane not found or `pane_current_command` starts with `claude`; otherwise send `claude --resume <session_id>` (prepend `cd <project_path> && ` when `pane_current_path` differs)

## 4. Subcommand Wiring

- [x] 4.1 Add `resurrect` subcommand group to `cmd/claude-notify/main.go` with `save` and `restore` child commands
- [x] 4.2 Both subcommands resolve `dataDir` the same way as existing `notify`/`clear` commands

## 5. Docs

- [x] 5.1 Add `@claude-notify-resurrect-file` row to the Configuration table in `README.md` (default: `~/.local/share/tmux-claude-notify/resurrect.json`)
- [x] 5.2 Add a "Resurrect" section to `README.md` describing the two commands and how to wire them into tmux-resurrect hooks
- [x] 5.3 Update `architecture.md` to include the resurrect save/restore flow
- [x] 5.4 Add tmux-resurrect hook wiring example to `README.md` (show `@resurrect-hook-pre-save` and `@resurrect-hook-post-restore-all` with full binary path)
- [x] 5.5 Update `~/dotfiles/dots/tmux/tmux.conf` — chain `claude-notify resurrect save/restore` into the existing `@resurrect-hook-pre-save` and `@resurrect-hook-post-restore-all` lines
