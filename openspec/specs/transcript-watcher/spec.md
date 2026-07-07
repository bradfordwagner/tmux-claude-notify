# Spec: transcript-watcher

## Purpose

The transcript watcher is a component that monitors Claude Code transcript files under `~/.claude/projects/` using fsnotify. It discovers active claude panes via `tmux list-panes`, maps each pane to its transcript file, derives agent state (`running`, `waiting`, `stale`) from JSONL transcript events, and emits `StateChange` events to the dashboard. It also keeps `sessions.jsonl` up to date as panes appear and disappear.

## Requirements

### Requirement: Watcher discovers Claude Code transcript files via live panes
The transcript watcher SHALL discover transcript files by calling `tmux list-panes -a` and filtering for panes whose `pane_current_command` matches the prefix `claude*`. For each matching pane, it forward-encodes `pane_current_path` (replacing `/` and `.` with `-`) to locate the project directory under `~/.claude/projects/`, then picks the most recently modified `.jsonl` file in that directory (modified within the past 24 hours). Only those specific transcript files are registered for fsnotify watching.

#### Scenario: Pane path forward-encoded to project dir
- **WHEN** a live tmux pane has `pane_current_path=/home/bw/foo.bar` and `pane_current_command=claude`
- **THEN** the encoded project directory is `-home-bw-foo-bar`
- **AND** the watcher looks for `~/.claude/projects/-home-bw-foo-bar/` to find transcripts

#### Scenario: Pane matched by encoded working directory
- **WHEN** the forward-encoded path matches an existing project directory
- **THEN** that pane is associated with the most recently modified transcript in that directory (within 24 hours)

#### Scenario: No matching pane — transcript not watched
- **WHEN** no live tmux pane with a `claude*` command has a path that encodes to the project directory
- **THEN** no fsnotify watch is registered for transcripts in that directory

#### Scenario: Non-claude pane excluded
- **WHEN** a pane's `pane_current_command` does not match the prefix `claude*`
- **THEN** it is not considered for pane correlation, regardless of its working directory

### Requirement: Watcher upserts session-index record on active pane discovery
When the watcher discovers a live tmux pane running `claude*`, it SHALL upsert a record in `sessions.jsonl` via `sessions.Upsert` with `project_path` set to the pane's real `pane_current_path` and all available pane identifiers (`pane_id`, `window_id`, `window_name`, `tmux_session`).

#### Scenario: Active pane discovery writes real path to sessions.jsonl
- **WHEN** the watcher calls `listClaudePanes` and finds pane `%3` with `pane_current_path=/home/bw/dotfiles`
- **THEN** a sessions.jsonl record is upserted with `project_path: "/home/bw/dotfiles"` and `pane_id: "%3"`

#### Scenario: Reconcile on startup upserts all live pane sessions
- **WHEN** `watcher.Reconcile()` runs at dashboard open
- **THEN** all discovered live panes have their session records upserted in sessions.jsonl before the dashboard renders

### Requirement: Watcher clears pane fields in session-index when pane disappears
When `clearInactivePaneSessions` runs and detects that a previously-tracked pane no longer exists in the tmux server, it SHALL upsert the corresponding sessions.jsonl record with `pane_id`, `window_id`, `window_name`, and `tmux_session` all set to empty string and `status` set to `"idle"`. The session record itself (session_id, project_path, last_activity) is retained.

#### Scenario: Pane closes — session record pane fields cleared
- **WHEN** a pane that had an active sessions.jsonl record is no longer returned by `tmux list-panes -a`
- **THEN** the sessions.jsonl record is upserted with `pane_id: ""` and `status: "idle"`
- **AND** the session_id, project_path, and last_activity are retained

### Requirement: Watcher derives agent state from transcript events
On each write to a watched transcript file, the watcher SHALL read the tail of the file (last 20 lines), parse JSONL events, and derive one of: `running`, `waiting`. A stale timer is (re)started on each event; if no new events arrive within `@claude-notify-stale-minutes` (default 5), the state transitions to `stale`. Parse errors on individual lines SHALL be skipped.

#### Scenario: Tool use event → running
- **WHEN** the most recent parsed event in the transcript tail has `tool_use` content
- **THEN** the derived state is `running`

#### Scenario: User tool_result content → running
- **WHEN** the most recent event is a `user` message with `tool_result` content
- **THEN** the derived state is `running` (tool output returning to Claude)

#### Scenario: Assistant message with end_turn stop_reason → waiting
- **WHEN** the most recent event is an `assistant` message with `stop_reason: "end_turn"`
- **THEN** the derived state is `waiting`

#### Scenario: Real user message → clear notification
- **WHEN** the most recent event is a `user` message with text content (not tool_result)
- **THEN** `StateChange{Clear: true}` is emitted — the notification is removed

#### Scenario: No activity for stale duration → stale
- **WHEN** no new events arrive in the transcript for more than `@claude-notify-stale-minutes` minutes (default 5)
- **THEN** `StateChange{Status: "stale"}` is emitted via the stale timer

#### Scenario: Malformed JSONL line — skipped
- **WHEN** a transcript line cannot be parsed as JSON
- **THEN** that line is skipped and parsing continues with the next line

### Requirement: Watcher limits active file watches to recent transcripts
The watcher SHALL only register fsnotify watches for transcript files last modified within the past 24 hours, to avoid accumulating watches for old sessions.

#### Scenario: Old transcript not watched
- **WHEN** a transcript file has not been modified in more than 24 hours
- **THEN** no fsnotify watch is registered for it

#### Scenario: Active transcript is watched
- **WHEN** a transcript file has been modified within the past 24 hours
- **THEN** an fsnotify watch is registered for it
