# Spec: dashboard-row-layout

## Purpose

Defines the visual structure and column ordering of rows in the dashboard TUI — both the Notifications view and the two-level Sessions view. Covers column headers, status icons, path display, pane ID visibility, and ANSI-safe alignment.

## Requirements

### Requirement: Tab-toggle header displays active view name
The dashboard SHALL render a tab header at the top showing both view names ("Notifications" and "Sessions"), with the active view highlighted and the inactive view dimmed.

#### Scenario: Notifications tab highlighted when in Notifications view
- **WHEN** the active view is Notifications
- **THEN** the header renders "[ Notifications ]  Sessions" with the active tab in the accent color

#### Scenario: Sessions tab highlighted when in Sessions view
- **WHEN** the active view is Sessions
- **THEN** the header renders "  Notifications  [ Sessions ]" with the active tab in the accent color

### Requirement: Notification rows display column headers
The dashboard notification list SHALL render a header row and a separator line above the entries so users can identify each column without prior knowledge of the layout.

#### Scenario: Headers visible above entries
- **WHEN** one or more notification entries are present
- **THEN** a header row reading "STATUS", "PIN", "WINDOW", "PATH", "SESSION", "AGE" is rendered above the entries
- **AND** a separator line (e.g. "──────") appears between the header row and the first entry
- **AND** PATH values longer than 22 visual columns are truncated with a `…` prefix

#### Scenario: No headers when list is empty
- **WHEN** there are no pending notifications
- **THEN** the header row and separator are not rendered (empty state message is shown instead)

### Requirement: Notification rows lead with status icon and status text
Each notification row SHALL begin with a status icon followed by the status text, making the agent state immediately visible at the start of each line.

#### Scenario: Waiting entry shows hourglass icon
- **WHEN** an entry has `status: waiting`
- **THEN** the row begins with "⏳ waiting" in the accent color

#### Scenario: Running entry shows gear icon
- **WHEN** an entry has `status: running`
- **THEN** the row begins with "⚙  running" in the warn/orange color

#### Scenario: Stale entry shows sleep icon
- **WHEN** an entry has `status: stale`
- **THEN** the row begins with "💤 stale" in the dim/subtle color

#### Scenario: Idle entry shows sleep icon
- **WHEN** an entry has `status: idle` (closed session, pane gone)
- **THEN** the row begins with "💤 idle" in the dim/subtle color

### Requirement: Notification rows include a PIN indicator column
Each notification row SHALL include a PIN column (between STATUS and WINDOW) showing 📌 if the entry is backed by a pinned `sessions.jsonl` record, and three blank spaces otherwise.

#### Scenario: Pinned session entry shows pin indicator
- **WHEN** a notification entry is derived from a pinned sessions.jsonl record
- **THEN** the PIN column shows 📌

#### Scenario: Stop-hook-only entry shows blank pin column
- **WHEN** a notification entry comes only from notifications.jsonl with no matching pinned sessions record
- **THEN** the PIN column shows blank space of consistent width

### Requirement: Notification rows display pane current path
Each notification row SHALL include a `PATH` column showing the current working directory of the tmux pane, truncated to the last two path components with `$HOME` replaced by `~`.

#### Scenario: Path shows last two components
- **WHEN** the pane's current path is `/home/user/workspace/myproject`
- **THEN** the PATH column displays `workspace/myproject`

#### Scenario: Path abbreviates home directory
- **WHEN** the pane's current path starts with the user's home directory
- **THEN** the leading home path is replaced with `~` before truncation

#### Scenario: Path unavailable
- **WHEN** `tmux display-message` fails to return a path for the pane
- **THEN** the PATH column displays an empty string (no error surfaced to user)

### Requirement: Notification rows omit raw pane ID
Notification rows SHALL NOT display the raw tmux pane ID (e.g. `%34`) in the visible table. Pane identification is an internal concern.

#### Scenario: Pane ID not in rendered row
- **WHEN** an entry is rendered
- **THEN** no column containing the raw `%NN` pane ID string is present in the visible output

### Requirement: Status column alignment is correct despite ANSI codes
Status text SHALL be padded to a fixed visual width (12 columns) using `runewidth.StringWidth` before lipgloss color codes are applied, so that ANSI escape sequences do not shift subsequent column positions.

#### Scenario: Columns align across rows with different status values
- **WHEN** entries with different statuses are displayed
- **THEN** the PIN, WINDOW, PATH, SESSION, and AGE columns start at the same horizontal position on every row

### Requirement: Dashboard shows all uncleared notifications regardless of pane liveness
The dashboard SHALL display all uncleared JSONL entries without filtering by whether the source pane currently exists in the tmux server. An entry is visible as long as it is uncleared, enabling the user to dismiss notifications from sessions that were restarted or panes that were closed since the Stop hook fired.

#### Scenario: Gone-pane entry visible in dashboard
- **WHEN** an uncleared JSONL entry exists for a pane that no longer appears in `tmux list-panes -a`
- **THEN** the entry is still rendered in the dashboard notification list

#### Scenario: Gone-pane entry can be cleared from dashboard
- **WHEN** the user selects a gone-pane entry and confirms dismissal
- **THEN** the JSONL entry is marked cleared
- **AND** any window-style teardown proceeds normally (tmux errors are non-fatal if window also gone)

### Requirement: Gone pane indicator in PATH column
When a notification entry's pane no longer exists in the current tmux server, the PATH column SHALL display `(gone)` instead of the pane's current working directory, making it clear that the pane is no longer live.

#### Scenario: Path column shows gone for missing pane
- **WHEN** `tmux display-message -t <paneID>` fails because the pane no longer exists
- **THEN** the PATH column displays `(gone)`

#### Scenario: Path column shows real path for live pane
- **WHEN** the pane still exists and `tmux display-message` succeeds
- **THEN** the PATH column displays the truncated real path as before

### Requirement: Sessions view Level-1 displays project-level rows
The Sessions view Level-1 SHALL render one row per distinct project (plus a "📌 Pinned" group if any sessions are pinned). Columns: STATUS (most urgent status among the project's sessions), PROJECT (trimmed path, truncated to 30 visual columns), COUNT (number of sessions), LAST USED (age of most recently active session).

#### Scenario: Level-1 column header row
- **WHEN** the Sessions view is active in Level-1 and records exist
- **THEN** a column header row reading "STATUS", "PROJECT", "COUNT", "LAST USED" is rendered with a separator line

#### Scenario: "📌 Pinned" group row appears first
- **WHEN** any sessions have `pinned: true`
- **THEN** a "📌 Pinned" row is the first row, showing count and most recent activity of all pinned sessions

#### Scenario: Level-1 STATUS shows most urgent status for the project
- **WHEN** a project has one session waiting and one idle
- **THEN** the project row STATUS shows `⏳ waiting`

#### Scenario: Sort indicator shown in Sessions tab header
- **WHEN** the Sessions view is active
- **THEN** the currently active sort field is shown in the tab header (e.g. "sort: age", "sort: status")

#### Scenario: Filter indicator shown in tab header when active
- **WHEN** the active-pane filter is on
- **THEN** the tab header shows "filter: active panes" alongside the sort indicator

### Requirement: Sessions view Level-2 displays session-specific rows
Level-2 (drilled into a project) SHALL display a breadcrumb header `← <project>` followed by a column header and session rows with columns: STATUS icon, PIN (📌 or blank), SESSION (first 8 chars of UUID), AGE (time since `last_activity`).

#### Scenario: Level-2 column header row
- **WHEN** Level-2 is active for a project
- **THEN** a breadcrumb `← <project>` followed by column header "STATUS", "PIN", "SESSION", "AGE" is rendered

#### Scenario: "esc: back" shown in tab header when drilled in
- **WHEN** the Sessions view is in Level-2
- **THEN** the tab header shows "esc: back" as a hint

#### Scenario: Pinned session row shows pin indicator in Level-2
- **WHEN** a sessions.jsonl record has `pinned: true`
- **THEN** the PIN column shows 📌

#### Scenario: Unpinned session row shows blank pin column
- **WHEN** a sessions.jsonl record has `pinned: false`
- **THEN** the PIN column shows a blank space of consistent width
