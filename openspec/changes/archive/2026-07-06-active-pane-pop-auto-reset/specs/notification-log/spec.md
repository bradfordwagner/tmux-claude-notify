## ADDED Requirements

### Requirement: Auto-reset clears the JSONL entry after timer fires
When the auto-reset subprocess fires and finds an uncleared entry for the pane, it SHALL call `store.ClearPane(paneID)` to remove the entry. This is identical to the manual dashboard-clear path; no new store API is needed.

#### Scenario: Auto-reset calls ClearPane — entry removed
- **WHEN** the auto-reset subprocess fires after the configured delay
- **AND** an uncleared entry for the pane exists in the JSONL store
- **THEN** `ClearPane` is called and the entry is removed from the file

#### Scenario: Auto-reset finds no entry — no store write
- **WHEN** the auto-reset subprocess fires
- **AND** no uncleared entry exists for the pane (already dismissed manually)
- **THEN** `ClearPane` is NOT called and the JSONL file is not modified
