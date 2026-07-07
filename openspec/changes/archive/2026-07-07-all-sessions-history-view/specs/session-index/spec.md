## ADDED Requirements

### Requirement: SessionRecord schema persists per-session metadata
The `sessions.jsonl` file at `~/.local/share/tmux-claude-notify/sessions.jsonl` SHALL store one record per Claude session UUID. Each record SHALL include: `session_id` (UUID string), `encoded_path` (raw directory name under `~/.claude/projects/`), `project_path` (real filesystem path; empty string if not yet recovered), `pinned` (bool), `last_activity` (UnixNano int64), `status` (string: `running`, `waiting`, `idle`, `stale`), `pane_id` (string; empty when pane not active), `window_id` (string; empty when pane not active), `window_name` (string; empty when pane not active), `tmux_session` (string; empty when pane not active).

#### Scenario: Full record written for active pane session
- **WHEN** the transcript watcher discovers a live tmux pane running `claude*`
- **THEN** a record is upserted in sessions.jsonl with `session_id`, `encoded_path`, `project_path` set to the real `pane_current_path`, `pane_id`, `window_id`, `window_name`, `tmux_session`, and the derived status

#### Scenario: Partial record written for closed session
- **WHEN** the filesystem scan discovers a session UUID with no matching live pane
- **THEN** a record is upserted with `session_id`, `encoded_path`, `last_activity` from file mtime, `pane_id` empty, and `project_path` set to recovered path or empty string

### Requirement: Filesystem scan discovers all sessions across projects
The sessions package SHALL scan `~/.claude/projects/*/` on demand and enumerate all `*.jsonl` files. For each file, it SHALL extract the session UUID from the filename and the encoded project path from the parent directory name.

#### Scenario: All JSONL files under projects dir are discovered
- **WHEN** `DiscoverAll()` is called
- **THEN** every `*.jsonl` file under `~/.claude/projects/` is enumerated
- **AND** a SessionRecord is returned for each unique session UUID

#### Scenario: Non-JSONL files are ignored
- **WHEN** files other than `*.jsonl` exist under a project directory
- **THEN** they are not included in the discovery result

#### Scenario: Projects directory does not exist
- **WHEN** `~/.claude/projects/` does not exist
- **THEN** `DiscoverAll()` returns an empty slice without error

### Requirement: Path recovery resolves real filesystem path from encoded directory name
The sessions package SHALL attempt to recover the real project path from the encoded directory name using a filesystem walk. Recovery SHALL use three strategies in priority order: (1) stored `project_path` from a previous active-pane discovery, (2) BFS filesystem walk replacing leading `-` with `/` and testing each `-`-separated segment as either a `/` separator or literal character, (3) display fallback replacing leading `-` with `/` with no filesystem verification.

#### Scenario: Stored path used directly
- **WHEN** a sessions.jsonl record already has a non-empty `project_path`
- **THEN** path recovery returns that path without any filesystem check

#### Scenario: Filesystem walk resolves ambiguous path
- **WHEN** `project_path` is empty and the encoded path is `-home-bw-foo-bar`
- **AND** `/home/bw/foo/bar` exists on the filesystem but `/home/bw/foo-bar` does not
- **THEN** path recovery returns `/home/bw/foo/bar`

#### Scenario: Filesystem walk falls back to display path
- **WHEN** `project_path` is empty and no filesystem path matching the encoding can be verified
- **THEN** path recovery returns the encoded path with leading `-` replaced by `/` and no filesystem check

### Requirement: Upsert creates or updates by session ID
The sessions package SHALL expose `Upsert(record SessionRecord) error` that finds an existing record with matching `session_id` and updates it, or appends a new record if none exists. Updates SHALL use atomic file rewrite (write to `.tmp`, then rename).

#### Scenario: New session upserted
- **WHEN** `Upsert` is called with a session_id not present in sessions.jsonl
- **THEN** a new record is appended and the file is rewritten atomically

#### Scenario: Existing session updated
- **WHEN** `Upsert` is called with a session_id already in sessions.jsonl
- **THEN** the existing record is replaced with the new data and the file is rewritten atomically

### Requirement: SetPinned toggles the pinned flag by session ID
The sessions package SHALL expose `SetPinned(sessionID string, pinned bool) error` that updates the `pinned` field of the matching record.

#### Scenario: Pin a session
- **WHEN** `SetPinned("abc-uuid", true)` is called
- **THEN** the matching record's `pinned` field is set to `true` and the file is rewritten atomically

#### Scenario: Unpin a session
- **WHEN** `SetPinned("abc-uuid", false)` is called
- **THEN** the matching record's `pinned` field is set to `false` and the file is rewritten atomically

#### Scenario: Session ID not found
- **WHEN** `SetPinned` is called with a session_id not present in sessions.jsonl
- **THEN** the function returns an error

### Requirement: ReadAll returns all records sorted by last_activity descending
The sessions package SHALL expose `ReadAll() ([]SessionRecord, error)` that reads all records from sessions.jsonl and returns them sorted by `last_activity` descending (most recent first), with pinned records floated to the top.

#### Scenario: Records returned newest first
- **WHEN** `ReadAll()` is called with records having different `last_activity` values
- **THEN** the record with the greatest `last_activity` is first in the result

#### Scenario: Pinned records float to top
- **WHEN** `ReadAll()` is called and some records have `pinned: true`
- **THEN** all pinned records appear before any unpinned records
- **AND** within each group the newest-first ordering is maintained

### Requirement: Compact removes stale unpinned inactive records
The sessions package SHALL expose `Compact(maxAge time.Duration) error` that removes unpinned records whose `last_activity` is older than `maxAge` AND whose `pane_id` is empty (not currently active).

#### Scenario: Old unpinned inactive record removed
- **WHEN** `Compact(90 * 24 * time.Hour)` is called
- **AND** a record has `pinned: false`, `pane_id: ""`, and `last_activity` older than 90 days
- **THEN** that record is removed from sessions.jsonl

#### Scenario: Old pinned record retained
- **WHEN** `Compact` is called
- **AND** a record has `pinned: true` regardless of age
- **THEN** that record is NOT removed

#### Scenario: Old active record retained
- **WHEN** `Compact` is called
- **AND** a record has a non-empty `pane_id` (currently active)
- **THEN** that record is NOT removed
