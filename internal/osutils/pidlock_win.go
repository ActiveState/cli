// +build windows

package osutils

import (
	"fmt"
	"os"
	"syscall"
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
			return fmt.Errorf("cannot obtain lock")
		}

		return fmt.Errorf("unknown error: %d", errNo)
	}
	return nil
}

// LockRelease releases a file lock
func LockRelease(f *os.File) error {
	// mark the file as unlocked
	if _, errNo := unlockFileEx(syscall.Handle(f.Fd()), 0, 1, 0, &syscall.Overlapped{}); errNo > 0 {
		return fmt.Errorf("error releasing windows file lock. errno: %d", errNo)
	}
	return nil
}
