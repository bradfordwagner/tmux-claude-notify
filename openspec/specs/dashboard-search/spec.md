# Spec: dashboard-search

## Purpose

Defines the in-TUI search/filter capability for the always-on dashboard. When active, a text input appears below the tab header and the visible row list is filtered in real time using fuzzy character-subsequence matching. Search mode operates across all views (Notifications, Sessions L1, Sessions L2) with a consistent two-focus-state model (input-focused vs. table-focused).

## Requirements

### Requirement: `/` hotkey enters search mode with input focused
Pressing `/` in any view SHALL activate search mode: a text input appears below the tab header, the input has keyboard focus, and keystrokes are routed to the filter input. The visible row list is filtered in real time.

#### Scenario: / enters search mode in Notifications view
- **WHEN** the user presses `/` while in Notifications view
- **THEN** `searchMode` becomes `true`, `searchFocus` becomes `true`, the filter input is focused and rendered below the tab header
- **AND** all subsequent alphanumeric keystrokes update the filter query

#### Scenario: / enters search mode in Sessions view L1
- **WHEN** the user presses `/` while in Sessions view at Level-1
- **THEN** `searchMode` becomes `true`, `searchFocus` becomes `true`, and the project list is filtered as the user types

#### Scenario: / enters search mode in Sessions view L2
- **WHEN** the user presses `/` while drilled into a project (Level-2)
- **THEN** `searchMode` becomes `true`, `searchFocus` becomes `true`, and the session list for that project is filtered

### Requirement: Search mode has two focus sub-states — input-focused and table-focused
When `searchMode` is `true`, the input can hold focus (`searchFocus=true`) or the table can hold focus (`searchFocus=false`). The filter remains active in both sub-states. `Tab` toggles between them without clearing the filter or switching views.

#### Scenario: Tab from input-focused moves focus to the table
- **WHEN** `searchMode` is `true`, `searchFocus` is `true`, and the user presses `Tab`
- **THEN** `searchFocus` becomes `false`, the textinput is blurred, and the filter stays active
- **AND** all normal navigation and action keys become available on the filtered list

#### Scenario: Tab from table-focused returns focus to the input
- **WHEN** `searchMode` is `true`, `searchFocus` is `false`, and the user presses `Tab`
- **THEN** `searchFocus` becomes `true`, the textinput is focused again
- **AND** alphanumeric keys route to the input

### Requirement: Fuzzy matching filters the active row list
While search mode is active (`searchMode=true`), the visible row list SHALL be filtered using fuzzy character-subsequence matching (`github.com/sahilm/fuzzy`). Rows that do not match the current query are hidden. Rows are displayed in match-score order (best match first).

#### Scenario: Fuzzy match narrows Notifications list
- **WHEN** the user types `dot` in search mode in Notifications view
- **THEN** only rows whose match target (window name + path + session) contain `dot` as a fuzzy subsequence are shown

#### Scenario: Fuzzy match narrows Sessions L1 list
- **WHEN** the user types `tmux` in search mode in Sessions L1
- **THEN** only project rows whose path contains `tmux` as a fuzzy subsequence are shown

#### Scenario: Fuzzy match narrows Sessions L2 list
- **WHEN** the user types `abc1` in search mode in Sessions L2
- **THEN** only session rows whose session ID or window name matches `abc1` as a fuzzy subsequence are shown

#### Scenario: Empty query shows all rows
- **WHEN** the filter query is empty (user cleared it or just entered search mode)
- **THEN** all rows are shown, as if search mode were inactive

#### Scenario: No match shows empty state
- **WHEN** the filter query matches no rows
- **THEN** the row area renders "No results for `<query>`" in dim style

### Requirement: Navigation and action keys operate on the filtered list
When `searchMode` is `true` (regardless of focus sub-state), `j/k/up/down` navigate the filtered list. `enter`, `p`, `w`, `h`, `v` act on the currently highlighted filtered row. The cursor is clamped to the filtered list length after every keystroke.

#### Scenario: enter acts on filtered row
- **WHEN** the user navigates to a row in the filtered list and presses `enter`
- **THEN** the same action fires as if search mode were not active (clear notification, drill in, resume, etc.)

#### Scenario: Cursor clamped after query update
- **WHEN** the user types a character that reduces the filtered list below the current cursor position
- **THEN** the cursor is moved to `max(0, len(filteredList)-1)`

### Requirement: `esc` exits search mode entirely
When `searchMode` is `true` (either focus sub-state), pressing `esc` SHALL clear the query, deactivate search mode, restore the full list, and return both `searchMode` and `searchFocus` to `false`. The dashboard does not quit and navigation does not change.

#### Scenario: esc exits search in Notifications view (input focused)
- **WHEN** `searchMode` is `true`, `searchFocus` is `true`, and the user presses `esc`
- **THEN** `searchMode` and `searchFocus` become `false`, `searchQuery` is cleared, and the full notification list is restored
- **AND** the dashboard does not close

#### Scenario: esc exits search in Sessions L2 before back-navigating
- **WHEN** the user is drilled into a project, `searchMode` is `true`, and presses `esc`
- **THEN** search mode exits; the user remains in L2 with the full session list restored
- **AND** a second `esc` then back-navigates to L1

### Requirement: Filter clears on view switch or drill navigation
`searchQuery`, `searchMode`, and `searchFocus` SHALL be reset when the user switches views (Tab when `searchMode=false`), drills into a project (enter at L1), or back-navigates out of a project (esc at L2 when not in search mode).

#### Scenario: View switch (Tab when not searching) clears filter
- **WHEN** `searchMode` is `false` and the user presses `Tab`
- **THEN** the view switches and any residual filter state is absent

#### Scenario: Drill-in clears filter
- **WHEN** the user presses `enter` on an L1 project row while a filter is active
- **THEN** `searchQuery` is cleared before the L2 session list is displayed

#### Scenario: Back-navigate clears filter
- **WHEN** the user presses `esc` at L2 (when `searchMode` is false) to return to L1
- **THEN** `searchQuery` is cleared and the full project list is shown
