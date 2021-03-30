// +build windows

package lockfile

import (
	"os"
	"syscall"

	"github.com/ActiveState/cli/internal/errs"
)

// ErrorLockViolation is the error no returned if a lock fails on Windows
const ErrorLockViolation syscall.Errno = 0x21 // 33

// PidExists checks if a process with the given PID exists and is running
func PidExists(pid int) bool {
	_, err := os.FindProcess(int(pid))
	return err == nil
}

// LockFile attempts to add a lock to the file f
func LockFile(f *os.File) error {
	_, errNo := lockFileEx(syscall.Handle(f.Fd()), winLockfileExclusiveLock|winLockfileFailImmediately, 0, 1, 0, &syscall.Overlapped{})

	if errNo > 0 {
		if errNo == ErrorLockViolation || errNo == syscall.ERROR_IO_PENDING {
			return errs.New("cannot obtain lock")
		}

		return errs.New("unknown error: %d", errNo)
	}
	return nil
}

// LockRelease releases a file lock
func LockRelease(f *os.File) error {
	// mark the file as unlocked
	if _, errNo := unlockFileEx(syscall.Handle(f.Fd()), 0, 1, 0, &syscall.Overlapped{}); errNo > 0 {
		return errs.New("error releasing windows file lock. errno: %d", errNo)
	}
	return nil
}

func (pl *PidLock) cleanLockFile(keep bool) error {
	err := LockRelease(pl.file)
	if err != nil {
		return errs.Wrap(err, "failed to release lock on lock file %s", pl.path)
	}
	err = pl.file.Close()
	if err != nil {
		return errs.Wrap(err, "failed to close lock file %s", pl.path)
	}
	// On Windows, we have to remove the file after it is closed.
	// It is not an error if it fails, because that could mean that another
	// process has acquired the lock for the file already.
	if !keep {
		_ = os.Remove(pl.path)
	}
	return nil
}
