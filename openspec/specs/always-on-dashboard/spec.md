# Spec: always-on-dashboard

## Purpose

The always-on dashboard is the bubbletea TUI that runs when `claude-notify` is invoked with no arguments. It displays current notification state for all tracked tmux panes, integrates a transcript watcher to keep state current without user interaction, and provides a Sessions view for browsing and resuming historical Claude sessions.

## Requirements

### Requirement: Dashboard starts transcript watcher on init
When the dashboard TUI starts (`claude-notify` with no args), it SHALL initialize a transcript watcher alongside the existing fsnotify settings/JSONL watcher. The watcher runs for the lifetime of the TUI process and is closed when the TUI exits.

#### Scenario: Watcher started on dashboard open
- **WHEN** `claude-notify` is invoked with no arguments
- **THEN** the bubbletea model initializes with a transcript watcher active
- **AND** the watcher begins scanning active claude panes via `tmux list-panes`

#### Scenario: Watcher stopped on dashboard close
- **WHEN** the user presses `q`, `esc`, or `ctrl+c`
- **THEN** the transcript watcher is closed and all fsnotify watches are released

### Requirement: Dashboard reconciles store state on open
On startup, after loading the JSONL notification log, the dashboard SHALL scan active transcript files and correct any store entries whose status no longer matches current transcript state.

#### Scenario: Stale "waiting" entry corrected on open
- **WHEN** the JSONL store has an uncleared entry with `status: waiting` for a pane
- **AND** the corresponding transcript shows a `user` message as the most recent event (user already responded)
- **THEN** `store.ClearPane` is called for that pane during reconciliation

#### Scenario: "Waiting" entry confirmed on open
- **WHEN** the JSONL store has an uncleared entry for a pane
- **AND** the transcript confirms the agent is still waiting
- **THEN** the entry is left unchanged

#### Scenario: No transcript found for stored entry
- **WHEN** the JSONL store has an uncleared entry for a pane but no matching transcript file is found
- **THEN** the entry is left unchanged (Stop hook wrote it; treated as valid waiting state)

### Requirement: Dashboard has two views toggled by Tab
The TUI SHALL have a Notifications view (default) and a Sessions view. Pressing `Tab` cycles between them. The active view is shown in a header bar with the active tab highlighted in the accent color (`#AD8EE6`) and the inactive tab dimmed.

#### Scenario: Default view is Notifications
- **WHEN** the dashboard opens
- **THEN** the Notifications view is active and "[ Notifications ]" is highlighted in the header

#### Scenario: Tab switches to Sessions view
- **WHEN** the user presses `Tab` while in Notifications view
- **THEN** the Sessions view becomes active and "[ Sessions ]" is highlighted

#### Scenario: Tab switches back to Notifications view
- **WHEN** the user presses `Tab` while in Sessions view
- **THEN** the Notifications view becomes active

### Requirement: Dashboard renders agent status per entry in Notifications view
Each Notifications view entry SHALL display STATUS (icon + text), PIN (📌 or blank), POP indicator (● or blank), WINDOW name, PATH (last two components, `~`-abbreviated, dynamic width), SESSION name, and AGE. The raw pane ID SHALL NOT appear. A header row and separator SHALL appear above the entry list when entries are present.

#### Scenario: Waiting entry styled with accent color
- **WHEN** an entry has `status: waiting`
- **THEN** it is rendered with "⏳ waiting" in the `#AD8EE6` accent color

#### Scenario: Running entry styled with warn color
- **WHEN** an entry has `status: running`
- **THEN** it is rendered with "⚙  running" in the warn/orange color

#### Scenario: Stale entry styled with dim color
- **WHEN** an entry has `status: stale`
- **THEN** it is rendered with "💤 stale" in the dim/subtle color

#### Scenario: Column headers rendered above entries
- **WHEN** one or more entries are displayed
- **THEN** a header row "STATUS  PIN  P  WINDOW  PATH  SESSION  AGE" and separator appear above the first entry

#### Scenario: Path column shows pane working directory
- **WHEN** an entry is rendered
- **THEN** the PATH column shows the pane's current directory, truncated to the last two components with `$HOME` replaced by `~`

#### Scenario: Pop indicator shows pane background pop state
- **WHEN** an entry's pane has an active pane-local background pop (`window-style` set via `set-option -p`)
- **THEN** the P column renders `●` in the accent color (`#AD8EE6`)
- **WHEN** the pane has no active pop
- **THEN** the P column renders a blank space

#### Scenario: Pop indicator queried at load time
- **WHEN** the Notifications view loads entries from `notifications.jsonl`
- **THEN** `IsPanePopped(paneID)` is called for each notification-backed entry to set the pop flag
- **AND** the flag is used for display only (not persisted)

### Requirement: Dashboard auto-refreshes on transcript state change
When the transcript watcher detects a state change while the dashboard is open, the dashboard SHALL re-render within one bubbletea event loop tick without user interaction.

#### Scenario: State change triggers re-render
- **WHEN** the transcript watcher emits a state change event
- **THEN** the bubbletea model receives a message and re-renders the entry list

#### Scenario: User message in transcript clears notification
- **WHEN** the transcript watcher sees a `user` message event (user responded to claude)
- **THEN** `store.ClearPane` is called and the entry is removed from the dashboard

### Requirement: Sessions view is a two-level drill-in table
The Sessions view SHALL display a Level-1 projects table (one row per project, plus "📌 Pinned" group). `enter` on a project row drills into Level-2 (session rows for that project). `esc` in Level-2 returns to Level-1; `esc` in Level-1 quits the dashboard.

#### Scenario: Level-1 shows one row per project
- **WHEN** the Sessions view is activated
- **THEN** a table is rendered with columns STATUS, PROJECT, COUNT, LAST USED

#### Scenario: enter on Level-1 row drills in
- **WHEN** the user presses `enter` on a project row
- **THEN** Level-2 shows session rows for that project

#### Scenario: esc in Level-2 returns to Level-1
- **WHEN** the user presses `esc` while in Level-2
- **THEN** the view returns to Level-1

#### Scenario: esc in Level-1 quits
- **WHEN** the user presses `esc` while in Level-1
- **THEN** the dashboard closes

### Requirement: `s` key cycles sort in Sessions view
Pressing `s` in the Sessions view SHALL cycle the sort field between `age` (default, most recently active first) and `status` (most urgent first). The active sort field is shown in the tab header. Pinned sessions always float above unpinned within any sort order.

#### Scenario: Default sort is age
- **WHEN** the Sessions view first opens
- **THEN** sessions are sorted by most recent `last_activity`
- **AND** the header shows "sort: age"

#### Scenario: s advances sort to status
- **WHEN** the user presses `s` with sort=age active
- **THEN** sessions are sorted by urgency (waiting > running > stale > idle)
- **AND** the header shows "sort: status"

### Requirement: `f` key toggles active-pane filter in Sessions view
Pressing `f` SHALL toggle a filter that limits displayed sessions to those with a non-empty `pane_id`. Pinned sessions always appear regardless of filter state.

#### Scenario: f activates filter
- **WHEN** the user presses `f`
- **THEN** only projects with active sessions (and "📌 Pinned") are shown
- **AND** the header shows "filter: active panes"

#### Scenario: f deactivates filter
- **WHEN** the user presses `f` again
- **THEN** all sessions are shown and the filter indicator is removed

### Requirement: `p` key toggles pin on selected session in Level-2
In Level-2, pressing `p` SHALL toggle the `pinned` flag on the currently selected session. Pinned sessions persist in `sessions.jsonl` and appear in both the Sessions view and the Notifications view (even when no active pane).

#### Scenario: p pins a session
- **WHEN** the cursor is on an unpinned session and the user presses `p`
- **THEN** the session's `pinned` flag is set to `true` and 📌 appears in the PIN column

#### Scenario: p unpins a session
- **WHEN** the cursor is on a pinned session and the user presses `p`
- **THEN** the session's `pinned` flag is set to `false`

### Requirement: Sessions L1 — `w`/`h`/`v` open a new claude session in the selected project
At Level-1 (project list), `w` opens `tmux neww -c <path> -n <leaf> -t <outer-session> -- claude`. `h`/`v` split horizontally/vertically in the outer (non-shpell) session. These create a FRESH session, not a resume. All commands target the outer session to avoid creating windows in `_shpell-session`.

#### Scenario: w at L1 opens new window named after project leaf
- **WHEN** the user presses `w` at Sessions L1
- **THEN** `tmux neww -c <project_path> -n <leaf_dir> -t <outer-session> -- claude` is executed and the popup closes

#### Scenario: h/v at L1 split in outer session
- **WHEN** the user presses `h` or `v` at Sessions L1
- **THEN** `tmux split-window -h/-v -c <path> -t <outer-session> -- claude` is executed in the user's real working session

### Requirement: Sessions L2 — `w`/`h`/`v` navigate to or resume selected session
At Level-2, `w`/`enter` on an active session navigates to its pane. On a closed session (no `pane_id`), `w` resumes via `tmux neww -c <path> -t <outer-session> -- claude --resume <session_id>`. `h`/`v` split horizontally/vertically for closed sessions.

#### Scenario: enter/w on active session navigates to pane
- **WHEN** the selected session has a non-empty `pane_id`
- **AND** the user presses `enter` or `w`
- **THEN** `SelectPane(paneID)` and `SelectWindow(session, windowID)` are called, then `DetachIfShpell`

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

#### Scenario: enter clears and navigates
- **WHEN** the user presses `enter` on a notifications.jsonl-backed entry
- **THEN** `ClearPopStyle`, `UnregisterClearHook`, and `store.ClearPane` are called
- **AND** `SelectPane(paneID)` + `SelectWindow(session, windowID)` navigate to the pane
- **AND** `DetachIfShpell` closes the popup
