## MODIFIED Requirements

### Requirement: Dashboard renders agent status per entry
Each dashboard entry SHALL display the agent status with an icon and color as the first column, followed by window name, pane current path, session name, and age. The raw pane ID SHALL NOT appear in the rendered row. A header row and separator SHALL appear above the entry list when entries are present.

#### Scenario: Waiting entry styled with accent color
- **WHEN** an entry has `status: waiting`
- **THEN** it is rendered with "⏳ waiting" in the `#AD8EE6` accent color as the first column

#### Scenario: Running entry styled with warn color
- **WHEN** an entry has `status: running`
- **THEN** it is rendered with "⚙  running" in the warn/orange color as the first column

#### Scenario: Stale entry styled with dim color
- **WHEN** an entry has `status: stale`
- **THEN** it is rendered with "💤 stale" in the dim/subtle color as the first column

#### Scenario: Column headers rendered above entries
- **WHEN** one or more entries are displayed
- **THEN** a header row ("STATUS WINDOW PATH SESSION AGE") and separator line appear above the first entry

#### Scenario: Path column shows pane working directory
- **WHEN** an entry is rendered
- **THEN** the PATH column shows the pane's current directory, truncated to the last two components with `$HOME` replaced by `~`
