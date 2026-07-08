## 1. Fix path recovery

- [x] 1.1 In `internal/sessions/sessions.go`, update `walkRecover` so that for a candidate span of `n` segments it tries every `-`/`.` boundary combination (not just the single all-hyphen join), `os.Stat`-verifying each candidate directory before recursing
- [x] 1.2 Keep the existing single-hyphen-join behavior working for directory names that contain literal hyphens (e.g. `tmux-claude-notify`) — it must remain one of the tried combinations
- [x] 1.3 Preserve the existing priority order and both surrounding fallbacks in `RecoverPath` (stored path first, display-fallback last) — only `walkRecover`'s internal candidate generation changes

## 2. Verify

- [x] 2.1 Add `internal/sessions/sessions_test.go` covering: hyphen-named directory (regression), dot-named directory (the bug), mixed dot/hyphen directory, and the no-match display-fallback case — using `t.TempDir()` to create real directories so `os.Stat` verification is exercised
- [x] 2.2 `go build ./...`, `go vet ./...`, and `go test ./...` pass
- [x] 2.3 Manually verify: in the dashboard Sessions view, drill to (or select at L1) a project whose real directory name contains dots and has no cached `project_path` in `sessions.jsonl`, press `w`, confirm the new `claude` window opens with the correct working directory — verified both directly via `sessions.RecoverPath` and live in the dashboard: user confirmed `w` on the `keyball/44` project now opens the correct working directory

## 3. Close out

- [x] 3.1 Confirm no architecture.md diagram update is needed (internal path-recovery heuristic fix only, no data-flow/component-boundary/hook-wiring change)
- [ ] 3.2 Run `/opsx:archive` once merged
