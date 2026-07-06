## ADDED Requirements

### Requirement: Default invocation opens the dashboard
Running `claude-notify` with no subcommand SHALL open an interactive bubbletea TUI displaying all uncleared notifications, sorted most-recent first, filtered to panes that are currently alive in tmux.

#### Scenario: Dashboard shows uncleared notifications for live panes
- **WHEN** `claude-notify` is invoked with no arguments
- **AND** there are uncleared notifications whose panes still exist in tmux
- **THEN** the TUI renders a list with the most recent notification at the top

#### Scenario: Dashboard excludes notifications for dead panes
- **WHEN** a notification exists in the log for a pane that no longer exists
- **THEN** that notification is not shown in the list

#### Scenario: Dashboard excludes already-cleared notifications
- **WHEN** a notification has been marked cleared in the log
- **THEN** it does not appear in the dashboard list

#### Scenario: Empty state when no pending notifications
- **WHEN** there are no uncleared notifications for live panes
- **THEN** the dashboard displays an empty state message rather than an empty list

#### Scenario: Selecting an entry navigates to that window
- **WHEN** the user selects a notification entry
- **THEN** `SelectWindow` sets the window as active in its session (session-level, not client-level)
- **AND** the notification is marked cleared in the log
- **AND** if other uncleared entries remain, the dashboard stays open
- **AND** if the list is now empty, the dashboard quits

#### Scenario: Dashboard auto-quits when last entry is selected
- **WHEN** the user selects the last remaining notification
- **THEN** the dashboard quits after clearing the entry
- **AND** if inside a grimoire shpell, `detach-client` closes the popup before quitting

#### Scenario: Dashboard accent matches grimoire color scheme
- **WHEN** the dashboard is rendered
- **THEN** highlighted/selected items use `#AD8EE6` to match the tmux window highlight color

### Requirement: Dashboard auto-configures the Stop hook on startup
When the dashboard opens, `claude-notify` SHALL check `~/.claude/settings.json` and automatically add the Stop hook if it is missing. The status indicator SHALL reflect the result.

#### Scenario: Hook already configured — status shown as ok
- **WHEN** `~/.claude/settings.json` contains a Stop hook pointing at `claude-notify notify`
- **THEN** the dashboard shows a green configured indicator

#### Scenario: Hook missing — auto-configured and status updated
- **WHEN** the Stop hook is absent or the file does not exist
- **THEN** the binary writes the hook and shows a configured indicator with a message describing what changed

#### Scenario: Malformed settings.json — error shown, file untouched
- **WHEN** `~/.claude/settings.json` cannot be parsed
- **THEN** the dashboard shows an error indicator and does not modify the file

### Requirement: Dashboard live-updates notifications via fsnotify
The dashboard SHALL watch the notification log directory via fsnotify and reload the entry list whenever `notifications.jsonl` changes, without requiring the user to restart.

#### Scenario: New notification appears while dashboard is open
- **WHEN** `claude-notify notify` appends a record to `notifications.jsonl`
- **THEN** the dashboard entry list updates immediately to include the new notification

#### Scenario: Notification cleared externally while dashboard is open
- **WHEN** a record in `notifications.jsonl` is marked cleared (e.g. by focusing the pane)
- **THEN** the dashboard removes that entry from the list immediately

#### Scenario: Cursor stays in bounds after reload
- **WHEN** the entry list shrinks due to a reload
- **AND** the cursor was on or past the last item
- **THEN** the cursor moves to the new last item without error

### Requirement: Dashboard watches settings.json via fsnotify
The dashboard SHALL watch the `~/.claude/` directory via fsnotify and re-run check-and-configure whenever `settings.json` changes, keeping the hook in place if an external editor removes it.

#### Scenario: Hook removed externally while dashboard is open
- **WHEN** `settings.json` is modified to remove the Stop hook while the dashboard is running
- **THEN** the binary re-adds the hook and updates the status indicator

### Requirement: Watcher closed on exit
The fsnotify watcher SHALL be closed when the dashboard exits (on selection, quit key, or error) to release OS resources.

#### Scenario: Watcher closed on q/esc
- **WHEN** the user presses q or esc
- **THEN** the watcher is closed before the program exits

#### Scenario: Watcher closed on selection when list empties
- **WHEN** the user selects the last notification and the TUI exits
- **THEN** the watcher is closed before the program exits

#### Scenario: Watcher remains open while entries exist
- **WHEN** the user selects a notification but other entries remain
- **THEN** the watcher continues running so live-updates still fire
