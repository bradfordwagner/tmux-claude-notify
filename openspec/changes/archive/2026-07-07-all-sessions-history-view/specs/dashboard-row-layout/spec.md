## ADDED Requirements

### Requirement: Tab-toggle header displays active view name
The dashboard SHALL render a tab header at the top showing both view names ("Notifications" and "Sessions"), with the active view highlighted and the inactive view dimmed.

#### Scenario: Notifications tab highlighted when in Notifications view
- **WHEN** the active view is Notifications
- **THEN** the header renders "[ Notifications ]  Sessions" with the active tab in the accent color

#### Scenario: Sessions tab highlighted when in Sessions view
- **WHEN** the active view is Sessions
- **THEN** the header renders "  Notifications  [ Sessions ]" with the active tab in the accent color

### Requirement: Sessions view Level-1 rows display project-level columns
Level-1 rows in the Sessions view SHALL display: STATUS (most urgent status among the project's sessions), PROJECT (trimmed path, truncated to 30 characters), COUNT (number of sessions), and LAST USED (age of most recent session activity). These columns apply only to Level-1 project rows.

#### Scenario: Level-1 column header row
- **WHEN** the Sessions view is active in Level-1 and records exist
- **THEN** a column header row reading "STATUS", "PROJECT", "COUNT", "LAST USED" is rendered with a separator line

#### Scenario: Sort field shown in Sessions view tab header
- **WHEN** the Sessions view is active
- **THEN** the currently active sort field is shown in the tab header (e.g., "sort: age", "sort: status")

#### Scenario: Filter indicator shown in tab header when active
- **WHEN** the active-pane filter is on
- **THEN** the tab header shows "filter: active panes" alongside the sort indicator

#### Scenario: "esc: back" shown in tab header when drilled in
- **WHEN** the Sessions view is in Level-2 (drilled into a project)
- **THEN** the tab header shows "esc: back" as a hint

### Requirement: Sessions view Level-2 rows display session-specific columns
Level-2 rows in the Sessions view (after drilling into a project) SHALL display: STATUS icon, PIN indicator (📌 if pinned, blank space if not), SESSION (first 8 characters of the UUID), and AGE (time since `last_activity`). A breadcrumb header showing `← <project>` is rendered above the column header row.

#### Scenario: Level-2 column header row
- **WHEN** Level-2 is active
- **THEN** a breadcrumb `← <project>` followed by column header "STATUS", "PIN", "SESSION", "AGE" is rendered

#### Scenario: Pinned session row shows pin indicator
- **WHEN** a sessions.jsonl record has `pinned: true`
- **THEN** the PIN column shows 📌

#### Scenario: Unpinned session row shows blank pin column
- **WHEN** a sessions.jsonl record has `pinned: false`
- **THEN** the PIN column shows a blank space of consistent width

### Requirement: Pinned indicator column in Notifications view rows
Notification rows in the Notifications view SHALL include a PIN indicator column showing 📌 if the entry is backed by a pinned sessions.jsonl record, and a blank space otherwise. This column SHALL be positioned between STATUS and WINDOW.

#### Scenario: Pinned session entry shows pin indicator in Notifications view
- **WHEN** a Notifications view entry is derived from a pinned sessions.jsonl record
- **THEN** the PIN column shows 📌

#### Scenario: Stop-hook-only entry shows blank pin column
- **WHEN** a Notifications view entry comes only from notifications.jsonl with no matching sessions.jsonl record
- **THEN** the PIN column shows a blank space of consistent width

## MODIFIED Requirements

### Requirement: Notification rows display column headers
The dashboard notification list SHALL render a header row and a separator line above the entries so users can identify each column without prior knowledge of the layout.

#### Scenario: Headers visible above entries
- **WHEN** one or more notification entries are present
- **THEN** a header row reading "STATUS", "PIN", "WINDOW", "PATH", "SESSION", "AGE" is rendered above the entries
- **AND** a separator line appears between the header row and the first entry
- **AND** PATH values longer than 22 visual columns are truncated with a `…` prefix

#### Scenario: No headers when list is empty
- **WHEN** there are no pending notifications
- **THEN** the header row and separator are not rendered (empty state message is shown instead)
