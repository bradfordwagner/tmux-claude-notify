## ADDED Requirements

### Requirement: Notifications view merges active sessions from session-index
The dashboard Notifications view SHALL load two record sets: (a) all uncleared records from `notifications.jsonl` (existing behavior), and (b) `sessions.jsonl` records whose `pane_id` is non-empty and whose `session_id` has no corresponding uncleared `notifications.jsonl` entry. The merged result is the full Notifications view entry list.

#### Scenario: Active pane with no notification entry appears in Notifications view
- **WHEN** a sessions.jsonl record has `pane_id: "%7"` and no uncleared notifications.jsonl entry exists for pane `%7`
- **THEN** a row for that session appears in the Notifications view

#### Scenario: Notification entry takes priority over sessions entry for same pane
- **WHEN** both a sessions.jsonl record and an uncleared notifications.jsonl entry exist for pane `%7`
- **THEN** only the notifications.jsonl entry is shown (sessions entry is suppressed)

#### Scenario: Closed session without notification not shown
- **WHEN** a sessions.jsonl record has `pane_id: ""` and `pinned: false`
- **AND** there is no uncleared notifications.jsonl entry for that session
- **THEN** no row for that session appears in the Notifications view

### Requirement: Pinned sessions appear at the top of the Notifications view
Sessions with `pinned: true` in sessions.jsonl SHALL appear at the top of the Notifications view entry list, above all uncleared notifications.jsonl entries and above all unpinned active sessions. Pinned entries SHALL be rendered identically to active session entries but with the pinned indicator, and their STATUS SHALL reflect the last known status from the session record. A pinned session SHALL appear in the Notifications view regardless of pane state.

#### Scenario: Pinned closed session in Notifications view
- **WHEN** a sessions.jsonl record has `pinned: true` and `pane_id: ""`
- **THEN** a row for that session appears in the Notifications view regardless of notification-log state

#### Scenario: Pinned active session not duplicated
- **WHEN** a sessions.jsonl record has `pinned: true` and `pane_id: "%9"`
- **AND** an uncleared notifications.jsonl entry also exists for pane `%9`
- **THEN** only one row appears (the notifications.jsonl entry, with pin indicator added)
