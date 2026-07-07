# Spec: session-index (delta)

## MODIFIED Requirements

### Requirement: Filesystem scan discovers all sessions across projects
The sessions package SHALL scan `~/.claude/projects/*/` on demand and enumerate all `*.jsonl` files modified within `@claude-notify-transcript-age-days` (default: 14 days). Files older than this cutoff are excluded from discovery. For each qualifying file, it SHALL extract the session UUID from the filename and the encoded project path from the parent directory name.

#### Scenario: All JSONL files within the age cutoff are discovered
- **WHEN** `DiscoverAll()` is called
- **THEN** every `*.jsonl` file under `~/.claude/projects/` modified within `@claude-notify-transcript-age-days` is enumerated
- **AND** a SessionRecord is returned for each unique session UUID

#### Scenario: Files older than the age cutoff are excluded
- **WHEN** a transcript file exists but was last modified more than `@claude-notify-transcript-age-days` ago
- **THEN** it is not included in the discovery result

#### Scenario: Non-JSONL files are ignored
- **WHEN** files other than `*.jsonl` exist under a project directory
- **THEN** they are not included in the discovery result

#### Scenario: Projects directory does not exist
- **WHEN** `~/.claude/projects/` does not exist
- **THEN** `DiscoverAll()` returns an empty slice without error

#### Scenario: Default age cutoff is 14 days
- **WHEN** `@claude-notify-transcript-age-days` is not set
- **THEN** only transcripts modified within the past 14 days are considered

#### Scenario: Custom age cutoff applied
- **WHEN** `@claude-notify-transcript-age-days` is set to a positive integer N
- **THEN** only transcripts modified within the past N days are considered
