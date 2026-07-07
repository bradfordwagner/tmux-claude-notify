## 1. Dependency

- [x] 1.1 Add `github.com/sahilm/fuzzy` to go.mod and go.sum via `go get`

## 2. Model State

- [x] 2.1 Add `searchMode bool`, `searchQuery string`, and `searchInput textinput.Model` fields to the bubbletea model struct in `internal/ui/model.go`
- [x] 2.2 Initialize `searchInput` (placeholder, character limit) in the model constructor

## 3. Key Handling

- [x] 3.1 Wire `/` key in `Update` to set `searchMode = true` and focus `searchInput` (both Notifications and Sessions views, when not already in search mode)
- [x] 3.2 Route printable keystrokes to `searchInput.Update` when `searchMode` is true, updating `searchQuery` from `searchInput.Value()`
- [x] 3.3 Extend `esc` handler: when `searchMode` is true, clear query and exit search mode without quitting or back-navigating
- [x] 3.4 Extend `Tab` handler: clear `searchMode` and `searchQuery` before switching views
- [x] 3.5 Clear `searchMode` and `searchQuery` on drill-in (enter at Sessions L1 row)
- [x] 3.6 Clear `searchMode` and `searchQuery` on drill-out (esc at Sessions L2 when `searchMode` is false)

## 4. Fuzzy Filtering

- [x] 4.1 Implement filtered entry slice for Notifications view: match target = window name + path + session; apply `sahilm/fuzzy` when `searchQuery` is non-empty; preserve existing sort order within matches
- [x] 4.2 Implement filtered slice for Sessions L1: match target = project path; apply fuzzy filter when `searchQuery` is non-empty
- [x] 4.3 Implement filtered slice for Sessions L2 (drill): match target = session ID + window name; apply fuzzy filter when `searchQuery` is non-empty
- [x] 4.4 Clamp cursor to `max(0, len(filteredList)-1)` after every query update

## 5. Rendering

- [x] 5.1 Render search input line below tab header when `searchMode` is true: `/ ` prefix in accent color followed by `searchInput.View()`
- [x] 5.2 Render `dimStyle("No results for \"<query>\"")` in the row area when the filtered list is empty and `searchMode` is true
