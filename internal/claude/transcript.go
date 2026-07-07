package claude

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// TranscriptMaxAge reads @claude-notify-transcript-age-days from tmux options
// and returns it as a duration. Defaults to 14 days if unset or non-numeric.
func TranscriptMaxAge() time.Duration {
	out, err := exec.Command("tmux", "show-option", "-gqv", "@claude-notify-transcript-age-days").Output()
	if err == nil {
		if days, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil && days > 0 {
			return time.Duration(days) * 24 * time.Hour
		}
	}
	return 14 * 24 * time.Hour
}

// EncodeProjectPath maps a filesystem path to the directory name Claude Code
// uses in ~/.claude/projects/. Claude Code replaces both "/" and "." with "-".
func EncodeProjectPath(path string) string {
	return strings.NewReplacer("/", "-", ".", "-").Replace(path)
}

// LatestTranscriptPath returns the most recently modified .jsonl file in dir
// that was modified within TranscriptMaxAge().
func LatestTranscriptPath(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	cutoff := time.Now().Add(-TranscriptMaxAge())
	var latest string
	var latestMod time.Time
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil || info.ModTime().Before(cutoff) {
			continue
		}
		if info.ModTime().After(latestMod) {
			latestMod = info.ModTime()
			latest = filepath.Join(dir, e.Name())
		}
	}
	return latest
}

// LatestTranscriptID returns the session UUID (filename without .jsonl) from the
// most recently modified transcript in the project directory for paneCurrentPath.
func LatestTranscriptID(paneCurrentPath string) string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".claude", "projects", EncodeProjectPath(paneCurrentPath))
	path := LatestTranscriptPath(dir)
	if path == "" {
		return ""
	}
	return strings.TrimSuffix(filepath.Base(path), ".jsonl")
}
