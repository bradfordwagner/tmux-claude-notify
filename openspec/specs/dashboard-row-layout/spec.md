# Spec: dashboard-row-layout

## Purpose

Defines the visual structure and column ordering of each notification row in the dashboard TUI. Covers column headers, status icons, path display, pane ID visibility, and ANSI-safe alignment.

## Requirements

### Requirement: Notification rows display column headers
The dashboard notification list SHALL render a header row and a separator line above the entries so users can identify each column without prior knowledge of the layout.

#### Scenario: Headers visible above entries
- **WHEN** one or more notification entries are present
- **THEN** a header row reading "STATUS", "WINDOW", "PATH", "SESSION", "AGE" is rendered above the entries
- **AND** a separator line (e.g. "──────") appears between the header row and the first entry

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
Status text SHALL be padded to a fixed width before lipgloss color codes are applied, so that ANSI escape sequences do not shift subsequent column positions.

#### Scenario: Columns align across rows with different status values
- **WHEN** entries with different statuses are displayed
- **THEN** the WINDOW, PATH, SESSION, and AGE columns start at the same horizontal position on every row

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
