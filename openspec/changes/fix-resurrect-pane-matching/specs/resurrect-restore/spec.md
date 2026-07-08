## MODIFIED Requirements

### Requirement: Restore replays claude --resume into matching panes
`claude-notify resurrect restore` SHALL read the resurrect sidecar at `~/.local/share/tmux-claude-notify/resurrect.json` and, for each saved entry, find a live tmux pane matching the same `(tmux_session, window_name, pane_index)`. If a match is found AND the pane is not already running `claude*` AND the pane's `pane_current_path` matches the saved `project_path`, the command `claude --resume <session_id>` SHALL be sent to that pane via `tmux send-keys`. Panes whose path does not match SHALL be skipped; no `cd` prefix is used.

#### Scenario: Matching idle pane in correct directory receives resume command
- **WHEN** a saved entry matches a live pane (by session, window_name, pane_index) that is running a shell (not `claude*`) and whose `pane_current_path` equals `project_path`
- **THEN** `tmux send-keys -t <pane_id> "claude --resume <session_id>" Enter` is executed

#### Scenario: Active pane is skipped
- **WHEN** a saved entry matches a live pane whose `pane_current_command` starts with `claude`
- **THEN** no keys are sent to that pane (restore is idempotent for live sessions)

#### Scenario: No matching live pane — entry skipped silently
- **WHEN** a saved entry references a session/window_name/pane that no longer exists
- **THEN** the entry is skipped without error

#### Scenario: Path mismatch — pane skipped
- **WHEN** a saved entry's `project_path` does not match the live pane's `pane_current_path`
- **THEN** the entry is skipped; no keys are sent and no `cd` command is issued

### Requirement: Restore skips v1 sidecars
If the sidecar `version` field is less than 2, `restore` SHALL exit without sending any keys. This prevents incorrect matches against sidecars that lack the `window_name` field.

#### Scenario: v1 sidecar is ignored
- **WHEN** the sidecar exists and has `version: 1`
- **THEN** restore exits 0 without sending any keys

#### Scenario: v2 sidecar is processed normally
- **WHEN** the sidecar exists and has `version: 2`
- **THEN** restore proceeds with name-based matching

### Requirement: Restore is a no-op when sidecar is absent or empty
If the sidecar file does not exist or contains an empty panes array, `restore` SHALL exit without error and without sending any keys.

#### Scenario: Missing sidecar
- **WHEN** the sidecar file does not exist
- **THEN** restore exits 0 with no output

#### Scenario: Empty panes array
- **WHEN** the sidecar exists but `panes` is `[]`
- **THEN** restore exits 0 with no output

## REMOVED Requirements

### Requirement: Restore prepends cd when pane's current path differs
**Reason**: The `cd` fallback caused wrong-directory panes (including the "cn" grimoire placeholder) to receive `claude --resume` commands. Path mismatch now skips the pane entirely.
**Migration**: If a pane is not in the correct project directory after restore, manually `cd <project_path>` then run `claude --resume <session_id>`.
