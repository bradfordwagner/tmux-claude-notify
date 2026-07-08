## MODIFIED Requirements

### Requirement: Save snapshots all panes running claude
`claude-notify resurrect save` SHALL enumerate all tmux panes (across all sessions) whose `pane_current_command` starts with `claude`. For each such pane it SHALL record: `tmux_session`, `window_name`, `pane_index`, `session_id`, and `project_path`. The result is written atomically to the resurrect sidecar. `window_name` replaces the previous `window_index` field; the sidecar version is bumped to 2.

#### Scenario: Active panes persisted
- **WHEN** one or more panes are running `claude*`
- **THEN** each pane is written as an entry in the sidecar with `window_name` (not `window_index`), `pane_index`, `session_id`, and `project_path`

#### Scenario: No active panes — sidecar cleared
- **WHEN** no panes are running `claude*`
- **THEN** the sidecar is written with an empty panes array (not left from a previous save)

#### Scenario: Save is idempotent
- **WHEN** `save` is called twice in a row without tmux state changing
- **THEN** the sidecar content is identical after both calls

#### Scenario: Sidecar version field is 2
- **WHEN** save writes the sidecar
- **THEN** the JSON `version` field is `2`
