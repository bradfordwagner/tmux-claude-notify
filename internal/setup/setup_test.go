package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigure_AddsMissingUserPromptSubmitHook(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	executable = func() (string, error) { return "/x/bin/claude-notify", nil }
	defer func() { executable = os.Executable }()
	dir := filepath.Join(home, ".claude")
	_ = os.MkdirAll(dir, 0o755)
	existing := `{"hooks":{"Stop":[{"matcher":"","hooks":[{"type":"command","command":"/x/bin/claude-notify notify"}]}]}}`
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Check().Status; got != StatusNotConfigured {
		t.Fatalf("Check before = %v, want NotConfigured", got)
	}
	if _, err := Configure(); err != nil {
		t.Fatal(err)
	}
	if got := Check().Status; got != StatusConfigured {
		t.Fatalf("Check after = %v, want Configured", got)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	if n := strings.Count(string(data), "claude-notify notify"); n != 1 {
		t.Fatalf("Stop hook duplicated or lost: %d\n%s", n, data)
	}
	if !strings.Contains(string(data), "UserPromptSubmit") {
		t.Fatalf("UserPromptSubmit missing:\n%s", data)
	}
}
