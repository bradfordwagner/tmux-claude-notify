## ADDED Requirements

### Requirement: Transient toast notification
The TUI model SHALL support a transient toast message that displays for 10 seconds and then auto-dismisses without user interaction.

#### Scenario: Toast appears when set
- **WHEN** a toast message is set on the model
- **THEN** the toast text is rendered in the TUI output

#### Scenario: Toast auto-dismisses after 10 seconds
- **WHEN** 10 seconds have elapsed since the toast was set
- **THEN** the toast is cleared and no longer appears in the TUI output

#### Scenario: No toast in steady state
- **WHEN** no toast event has occurred
- **THEN** the toast area renders nothing and does not occupy a line in the output
