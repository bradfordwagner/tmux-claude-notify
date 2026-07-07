# Spec: resurrect-restore

## Purpose

Defines how `claude-notify resurrect restore` replays `claude --resume` commands into the correct panes after a tmux-resurrect cycle, using the positional key saved by resurrect-save.

## Requirements

### Requirement: Restore replays claude --resume into matching panes
`claude-notify resurrect restore` SHALL read the resurrect sidecar at `~/.local/share/tmux-claude-notify/resurrect.json` and, for each saved entry, find a live tmux pane matching the same `(tmux_session, window_index, pane_index)`. If a match is found AND the pane is not already running `claude*`, the command `claude --resume <session_id>` SHALL be sent to that pane via `tmux send-keys`.

#### Scenario: Matching idle pane receives resume command
- **WHEN** a saved entry matches a live pane that is running a shell (not `claude*`)
- **THEN** `tmux send-keys -t <pane_id> "claude --resume <session_id>" Enter` is executed

#### Scenario: Active pane is skipped
- **WHEN** a saved entry matches a live pane whose `pane_current_command` starts with `claude`
- **THEN** no keys are sent to that pane (restore is idempotent for live sessions)

#### Scenario: No matching live pane — entry skipped silently
- **WHEN** a saved entry references a session/window/pane that no longer exists
- **THEN** the entry is skipped without error

### Requirement: Restore prepends cd when pane's current path differs
If the saved `project_path` is non-empty and the live pane's `pane_current_path` does not match, the restore command SHALL be prefixed with `cd <project_path> && ` so claude opens in the correct directory.

#### Scenario: Working directory already correct — no cd needed
- **WHEN** the pane's `pane_current_path` matches `project_path`
- **THEN** only `claude --resume <session_id>` is sent (no `cd`)

#### Scenario: Working directory differs — cd prepended
- **WHEN** the pane's `pane_current_path` does not match `project_path`
- **THEN** `cd <project_path> && claude --resume <session_id>` is sent as a single send-keys call

### Requirement: Restore is a no-op when sidecar is absent or empty
If the sidecar file does not exist or contains an empty panes array, `restore` SHALL exit without error and without sending any keys.

#### Scenario: Missing sidecar
- **WHEN** the sidecar file does not exist
- **THEN** restore exits 0 with no output

#### Scenario: Empty panes array
- **WHEN** the sidecar exists but `panes` is `[]`
- **THEN** restore exits 0 with no output
