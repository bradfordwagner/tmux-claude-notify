## ADDED Requirements

### Requirement: Dashboard shows all uncleared notifications regardless of pane liveness
The dashboard SHALL display all uncleared JSONL entries without filtering by whether the source pane currently exists in the tmux server. An entry is visible as long as it is uncleared, enabling the user to dismiss notifications from sessions that were restarted or panes that were closed since the Stop hook fired.

#### Scenario: Gone-pane entry visible in dashboard
- **WHEN** an uncleared JSONL entry exists for a pane that no longer appears in `tmux list-panes -a`
- **THEN** the entry is still rendered in the dashboard notification list

#### Scenario: Gone-pane entry can be cleared from dashboard
- **WHEN** the user selects a gone-pane entry and confirms dismissal
- **THEN** the JSONL entry is marked cleared
- **AND** any window-style teardown proceeds normally (tmux errors are non-fatal if window also gone)

### Requirement: Gone pane indicator in PATH column
When a notification entry's pane no longer exists in the current tmux server, the PATH column SHALL display `(gone)` instead of the pane's current working directory, making it clear that the pane is no longer live.

#### Scenario: Path column shows gone for missing pane
- **WHEN** `tmux display-message -t <paneID>` fails because the pane no longer exists
- **THEN** the PATH column displays `(gone)`

#### Scenario: Path column shows real path for live pane
- **WHEN** the pane still exists and `tmux display-message` succeeds
- **THEN** the PATH column displays the truncated real path as before
