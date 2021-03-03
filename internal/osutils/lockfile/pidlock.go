package lockfile

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/osutils/stacktrace"
)

// PidLock represents a lock file that can be used for exclusive access to
// resources that should be accessed by only one process at a time.
//
// The characteristics of the lock are:
// - Lockfiles are removed after use
// - Even if the lockfiles are not removed (because a process has been terminated prematurely), it is unlocked
// - On file-systems that support advisory locks via fcntl or LockFileEx, all file system operations are atomic
//
// Notes:
// - The implementation currently does not support a blocking wait operation that returns once the lock can be acquired. If required, it can be extended this way.
// - Storing the PID inside the lockfile was initially intended to be fall-back mechanism for file systems that do not support locking files.  This is probably unnecessary, but could be extended to communicate with the process currently holding the lock via its PID.
type PidLock struct {
	path   string
	file   *os.File
	locked bool
}

// NewPidLock creates a new PidLock that can be used to get exclusive access to resources between processes
func NewPidLock(path string) (pl *PidLock, err error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, errs.Wrap(err, "failed to open lock file %s", path)
	}
	return &PidLock{
		path: path,
		file: f,
	}, nil
}

// TryLock attempts to lock the created lock file. If the lock cannot be acquired, it returns false and an error.
func (pl *PidLock) TryLock() (locked bool, err error) {
	err = LockFile(pl.file)
	if err != nil {
		// if lock cannot be acquired it usually means that another process is holding the lock
		return false, NewAlreadyLockedError(err, pl.path, "cannot acquire exclusive lock")
	}

	// check if PID can be read and if so, if the process is running
	b := make([]byte, 100)
	n, err := pl.file.Read(b)
	if err != nil && err != io.EOF {
		return false, errs.Wrap(err, "failed to read PID from lockfile %s", pl.path)
	}
	if n > 0 {
		pid, err := strconv.ParseInt(string(b[:n]), 10, 64)
		if err != nil {
			return false, errs.Wrap(err, "failed to parse PID from lockfile %s", pl.path)
		}
		if PidExists(int(pid)) {
			err := fmt.Errorf("pid %d exists", pid)
			return false, NewAlreadyLockedError(err, pl.path, "pid parsed")
		}
	}

	// write PID into lock file
	_, err = pl.file.Write([]byte(fmt.Sprintf("%d", os.Getpid())))
	if err != nil {
		return false, errs.Wrap(err, "failed to write pid to lockfile %s", pl.path)
	}

	pl.locked = true
	return true, nil
}

// Close removes the lock file and releases the lock
func (pl *PidLock) Close(keepFile ...bool) error {
	keep := false
	if len(keepFile) == 1 {
		keep = keepFile[0]
	}
	if !pl.locked {
		err := pl.file.Close()
		if err != nil {
			return errs.Wrap(err, "failed to close unlocked lock file %s", pl.path)
		}
		return nil
	}
	err := pl.cleanLockFile(keep)
	if err != nil {
		return errs.Wrap(err, "failed to remove lock file")
	}
	return nil
}

// WaitForLock will attempt to acquire the lock for the duration given
func (pl *PidLock) WaitForLock(timeout time.Duration) error {
	expiration := time.Now().Add(timeout)
	for {
		_, err := pl.TryLock()
		if err != nil {
			if !errs.Matches(err, &AlreadyLockedError{}) {
				return errs.Wrap(err, "Could not acquire lock")
			}

			if time.Now().After(expiration) {
				return err
			}
			time.Sleep(100 * time.Millisecond)
		}
		return nil
	}
}

// AlreadyLockedError manages info that clarifies why a lock has failed, but
// is still likely valid.
type AlreadyLockedError struct {
	err   error
	file  string
	msg   string
	stack *stacktrace.Stacktrace
}

// NewAlreadyLockedError returns a new AlreadyLockedError.
func NewAlreadyLockedError(err error, file, msg string) *AlreadyLockedError {
	return &AlreadyLockedError{err, file, msg, stacktrace.Get()}
}

// Error implements the error interface.
func (e *AlreadyLockedError) Error() string {
	return fmt.Sprintf("file %q is already locked: %s", e.file, e.msg)
}

// Unwrap allows the unwrapping of a causing error.
func (e *AlreadyLockedError) Unwrap() error {
	return e.err
}

// Stack implements the errs.WrapperError interface.
func (e *AlreadyLockedError) Stack() *stacktrace.Stacktrace {
	return e.stack
}
