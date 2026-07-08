## MODIFIED Requirements

### Requirement: Dashboard renders agent status per entry in Notifications view
Each Notifications view entry SHALL display STATUS (icon + text), PIN (📌 or blank), POP indicator (● or blank), WINDOW name, PATH (last two components, `~`-abbreviated, dynamic width), SESSION name, and AGE. The raw pane ID SHALL NOT appear. A header row and separator SHALL appear above the entry list when entries are present. Only the entries within the current scroll window (as defined by the tui-viewport capability) SHALL be rendered.

#### Scenario: Waiting entry styled with accent color
- **WHEN** an entry has `status: waiting`
- **THEN** it is rendered with "⏳ waiting" in the `#AD8EE6` accent color

#### Scenario: Running entry styled with warn color
- **WHEN** an entry has `status: running`
- **THEN** it is rendered with "⚙  running" in the warn/orange color

#### Scenario: Stale entry styled with dim color
- **WHEN** an entry has `status: stale`
- **THEN** it is rendered with "💤 stale" in the dim/subtle color

#### Scenario: Column headers rendered above entries
- **WHEN** one or more entries are displayed
- **THEN** a header row "STATUS  PIN  P  WINDOW  PATH  SESSION  AGE" and separator appear above the first entry

#### Scenario: Path column shows pane working directory
- **WHEN** an entry is rendered
- **THEN** the PATH column shows the pane's current directory, truncated to the last two components with `$HOME` replaced by `~`

#### Scenario: Pop indicator shows pane background pop state
- **WHEN** an entry's pane has an active pane-local background pop (`window-style` set via `set-option -p`)
- **THEN** the P column renders `●` in the accent color (`#AD8EE6`)
- **WHEN** the pane has no active pop
- **THEN** the P column renders a blank space

#### Scenario: Pop indicator queried at load time
- **WHEN** the Notifications view loads entries from `notifications.jsonl`
- **THEN** `IsPanePopped(paneID)` is called for each notification-backed entry to set the pop flag
- **AND** the flag is used for display only (not persisted)

#### Scenario: Entries below viewport height are not rendered
- **WHEN** the number of entries exceeds the available viewport height
- **THEN** only the entries within `[scrollOffset, scrollOffset + viewportHeight)` are rendered
- **AND** the user can reach off-screen entries by moving the cursor past the visible boundary

### Requirement: Sessions view is a two-level drill-in table
The Sessions view SHALL display a Level-1 projects table (one row per project, plus "📌 Pinned" group). `enter` on a project row drills into Level-2 (session rows for that project). Navigation back and quit behavior is governed by the `esc key closes the dashboard or navigates back` requirement. Only the rows within the current scroll window (as defined by the tui-viewport capability) SHALL be rendered at each level.

#### Scenario: Level-1 shows one row per project
- **WHEN** the Sessions view is activated
- **THEN** a table is rendered with columns STATUS, PROJECT, COUNT, LAST USED

#### Scenario: enter on Level-1 row drills in
- **WHEN** the user presses `enter` on a project row
- **THEN** Level-2 shows session rows for that project

#### Scenario: Level-1 rows beyond viewport height are not rendered
- **WHEN** the number of project rows exceeds `viewportHeight`
- **THEN** only rows within the scroll window are rendered and the cursor can scroll to reveal the rest

#### Scenario: Level-2 rows beyond viewport height are not rendered
- **WHEN** the number of session rows for a project exceeds `viewportHeight`
- **THEN** only rows within the scroll window are rendered and the cursor can scroll to reveal the rest
