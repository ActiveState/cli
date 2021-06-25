// +build linux darwin

package lockfile

import (
	"io"
	"os"
	"syscall"

	"github.com/ActiveState/cli/internal/errs"
)

// LockFile tries to acquire a read lock on the file f
func LockFile(f *os.File) error {
	ft := &syscall.Flock_t{
		Whence: io.SeekStart,
		Start:  0,
		Len:    0,
		Pid:    int32(os.Getpid()),
		Type:   syscall.F_WRLCK,
	}

	err := syscall.FcntlFlock(f.Fd(), syscall.F_SETLK, ft)
	if err != nil {
		return errs.Wrap(err, "failed to lock file")
	}
	return nil
}

func LockRelease(f *os.File) error {
	ft := &syscall.Flock_t{
		Whence: io.SeekStart,
		Start:  0,
		Len:    0,
		Pid:    int32(os.Getpid()),
		Type:   syscall.F_UNLCK,
	}

	return syscall.FcntlFlock(f.Fd(), syscall.F_SETLK, ft)
}
