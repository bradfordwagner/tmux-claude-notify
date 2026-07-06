## ADDED Requirements

### Requirement: Binary checks Stop and PreToolUse hook configuration in settings.json
`claude-notify` SHALL read `~/.claude/settings.json` and determine whether both a `Stop` hook and a `PreToolUse` hook pointing at the binary are configured. Both must be present for the status to report as configured.

#### Scenario: Both hooks present and pointing at claude-notify
- **WHEN** `~/.claude/settings.json` contains both a `Stop` hook and a `PreToolUse` hook with commands that include `claude-notify notify`
- **THEN** the binary reports hook status as configured
- **AND** the dashboard shows `[Stop,PreToolUse] hooks configured`

#### Scenario: Either hook absent from settings.json — auto-configured
- **WHEN** `~/.claude/settings.json` exists but is missing either `Stop` or `PreToolUse` hook
- **THEN** the binary automatically adds whichever hooks are missing, preserving all existing keys
- **AND** the dashboard shows a configured indicator with a message describing what was written

#### Scenario: settings.json missing — created automatically
- **WHEN** `~/.claude/settings.json` does not exist
- **THEN** the binary creates the file containing both hooks
- **AND** the dashboard shows a configured indicator with a message describing what was created

#### Scenario: settings.json is malformed JSON — not overwritten
- **WHEN** `~/.claude/settings.json` exists but cannot be parsed
- **THEN** the binary reports hook status as unknown and displays a parse error warning
- **AND** the file is not modified

### Requirement: Binary automatically configures settings.json when hooks are missing
When either `Stop` or `PreToolUse` hook is not present, the binary SHALL add it to `~/.claude/settings.json` automatically, creating the file if it does not exist. The command written SHALL use the binary's resolved absolute path from `os.Executable()`.

Hook events registered: `Stop`, `PreToolUse`. The `hookEvents` slice in `setup.go` is the single source of truth — adding a new event name there is sufficient to include it in both the check and configure paths.

#### Scenario: Hooks added when settings.json exists but missing hooks
- **WHEN** `~/.claude/settings.json` exists and is missing one or both hooks
- **THEN** the binary merges the missing hook entries into the existing JSON and writes the file

#### Scenario: settings.json created when it does not exist
- **WHEN** `~/.claude/settings.json` does not exist
- **THEN** the binary creates it with both hooks as the only entries

#### Scenario: No duplicate entries added
- **WHEN** a hook pointing at `claude-notify notify` already exists for a given event type
- **THEN** the binary does not add a second entry for that event type

#### Scenario: Malformed settings.json is not overwritten
- **WHEN** `~/.claude/settings.json` exists but cannot be parsed
- **THEN** the binary reports an error and does not modify the file

### Requirement: Dashboard re-configures if settings.json is modified externally
The dashboard SHALL watch `~/.claude/settings.json` via fsnotify and re-run the check-and-configure logic whenever the file changes, so that hooks removed by an external editor are immediately restored.

#### Scenario: Hook removed while dashboard is open
- **WHEN** either hook is removed from `~/.claude/settings.json` while the dashboard is running
- **THEN** the binary detects the change via fsnotify and re-adds the missing hook(s)
- **AND** the dashboard status indicator updates to reflect the current state

#### Scenario: settings.json deleted while dashboard is open
- **WHEN** `~/.claude/settings.json` is deleted while the dashboard is running
- **THEN** the binary detects the deletion via fsnotify and recreates the file with both hooks
