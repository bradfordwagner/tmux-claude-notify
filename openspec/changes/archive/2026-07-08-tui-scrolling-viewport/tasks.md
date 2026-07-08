## 1. Model Setup

- [x] 1.1 Add `height int` field to the `model` struct in `internal/ui/model.go`
- [x] 1.2 Add `vp viewport.Model` field to the `model` struct (import `github.com/charmbracelet/bubbles/viewport`)
- [x] 1.3 Add `termHeight() int` helper that returns `m.height` if > 0, else `24`
- [x] 1.4 Initialize `m.vp = viewport.New(termWidth(), termHeight()-fixedRows)` in `newModel()` before the bubbletea program starts

## 2. Viewport Resize Helper

- [x] 2.1 Add `countLines(s string) int` helper: `strings.Count(strings.TrimRight(s, "\n"), "\n") + 1` if s non-empty, else `0`
- [x] 2.2 Add `fixedRowCount() int` method: 1 (tab header) + (1 if searchMode) + countLines(renderSetupStatus()) for Notifications view + (1 if toast active) + 1 (blank) + 1 (footer)
- [x] 2.3 Add `recalcViewport()` method: sets `m.vp.Width = m.termWidth()`, `m.vp.Height = max(1, m.termHeight() - m.fixedRowCount())`, then calls `m.vp.SetContent(m.renderListContent())`
- [x] 2.4 Call `recalcViewport()` in the `tea.WindowSizeMsg` handler (set `m.height = msg.Height` and `m.width = msg.Width` first)

## 3. List Content Renderer

- [x] 3.1 Extract list-only rendering from `renderNotificationsView` into `renderNotificationsContent() string` (header + separator + all data rows, no footer hint)
- [x] 3.2 Extract list-only rendering from `renderSessionsView` (L1 path) into `renderSessionsL1Content() string`
- [x] 3.3 Extract list-only rendering from `renderSessionsView` (L2 path) into `renderSessionsL2Content() string`
- [x] 3.4 Add `renderListContent() string` dispatcher that calls the correct content renderer based on `m.activeView` and `m.drillProject`

## 4. View() Layout Wiring

- [x] 4.1 Rewrite `View()` to render fixed elements (tab header, search bar, setup status, toast, blank) as a header string above `m.vp.View()`
- [x] 4.2 Render the footer hint line below `m.vp.View()` — append `  <cursor+1>/<total>` right-padded when `total > m.vp.Height`
- [x] 4.3 Remove the old footer rendering from all three per-view render functions (footer is now always rendered by `View()`)

## 5. Half-Page Scroll Keys

- [x] 5.1 In the `tea.KeyMsg` handler, intercept `ctrl+d`: compute `half = max(1, m.vp.Height/2)`, advance `m.cursor` by `half` clamped to `len(filtered)-1`, call `m.vp.HalfViewDown()`, then `ensureCursorVisible()`
- [x] 5.2 Intercept `ctrl+u`: retreat `m.cursor` by `half` clamped to `0`, call `m.vp.HalfViewUp()`, then `ensureCursorVisible()`
- [x] 5.3 Confirm these keys are intercepted before the viewport's own key handler so they do not double-fire

## 6. Scroll-Follows-Cursor

- [x] 6.1 Add `ensureCursorVisible()` method: adjusts `m.vp.SetYOffset` using minimum-scroll logic so cursor line is within `[vp.YOffset, vp.YOffset + vp.Height)`, accounting for header rows in content (offset cursor index by 2 for header + separator)
- [x] 6.2 Call `ensureCursorVisible()` after `up`/`k` key handler
- [x] 6.3 Call `ensureCursorVisible()` after `down`/`j` key handler
- [x] 6.4 Call `recalcViewport()` (which resets content) then `ensureCursorVisible()` in `clampCursor` and `clampCursorToFiltered`

## 7. Scroll Reset on Context Change

- [x] 7.1 Call `recalcViewport()` (resets content and scroll to top via `SetContent`) on view switch (tab key, both directions)
- [x] 7.2 Call `recalcViewport()` when `drillProject` changes (enter to drill, esc to back-navigate)
- [x] 7.3 Call `recalcViewport()` when `searchQuery` changes (character typed or cleared via the textinput update path)
- [x] 7.4 Call `recalcViewport()` in `notificationsChangedMsg` and `watcher.StateChange` handlers so data reloads update viewport content

## 8. Recalc on Fixed-Element Visibility Change

- [x] 8.1 Call `recalcViewport()` in `settingsChangedMsg` handler (setup status may appear/disappear, changing fixedRows)
- [x] 8.2 Call `recalcViewport()` after toast timer starts (toast line appears) and in `timer.TimeoutMsg` handler (toast disappears)
- [x] 8.3 Call `recalcViewport()` after search mode is toggled on or off (search bar line appears/disappears)

## 9. Verification

- [x] 9.1 Build (`task build`) and confirm no compile errors
- [x] 9.2 Open the dashboard with more sessions than terminal height and confirm entries scroll as cursor moves with j/k
- [x] 9.3 Confirm `ctrl+d` scrolls down half a page and moves the cursor with it
- [x] 9.4 Confirm `ctrl+u` scrolls up half a page and moves the cursor with it
- [x] 9.5 Confirm the position indicator (e.g. `3/20`) appears in the footer when scrolling is active and is absent when all entries fit
- [x] 9.6 Confirm switching views (Tab) resets scroll to top
- [x] 9.7 Confirm resizing the terminal window dynamically recalculates viewport height
- [x] 9.8 Confirm searching resets scroll to top and selected row stays visible while typing
