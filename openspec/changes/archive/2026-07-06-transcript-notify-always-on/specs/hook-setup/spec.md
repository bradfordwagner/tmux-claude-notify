## MODIFIED Requirements

### Requirement: Binary checks Stop hook configuration in settings.json
`claude-notify` SHALL read `~/.claude/settings.json` and determine whether a `Stop` hook pointing at the binary is configured. The Stop hook is now a supplementary signal alongside transcript-based detection; the dashboard status indicator SHALL reflect this role.

#### Scenario: Hook present and pointing at claude-notify
- **WHEN** `~/.claude/settings.json` contains a `Stop` hook with a command that includes `claude-notify notify`
- **THEN** the binary reports hook status as configured
- **AND** the dashboard shows `[Stop] hook configured (transcript watcher active)`

#### Scenario: Hook absent from settings.json — auto-configured
- **WHEN** `~/.claude/settings.json` exists but contains no Stop hook
- **THEN** the binary automatically adds the Stop hook, preserving all existing keys
- **AND** the dashboard shows a configured indicator with a message describing what was written

#### Scenario: settings.json missing — created automatically
- **WHEN** `~/.claude/settings.json` does not exist
- **THEN** the binary creates the file containing the Stop hook
- **AND** the dashboard shows a configured indicator with a message describing what was created

#### Scenario: settings.json is malformed JSON — not overwritten
- **WHEN** `~/.claude/settings.json` exists but cannot be parsed
- **THEN** the binary reports hook status as unknown and displays a parse error warning
- **AND** the file is not modified

### Requirement: Stop hook is supplementary to transcript detection
When the transcript watcher is active, the Stop hook SHALL still be registered but is not the primary source of notification state. A Stop hook firing for a pane that already has a `waiting` state from the transcript watcher SHALL be a no-op.

#### Scenario: Stop hook fires while transcript already shows waiting
- **WHEN** the transcript watcher has already set the pane state to `waiting`
- **AND** the Stop hook fires `claude-notify notify` for the same pane
- **THEN** `HasUnclearedPane` returns true and no duplicate entry is created

#### Scenario: Stop hook fires when transcript watcher missed the session
- **WHEN** no transcript file was found for the pane's project directory
- **AND** the Stop hook fires `claude-notify notify`
- **THEN** a new notification entry is created as in the existing flow
