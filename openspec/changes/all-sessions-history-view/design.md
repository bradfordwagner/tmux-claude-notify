## Context

The dashboard currently surfaces only Stop-hook-triggered notifications stored in `notifications.jsonl`. It is blind to: (a) sessions with an active tmux pane but no pending notification, (b) sessions that ended cleanly without a Stop-hook firing, and (c) all historical sessions stored in `~/.claude/projects/` but never associated with the current tmux server. Claude Code stores every session as `~/.claude/projects/<encoded-path>/<uuid>.jsonl`, giving us a rich data set that goes untapped.

## Goals / Non-Goals

**Goals:**
- Single persistent `sessions.jsonl` index tracking all discovered Claude sessions with real project path, status, and pin flag.
- New Sessions tab in the dashboard (Tab-toggle) listing all sessions across all projects with sort and resume.
- Active/idle panes (live tmux pane, no pending notification) visible in the Notifications view.
- Pinned sessions visible in both views, regardless of activity state.

**Non-Goals:**
- Background agent / fleet-view integration.
- Session content browsing, search, or summarization.
- Multi-machine or remote session sync.
- Changing the Stop-hook notification flow.

## Decisions

### 1. Separate `sessions.jsonl` from `notifications.jsonl`

Sessions and notifications have different lifecycles. Notifications are ephemeral (cleared on acknowledgement). Sessions persist indefinitely until compacted. Commingling them would break the `HasUnclearedPane` idempotency check used by the Stop hook and bloat existing records with fields unused in the notification path.

Alternative considered: Add `session_id` and `pinned` fields to `notifications.jsonl` records. Rejected — the Stop hook's idempotency and the JSONL compaction/rewrite logic assume a narrow record schema.

### 2. New `internal/sessions` package

A dedicated package owns `sessions.jsonl` CRUD, filesystem scan discovery, and path recovery. Both the transcript watcher (writes real paths) and the dashboard UI (reads for both views) import it. This prevents the sessions concern from spreading into `internal/store` or `internal/watcher`.

### 3. Tab-toggle view inside the existing bubbletea model

The dashboard bubbletea model gains a `view` field (Notifications | Sessions). `Tab` cycles it. Both views reuse the same model lifecycle; only the `View()` render path changes.

Alternative: Separate TUI binary or subcommand for sessions. Rejected — the popup is launched once and the keybinding context (DetachIfShpell, selection actions) must stay unified.

### 4. Path recovery strategy

The encoding `strings.NewReplacer("/", "-", ".", "-")` is lossy: both `/` and `.` become `-`. Three-tier recovery:

1. **Exact**: Real path stored in `sessions.jsonl` from active-pane discovery (watcher writes `pane_current_path`). No ambiguity.
2. **Filesystem walk**: For sessions never seen with a live pane, replace the leading `-` with `/` then BFS through path segments — at each `-`, try it as `/` and check if the resulting prefix exists on the filesystem. First full match wins.
3. **Display fallback**: Replace leading `-` with `/` and show the result with a `(unverified)` marker. Does not block any functionality.

Alternative: Query the Claude API for session metadata. Rejected — requires network, adds latency, creates an auth dependency in the UI.

### 5. Active/idle sessions in Notifications view

The watcher, when discovering a live pane, upserts a record in `sessions.jsonl`. The dashboard Notifications view loads two record sets: (a) all uncleared `notifications.jsonl` entries (existing), (b) `sessions.jsonl` entries whose `pane_id` matches a currently-live tmux pane and that have no corresponding uncleared `notifications.jsonl` entry.

Dedup key: pane ID. `notifications.jsonl` takes priority — if a stop-hook entry exists for a pane, the `sessions.jsonl` entry for that pane does not appear as a separate row.

### 6. Pinned sessions float across both views

Pinned sessions appear in the Notifications view even when idle (no stop-hook entry, pane may or may not be alive). They are rendered with a `📌` marker and sorted above unpinned entries within each status group.

## Risks / Trade-offs

- [Performance] Scanning all `~/.claude/projects/` on each dashboard open may be slow for large session histories. → Mitigation: limit initial discovery to files modified within 30 days; display known (already-indexed) sessions immediately and load new ones in a goroutine.
- [Path ambiguity] Filesystem BFS may find an incorrect path if two directories encode identically. → Mitigation: show `(unverified)` marker; the resume action confirms path before executing.
- [sessions.jsonl growth] Append-only log grows unboundedly. → Mitigation: compact on open — remove unpinned inactive records older than 90 days.

## Migration Plan

`sessions.jsonl` is new; no migration of existing data. On the first dashboard open after upgrade, the watcher populates it from active panes and the filesystem scan backfills closed sessions. No changes to `notifications.jsonl` format or Stop-hook behavior.
