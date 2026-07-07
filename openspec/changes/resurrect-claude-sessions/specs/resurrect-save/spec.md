# Spec: resurrect-save

## Purpose

Defines how `claude-notify resurrect save` discovers active claude sessions and persists them to a JSON sidecar so they can be restored after a tmux-resurrect cycle.

## ADDED Requirements

### Requirement: Save snapshots all panes running claude
`claude-notify resurrect save` SHALL enumerate all tmux panes (across all sessions) whose `pane_current_command` starts with `claude`. For each such pane it SHALL record: `tmux_session`, `window_index`, `pane_index`, `session_id`, and `project_path`. The result is written atomically to the resurrect sidecar.

#### Scenario: Active panes persisted
- **WHEN** one or more panes are running `claude*`
- **THEN** each pane is written as an entry in the sidecar with its positional key, session ID, and project path

#### Scenario: No active panes — sidecar cleared
- **WHEN** no panes are running `claude*`
- **THEN** the sidecar is written with an empty panes array (not left from a previous save)

#### Scenario: Save is idempotent
- **WHEN** `save` is called twice in a row without tmux state changing
- **THEN** the sidecar content is identical after both calls

### Requirement: Session ID derived from latest transcript file
The save command SHALL derive each pane's `session_id` from the most recently modified `.jsonl` transcript file in `~/.claude/projects/<encoded-path>/`, subject to the `@claude-notify-transcript-age-days` cutoff. `project_path` is always taken from the live `pane_current_path` reported by tmux. `sessions.jsonl` is not consulted.

#### Scenario: Latest transcript used for session ID
- **WHEN** one or more `.jsonl` files exist in the pane's project directory within the age cutoff
- **THEN** the filename stem (without `.jsonl`) of the most recently modified file is used as `session_id`

#### Scenario: Pane with no discoverable session ID is skipped
- **WHEN** no `.jsonl` file within the age cutoff exists for the pane's project path
- **THEN** that pane is omitted from the sidecar silently

### Requirement: Sidecar written atomically
The resurrect sidecar SHALL be written via a temp file + rename to prevent partial reads by a concurrent restore. The sidecar path is fixed at `~/.local/share/tmux-claude-notify/resurrect.json`.

#### Scenario: Write succeeds
- **WHEN** save completes without error
- **THEN** the sidecar file is a valid JSON file at `~/.local/share/tmux-claude-notify/resurrect.json`

### Requirement: Transcript age cutoff is configurable
The maximum age of transcript files considered during save SHALL be controlled by the `@claude-notify-transcript-age-days` TPM option (default: 14 days). This option is shared with the dashboard's session discovery.

#### Scenario: Default cutoff is 14 days
- **WHEN** `@claude-notify-transcript-age-days` is not set
- **THEN** only transcripts modified within the past 14 days are considered

#### Scenario: Custom cutoff applied
- **WHEN** `@claude-notify-transcript-age-days` is set to a positive integer
- **THEN** only transcripts modified within that many days are considered
