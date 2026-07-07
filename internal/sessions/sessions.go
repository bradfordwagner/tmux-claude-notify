package sessions

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type SessionRecord struct {
	SessionID    string `json:"session_id"`
	EncodedPath  string `json:"encoded_path"`
	ProjectPath  string `json:"project_path"`
	Pinned       bool   `json:"pinned"`
	LastActivity int64  `json:"last_activity"`
	Status       string `json:"status"`
	PaneID       string `json:"pane_id"`
	WindowID     string `json:"window_id"`
	WindowName   string `json:"window_name"`
	TmuxSession  string `json:"tmux_session"`
}

func LogPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "tmux-claude-notify", "sessions.jsonl")
}

func claudeProjectsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "projects")
}

// Upsert creates or updates a session record by SessionID. The existing Pinned
// flag is always preserved — callers must use SetPinned to change it.
func Upsert(r SessionRecord) error {
	path := LogPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	records, err := readRaw()
	if err != nil {
		return err
	}
	found := false
	for i, existing := range records {
		if existing.SessionID == r.SessionID {
			r.Pinned = existing.Pinned // always preserve user's pin choice
			records[i] = r
			found = true
			break
		}
	}
	if !found {
		records = append(records, r)
	}
	return writeAll(path, records)
}

// ReadAll returns all session records sorted: pinned first, then newest-first by LastActivity.
func ReadAll() ([]SessionRecord, error) {
	records, err := readRaw()
	if err != nil {
		return nil, err
	}
	sortRecords(records)
	return records, nil
}

func sortRecords(records []SessionRecord) {
	sort.SliceStable(records, func(i, j int) bool {
		ri, rj := records[i], records[j]
		if ri.Pinned != rj.Pinned {
			return ri.Pinned
		}
		return ri.LastActivity > rj.LastActivity
	})
}

// SetPinned updates the Pinned flag of the matching session record.
func SetPinned(sessionID string, pinned bool) error {
	path := LogPath()
	records, err := readRaw()
	if err != nil {
		return err
	}
	found := false
	for i, r := range records {
		if r.SessionID == sessionID {
			records[i].Pinned = pinned
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	return writeAll(path, records)
}

// Compact removes unpinned inactive (empty PaneID) records older than maxAge.
func Compact(maxAge time.Duration) error {
	path := LogPath()
	records, err := readRaw()
	if err != nil {
		return err
	}
	cutoff := time.Now().Add(-maxAge).UnixNano()
	var kept []SessionRecord
	for _, r := range records {
		if r.Pinned || r.PaneID != "" || r.LastActivity > cutoff {
			kept = append(kept, r)
		}
	}
	if len(kept) == len(records) {
		return nil
	}
	return writeAll(path, kept)
}

// DiscoverAll scans ~/.claude/projects/*/*.jsonl and returns a SessionRecord for
// every session file found, with no ProjectPath (caller must call RecoverPath).
func DiscoverAll() ([]SessionRecord, error) {
	dir := claudeProjectsDir()
	projectDirs, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var records []SessionRecord
	for _, pd := range projectDirs {
		if !pd.IsDir() {
			continue
		}
		encodedPath := pd.Name()
		projectDir := filepath.Join(dir, encodedPath)
		files, err := os.ReadDir(projectDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
				continue
			}
			sessionID := strings.TrimSuffix(f.Name(), ".jsonl")
			var lastActivity int64
			if info, err := f.Info(); err == nil {
				lastActivity = info.ModTime().UnixNano()
			}
			records = append(records, SessionRecord{
				SessionID:    sessionID,
				EncodedPath:  encodedPath,
				LastActivity: lastActivity,
				Status:       "idle",
			})
		}
	}
	return records, nil
}

// RecoverPath resolves the real filesystem path from an encoded project directory name.
// Priority: (1) stored if non-empty, (2) BFS filesystem walk, (3) naive decode fallback.
func RecoverPath(encodedPath, stored string) string {
	if stored != "" {
		return stored
	}
	if !strings.HasPrefix(encodedPath, "-") {
		return ""
	}
	// Leading `-` was the initial `/` of the absolute path.
	segments := strings.Split(encodedPath[1:], "-")
	if path := walkRecover("/", segments, 0); path != "" {
		return path
	}
	// Fallback: treat every `-` as a `/`.
	return "/" + strings.Join(segments, "/")
}

// walkRecover attempts to reconstruct a real path by trying each `-`-separated
// segment as either a directory-name component (literal `-`) or a path separator.
func walkRecover(base string, segments []string, idx int) string {
	if idx == len(segments) {
		if _, err := os.Stat(base); err == nil {
			return base
		}
		return ""
	}
	// Try combining 1..N remaining segments as the next path component (handles
	// directory names that contain hyphens, e.g. "tmux-claude-notify").
	for n := 1; idx+n <= len(segments); n++ {
		component := strings.Join(segments[idx:idx+n], "-")
		candidate := filepath.Join(base, component)
		info, err := os.Stat(candidate)
		if err != nil || !info.IsDir() {
			continue
		}
		if result := walkRecover(candidate, segments, idx+n); result != "" {
			return result
		}
	}
	return ""
}

func readRaw() ([]SessionRecord, error) {
	path := LogPath()
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var records []SessionRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var r SessionRecord
		if err := json.Unmarshal(line, &r); err != nil {
			continue
		}
		records = append(records, r)
	}
	return records, scanner.Err()
}

func writeAll(path string, records []SessionRecord) error {
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
