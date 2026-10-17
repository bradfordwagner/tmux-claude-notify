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
	// Point stdio at /dev/null rather than leaving fds 0-2 closed: with them
	// closed, the child's next open() (e.g. the JSONL store) lands on fd 1/2 and
	// any stray write or panic trace would corrupt that file.
	devnull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer devnull.Close()
	p, err := os.StartProcess(exe, append([]string{exe}, args...), &os.ProcAttr{
		Files: []*os.File{devnull, devnull, devnull},
		Sys:   &syscall.SysProcAttr{Setsid: true},
	})
	if err != nil {
		return err
	}
	// Release so the parent doesn't accumulate a zombie entry.
	return p.Release()
}
