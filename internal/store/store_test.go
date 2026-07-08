package store

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// setHome redirects LogPath() to a fresh temp directory for the duration of
// the test, so tests never touch the real ~/.local/share/tmux-claude-notify.
func setHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

func TestClearPane_ConcurrentDistinctPanes_NoLostUpdates(t *testing.T) {
	setHome(t)

	const n = 20
	for i := range n {
		if err := Append(Record{TS: int64(i), Pane: fmt.Sprintf("%%%d", i)}); err != nil {
			t.Fatalf("Append(%d): %v", i, err)
		}
	}

	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(pane string) {
			defer wg.Done()
			if err := ClearPane(pane); err != nil {
				t.Errorf("ClearPane(%s): %v", pane, err)
			}
		}(fmt.Sprintf("%%%d", i))
	}
	wg.Wait()

	records, err := ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	cleared := make(map[string]bool, len(records))
	for _, r := range records {
		cleared[r.Pane] = r.Cleared
	}
	for i := range n {
		pane := fmt.Sprintf("%%%d", i)
		if !cleared[pane] {
			t.Errorf("pane %s: expected cleared, got uncleared (lost update)", pane)
		}
	}
}

func TestClearOldestUncleared_OldestFirstNoRepeats(t *testing.T) {
	setHome(t)

	if err := Append(Record{TS: 300, Pane: "%C", Session: "s", Window: "@3"}); err != nil {
		t.Fatalf("Append %%C: %v", err)
	}
	if err := Append(Record{TS: 100, Pane: "%A", Session: "s", Window: "@1"}); err != nil {
		t.Fatalf("Append %%A: %v", err)
	}
	if err := Append(Record{TS: 200, Pane: "%B", Session: "s", Window: "@2"}); err != nil {
		t.Fatalf("Append %%B: %v", err)
	}

	want := []string{"%A", "%B", "%C"}
	seen := make(map[string]bool)
	for i, wantPane := range want {
		r, err := ClearOldestUncleared()
		if err != nil {
			t.Fatalf("call %d: ClearOldestUncleared: %v", i, err)
		}
		if r == nil {
			t.Fatalf("call %d: expected record for pane %s, got nil", i, wantPane)
		}
		if r.Pane != wantPane {
			t.Fatalf("call %d: got pane %s, want %s", i, r.Pane, wantPane)
		}
		if !r.Cleared {
			t.Fatalf("call %d: returned record for pane %s is not marked cleared", i, r.Pane)
		}
		if seen[r.Pane] {
			t.Fatalf("call %d: pane %s was already returned by an earlier call", i, r.Pane)
		}
		seen[r.Pane] = true
	}

	r, err := ClearOldestUncleared()
	if err != nil {
		t.Fatalf("4th call: unexpected error: %v", err)
	}
	if r != nil {
		t.Fatalf("4th call: expected nil (no uncleared entries remain), got %+v", r)
	}
}

// TestClearOldestUncleared_ConcurrentNoise reproduces the reported bug: while
// background goroutines simulate auto-reset subprocesses clearing OTHER panes
// concurrently, sequential ClearOldestUncleared calls (simulating repeated
// jump presses) must still visit each of the original waiting panes exactly
// once, in oldest-first order, with no reappearance caused by a lost update.
func TestClearOldestUncleared_ConcurrentNoise(t *testing.T) {
	setHome(t)

	targets := []string{"%A", "%B", "%C"}
	for i, pane := range targets {
		if err := Append(Record{TS: int64(i + 1), Pane: pane, Session: "s", Window: fmt.Sprintf("@%d", i+1)}); err != nil {
			t.Fatalf("Append %s: %v", pane, err)
		}
	}

	stop := make(chan struct{})
	var noiseWG sync.WaitGroup
	for i := range 4 {
		noiseWG.Add(1)
		go func(worker int) {
			defer noiseWG.Done()
			n := 0
			for {
				select {
				case <-stop:
					return
				default:
				}
				pane := fmt.Sprintf("%%noise-%d-%d", worker, n)
				_ = Append(Record{TS: time.Now().UnixNano(), Pane: pane, Session: "s", Window: "@noise"})
				_ = ClearPane(pane)
				n++
			}
		}(i)
	}

	got := make([]string, 0, len(targets))
	for range targets {
		r, err := ClearOldestUncleared()
		if err != nil {
			t.Fatalf("ClearOldestUncleared: %v", err)
		}
		if r == nil {
			t.Fatal("ClearOldestUncleared returned nil while target panes remained uncleared")
		}
		got = append(got, r.Pane)
	}

	close(stop)
	noiseWG.Wait()

	if len(got) != len(targets) {
		t.Fatalf("got %d results, want %d", len(got), len(targets))
	}
	for i, pane := range targets {
		if got[i] != pane {
			t.Errorf("call %d: got pane %s, want %s (full sequence: %v)", i, got[i], pane, got)
		}
	}
}
