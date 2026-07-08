package sessions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// encode mirrors Claude Code's own directory-name encoding: replace `/` and
// `.` with `-`.
func encode(path string) string {
	return strings.NewReplacer("/", "-", ".", "-").Replace(path)
}

func TestRecoverPath_StoredPathUsedDirectly(t *testing.T) {
	got := RecoverPath("-anything-not-checked", "/some/stored/path")
	if got != "/some/stored/path" {
		t.Fatalf("got %q, want stored path returned verbatim", got)
	}
}

func TestRecoverPath_HyphenDirectoryName(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "tmux-claude-notify")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	encoded := encode(real)
	got := RecoverPath(encoded, "")
	if got != real {
		t.Fatalf("got %q, want %q", got, real)
	}
}

func TestRecoverPath_DotDirectoryName(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "src", "bradfordwagner.src.zmk.config.keyball.44")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	encoded := encode(real)
	got := RecoverPath(encoded, "")
	if got != real {
		t.Fatalf("got %q, want %q", got, real)
	}
}

func TestRecoverPath_MixedDotAndHyphenDirectoryName(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "src", "foo.bar-baz.qux")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	encoded := encode(real)
	got := RecoverPath(encoded, "")
	if got != real {
		t.Fatalf("got %q, want %q", got, real)
	}
}

func TestRecoverPath_NoMatchFallsBackToDisplayPath(t *testing.T) {
	encoded := "-nonexistent-path-that-does-not-exist-anywhere"
	got := RecoverPath(encoded, "")
	want := "/nonexistent/path/that/does/not/exist/anywhere"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRecoverPath_EmptyWhenNoLeadingDash(t *testing.T) {
	got := RecoverPath("no-leading-dash", "")
	if got != "" {
		t.Fatalf("got %q, want empty string", got)
	}
}
