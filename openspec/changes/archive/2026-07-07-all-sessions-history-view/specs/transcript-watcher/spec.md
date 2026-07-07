## ADDED Requirements

### Requirement: Watcher upserts session-index record on active pane discovery
When the watcher discovers a live tmux pane running `claude*`, it SHALL upsert a record in `sessions.jsonl` via `sessions.Upsert` with the `project_path` set to the pane's real `pane_current_path`, and all available pane identifiers (`pane_id`, `window_id`, `window_name`, `tmux_session`).

#### Scenario: Active pane discovery writes real path to sessions.jsonl
- **WHEN** the watcher calls `listClaudePanes` and finds pane `%3` with `pane_current_path=/home/bw/dotfiles`
- **THEN** a sessions.jsonl record is upserted with `project_path: "/home/bw/dotfiles"` and `pane_id: "%3"`

#### Scenario: Reconcile on startup upserts all live pane sessions
- **WHEN** `watcher.Reconcile()` runs at dashboard open
- **THEN** all discovered live panes have their session records upserted in sessions.jsonl before the dashboard renders

### Requirement: Watcher clears pane fields in session-index when pane disappears
When the watcher detects that a previously-tracked pane no longer exists in the tmux server, it SHALL upsert the corresponding sessions.jsonl record with `pane_id`, `window_id`, `window_name`, and `tmux_session` all set to empty string, preserving the session record itself.

#### Scenario: Pane closes — session record pane fields cleared
- **WHEN** a pane that had an active sessions.jsonl record is no longer returned by `tmux list-panes -a`
- **THEN** the sessions.jsonl record for that session is upserted with `pane_id: ""`
- **AND** the session record itself (session_id, project_path, last_activity, status) is retained

## MODIFIED Requirements

### Requirement: Watcher maps transcript path to tmux pane
The watcher SHALL correlate each transcript with a live tmux pane by forward-encoding the pane's `pane_current_path` (replacing `/` and `.` with `-`) and looking up the resulting directory under `~/.claude/projects/`. Only panes whose `pane_current_command` matches the prefix `claude*` are considered. When a pane is correlated, the watcher SHALL also upsert a sessions.jsonl record with the real `pane_current_path`.

#### Scenario: Pane path forward-encoded to project dir
- **WHEN** a live tmux pane has `pane_current_path=/home/bw/foo.bar` and `pane_current_command=claude`
- **THEN** the encoded project directory is `-home-bw-foo-bar`
- **AND** the watcher looks for `~/.claude/projects/-home-bw-foo-bar/` to find transcripts

#### Scenario: Pane matched by encoded working directory
- **WHEN** the forward-encoded path matches an existing project directory under `~/.claude/projects/`
- **THEN** that pane is associated with the most recently modified transcript in that directory

#### Scenario: Pane correlated — session-index updated with real path
- **WHEN** pane `%5` with `pane_current_path=/home/bw/myproject` is correlated to a transcript
- **THEN** a sessions.jsonl record is upserted with `project_path: "/home/bw/myproject"` and `pane_id: "%5"`

#### Scenario: No matching pane — transcript not watched
- **WHEN** no live tmux pane with a `claude*` command has a path that encodes to the project directory
- **THEN** no fsnotify watch is registered for transcripts in that directory

#### Scenario: Non-claude pane excluded
- **WHEN** a pane's `pane_current_command` does not match the prefix `claude*`
- **THEN** it is not considered for pane correlation, regardless of its working directory
