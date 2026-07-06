## ADDED Requirements

### Requirement: Auto-reset delay is configurable via @claude-notify-active-reset-seconds
The TPM option `@claude-notify-active-reset-seconds` SHALL control how long (in seconds) to wait before auto-clearing a notification on the currently focused pane. The option is read by the Go binary at notify time via `tmux show-option -gqv`; no logic is added to the `.tmux` entry point.

#### Scenario: Option not set — binary uses default of 15 seconds
- **WHEN** `@claude-notify-active-reset-seconds` is absent from `tmux.conf`
- **THEN** `tmux show-option -gqv "@claude-notify-active-reset-seconds"` returns empty
- **AND** the binary defaults to 15 seconds

#### Scenario: Option set in tmux.conf — binary reads custom value
- **WHEN** `set -g @claude-notify-active-reset-seconds 30` is in `tmux.conf`
- **THEN** the binary reads `30` and uses that as the auto-reset delay

#### Scenario: Entry point stays thin — no new shell logic
- **WHEN** `tmux-claude-notify.tmux` runs
- **THEN** it does NOT read or validate `@claude-notify-active-reset-seconds`
- **AND** no logic related to auto-reset is added to the shell entry point
