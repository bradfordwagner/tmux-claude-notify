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
- **THEN** `window-status-style` and `window-status-current-style` are set to `fg=<@claude-notify-highlight-color>,bold` (default `#a6e3a1`)

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

### Requirement: Store read-modify-write operations are race-free under concurrent writers
`ClearPane` and `UpdateStatus` SHALL serialize their read-modify-write cycle against every other process performing a read-modify-write cycle on the same JSONL file, using a file lock held for the duration of the read, mutation, and atomic rename. A clear or status update committed by one process SHALL NOT be silently reverted by another process's concurrent read-modify-write cycle.

#### Scenario: Two concurrent ClearPane calls for different panes both survive
- **WHEN** `ClearPane("%1")` and `ClearPane("%2")` are invoked from two separate processes at approximately the same time, both against a store where `%1` and `%2` have uncleared entries
- **THEN** after both calls return, both `%1`'s and `%2`'s entries are marked cleared
- **AND** neither call's result is lost regardless of the order the two processes' file renames land in

#### Scenario: ClearPane blocks while another writer holds the lock
- **WHEN** `ClearPane` is invoked while another process is mid-way through its own read-modify-write cycle on the same file
- **THEN** the invocation waits for the lock rather than reading a stale snapshot
- **AND** proceeds with its read-modify-write cycle once the lock is released

### Requirement: Store exposes an atomic clear-oldest-uncleared operation
The store SHALL provide a function that finds the uncleared record with the smallest `ts` and marks it cleared as a single operation performed under one lock acquisition, so no other writer can observe or act on the intermediate state between "find oldest" and "mark cleared".

#### Scenario: Atomic clear-oldest returns the cleared record
- **WHEN** the atomic clear-oldest function is called against a store with multiple uncleared entries
- **THEN** it returns the record that had the smallest `ts` among uncleared entries
- **AND** that record is marked cleared in the persisted file before the function returns

#### Scenario: No uncleared entries — atomic clear-oldest returns nil
- **WHEN** the atomic clear-oldest function is called and no uncleared entries exist
- **THEN** it returns a nil record and does not modify the file
