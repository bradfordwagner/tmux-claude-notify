## MODIFIED Requirements

### Requirement: Record schema includes agent status field
The JSONL record struct SHALL include a `status` field (string) representing the current agent state: `running`, `waiting`, `idle`, or `stale`. Existing records without a `status` field SHALL be read as `waiting` for backwards compatibility.

#### Scenario: New record written with status
- **WHEN** the transcript watcher or notify subcommand appends a record
- **THEN** the record includes a `status` field with the current derived state

#### Scenario: Legacy record without status field — treated as waiting
- **WHEN** a JSONL record is read that has no `status` field
- **THEN** the parsed record's status defaults to `waiting`

#### Scenario: Running record does not set window highlight
- **WHEN** a record with `status: running` is appended
- **THEN** the window tab is NOT highlighted (running is informational only)

#### Scenario: Waiting record sets window highlight
- **WHEN** a record with `status: waiting` is appended
- **THEN** `window-status-style` and `window-status-current-style` are set to `fg=#AD8EE6,bold`

## ADDED Requirements

### Requirement: Store exposes UpdateStatus for in-place state transitions
The store SHALL provide `UpdateStatus(paneID string, status string) error` to update the status of the most recent uncleared record for a pane without clearing it.

#### Scenario: Status updated from running to waiting
- **WHEN** `UpdateStatus("%3", "waiting")` is called
- **THEN** the most recent uncleared record for pane `%3` has its `status` field updated to `waiting`
- **AND** the JSONL file is rewritten atomically

#### Scenario: No uncleared record — no-op
- **WHEN** `UpdateStatus` is called for a pane with no uncleared record
- **THEN** the function returns nil without modifying the file
