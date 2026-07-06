# Spec: transcript-watcher

## Purpose

The transcript watcher is a new component that monitors Claude Code transcript files under `~/.claude/projects/` using fsnotify. It derives agent state (`running`, `waiting`, `idle`, `stale`) from JSONL transcript events and maps each transcript to a live tmux pane, enabling proactive notification without relying solely on the Stop hook.

## Requirements

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
- **WHEN** no new events have appeared in the transcript for more than `@claude-notify-stale-minutes` minutes (default 5)
- **THEN** the derived state is `stale`

#### Scenario: Malformed JSONL line — skipped
- **WHEN** a transcript line cannot be parsed as JSON
- **THEN** that line is skipped and parsing continues with the next line

### Requirement: Watcher maps transcript path to tmux pane
The watcher SHALL correlate each transcript with a live tmux pane by forward-encoding the pane's `pane_current_path` (replacing `/` and `.` with `-`) and looking up the resulting directory under `~/.claude/projects/`. Only panes whose `pane_current_command` matches the prefix `claude*` are considered.

#### Scenario: Pane path forward-encoded to project dir
- **WHEN** a live tmux pane has `pane_current_path=/home/bw/foo.bar` and `pane_current_command=claude`
- **THEN** the encoded project directory is `-home-bw-foo-bar`
- **AND** the watcher looks for `~/.claude/projects/-home-bw-foo-bar/` to find transcripts

#### Scenario: Pane matched by encoded working directory
- **WHEN** the forward-encoded path matches an existing project directory under `~/.claude/projects/`
- **THEN** that pane is associated with the most recently modified transcript in that directory

#### Scenario: No matching pane — transcript not watched
- **WHEN** no live tmux pane with a `claude*` command has a path that encodes to the project directory
- **THEN** no fsnotify watch is registered for transcripts in that directory

#### Scenario: Non-claude pane excluded
- **WHEN** a pane's `pane_current_command` does not match the prefix `claude*`
- **THEN** it is not considered for pane correlation, regardless of its working directory

### Requirement: Watcher limits active file watches
The watcher SHALL only register fsnotify watches for transcript files last modified within the past 24 hours, to avoid accumulating watches for old sessions.

#### Scenario: Old transcript not watched
- **WHEN** a transcript file has not been modified in more than 24 hours
- **THEN** no fsnotify watch is registered for it

#### Scenario: Active transcript is watched
- **WHEN** a transcript file has been modified within the past 24 hours
- **THEN** an fsnotify watch is registered for it
