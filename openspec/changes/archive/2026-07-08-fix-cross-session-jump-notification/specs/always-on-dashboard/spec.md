## MODIFIED Requirements

### Requirement: Sessions L2 — `w`/`h`/`v` navigate to or resume selected session
At Level-2, `w`/`enter` on an active session SHALL navigate to its pane, resolving the outer client deterministically even across tmux sessions. On a closed session (no `pane_id`), `w` resumes via `tmux neww -c <path> -t <outer-session> -- claude --resume <session_id>`. `h`/`v` split horizontally/vertically for closed sessions.

#### Scenario: enter/w on active session navigates to pane — same session
- **WHEN** the selected session has a non-empty `pane_id`
- **AND** the user presses `enter` or `w`
- **AND** the target pane lives in the same outer tmux session
- **THEN** `SelectPane(paneID)`, `SelectWindow(session, windowID)`, and `SwitchOuterClientToSessionWindow(session, windowID)` are called, then `DetachIfShpell`

#### Scenario: enter/w on active session navigates to pane — different session
- **WHEN** the selected session has a non-empty `pane_id`
- **AND** the user presses `enter` or `w`
- **AND** the target pane lives in a different tmux session than the one the outer client is attached to
- **THEN** `SwitchOuterClientToSessionWindow` resolves the outer client by querying `#{client_name}` and `#{client_session}` with an explicit `-t <dashboard-pane-id>` target rather than tmux's implicit current-client resolution
- **AND** it then issues `switch-client -c <outer-client> -t <session>:<window>` before the detach
- **AND** after the popup closes the outer client is in the target session at the correct window

#### Scenario: w on closed session resumes claude in new window
- **WHEN** the selected session has an empty `pane_id`
- **AND** the user presses `w`
- **THEN** `tmux neww -c <recovered_path> -t <outer-session> -- claude --resume <session_id>` is executed

#### Scenario: h/v on closed session resumes in split pane
- **WHEN** the selected session has an empty `pane_id`
- **AND** the user presses `h` or `v`
- **THEN** `tmux split-window -h/-v -c <path> -t <outer-session> -- claude --resume <session_id>` is executed

#### Scenario: No active pane in Notifications view
- **WHEN** the selected Notifications entry is a session-only entry with no active pane
- **AND** the user presses `enter`
- **THEN** a toast "No active pane — switch to Sessions tab to resume" is shown

### Requirement: enter in Notifications view clears notification and navigates
Pressing `enter` on a Notifications entry SHALL clear the notification (if from notifications.jsonl), navigate to the window via `SelectPane` + `SelectWindow`, and close the popup.

#### Scenario: enter clears and navigates — same session
- **WHEN** the user presses `enter` on a notifications.jsonl-backed entry in the same outer session
- **THEN** `ClearPopStyle`, `UnregisterClearHook`, and `store.ClearPane` are called
- **AND** `SelectPane(paneID)` + `SelectWindow(session, windowID)` + `SwitchOuterClientToSessionWindow(session, windowID)` navigate to the pane
- **AND** `DetachIfShpell` closes the popup

#### Scenario: enter clears and navigates — different session
- **WHEN** the user presses `enter` on a notifications.jsonl-backed entry whose pane lives in a different tmux session
- **THEN** `ClearPopStyle`, `UnregisterClearHook`, and `store.ClearPane` are called
- **AND** `SwitchOuterClientToSessionWindow` resolves the outer client by querying `#{client_name}` and `#{client_session}` with an explicit `-t <dashboard-pane-id>` target rather than tmux's implicit current-client resolution
- **AND** it then issues `switch-client -c <outer-client> -t <session>:<window>`
- **AND** after the popup closes the outer client is in the target session at the correct window
