# Spec: desktop-notification

## Purpose

Optional desktop notification fired via `notify-send` when claude finishes a turn and returns to the prompt. Guards against missing binary; non-fatal.

## Requirements

### Requirement: Desktop notification fired on Stop hook
`claude-notify notify` SHALL call `notify-send` with a message indicating claude is waiting for input when invoked inside a tmux session.

#### Scenario: notify-send present and tmux context valid
- **WHEN** `claude-notify notify` is invoked with valid `$TMUX_PANE`
- **AND** `notify-send` is on PATH
- **THEN** a desktop notification is sent with a title and body identifying the window

#### Scenario: notify-send not available — silent skip
- **WHEN** `notify-send` is not on PATH
- **THEN** `claude-notify notify` skips the desktop notification and continues without error

#### Scenario: Notification identifies the tmux window
- **WHEN** the desktop notification is sent
- **THEN** the notification body SHALL include the window name so the user can locate it without switching to tmux
