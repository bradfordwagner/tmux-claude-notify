## ADDED Requirements

### Requirement: Watcher discovers Claude Code transcript files
The transcript watcher SHALL scan `~/.claude/projects/` recursively for `*.jsonl` files and begin watching each one for changes. It SHALL also watch the projects directory itself to detect new transcript files created by new claude sessions.

#### Scenario: Existing transcripts discovered on startup
- **WHEN** the daemon starts
- **THEN** all `*.jsonl` files under `~/.claude/projects/` are discovered and registered for watching

#### Scenario: New transcript detected during run
- **WHEN** a new `*.jsonl` file appears under `~/.claude/projects/`
- **THEN** the watcher registers it within one fsnotify event cycle

#### Scenario: Projects directory does not exist
- **WHEN** `~/.claude/projects/` does not exist at startup
- **THEN** the watcher starts without error and watches for the directory to be created

### Requirement: Watcher derives agent state from transcript events
On each write to a transcript file, the watcher SHALL read the tail of the file, parse JSONL events, and derive one of: `running`, `waiting`, `idle`, `stale`. Parse errors on individual lines SHALL be skipped; unrecognized event types SHALL be ignored.

#### Scenario: Tool use event → running
- **WHEN** the most recent event in the transcript is a `tool_use` event
- **THEN** the derived state is `running`

#### Scenario: Assistant message followed by silence → waiting
- **WHEN** the most recent event is an `assistant` message
- **AND** no new events have appeared for at least 2 seconds
- **THEN** the derived state is `waiting`

#### Scenario: No transcript activity for extended period → stale
- **WHEN** no new events have appeared in the transcript for more than 5 minutes
- **THEN** the derived state is `stale`

#### Scenario: Malformed JSONL line — skipped
- **WHEN** a transcript line cannot be parsed as JSON
- **THEN** that line is skipped and parsing continues with the next line

### Requirement: Watcher maps transcript path to tmux pane
The watcher SHALL decode the Claude Code project directory from the encoded path segment of the transcript file (`~/.claude/projects/<encoded-path>/`) and correlate it with a live tmux pane running in that directory.

#### Scenario: Project dir decoded from encoded path
- **WHEN** a transcript file lives at `~/.claude/projects/-home-bw-foo-bar/session.jsonl`
- **THEN** the decoded project directory is `/home/bw/foo/bar`

#### Scenario: Pane matched by working directory
- **WHEN** the decoded project directory matches the `pane_current_path` of a live tmux pane running `claude`
- **THEN** that pane is associated with the transcript's state

#### Scenario: No matching pane — notification suppressed
- **WHEN** no live tmux pane matches the decoded project directory
- **THEN** no notification is fired for that transcript

### Requirement: Watcher limits active file watches
The watcher SHALL only register fsnotify watches for transcript files last modified within the past 24 hours, to avoid accumulating watches for old sessions.

#### Scenario: Old transcript not watched
- **WHEN** a transcript file has not been modified in more than 24 hours
- **THEN** no fsnotify watch is registered for it

#### Scenario: Active transcript is watched
- **WHEN** a transcript file has been modified within the past 24 hours
- **THEN** an fsnotify watch is registered for it
