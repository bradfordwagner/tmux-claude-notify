package store

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Record struct {
	TS         int64  `json:"ts"`
	Pane       string `json:"pane"`
	Window     string `json:"window"`
	WindowName string `json:"window_name"`
	Session    string `json:"session"`
	Cleared    bool   `json:"cleared"`
}

func NowNano() int64 {
	return time.Now().UnixNano()
}

func LogPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "tmux-claude-notify", "notifications.jsonl")
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
	path := LogPath()
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

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
	f.Close()
	if err := scanner.Err(); err != nil {
		return err
	}

	// Find most recent uncleared record for this pane (highest TS)
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
	records[bestIdx].Cleared = true

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
