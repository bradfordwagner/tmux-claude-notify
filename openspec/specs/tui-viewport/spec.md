# Spec: tui-viewport

## Purpose

Scrollable viewport for the dashboard TUI — cursor movement scrolls the content window; the selected row is always kept visible; scroll indicators appear when content overflows. Prevents entries beyond the terminal height from being silently invisible and unreachable.

## Requirements

### Requirement: Dashboard tracks terminal height
The model SHALL store the terminal height received from `tea.WindowSizeMsg` and expose a `termHeight()` helper that returns a safe default of `24` when height has not yet been received.

#### Scenario: Height stored on WindowSizeMsg
- **WHEN** a `tea.WindowSizeMsg` is received
- **THEN** `m.height` is set to `msg.Height`

#### Scenario: Safe default before first WindowSizeMsg
- **WHEN** `termHeight()` is called before any `WindowSizeMsg` has been received
- **THEN** it returns `24`

### Requirement: Dashboard computes viewport height from fixed overhead
Before rendering the entry list, the model SHALL compute `viewportHeight = max(1, termHeight() - fixedRows)` where `fixedRows` counts:
- 1 for the tab header
- 1 if `searchMode` is true
- the number of non-empty lines returned by `renderSetupStatus()` (0 or 2), counted only in Notifications view
- 1 if a toast is active
- 1 for the blank separator
- 1 for the footer hint line

#### Scenario: Viewport height shrinks when search bar is visible
- **WHEN** `searchMode` is true and the terminal is 30 rows tall with no toast and no setup warning
- **THEN** `viewportHeight` is `30 - 1 - 1 - 0 - 0 - 1 - 1 = 26`

#### Scenario: Viewport height shrinks when setup warning is shown
- **WHEN** the Notifications view is active, `setupStatus` is `StatusNotConfigured`, and the terminal is 30 rows tall with no search and no toast
- **THEN** `viewportHeight` is `30 - 1 - 0 - 2 - 0 - 1 - 1 = 25`

### Requirement: Entry list rendered as a bounded window using scroll offset
Each view's render function SHALL slice the filtered list to `visible = filtered[scrollOffset : min(scrollOffset+viewportHeight, len(filtered))]` and render only that window. The header row and separator appear above the visible window regardless of scroll position.

#### Scenario: First viewport window shown when list exceeds height
- **WHEN** there are 20 entries and `viewportHeight` is 10 and `scrollOffset` is 0
- **THEN** entries 0–9 are rendered (10 rows)
- **AND** entries 10–19 are not rendered

#### Scenario: Scroll reveals entries below fold
- **WHEN** `scrollOffset` is 5 and `viewportHeight` is 10
- **THEN** entries 5–14 are rendered

### Requirement: Scroll offset is a single field reset on context change
The model SHALL maintain one `scrollOffset int` field. It SHALL be reset to `0` whenever:
- the active view changes (tab key)
- `drillProject` changes (enter to drill, esc to back)
- `searchQuery` changes

#### Scenario: scrollOffset resets on tab
- **WHEN** the user presses `tab` to switch views
- **THEN** `scrollOffset` is set to `0`

#### Scenario: scrollOffset resets on drill
- **WHEN** the user presses `enter` to drill into a project in Sessions L1
- **THEN** `scrollOffset` is set to `0`

#### Scenario: scrollOffset resets on search query change
- **WHEN** the search query changes (character typed or cleared)
- **THEN** `scrollOffset` is set to `0`

### Requirement: Cursor movement enforces scroll-follows-cursor invariant
After any cursor position change, the model SHALL adjust `scrollOffset` to satisfy:
```
scrollOffset <= cursor < scrollOffset + viewportHeight
```
using minimum-scroll logic:
- if `cursor < scrollOffset`: set `scrollOffset = cursor`
- if `cursor >= scrollOffset + viewportHeight`: set `scrollOffset = cursor - viewportHeight + 1`

#### Scenario: Cursor moves below visible window
- **WHEN** the cursor advances past `scrollOffset + viewportHeight - 1`
- **THEN** `scrollOffset` is incremented so the cursor row becomes the last visible row

#### Scenario: Cursor moves above visible window
- **WHEN** the cursor retreats past `scrollOffset`
- **THEN** `scrollOffset` is decremented so the cursor row becomes the first visible row

#### Scenario: Cursor within window — no scroll
- **WHEN** the cursor moves but stays within `[scrollOffset, scrollOffset + viewportHeight)`
- **THEN** `scrollOffset` is unchanged

### Requirement: ctrl+d / ctrl+u scroll half a page and move the cursor
`ctrl+d` SHALL scroll the viewport down by half a page and advance the cursor by the same amount (clamped to the last entry). `ctrl+u` SHALL scroll the viewport up by half a page and retreat the cursor by the same amount (clamped to 0). After either key, the cursor is always within the visible window.

#### Scenario: ctrl+d advances cursor half a page
- **WHEN** the cursor is at index 3, `vp.Height` is 10, and the list has 20 entries
- **THEN** the cursor moves to index 8 (3 + 5) and the viewport scrolls so the cursor is visible

#### Scenario: ctrl+d at end of list clamps cursor
- **WHEN** the cursor is at index 18 and the list has 20 entries
- **THEN** the cursor moves to index 19 (last entry) and the viewport does not scroll past the end

#### Scenario: ctrl+u retreats cursor half a page
- **WHEN** the cursor is at index 12 and `vp.Height` is 10
- **THEN** the cursor moves to index 7 (12 - 5) and the viewport scrolls so the cursor is visible

#### Scenario: ctrl+u at top of list clamps cursor
- **WHEN** the cursor is at index 2 and `vp.Height` is 10
- **THEN** the cursor moves to index 0 and the viewport scrolls to the top

### Requirement: Scroll position indicator in footer when list overflows
When `len(filtered) > viewportHeight`, the footer hint line SHALL append a right-padded position indicator ` <cursor+1>/<total>` to the existing hint text (right-aligned using remaining terminal width).

#### Scenario: Position indicator shown when scrolling is active
- **WHEN** there are 20 entries, `viewportHeight` is 10, and the cursor is at index 4
- **THEN** the footer shows `5/20` appended after the hint text

#### Scenario: Position indicator absent when all entries fit
- **WHEN** there are 5 entries and `viewportHeight` is 10
- **THEN** the footer shows no position indicator
