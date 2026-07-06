## ADDED Requirements

### Requirement: Store exposes UnclearedForWindow for window-scoped notification check
The store SHALL provide `UnclearedForWindow(windowID string) ([]Record, error)` that returns all uncleared records whose `window` field matches the given window ID. This is used by `runClear` to determine whether window-level tmux styles should be torn down after clearing a single pane's entry.

#### Scenario: Returns remaining notifications for window
- **WHEN** `UnclearedForWindow("%W3")` is called
- **AND** pane `%1` in window `%W3` has an uncleared entry and pane `%2` in `%W3` also has an uncleared entry
- **AND** pane `%1`'s entry was just cleared via `ClearPane`
- **THEN** `UnclearedForWindow` returns one record (pane `%2`'s entry)

#### Scenario: Returns empty slice when all panes in window are cleared
- **WHEN** all panes in a window have had their entries cleared
- **THEN** `UnclearedForWindow` returns an empty slice
- **AND** the caller MAY proceed to clear window-level tmux styles

#### Scenario: Returns empty slice when window has no entries
- **WHEN** `UnclearedForWindow` is called for a window with no entries in the JSONL
- **THEN** it returns an empty slice without error
