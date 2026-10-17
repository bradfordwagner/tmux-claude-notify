package proc

import (
	"os"
	"syscall"
)

// SpawnSelf starts a detached copy of the current executable with args
// (Setsid, closed stdio) and returns without waiting. Used by hook paths that
// must return immediately while slow work continues in the background.
func SpawnSelf(args ...string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	p, err := os.StartProcess(exe, append([]string{exe}, args...), &os.ProcAttr{
		Files: []*os.File{nil, nil, nil},
		Sys:   &syscall.SysProcAttr{Setsid: true},
	})
	if err != nil {
		return err
	}
	// Release so the parent doesn't accumulate a zombie entry.
	return p.Release()
}
