# Spec: always-on-dashboard

## Purpose

The always-on dashboard is the bubbletea TUI that runs when `claude-notify` is invoked with no arguments. It displays the current notification state for all tracked tmux panes, integrates a transcript watcher to keep state current without user interaction, and auto-refreshes as transcript events arrive.

## Requirements

### Requirement: Dashboard starts transcript watcher on init
When the dashboard TUI starts (`claude-notify` with no args), it SHALL initialize a transcript watcher alongside the existing fsnotify settings/JSONL watcher. The watcher runs for the lifetime of the TUI process and is closed when the TUI exits.

#### Scenario: Watcher started on dashboard open
- **WHEN** `claude-notify` is invoked with no arguments
- **THEN** the bubbletea model initializes with a transcript watcher active
- **AND** the watcher begins scanning `~/.claude/projects/` for active transcript files

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
- **AND** the transcript confirms the agent is still waiting (last event was `assistant` message with silence)
- **THEN** the entry is left unchanged

#### Scenario: No transcript found for stored entry
- **WHEN** the JSONL store has an uncleared entry for a pane but no matching transcript file is found
- **THEN** the entry is left unchanged (Stop hook wrote it; treated as valid waiting state)

### Requirement: Dashboard renders agent status per entry
Each dashboard entry SHALL display the agent status with an icon and color as the first column, followed by window name, pane current path, session name, and age. The raw pane ID SHALL NOT appear in the rendered row. A header row and separator SHALL appear above the entry list when entries are present.

#### Scenario: Waiting entry styled with accent color
- **WHEN** an entry has `status: waiting`
- **THEN** it is rendered with "⏳ waiting" in the `#AD8EE6` accent color as the first column

#### Scenario: Running entry styled with warn color
- **WHEN** an entry has `status: running`
- **THEN** it is rendered with "⚙  running" in the warn/orange color as the first column

#### Scenario: Stale entry styled with dim color
- **WHEN** an entry has `status: stale`
- **THEN** it is rendered with "💤 stale" in the dim/subtle color as the first column

#### Scenario: Column headers rendered above entries
- **WHEN** one or more entries are displayed
- **THEN** a header row ("STATUS WINDOW PATH SESSION AGE") and separator line appear above the first entry

#### Scenario: Path column shows pane working directory
- **WHEN** an entry is rendered
- **THEN** the PATH column shows the pane's current directory, truncated to the last two components with `$HOME` replaced by `~`

### Requirement: Dashboard auto-refreshes on transcript state change
When the transcript watcher detects a state change while the dashboard is open, the dashboard SHALL re-render within one bubbletea event loop tick without user interaction.

#### Scenario: State change triggers re-render
- **WHEN** the transcript watcher emits a state change event
- **THEN** the bubbletea model receives a message and re-renders the entry list

#### Scenario: User message in transcript clears notification
- **WHEN** the transcript watcher sees a `user` message event (user responded to claude)
- **THEN** `store.ClearPane` is called and the entry is removed from the dashboard
