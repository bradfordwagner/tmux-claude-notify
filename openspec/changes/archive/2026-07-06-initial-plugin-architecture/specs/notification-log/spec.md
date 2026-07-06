## ADDED Requirements

### Requirement: Notifications persisted as JSON Lines
The binary SHALL append one JSON record per notification to `~/.local/share/tmux-claude-notify/notifications.jsonl`. All reads and writes to the log SHALL go through the binary — no direct file manipulation from bash.

#### Scenario: Record appended on notify
- **WHEN** `claude-notify notify` runs successfully
- **THEN** a JSON record is appended to the log with fields: `ts` (unix nanoseconds), `pane`, `window`, `window_name`, `session`, `cleared` (false)

#### Scenario: Record marked cleared on pane focus
- **WHEN** `claude-notify clear --pane <id>` runs
- **THEN** the most recent uncleared record for that pane has `cleared` set to true

#### Scenario: Log directory created if missing
- **WHEN** `~/.local/share/tmux-claude-notify/` does not exist
- **THEN** the binary creates it before writing the first record

#### Scenario: Log survives tmux restart
- **WHEN** tmux is killed and restarted
- **THEN** the notification log file persists on disk and is readable by the binary on next invocation

### Requirement: Dashboard reads log and cross-references live panes
The binary SHALL cross-reference log entries against `tmux list-panes -a` output to determine which notifications are still actionable.

#### Scenario: Pane no longer in tmux is excluded from dashboard
- **WHEN** a log entry references a pane ID not present in `tmux list-panes -a`
- **THEN** that entry is not shown in the dashboard

#### Scenario: Most recent notification appears first
- **WHEN** multiple uncleared notifications exist for live panes
- **THEN** they are sorted descending by `ts` so the newest is at the top of the list
