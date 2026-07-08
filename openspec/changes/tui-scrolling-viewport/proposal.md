## Why

The dashboard TUI renders all entries as a flat list with no viewport — when the number of notifications plus session rows exceeds the terminal height, entries below the fold are silently invisible and unreachable. Users with many active or pinned sessions have no way to see or act on entries that don't fit on screen.

## What Changes

- Add a scrollable viewport to the Notifications view so entries beyond terminal height are reachable via cursor movement
- Add a scrollable viewport to the Sessions view (both Level-1 projects table and Level-2 sessions drill-down)
- Ensure the selected cursor row is always kept visible (scroll-follows-cursor)
- Show a scroll position indicator (e.g. `3/12`) in each view's footer when the list is longer than the viewport
- Remove the implicit assumption of a fixed terminal height from all three render paths

## Capabilities

### New Capabilities

- `tui-viewport`: Scrollable viewport for the dashboard TUI — cursor movement scrolls the content window; the selected row is always kept visible; scroll indicators appear when content overflows.

### Modified Capabilities

- `always-on-dashboard`: The dashboard render model changes — views now render into a bounded height rather than unbounded strings. The header, search bar, toast, and footer are fixed; the remaining height is the viewport.

## Impact

- `internal/ui/model.go`: Primary change — model gains a `viewport.Model` (or a manual `scrollOffset int`); `renderNotificationsView`, `renderSessionsView`, and `renderTabHeader` adapt to bounded height; `Update` handles `tea.WindowSizeMsg` to recalculate viewport height; cursor movement triggers scroll offset recalculation.
- `go.mod` / `go.sum`: `github.com/charmbracelet/bubbles/viewport` is already in the charmbracelet ecosystem; if not already a direct dependency, add it. (Current imports show `bubbles/textinput` and `bubbles/timer` are already used, so the module is present.)
- No changes to `store`, `sessions`, `watcher`, `tmux`, or `resurrect` packages.
- No changes to the TPM entry point or hook wiring.
