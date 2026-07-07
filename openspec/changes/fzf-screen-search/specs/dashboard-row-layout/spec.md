## ADDED Requirements

### Requirement: Search input line rendered below tab header when active
When `searchMode` is `true`, the dashboard SHALL render a search input line immediately below the tab header, before any setup status or toast messages. The line SHALL show the `textinput` component prefixed with `/ ` in the accent color.

#### Scenario: Search input visible when active
- **WHEN** `searchMode` is `true`
- **THEN** a line reading `/ <query>_` (with blinking cursor) appears below the tab header

#### Scenario: Search input hidden when inactive
- **WHEN** `searchMode` is `false`
- **THEN** no search line is rendered and the layout is identical to the current design

#### Scenario: "No results" message shown when filter matches nothing
- **WHEN** `searchMode` is `true` and the filtered row list is empty
- **THEN** the row area renders `dimStyle("No results for "<query>"")` in place of the normal table
