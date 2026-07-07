## ADDED Requirements

### Requirement: Tab key toggles between Notifications and Sessions views
The dashboard TUI SHALL have two views: Notifications (default) and Sessions. Pressing `Tab` SHALL cycle between them. The active view SHALL be indicated in a header bar at the top of the dashboard.

#### Scenario: Default view is Notifications
- **WHEN** the dashboard opens
- **THEN** the Notifications view is shown
- **AND** the header indicates "Notifications" as the active tab

#### Scenario: Tab switches to Sessions view
- **WHEN** the user presses `Tab` while Notifications view is active
- **THEN** the Sessions view is shown
- **AND** the header indicates "Sessions" as the active tab

#### Scenario: Tab switches back to Notifications view
- **WHEN** the user presses `Tab` while Sessions view is active
- **THEN** the Notifications view is shown

### Requirement: Sessions view is a two-level drill-in table
The Sessions view SHALL display a Level-1 projects table. Each row represents one project (or the special "📌 Pinned" group). Pressing `enter` on a project row drills into Level-2, which shows individual session rows for that project. Pressing `esc` in Level-2 returns to Level-1. Projects are sorted by their most recently active session (default) or by status; sessions within a project are sorted by the same field.

#### Scenario: Level-1 shows one row per project
- **WHEN** the Sessions view is activated
- **THEN** a table is rendered with columns STATUS, PROJECT, COUNT, LAST USED — one row per distinct project

#### Scenario: Level-1 projects sorted by most recent
- **WHEN** the default sort (age) is active
- **THEN** the project with the most recently active session appears first

#### Scenario: "📌 Pinned" project row appears first when pinned sessions exist
- **WHEN** any sessions have `pinned: true`
- **THEN** a "📌 Pinned" row is the first row in Level-1, showing the count and most recent activity of all pinned sessions

#### Scenario: Level-1 STATUS column shows most urgent session status for the project
- **WHEN** a project has one session waiting and one idle
- **THEN** the project row STATUS shows `⏳ waiting`

#### Scenario: `enter` on a Level-1 project row drills in
- **WHEN** the cursor is on a project row and the user presses `enter`
- **THEN** the view switches to Level-2 showing sessions for that project, with cursor reset to 0

#### Scenario: Level-2 shows session rows with STATUS, PIN, SESSION, AGE columns
- **WHEN** Level-2 is active for a project
- **THEN** a header displays the project path with a ← indicator, followed by a column header row and session rows

#### Scenario: Active session row shows live status
- **WHEN** a record has a non-empty `pane_id` and its transcript-derived status is `running`
- **THEN** the row shows the `⚙  running` status icon

#### Scenario: Closed session row shows idle status
- **WHEN** a record has an empty `pane_id`
- **THEN** the row shows `💤 idle` status

#### Scenario: `esc` in Level-2 returns to Level-1
- **WHEN** the user presses `esc` while in Level-2
- **THEN** the view returns to Level-1 with cursor reset to 0

#### Scenario: `esc` in Level-1 quits the dashboard
- **WHEN** the user presses `esc` while in Level-1 (not drilled in)
- **THEN** the dashboard closes (same as `q`)

#### Scenario: Sessions view empty state
- **WHEN** no session records exist
- **THEN** an empty state message "No sessions discovered" is shown

### Requirement: `p` key toggles pin on selected session in Level-2
In Level-2, pressing `p` SHALL toggle the `pinned` flag on the currently selected session record.

#### Scenario: Pin unpinned session
- **WHEN** the cursor is on an unpinned session and the user presses `p`
- **THEN** the session's `pinned` flag is set to `true`
- **AND** the PIN column shows 📌

#### Scenario: Unpin pinned session
- **WHEN** the cursor is on a pinned session and the user presses `p`
- **THEN** the session's `pinned` flag is set to `false`
- **AND** the PIN column shows blank

#### Scenario: `p` has no effect in Level-1
- **WHEN** the user presses `p` while in Level-1
- **THEN** nothing happens

### Requirement: `f` key toggles active-pane filter in Sessions view
In the Sessions view, pressing `f` SHALL toggle a filter that limits displayed sessions to those with a non-empty `pane_id`. The filter applies to both Level-1 (hides projects with no active sessions, except "📌 Pinned") and Level-2 (hides inactive sessions). Pinned sessions always appear regardless of filter state.

#### Scenario: Default filter is off — all sessions shown
- **WHEN** the Sessions view is first activated
- **THEN** all sessions.jsonl records are shown and the header shows no filter indicator

#### Scenario: `f` activates active-pane filter
- **WHEN** the user presses `f` with filter off
- **THEN** only projects with active sessions (and "📌 Pinned") appear in Level-1, and the header shows "filter: active panes"

#### Scenario: Filter active with no matching sessions
- **WHEN** the active-pane filter is on and no sessions have a non-empty `pane_id` (and no pinned sessions exist)
- **THEN** the empty state message "No active sessions" is shown

### Requirement: Pinned sessions sort to top in Sessions view
Pinned sessions SHALL always appear first in the Level-1 table ("📌 Pinned" row is always row 0) and first within a project's Level-2 session list.

#### Scenario: Pinned sessions float above unpinned
- **WHEN** the Sessions view is sorted by last activity
- **AND** a session pinned two weeks ago exists alongside an unpinned session from five minutes ago
- **THEN** the pinned session appears first in Level-2

### Requirement: Pinned sessions appear in Notifications view regardless of pane state
Sessions with `pinned: true` SHALL appear as rows in the Notifications view even when their `pane_id` is empty (not currently in an active tmux pane). They SHALL be rendered with a 📌 prefix in the STATUS column.

#### Scenario: Pinned idle session in Notifications view
- **WHEN** a sessions.jsonl record has `pinned: true` and `pane_id: ""`
- **THEN** it appears as a row in the Notifications view
- **AND** the STATUS column shows 📌 with `idle`

#### Scenario: Unpinned idle session not in Notifications view
- **WHEN** a sessions.jsonl record has `pinned: false` and `pane_id: ""`
- **AND** there is no matching uncleared notifications.jsonl entry
- **THEN** it does NOT appear in the Notifications view
