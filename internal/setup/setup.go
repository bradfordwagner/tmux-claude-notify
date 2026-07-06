package setup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Status int

const (
	StatusConfigured Status = iota
	StatusNotConfigured
	StatusUnknown
)

type Result struct {
	Status  Status
	Message string
}

// hookCommand is a single executable hook entry.
type hookCommand struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// hookMatcher is the correct Claude Code hook schema:
// {"matcher": "<tool-or-empty>", "hooks": [{...}]}
type hookMatcher struct {
	Matcher string        `json:"matcher"`
	Hooks   []hookCommand `json:"hooks"`
}

// rawSettings preserves unknown keys when round-tripping the JSON file.
type rawSettings map[string]json.RawMessage

func SettingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

func Check() Result {
	path := SettingsPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Result{Status: StatusNotConfigured, Message: "~/.claude/settings.json not found"}
	}
	if err != nil {
		return Result{Status: StatusUnknown, Message: "cannot read ~/.claude/settings.json: " + err.Error()}
	}

	configured, parseErr := isConfigured(data)
	if parseErr != nil {
		return Result{Status: StatusUnknown, Message: "cannot parse ~/.claude/settings.json: " + parseErr.Error()}
	}
	if configured {
		return Result{Status: StatusConfigured, Message: "[" + strings.Join(hookEvents, ",") + "] hooks configured"}
	}
	return Result{Status: StatusNotConfigured}
}

// hookEvents lists all event types this plugin registers.
var hookEvents = []string{"Stop", "PreToolUse"}

// Configure adds Stop and PreToolUse hooks to settings.json if missing,
// creating the file if it does not exist. Returns a message describing what happened.
func Configure() (string, error) {
	binary, _ := os.Executable()
	path := SettingsPath()

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	var raw rawSettings
	if os.IsNotExist(err) || len(data) == 0 {
		raw = make(rawSettings)
	} else {
		if err := json.Unmarshal(data, &raw); err != nil {
			return "", err
		}
		if ok, _ := isConfigured(data); ok {
			return "hooks already configured", nil
		}
	}

	newMatcher := hookMatcher{
		Matcher: "",
		Hooks:   []hookCommand{{Type: "command", Command: binary + " notify"}},
	}

	// Merge with any other hook event types already present.
	mergedHooks := map[string]json.RawMessage{}
	if hooksRaw, ok := raw["hooks"]; ok {
		var existingMap map[string]json.RawMessage
		if err := json.Unmarshal(hooksRaw, &existingMap); err == nil {
			for k, v := range existingMap {
				mergedHooks[k] = v
			}
		}
	}

	// Add each hook event if not already present.
	var hooksMap map[string][]hookMatcher
	if hooksRaw, ok := raw["hooks"]; ok {
		_ = json.Unmarshal(hooksRaw, &hooksMap)
	}
	if hooksMap == nil {
		hooksMap = make(map[string][]hookMatcher)
	}
	for _, event := range hookEvents {
		if !hasNotifyHook(hooksMap[event]) {
			hooksMap[event] = append(hooksMap[event], newMatcher)
		}
	}

	for _, event := range hookEvents {
		raw, err := json.Marshal(hooksMap[event])
		if err != nil {
			return "", err
		}
		mergedHooks[event] = raw
	}

	hooksRaw, err := json.Marshal(mergedHooks)
	if err != nil {
		return "", err
	}
	raw["hooks"] = hooksRaw

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		return "", err
	}

	action := "created"
	if len(data) > 0 {
		action = "updated"
	}
	return "hooks added — " + action + " ~/.claude/settings.json", nil
}

func isConfigured(data []byte) (bool, error) {
	var s struct {
		Hooks map[string][]hookMatcher `json:"hooks"`
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return false, err
	}
	for _, event := range hookEvents {
		if !hasNotifyHook(s.Hooks[event]) {
			return false, nil
		}
	}
	return true, nil
}

func hasNotifyHook(matchers []hookMatcher) bool {
	for _, m := range matchers {
		for _, cmd := range m.Hooks {
			if strings.Contains(cmd.Command, "claude-notify notify") {
				return true
			}
		}
	}
	return false
}
