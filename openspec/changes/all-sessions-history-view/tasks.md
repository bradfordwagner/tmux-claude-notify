## 1. Session Index Package

- [x] 1.1 Create `internal/sessions/sessions.go` with `SessionRecord` struct (`session_id`, `encoded_path`, `project_path`, `pinned`, `last_activity`, `status`, `pane_id`, `window_id`, `window_name`, `tmux_session`)
- [x] 1.2 Implement `Upsert(record SessionRecord) error` — find-and-replace by `session_id`, atomic rewrite to `.tmp` then rename
- [x] 1.3 Implement `ReadAll() ([]SessionRecord, error)` — read all records, sort pinned-first then newest-first by `last_activity`
- [x] 1.4 Implement `SetPinned(sessionID string, pinned bool) error` — update `pinned` field by `session_id`, atomic rewrite
- [x] 1.5 Implement `Compact(maxAge time.Duration) error` — remove unpinned + inactive (`pane_id == ""`) records older than `maxAge`
- [x] 1.6 Implement `DiscoverAll() ([]SessionRecord, error)` — scan `~/.claude/projects/*/` for all `*.jsonl` files, extract session UUID from filename and encoded path from parent directory name
- [x] 1.7 Implement `RecoverPath(encodedPath string, stored string) string` — three-tier recovery: (1) return `stored` if non-empty, (2) BFS filesystem walk replacing leading `-` with `/` and testing each `-` as `/` separator, (3) return encoded path with leading `-` replaced by `/`

## 2. Transcript Watcher — Session Index Integration

- [x] 2.1 In `listClaudePanes` / reconcile path, call `sessions.Upsert` for each discovered live pane with real `pane_current_path`, pane identifiers, and derived status
- [x] 2.2 When a previously-tracked pane is no longer returned by `tmux list-panes -a`, call `sessions.Upsert` with the same `session_id` but `pane_id`, `window_id`, `window_name`, `tmux_session` set to empty string

## 3. Dashboard — Tab Toggle Infrastructure

- [x] 3.1 Add `activeView` field to bubbletea `model` (type: enum `viewNotifications | viewSessions`); handle `Tab` key to toggle
- [x] 3.2 Render tab header bar showing both view names, active view highlighted in accent color (`#AD8EE6`), inactive dimmed

## 4. Sessions View Rendering

- [x] 4.1 On `viewSessions` activation, call `sessions.ReadAll()` and store in model; also call `sessions.DiscoverAll()` for new sessions not yet indexed (merge + dedup by `session_id`)
- [x] 4.2 Render Sessions view rows: STATUS icon + badge, PIN column (📌 or blank), PROJECT (last two path components of recovered path, `~`-abbreviated), SESSION (first 8 chars of UUID), AGE (time since `last_activity`)
- [x] 4.3 Render Sessions view header row: "STATUS / PIN / PROJECT / SESSION / AGE" with separator line; show empty state "No sessions discovered" when list is empty
- [x] 4.4 Add `sortField` to model (cycles: `sortAge → sortProject → sortStatus`); `s` key advances sort; display current sort in header (e.g. "sort: age")
- [x] 4.5 Apply sort to Sessions view entries, with pinned records always floating above unpinned within any sort order

## 5. Pin Feature

- [x] 5.1 Handle `p` key in Sessions view: call `sessions.SetPinned(selected.session_id, !selected.pinned)`; refresh session list immediately in model
- [x] 5.2 In Notifications view entry loading, include `sessions.ReadAll()` records where `pinned: true` and `pane_id: ""` (idle pinned sessions)
- [x] 5.3 Add PIN column to Notifications view header ("STATUS / PIN / WINDOW / PATH / SESSION / AGE"); render 📌 for entries backed by a pinned sessions record, blank space otherwise

## 6. Notifications View Merge

- [x] 6.1 Load sessions.jsonl active entries (non-empty `pane_id`) alongside uncleared notifications.jsonl entries into the Notifications view entry list
- [x] 6.2 Dedup merged list by pane ID: if both sources have an entry for the same pane, use the notifications.jsonl entry and suppress the sessions entry
- [x] 6.3 Ensure idle pinned sessions (from task 5.2) also follow the dedup rule: suppress if a notifications.jsonl entry exists for the same pane

## 7. Active-Pane Filter

- [x] 7.1 Add `filterActive bool` field to model; `f` key toggles it; filter applies to Sessions view entry list
- [x] 7.2 When filter is on, render only records with non-empty `pane_id` plus all pinned records (regardless of pane state)
- [x] 7.3 Show "filter: active panes" in Sessions view header alongside sort indicator when filter is on; omit indicator when off
- [x] 7.4 Show empty state "No active sessions" when filter is on and no matching (or pinned) records exist

## 8. Session Resume

- [x] 8.1 Handle `r` key in Sessions view: for entries with empty `pane_id`, call `sessions.RecoverPath` to get the project path, then execute `tmux neww -c <path> -- claude --resume <session_id>` via `exec.Command`
- [x] 8.2 After issuing `tmux neww`, call `DetachIfShpell` to close the popup
- [x] 8.3 For entries with non-empty `pane_id` (active session), `r` behaves like `enter`: call `tmux select-window -t <window_id>` then `DetachIfShpell`
- [x] 8.4 Show toast message "Cannot resume: project path unknown" if `RecoverPath` returns empty; show "Cannot resume: not in tmux" if `$TMUX` is unset; do not close popup on error

## 9. Housekeeping

- [x] 9.1 Call `sessions.Compact(90 * 24 * time.Hour)` on dashboard open (after initial data load)
- [x] 9.2 Update `architecture.md` to add `sessions.jsonl` data flow: watcher → sessions.Upsert, dashboard → sessions.ReadAll/DiscoverAll, Sessions view → session-resume
- [x] 9.3 Update `README.md` Configuration table if any new `@claude-notify-*` TPM options are introduced by this change

## 10. Group-by Project in Sessions View

- [x] 10.1 Update specs: `sessions-view/spec.md` — rewrite "Sessions view displays all discovered sessions" to group-by-project layout (📌 Pinned group first, then alpha-sorted project groups, STATUS/PIN/SESSION/AGE columns, no PROJECT column in rows)
- [x] 10.2 Update specs: `dashboard-row-layout/spec.md` — update column header to STATUS/PIN/SESSION/AGE; add "Sessions view renders project group headers" requirement (accent-colored header, `─` separator, blank line between groups)
- [x] 10.3 Reduce `sortField` enum from 3 to 2 values (remove `sortProject`); update `s` key handler to `% 2`; update `renderTabHeader` sort names to `["age", "status"]`
- [x] 10.4 Rewrite `loadSessionEntries` to produce grouped-flat order: pinned first (sorted by `sortBy`), then unpinned grouped by `projPath` (groups alpha-sorted, within-group sorted by `sortBy`)
- [x] 10.5 Rewrite `renderSessionsView` to detect group boundaries from the flat list and render accent-colored group headers with `─` separator and blank line between groups; rows show STATUS/PIN/SESSION/AGE (no PROJECT column)

## 11. Two-Level Drill-In Table for Sessions View

- [x] 11.1 Update specs: `sessions-view/spec.md` — rewrite grouped sessions requirement as two-level drill-in (Level-1 projects table, Level-2 sessions for project, `esc` back navigation)
- [x] 11.2 Update specs: `dashboard-row-layout/spec.md` — update Level-1 columns to STATUS/PROJECT/COUNT/LAST USED; update Level-2 columns to STATUS/PIN/SESSION/AGE with breadcrumb header
- [x] 11.3 Replace `collapsedGroups map[string]bool` with `drillProject string` in model struct; remove `isGroupHeader`/`groupKey` from `sessionEntry`
- [x] 11.4 Add `projRow` type and `buildProjRows(items []sessionEntry) []projRow` function (derives STATUS/COUNT/LAST USED per project from flat session list)
- [x] 11.5 Add `sessionsForDrill() []sessionEntry` and `sessionListLen() int` model methods; replace `reloadSessionItems`/`visibleSessionItems` with direct `loadSessionEntries` calls
- [x] 11.6 Update `esc` key: back to Level-1 when drilled in, quit otherwise; update `enter` key: drill in at Level-1, resume at Level-2; update `p`/`r` keys: only active at Level-2
- [x] 11.7 Rewrite `renderSessionsView`: Level-1 renders projects table (STATUS/PROJECT/COUNT/LAST USED); Level-2 renders breadcrumb + session rows (STATUS/PIN/SESSION/AGE)
- [x] 11.8 Update `renderTabHeader`: show "esc: back" hint when drilled in

## 12. bubbles/table Column Rendering for Alignment

- [x] 12.1 Replace `fmt.Sprintf` column layout in both views with `bubbles/table` component (already in dep tree as `github.com/charmbracelet/bubbles v1.0.0`); fix emoji-width misalignment using `github.com/mattn/go-runewidth`
- [x] 12.2 Add `notifTable`, `projTable`, `sessTable table.Model` fields to model; remove `cursor int` (table manages own cursor via `Cursor()`)
- [x] 12.3 Implement `makeTableStyles()`, `makeNotifTable()`, `makeProjTable()`, `makeSessTable()`, `buildNotifTableRows()`, `buildProjTableRows()`, `buildSessTableRows()` helpers
- [x] 12.4 Replace `renderStatusBadge` padding with `runewidth.StringWidth` so all status badges are exactly 12 visual columns regardless of emoji width
- [x] 12.5 Delegate navigation keys (j/k/up/down/g/G/pgup/pgdn/home/end) to active table via `table.Update(msg)`; use `table.Cursor()` for action keys (enter/r/p)
- [x] 12.6 Update `SetRows` on data change (watcher events, sort, filter, pin); recreate table via `makeProjTable`/`makeSessTable` on drill-in and tab switch
- [x] 12.7 PATH column in Notifications view truncated to 22 visual chars with `…` prefix using runewidth-aware trimming
