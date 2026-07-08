## ADDED Requirements

### Requirement: Store read-modify-write operations are race-free under concurrent writers
`ClearPane` and `UpdateStatus` SHALL serialize their read-modify-write cycle against every other process performing a read-modify-write cycle on the same JSONL file, using a file lock held for the duration of the read, mutation, and atomic rename. A clear or status update committed by one process SHALL NOT be silently reverted by another process's concurrent read-modify-write cycle.

#### Scenario: Two concurrent ClearPane calls for different panes both survive
- **WHEN** `ClearPane("%1")` and `ClearPane("%2")` are invoked from two separate processes at approximately the same time, both against a store where `%1` and `%2` have uncleared entries
- **THEN** after both calls return, both `%1`'s and `%2`'s entries are marked cleared
- **AND** neither call's result is lost regardless of the order the two processes' file renames land in

#### Scenario: ClearPane blocks while another writer holds the lock
- **WHEN** `ClearPane` is invoked while another process is mid-way through its own read-modify-write cycle on the same file
- **THEN** the invocation waits for the lock rather than reading a stale snapshot
- **AND** proceeds with its read-modify-write cycle once the lock is released

### Requirement: Store exposes an atomic clear-oldest-uncleared operation
The store SHALL provide a function that finds the uncleared record with the smallest `ts` and marks it cleared as a single operation performed under one lock acquisition, so no other writer can observe or act on the intermediate state between "find oldest" and "mark cleared".

#### Scenario: Atomic clear-oldest returns the cleared record
- **WHEN** the atomic clear-oldest function is called against a store with multiple uncleared entries
- **THEN** it returns the record that had the smallest `ts` among uncleared entries
- **AND** that record is marked cleared in the persisted file before the function returns

#### Scenario: No uncleared entries — atomic clear-oldest returns nil
- **WHEN** the atomic clear-oldest function is called and no uncleared entries exist
- **THEN** it returns a nil record and does not modify the file
