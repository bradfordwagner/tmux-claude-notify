package store

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"
)

type Record struct {
	TS         int64  `json:"ts"`
	Pane       string `json:"pane"`
	Window     string `json:"window"`
	WindowName string `json:"window_name"`
	Session    string `json:"session"`
	Cleared    bool   `json:"cleared"`
	Status     string `json:"status"` // "running", "waiting", "stale"; defaults to "waiting" if absent
}

func NowNano() int64 {
	return time.Now().UnixNano()
}

func LogPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "tmux-claude-notify", "notifications.jsonl")
}

// withStoreLock serializes read-modify-write cycles against the JSONL store
// across processes (Stop hook, auto-reset subprocesses, jump) via an exclusive
// flock on a sidecar lock file. The kernel releases the lock automatically if
// the holding process exits or crashes, so no explicit cleanup is needed.
func withStoreLock(fn func() error) error {
	path := LogPath() + ".lock"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return fn()
}

func Append(r Record) error {
	path := LogPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(r)
}

func ReadAll() ([]Record, error) {
	path := LogPath()
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var records []Record
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var r Record
		if err := json.Unmarshal(line, &r); err != nil {
			continue
		}
		if r.Status == "" {
			r.Status = "waiting"
		}
		records = append(records, r)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].TS > records[j].TS
	})
	return records, nil
}

// HasUnclearedPane returns true if there is already an uncleared notification
// for the given pane. Used by runNotify to avoid duplicate entries.
func HasUnclearedPane(paneID string) (bool, error) {
	records, err := ReadAll()
	if err != nil {
		return false, err
	}
	for _, r := range records {
		if !r.Cleared && r.Pane == paneID {
			return true, nil
		}
	}
	return false, nil
}

func ClearPane(paneID string) error {
	return withStoreLock(func() error {
		path := LogPath()
		records, err := readRecordsRaw(path)
		if err != nil {
			return err
		}

		// Clear all uncleared records for this pane. Multiple uncleared records can
		// accumulate from a race between the Stop hook subprocess and the watcher.
		found := false
		for i, r := range records {
			if r.Pane == paneID && !r.Cleared {
				records[i].Cleared = true
				found = true
			}
		}
		if !found {
			return nil
		}

		return writeRecords(path, records)
	})
}

// ClearOldestUncleared finds the uncleared record with the smallest TS and marks
// it cleared, as a single operation under one lock acquisition. This closes the
// gap between "find oldest" and "clear it" that a two-step
// OldestUncleared+ClearPane sequence would leave open to concurrent writers.
// Returns nil if no uncleared records exist.
func ClearOldestUncleared() (*Record, error) {
	var result *Record
	err := withStoreLock(func() error {
		path := LogPath()
		records, err := readRecordsRaw(path)
		if err != nil {
			return err
		}

		oldestIdx := -1
		for i, r := range records {
			if r.Cleared {
				continue
			}
			if oldestIdx == -1 || r.TS < records[oldestIdx].TS {
				oldestIdx = i
			}
		}
		if oldestIdx == -1 {
			return nil
		}

		records[oldestIdx].Cleared = true
		cleared := records[oldestIdx]
		result = &cleared
		return writeRecords(path, records)
	})
	return result, err
}

// readRecordsRaw reads records from path without defaulting Status or sorting
// (unlike ReadAll). Returns nil, nil if the file does not exist. Callers that
// mutate the result must hold withStoreLock.
func readRecordsRaw(path string) ([]Record, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var records []Record
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var r Record
		if err := json.Unmarshal(line, &r); err != nil {
			continue
		}
		records = append(records, r)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

// writeRecords atomically replaces the JSONL file at path with records.
// Callers are responsible for holding withStoreLock while calling this.
func writeRecords(path string, records []Record) error {
	tmp := path + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(out)
	for _, r := range records {
		if err := enc.Encode(r); err != nil {
			out.Close()
			os.Remove(tmp)
			return err
		}
	}
	out.Close()
	return os.Rename(tmp, path)
}

// WindowForPane returns the window ID from the most recent uncleared record for pane.
// Used by runClear to resolve the window ID even when the pane no longer exists in tmux.
func WindowForPane(paneID string) (string, error) {
	records, err := ReadAll()
	if err != nil {
		return "", err
	}
	for _, r := range records { // already sorted newest-first
		if !r.Cleared && r.Pane == paneID {
			return r.Window, nil
		}
	}
	return "", nil
}

// UnclearedForWindow returns all uncleared records whose Window field matches windowID.
// Used by runClear to gate window-level style teardown: only clear styles when the
// last uncleared entry for that window has been removed.
func UnclearedForWindow(windowID string) ([]Record, error) {
	records, err := ReadAll()
	if err != nil {
		return nil, err
	}
	var result []Record
	for _, r := range records {
		if !r.Cleared && r.Window == windowID {
			result = append(result, r)
		}
	}
	return result, nil
}

// OldestUncleared returns the uncleared record with the smallest TS (oldest waiting).
// Returns nil if no uncleared records exist.
func OldestUncleared() (*Record, error) {
	records, err := ReadAll()
	if err != nil {
		return nil, err
	}
	var oldest *Record
	for i, r := range records {
		if r.Cleared {
			continue
		}
		if oldest == nil || r.TS < oldest.TS {
			oldest = &records[i]
		}
	}
	return oldest, nil
}

// UpdateStatus updates the status field of the most recent uncleared record for
// paneID. Atomically rewrites the JSONL file. No-op if no uncleared record exists.
func UpdateStatus(paneID, status string) error {
	return withStoreLock(func() error {
		path := LogPath()
		records, err := readRecordsRaw(path)
		if err != nil {
			return err
		}

		bestIdx := -1
		for i, r := range records {
			if r.Pane == paneID && !r.Cleared {
				if bestIdx == -1 || records[i].TS > records[bestIdx].TS {
					bestIdx = i
				}
			}
		}
		if bestIdx == -1 {
			return nil
		}
		records[bestIdx].Status = status

		return writeRecords(path, records)
	})
}
