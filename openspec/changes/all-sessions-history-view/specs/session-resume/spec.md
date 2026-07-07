## ADDED Requirements

### Requirement: Sessions L1 — `w`/`h`/`v` open a new claude session in the selected project
In the Sessions view Level-1 (project list), pressing `w`, `h`, or `v` SHALL open a fresh `claude` session (no `--resume`) in the selected project's directory. `w` opens a new tmux window named after the project leaf directory. `h` and `v` split the current pane in the outer (non-shpell) session horizontally or vertically.

#### Scenario: w opens new window named after leaf dir
- **WHEN** the user is at Sessions L1 and presses `w`
- **AND** the project path is resolved
- **THEN** `tmux neww -c <path> -n <leaf_dir> -t <outer-session> -- claude` is executed
- **AND** `DetachIfShpell` is called to close the popup

#### Scenario: h/v split in outer session
- **WHEN** the user presses `h` or `v` at Sessions L1
- **THEN** `tmux split-window -h/-v -c <path> -t <outer-session> -- claude` is executed
- **AND** the split lands in the user's real working session, not in _shpell-session

#### Scenario: Path unresolvable at L1
- **WHEN** no sessions in the selected project have a recoverable path
- **THEN** a toast "Cannot open: project path unknown" is shown

### Requirement: Sessions L2 — `w`/`h`/`v` resume a specific closed session
In the Sessions view Level-2 (individual session list), pressing `r`, `h`, or `v` on a closed session (empty `pane_id`) SHALL open that session with `claude --resume <session-id>`.

#### Scenario: w resumes closed session in new window
- **WHEN** the user selects a closed session at L2 and presses `w`
- **AND** the session's `project_path` is known (non-empty)
- **THEN** the system executes `tmux neww -c <project_path> -t <outer-session> -- claude --resume <session_id>`
- **AND** `DetachIfShpell` is called to close the popup

#### Scenario: h/v resumes closed session in split pane
- **WHEN** the user presses `h` or `v` on a closed session at L2
- **THEN** `tmux split-window -h/-v -c <path> -t <outer-session> -- claude --resume <session_id>` is executed

#### Scenario: Resume uses recovered path when project_path empty
- **WHEN** the user presses `w`/`h`/`v` on a session with empty `project_path`
- **THEN** path recovery is attempted via filesystem walk
- **AND** if a path is found, the command is executed with the recovered path

#### Scenario: Resume blocked when path unresolvable
- **WHEN** the user presses `w`/`h`/`v` on a session where path recovery fails
- **THEN** a toast message "Cannot resume: project path unknown" is shown
- **AND** no tmux command is executed and the popup remains open

### Requirement: `w`/`h`/`v` on an active session at L2 focuses its pane
In the Sessions view L2, pressing `w` on a session whose `pane_id` is non-empty SHALL switch to that pane's window. `h` and `v` are no-ops for active sessions.

#### Scenario: w on active session focuses existing window
- **WHEN** the user presses `w` on a session with a non-empty `pane_id`
- **THEN** `SelectPane` + `SelectWindow` are called and `DetachIfShpell` closes the popup

### Requirement: Open/resume command is issued within the current tmux server
The action SHALL only execute when the dashboard is running inside a tmux session. If invoked outside tmux, the action SHALL be a no-op with a toast error.

#### Scenario: Action inside tmux executes correctly
- **WHEN** `$TMUX` is set and `w`/`h`/`v` is pressed
- **THEN** the appropriate tmux command is executed against the current tmux server

#### Scenario: Action outside tmux shows error
- **WHEN** `$TMUX` is not set
- **THEN** a toast error is shown and no tmux command is executed

### Requirement: Split commands target the outer (non-shpell) session
When the dashboard is running inside `_shpell-session`, split-window commands SHALL target the user's real working session by passing `-t <outer-session>` to `tmux split-window`.

#### Scenario: OuterSession resolved from session list
- **WHEN** `tmux list-sessions` returns multiple sessions including `_shpell-session`
- **THEN** `OuterSession()` returns the first non-`_shpell-session` session name
- **AND** split-window uses `-t <outer-session>` so the new pane appears in the user's real session
