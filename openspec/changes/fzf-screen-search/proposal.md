## Why

With many sessions and notifications visible, finding the right entry requires scanning rows manually. A `/`-triggered fuzzy filter lets users narrow any view instantly, matching the muscle memory of vim/fzf navigation already present throughout the toolchain.

## What Changes

- Pressing `/` in either view enters search mode: a filter input appears below the tab header and keystrokes update the query, narrowing the visible rows in real time using fuzzy matching.
- Search mode has two focus sub-states toggled by `Tab`:
  - **Input focused** (default on entry): alphanumeric keys go to the filter input.
  - **Table focused**: all normal navigation and action keys (`j/k/p/w/h/v/enter/q/s/f`) are active on the filtered list. The filter remains applied.
- `esc` exits search mode entirely from either sub-state, restoring the full list.
- `Tab` while in search mode toggles between input-focused and table-focused; it does **not** switch views or clear the filter.
- `Tab` while not in search mode switches views as normal.
- The filter query is shown inline below the tab header (e.g. `/ query_`). The textinput cursor blinks when input-focused and is hidden when table-focused.
- When the filtered list is empty, an appropriate "no results" message is shown.
- Search is cleared when the user drills in (enter at Sessions L1) or back-navigates out of a project (esc at L2 when not in search mode). View switches via Tab also start fresh with no active filter.

## Capabilities

### New Capabilities

- `dashboard-search`: `/` hotkey entry into search mode; two focus sub-states (input/table) toggled by Tab; real-time fuzzy filter applied to the active view's row list; search input rendering; cleared on drill navigation.

### Modified Capabilities

- `always-on-dashboard`: `/` key wired in both views; `esc` exits search mode before quitting/navigating; `Tab` toggles input/table focus while searching and switches views when not searching.
- `dashboard-row-layout`: Search input line rendered below tab header when search mode is active; "no results" message when filter matches nothing.

## Impact

- `internal/ui/model.go`: new `searchMode bool`, `searchFocus bool`, `searchQuery string` fields; `textinput.Model` from `github.com/charmbracelet/bubbles/textinput` (already in dep tree); fuzzy matching via `github.com/sahilm/fuzzy` (new dep); filtered entry/session slices derived before render.
- `go.mod` / `go.sum`: add `github.com/sahilm/fuzzy`.
- No changes to store, watcher, tmux helpers, or hook wiring.
