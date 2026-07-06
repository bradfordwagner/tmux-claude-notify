## 1. Tmux Client

- [x] 1.1 Add `PanePath(paneID string) (string, error)` to `internal/tmux/tmux.go` — runs `tmux display-message -p -t <pane> '#{pane_current_path}'` and returns the output trimmed of whitespace

## 2. UI Model — Data Layer

- [x] 2.1 Add `Path string` field to the `entry` struct in `internal/ui/model.go`
- [x] 2.2 In `loadEntries()`, call `tmuxclient.PanePath(r.Pane)` for each entry; truncate to last 2 path components and replace `$HOME` with `~`; store result in `entry.Path` (silently ignore errors, leaving `Path` empty)

## 3. UI Model — Rendering

- [x] 3.1 Fix `renderStatusBadge()`: pad the plain status string to fixed width first, then apply the lipgloss color style (so ANSI codes don't inflate column widths)
- [x] 3.2 Add status icons to `renderStatusBadge()`: prepend `⏳` for `waiting`, `⚙ ` for `running`, `💤` for `stale`
- [x] 3.3 Rewrite the row format in `View()`: column order is Status | Window | Path | Session | Age; remove the Pane ID column
- [x] 3.4 Add a header row (`STATUS`, `WINDOW`, `PATH`, `SESSION`, `AGE`) and a separator line rendered above the entry list when entries are present

## 4. Verification

- [x] 4.1 Run `task build` to confirm the binary compiles cleanly
- [x] 4.2 Open the dashboard (`C-M-p`) with at least one active claude session; confirm columns align, icons appear, path column shows correctly, and headers render above entries
- [x] 4.3 Confirm the empty state ("No pending notifications.") still renders without headers or separator

