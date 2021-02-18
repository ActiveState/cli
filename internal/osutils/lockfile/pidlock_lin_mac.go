// +build linux darwin

package lockfile

import (
	"os"
	"syscall"

	"github.com/ActiveState/cli/internal/errs"
)

// PidExists checks if a process with the given PID exists and is running
func PidExists(pid int) bool {
	// this always returns a process on linux...
	p, err := os.FindProcess(int(pid))
	if err != nil {
		return false
	}
	// ... so we have to send it a fake signal
	err = p.Signal(syscall.Signal(0))
	if err == nil {
		// process exists on success
		return true
	}
	if err.Error() == "os: process already finished" {
		return false
	}
	errno, ok := err.(syscall.Errno)
	if !ok {
		return false
	}
	switch errno {
	case syscall.ESRCH: // process does not exist if we get search failed error
		return false
	case syscall.EPERM:
		return true // process exists if we do not have permissions to send signal
	}
	return false
}

// LockFile tries to acquire a read lock on the file f
func LockFile(f *os.File) error {
	// attempting to obtain read lock on update file
	ft := &syscall.Flock_t{
		Whence: int16(os.SEEK_SET),
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
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
		Pid:    int32(os.Getpid()),
		Type:   syscall.F_UNLCK,
	}

	return syscall.FcntlFlock(f.Fd(), syscall.F_SETLK, ft)
}

func (pl *PidLock) cleanLockFile(keep bool) error {
	// On Linux we have to remove the file before removing the file lock to avoid race conditions.
	if !keep {
		err := os.Remove(pl.path)
		if err != nil {
			return errs.Wrap(err, "failed to remove lock file %s", pl.path)
		}
	}
	err := LockRelease(pl.file)
	if err != nil {
		return errs.Wrap(err, "failed to release lock on lock file %s", pl.path)
	}
	err = pl.file.Close()
	if err != nil {
		return errs.Wrap(err, "failed to close lock file %s", pl.path)
	}
	return nil
}
