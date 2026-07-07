# Spec: notification-log

## Purpose

The notification log is the JSONL file at `~/.local/share/tmux-claude-notify/notifications.jsonl` that persists notification state across dashboard sessions. This capability covers the record schema, read/write operations, and the store API exposed to the dashboard and transcript watcher.

## Requirements

### Requirement: Notifications view merges active sessions from session-index
The dashboard Notifications view SHALL load two record sets: (a) all uncleared records from `notifications.jsonl` (existing behavior), and (b) `sessions.jsonl` records whose `pane_id` is non-empty and whose `session_id` has no corresponding uncleared `notifications.jsonl` entry. The merged result is the full Notifications view entry list.

#### Scenario: Active pane with no notification entry appears in Notifications view
- **WHEN** a sessions.jsonl record has `pane_id: "%7"` and no uncleared notifications.jsonl entry exists for pane `%7`
- **THEN** a row for that session appears in the Notifications view

#### Scenario: Notification entry takes priority over sessions entry for same pane
- **WHEN** both a sessions.jsonl record and an uncleared notifications.jsonl entry exist for pane `%7`
- **THEN** only the notifications.jsonl entry is shown (sessions entry is suppressed)

#### Scenario: Closed session without notification not shown
- **WHEN** a sessions.jsonl record has `pane_id: ""` and `pinned: false`
- **AND** there is no uncleared notifications.jsonl entry for that session
- **THEN** no row for that session appears in the Notifications view

### Requirement: Pinned sessions appear at the top of the Notifications view
Sessions with `pinned: true` in sessions.jsonl SHALL appear at the top of the Notifications view entry list, above all uncleared notifications.jsonl entries and above all unpinned active sessions. Pinned entries SHALL be rendered identically to active session entries but with the pinned indicator, and their STATUS SHALL reflect the last known status from the session record. A pinned session SHALL appear in the Notifications view regardless of pane state.

#### Scenario: Pinned closed session in Notifications view
- **WHEN** a sessions.jsonl record has `pinned: true` and `pane_id: ""`
- **THEN** a row for that session appears in the Notifications view regardless of notification-log state

#### Scenario: Pinned active session not duplicated
- **WHEN** a sessions.jsonl record has `pinned: true` and `pane_id: "%9"`
- **AND** an uncleared notifications.jsonl entry also exists for pane `%9`
- **THEN** only one row appears (the notifications.jsonl entry, with pin indicator added)

### Requirement: Record schema includes agent status field
The JSONL record struct SHALL include a `status` field (string) representing the current agent state: `running`, `waiting`, `idle`, or `stale`. Existing records without a `status` field SHALL be read as `waiting` for backwards compatibility.

#### Scenario: New record written with status
- **WHEN** the transcript watcher or notify subcommand appends a record
- **THEN** the record includes a `status` field with the current derived state

#### Scenario: Legacy record without status field — treated as waiting
- **WHEN** a JSONL record is read that has no `status` field
- **THEN** the parsed record's status defaults to `waiting`

#### Scenario: Running status does not set window highlight
- **WHEN** a pane's status is updated to `running` via `UpdateStatus` (set by the transcript watcher; the notify subcommand never appends running records)
- **THEN** the window tab is NOT highlighted (running is informational only)

#### Scenario: Waiting record sets window highlight
- **WHEN** a record with `status: waiting` is appended
- **THEN** `window-status-style` and `window-status-current-style` are set to `fg=#AD8EE6,bold`

### Requirement: Store exposes UpdateStatus for in-place state transitions
The store SHALL provide `UpdateStatus(paneID string, status string) error` to update the status of the most recent uncleared record for a pane without clearing it.

#### Scenario: Status updated from running to waiting
- **WHEN** `UpdateStatus("%3", "waiting")` is called
- **THEN** the most recent uncleared record for pane `%3` has its `status` field updated to `waiting`
- **AND** the JSONL file is rewritten atomically

#### Scenario: No uncleared record — no-op
- **WHEN** `UpdateStatus` is called for a pane with no uncleared record
- **THEN** the function returns nil without modifying the file

### Requirement: Auto-reset clears the JSONL entry after timer fires
When the auto-reset subprocess fires and finds an uncleared entry for the pane, it SHALL call `store.ClearPane(paneID)` to remove the entry. This is identical to the manual dashboard-clear path; no new store API is needed.

#### Scenario: Auto-reset calls ClearPane — entry removed
- **WHEN** the auto-reset subprocess fires after the configured delay
- **AND** an uncleared entry for the pane exists in the JSONL store
- **THEN** `ClearPane` is called and the entry is removed from the file

#### Scenario: Auto-reset finds no entry — no store write
- **WHEN** the auto-reset subprocess fires
- **AND** no uncleared entry exists for the pane (already dismissed manually)
- **THEN** `ClearPane` is NOT called and the JSONL file is not modified

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
