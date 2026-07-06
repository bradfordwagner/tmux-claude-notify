## Why

The dashboard notification list is hard to read: columns have no headers, the raw pane ID (`%34`) is meaningless to users, status is buried mid-row rather than leading, and a subtle alignment bug (ANSI escape codes inflating `fmt.Sprintf` width counters) causes column drift. Adding a path column would also tell users which project each waiting session belongs to — something the window name alone doesn't convey.

## What Changes

- Add column headers and a separator line above the notification list
- Reorder columns: Status first (the decision signal), then Window, Path, Session, Age
- Add status icons alongside the existing status colors: ⏳ waiting, ⚙ running, 💤 stale
- Replace the raw pane ID column with a `Path` column showing the pane's current directory (truncated to the last 2 components, `$HOME` → `~`)
- Fix the alignment bug: pad plain status text before applying lipgloss color so ANSI codes don't skew column widths

## Capabilities

### New Capabilities

- `dashboard-row-layout`: The visual structure of each notification row in the TUI — column order, headers, icons, and path display.

### Modified Capabilities

- `always-on-dashboard`: The TUI rendering and entry model gain a Path field and revised View() output.

## Impact

- `internal/ui/model.go` — `entry` struct, `loadEntries()`, `View()`, `renderStatusBadge()`
- `internal/tmux/tmux.go` — new `PanePath(paneID string)` helper
- No changes to the notification store schema, hook wiring, or external behavior
