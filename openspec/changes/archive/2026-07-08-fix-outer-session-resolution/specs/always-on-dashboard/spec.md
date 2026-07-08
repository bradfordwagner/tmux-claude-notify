## MODIFIED Requirements

### Requirement: Sessions L1 — `w`/`h`/`v` open a new claude session in the selected project
At Level-1 (project list), `w` opens `tmux neww -c <path> -n <leaf> -t <outer-session> -- claude`. `h`/`v` split horizontally/vertically in the outer (non-shpell) session. These create a FRESH session, not a resume. All commands SHALL target the outer session, resolved from the actual outer client rather than an arbitrary session-list ordering, to avoid creating windows in `_shpell-session` or in the wrong tmux session when multiple sessions are attached.

#### Scenario: w at L1 opens new window named after project leaf
- **WHEN** the user presses `w` at Sessions L1
- **THEN** `tmux neww -c <project_path> -n <leaf_dir> -t <outer-session> -- claude` is executed and the popup closes

#### Scenario: h/v at L1 split in outer session
- **WHEN** the user presses `h` or `v` at Sessions L1
- **THEN** `tmux split-window -h/-v -c <path> -t <outer-session> -- claude` is executed in the user's real working session

#### Scenario: w at L1 targets the session the popup was opened from, not the first session in the list
- **WHEN** multiple tmux sessions besides `_shpell-session` are attached (e.g. `edit`, `fwd`, `k8s`)
- **AND** the dashboard was opened from `k8s`
- **AND** the user presses `w` at Sessions L1
- **THEN** `OuterSession()` resolves to `k8s` (the session the outer client is actually attached to), not whichever session sorts first in `tmux list-sessions`
- **AND** the new claude window is created in `k8s`

### Requirement: Sessions L2 — `w`/`h`/`v` navigate to or resume selected session
At Level-2, `w`/`enter` on an active session SHALL navigate to its pane. On a closed session (no `pane_id`), `w` resumes via `tmux neww -c <path> -t <outer-session> -- claude --resume <session_id>`, targeting the outer session resolved from the actual outer client. `h`/`v` split horizontally/vertically for closed sessions, targeting the same resolved outer session.

#### Scenario: enter/w on active session navigates to pane — same session
- **WHEN** the selected session has a non-empty `pane_id`
- **AND** the user presses `enter` or `w`
- **AND** the target pane lives in the same outer tmux session
- **THEN** `SelectPane(paneID)`, `SelectWindow(session, windowID)`, and `SwitchOuterClientToSessionWindow(session, windowID)` are called, then `DetachIfShpell`

#### Scenario: enter/w on active session navigates to pane — different session
- **WHEN** the selected session has a non-empty `pane_id`
- **AND** the user presses `enter` or `w`
- **AND** the target pane lives in a different tmux session than the one the outer client is attached to
- **THEN** `SwitchOuterClientToSessionWindow` issues `switch-client -c <outer-client> -t <session>:<window>` before the detach
- **AND** after the popup closes the outer client is in the target session at the correct window

#### Scenario: w on closed session resumes claude in new window
- **WHEN** the selected session has an empty `pane_id`
- **AND** the user presses `w`
- **THEN** `tmux neww -c <recovered_path> -t <outer-session> -- claude --resume <session_id>` is executed

#### Scenario: w on closed session targets the session the popup was opened from, not the first session in the list
- **WHEN** multiple tmux sessions besides `_shpell-session` are attached
- **AND** the dashboard was opened from a session other than the one that sorts first in `tmux list-sessions`
- **AND** the user presses `w` on a closed session
- **THEN** the resumed claude window is created in the session the dashboard was actually opened from

#### Scenario: h/v on closed session resumes in split pane
- **WHEN** the selected session has an empty `pane_id`
- **AND** the user presses `h` or `v`
- **THEN** `tmux split-window -h/-v -c <path> -t <outer-session> -- claude --resume <session_id>` is executed

#### Scenario: No active pane in Notifications view
- **WHEN** the selected Notifications entry is a session-only entry with no active pane
- **AND** the user presses `enter`
- **THEN** a toast "No active pane — switch to Sessions tab to resume" is shown
