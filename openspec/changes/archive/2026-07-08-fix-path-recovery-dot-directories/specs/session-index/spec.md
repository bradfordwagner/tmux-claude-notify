## MODIFIED Requirements

### Requirement: Path recovery resolves real filesystem path from encoded directory name
The sessions package SHALL attempt to recover the real project path from the encoded directory name using a filesystem walk. Recovery SHALL use three strategies in priority order: (1) stored `project_path` from a previous active-pane discovery, (2) BFS filesystem walk replacing leading `-` with `/` and testing each `-`-separated span of segments as either a `/` separator, a literal `-`, or a literal `.` within a directory-name component, (3) display fallback replacing leading `-` with `/` with no filesystem verification.

#### Scenario: Stored path used directly
- **WHEN** a sessions.jsonl record already has a non-empty `project_path`
- **THEN** path recovery returns that path without any filesystem check

#### Scenario: Filesystem walk resolves ambiguous path
- **WHEN** `project_path` is empty and the encoded path is `-home-bw-foo-bar`
- **AND** `/home/bw/foo/bar` exists on the filesystem but `/home/bw/foo-bar` does not
- **THEN** path recovery returns `/home/bw/foo/bar`

#### Scenario: Filesystem walk resolves directory names containing literal dots
- **WHEN** `project_path` is empty and the encoded path is `-home-bw-src-foo-bar-baz`
- **AND** `/home/bw/src/foo.bar.baz` exists on the filesystem but neither `/home/bw/src/foo-bar-baz` nor `/home/bw/src/foo/bar/baz` exists
- **THEN** path recovery returns `/home/bw/src/foo.bar.baz`

#### Scenario: Filesystem walk falls back to display path
- **WHEN** `project_path` is empty and no filesystem path matching the encoding (via `/` separators, literal `-`, or literal `.`) can be verified
- **THEN** path recovery returns the encoded path with leading `-` replaced by `/` and no filesystem check
