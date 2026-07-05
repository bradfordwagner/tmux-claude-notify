## ADDED Requirements

### Requirement: Binary checks Stop hook configuration in settings.json
`claude-notify` SHALL read `~/.claude/settings.json` and determine whether a Stop hook pointing at the binary is configured.

#### Scenario: Hook present and pointing at claude-notify
- **WHEN** `~/.claude/settings.json` contains a Stop hook with a command that includes `claude-notify notify`
- **THEN** the binary reports hook status as configured

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

### Requirement: Binary automatically configures settings.json when hook is missing
When the Stop hook is not present, the binary SHALL add it to `~/.claude/settings.json` automatically, creating the file if it does not exist. The command written SHALL use the binary's resolved absolute path from `os.Executable()`.

#### Scenario: Hook added when settings.json exists but has no Stop hook
- **WHEN** `~/.claude/settings.json` exists and contains no Stop hook
- **THEN** the binary merges the Stop hook entry into the existing JSON and writes the file

#### Scenario: settings.json created when it does not exist
- **WHEN** `~/.claude/settings.json` does not exist
- **THEN** the binary creates it with the Stop hook as the only entry

#### Scenario: No duplicate entries added
- **WHEN** a Stop hook pointing at `claude-notify notify` already exists
- **THEN** the binary does not add a second entry

#### Scenario: Malformed settings.json is not overwritten
- **WHEN** `~/.claude/settings.json` exists but cannot be parsed
- **THEN** the binary reports an error and does not modify the file

### Requirement: Dashboard re-configures if settings.json is modified externally
The dashboard SHALL watch `~/.claude/settings.json` via fsnotify and re-run the check-and-configure logic whenever the file changes, so that a hook removed by an external editor is immediately restored.

#### Scenario: Hook removed while dashboard is open
- **WHEN** the Stop hook is removed from `~/.claude/settings.json` while the dashboard is running
- **THEN** the binary detects the change via fsnotify and re-adds the hook
- **AND** the dashboard status indicator updates to reflect the current state

#### Scenario: settings.json deleted while dashboard is open
- **WHEN** `~/.claude/settings.json` is deleted while the dashboard is running
- **THEN** the binary detects the deletion via fsnotify and recreates the file with the Stop hook
