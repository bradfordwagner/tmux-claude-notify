## Why

`sessions.RecoverPath()` guesses a project's real filesystem path from its Claude-encoded transcript directory name (which replaces both `/` and `.` with `-`) whenever no cached `project_path` is already stored. The guessing walk (`walkRecover`) only tries treating a `-` as a literal **hyphen** within a directory-name component; it never tries a literal **dot**. Any project whose directory name contains dots (very common — `foo.bar`, `go.bin`, versioned config dirs, etc.) and that has never had an active-pane discovery (so no cached path exists yet) silently resolves to a wrong, nonexistent path. Pressing `w`/`h`/`v` on such a project in the dashboard Sessions view then opens `claude` with a `-c <bad-path>` that doesn't exist, so tmux silently falls back to the default directory instead — the new window ends up in the wrong place with no error surfaced to the user.

## What Changes

- `walkRecover()` in `internal/sessions/sessions.go` tries a literal `.` (in addition to the existing literal `-`) when testing whether a run of encoded segments forms a single real directory-name component
- When both interpretations (dot vs hyphen vs separator) are filesystem-verifiable at a given branch, prefer whichever produces a path that exists, consistent with the existing "filesystem walk" strategy already documented for `RecoverPath`
- No change to the "stored path used directly" fast path (unaffected — this only touches the walk used when no cached path exists) or the final display-fallback (unaffected — still used only when the walk can't verify anything)

## Capabilities

### New Capabilities
<!-- none: this is a bug fix, not a new capability -->

### Modified Capabilities
- `session-index`: the "Filesystem walk resolves ambiguous path" requirement is extended so the walk also tests literal `.` characters within a directory-name component, not just literal `-`, so directory names containing dots (not previously cached) resolve correctly instead of falling through to the broken display fallback.

## Impact

- `internal/sessions/sessions.go` — `walkRecover()`: try an additional literal-dot interpretation per candidate component
- Affects `doNewSession`/`doResume` in `internal/ui/model.go` (dashboard `w`/`h`/`v`) and any other caller of `sessions.RecoverPath` for projects without a cached `project_path` and dotted directory names
- No JSONL schema, config, or keybinding changes
